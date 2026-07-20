//go:build integration

package repository

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/morfostech/morfos-finance/internal/domain"
	"github.com/morfostech/morfos-finance/internal/migrate"
)

func TestInstallmentPaymentFeedsDashboardAtomically(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is required")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := migrate.Up(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		TRUNCATE attachments, project_proposals, transactions, project_members,
			project_installments, projects, expense_categories, users
		RESTART IDENTITY CASCADE`); err != nil {
		t.Fatal(err)
	}

	var actorID, projectID, installmentID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (nome, email, senha_hash, role, must_change_password)
		VALUES ('Integration Admin', 'integration@example.com', 'unused', 'admin', false)
		RETURNING id`).Scan(&actorID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO projects (nome, valor_implementacao)
		VALUES ('Integrated Project', 1000.00)
		RETURNING id`).Scan(&projectID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO project_installments (project_id, tipo, valor)
		VALUES ($1, 'entrada', 500.00)
		RETURNING id`, projectID).Scan(&installmentID); err != nil {
		t.Fatal(err)
	}

	projects := NewProjectRepository(pool)
	dashboard := NewDashboardRepository(pool)
	txs := NewTransactionRepository(pool)
	paidAt := domain.MustDate("2026-07-20")

	if _, err := projects.SetInstallment(ctx, projectID, installmentID, 50000, &paidAt, actorID); err != nil {
		t.Fatal(err)
	}
	assertDashboardAmounts(t, ctx, dashboard, 50000, 50000, 0)

	var transactionID int64
	if err := pool.QueryRow(ctx, `
		SELECT id FROM transactions
		WHERE installment_id = $1 AND deleted_at IS NULL`, installmentID).Scan(&transactionID); err != nil {
		t.Fatal(err)
	}
	transaction, err := txs.GetByID(ctx, transactionID)
	if err != nil {
		t.Fatal(err)
	}
	if transaction.InstallmentID == nil || *transaction.InstallmentID != installmentID {
		t.Fatalf("installment_id = %v, want %d", transaction.InstallmentID, installmentID)
	}

	if _, err := projects.SetInstallment(ctx, projectID, installmentID, 50000, nil, actorID); err != nil {
		t.Fatal(err)
	}
	assertDashboardAmounts(t, ctx, dashboard, 0, 0, 50000)
	if _, err := txs.GetByID(ctx, transactionID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("soft-deleted transaction err = %v, want ErrNotFound", err)
	}

	repaidAt := domain.MustDate("2026-07-21")
	if _, err := projects.SetInstallment(ctx, projectID, installmentID, 50000, &repaidAt, actorID); err != nil {
		t.Fatal(err)
	}
	assertDashboardAmounts(t, ctx, dashboard, 50000, 50000, 0)
	var activeCount int
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM transactions
		WHERE installment_id = $1 AND deleted_at IS NULL`, installmentID).Scan(&activeCount); err != nil {
		t.Fatal(err)
	}
	if activeCount != 1 {
		t.Fatalf("active managed transactions = %d, want 1", activeCount)
	}
}

func assertDashboardAmounts(t *testing.T, ctx context.Context, dashboard *DashboardRepository, wantSaldo, wantReceived, wantPending domain.Money) {
	t.Helper()
	saldo, err := dashboard.SaldoEmCaixa(ctx)
	if err != nil {
		t.Fatal(err)
	}
	implementation, err := dashboard.ImplementacaoTotals(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if saldo != wantSaldo || implementation.Recebido != wantReceived || implementation.AReceber != wantPending {
		t.Fatalf("saldo/recebido/a_receber = %d/%d/%d, want %d/%d/%d",
			saldo, implementation.Recebido, implementation.AReceber,
			wantSaldo, wantReceived, wantPending)
	}
}
