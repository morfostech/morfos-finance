package repository

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/morfostech/morfos-finance/internal/domain"
	"github.com/morfostech/morfos-finance/internal/migrate"
)

func TestViaPermutaRepositoryIntegration(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is required")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err := migrate.Up(ctx, pool); err != nil {
		t.Fatal(err)
	}

	var userID int64
	email := fmt.Sprintf("vp-integration-%d@invalid.local", time.Now().UnixNano())
	err = pool.QueryRow(ctx, `
		INSERT INTO users (nome, email, senha_hash, role, must_change_password)
		VALUES ('VP Integration Test', $1, 'invalid-test-hash', 'admin', false)
		RETURNING id`, email).Scan(&userID)
	if err != nil {
		t.Fatal(err)
	}

	repo := NewViaPermutaRepository(pool)
	originalSettings, err := repo.GetSettings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM vp_transactions WHERE created_by = $1`, userID)
		_, _ = pool.Exec(ctx, `DELETE FROM vp_offers WHERE created_by = $1`, userID)
		_, _ = pool.Exec(ctx, `UPDATE vp_settings SET credit_limit = $1::numeric WHERE id = 1`, originalSettings.CreditLimit.Numeric())
		_, _ = pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	})

	if _, err := repo.UpdateSettings(ctx, 800000); err != nil {
		t.Fatal(err)
	}
	day := domain.MustDate("2026-07-22")
	createTransaction := func(tipo domain.VPTransactionType, status domain.VPTransactionStatus, value domain.Money) *domain.VPTransaction {
		t.Helper()
		created, err := repo.CreateTransaction(ctx, &domain.VPTransaction{
			Tipo: tipo, Status: status, Valor: value, Data: day,
			Permutante: "Associado teste", Oferta: "Serviço teste", CreatedBy: userID,
		})
		if err != nil {
			t.Fatal(err)
		}
		return created
	}

	sale := createTransaction(domain.VPVenda, domain.VPConcluida, 100000)
	purchase := createTransaction(domain.VPCompra, domain.VPConcluida, 25000)
	createTransaction(domain.VPVenda, domain.VPNegociando, 12500)

	summary, err := repo.Summary(ctx, domain.VPTransactionFilter{From: &day, To: &day})
	if err != nil {
		t.Fatal(err)
	}
	if summary.Saldo != 75000 || summary.Disponivel != 875000 || summary.VendasPeriodo != 100000 || summary.ComprasPeriodo != 25000 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	if summary.NegociacoesAbertas != 1 || summary.TicketMedioVenda != 100000 || summary.TicketMedioCompra != 25000 {
		t.Fatalf("unexpected counters/averages: %+v", summary)
	}

	tipo := domain.VPVenda
	items, err := repo.ListTransactions(ctx, domain.VPTransactionFilter{Tipo: &tipo})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("sale filter returned %d items, want 2", len(items))
	}

	sale.Oferta = "Serviço atualizado"
	updated, err := repo.UpdateTransaction(ctx, sale)
	if err != nil || updated.Oferta != "Serviço atualizado" {
		t.Fatalf("update transaction = %+v, err = %v", updated, err)
	}
	if err := repo.DeleteTransaction(ctx, purchase.ID); err != nil {
		t.Fatal(err)
	}

	amount := domain.Money(70000)
	offer, err := repo.CreateOffer(ctx, &domain.VPOffer{
		Titulo: "Oferta integração", Valor: &amount, Status: domain.VPOfferAberta, CreatedBy: userID,
	})
	if err != nil {
		t.Fatal(err)
	}
	offers, err := repo.ListOffers(ctx)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range offers {
		if item.ID == offer.ID && item.Valor != nil && *item.Valor == amount {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("created offer %d not found", offer.ID)
	}
	if err := repo.DeleteOffer(ctx, offer.ID); err != nil {
		t.Fatal(err)
	}
}
