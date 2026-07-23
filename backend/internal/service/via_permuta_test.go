package service

import (
	"context"
	"errors"
	"testing"

	"github.com/morfostech/morfos-finance/internal/domain"
)

type fakeViaPermutaRepo struct {
	transactions map[int64]*domain.VPTransaction
	offers       map[int64]*domain.VPOffer
	limit        domain.Money
	nextID       int64
}

func newFakeViaPermutaRepo() *fakeViaPermutaRepo {
	return &fakeViaPermutaRepo{
		transactions: map[int64]*domain.VPTransaction{},
		offers:       map[int64]*domain.VPOffer{},
		nextID:       1,
	}
}

func (f *fakeViaPermutaRepo) CreateTransaction(_ context.Context, item *domain.VPTransaction) (*domain.VPTransaction, error) {
	item.ID = f.nextID
	f.nextID++
	f.transactions[item.ID] = item
	return item, nil
}
func (f *fakeViaPermutaRepo) GetTransaction(_ context.Context, id int64) (*domain.VPTransaction, error) {
	item, ok := f.transactions[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return item, nil
}
func (f *fakeViaPermutaRepo) UpdateTransaction(_ context.Context, item *domain.VPTransaction) (*domain.VPTransaction, error) {
	f.transactions[item.ID] = item
	return item, nil
}
func (f *fakeViaPermutaRepo) DeleteTransaction(_ context.Context, id int64) error {
	delete(f.transactions, id)
	return nil
}
func (f *fakeViaPermutaRepo) ListTransactions(_ context.Context, _ domain.VPTransactionFilter) ([]domain.VPTransaction, error) {
	return nil, nil
}
func (f *fakeViaPermutaRepo) GetSettings(_ context.Context) (*domain.VPSettings, error) {
	return &domain.VPSettings{CreditLimit: f.limit}, nil
}
func (f *fakeViaPermutaRepo) UpdateSettings(_ context.Context, limit domain.Money) (*domain.VPSettings, error) {
	f.limit = limit
	return &domain.VPSettings{CreditLimit: limit}, nil
}
func (f *fakeViaPermutaRepo) Summary(_ context.Context, _ domain.VPTransactionFilter) (*domain.VPSummary, error) {
	return &domain.VPSummary{LimiteCredito: f.limit}, nil
}
func (f *fakeViaPermutaRepo) CreateOffer(_ context.Context, offer *domain.VPOffer) (*domain.VPOffer, error) {
	offer.ID = f.nextID
	f.nextID++
	f.offers[offer.ID] = offer
	return offer, nil
}
func (f *fakeViaPermutaRepo) GetOffer(_ context.Context, id int64) (*domain.VPOffer, error) {
	offer, ok := f.offers[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return offer, nil
}
func (f *fakeViaPermutaRepo) UpdateOffer(_ context.Context, offer *domain.VPOffer) (*domain.VPOffer, error) {
	f.offers[offer.ID] = offer
	return offer, nil
}
func (f *fakeViaPermutaRepo) DeleteOffer(_ context.Context, id int64) error {
	delete(f.offers, id)
	return nil
}
func (f *fakeViaPermutaRepo) ListOffers(_ context.Context) ([]domain.VPOffer, error) {
	return nil, nil
}

func vpTestDate() *domain.Date {
	date := domain.MustDate("2026-07-22")
	return &date
}

func TestVPTransactionValidationAndCreator(t *testing.T) {
	repo := newFakeViaPermutaRepo()
	svc := NewViaPermutaService(repo)
	valid := VPTransactionInput{
		Tipo: domain.VPVenda, Status: domain.VPConcluida, Valor: 100000,
		Data: vpTestDate(), Permutante: "Cliente VP", Oferta: "Landing page",
	}
	created, err := svc.CreateTransaction(context.Background(), valid, 42)
	if err != nil {
		t.Fatal(err)
	}
	if created.CreatedBy != 42 {
		t.Fatalf("created_by = %d, want 42", created.CreatedBy)
	}

	tests := []VPTransactionInput{
		{Tipo: domain.VPTransactionType("x"), Status: domain.VPConcluida, Valor: 100, Data: vpTestDate(), Permutante: "A", Oferta: "B"},
		{Tipo: domain.VPCompra, Status: domain.VPTransactionStatus("x"), Valor: 100, Data: vpTestDate(), Permutante: "AB", Oferta: "CD"},
		{Tipo: domain.VPCompra, Status: domain.VPConcluida, Valor: 0, Data: vpTestDate(), Permutante: "AB", Oferta: "CD"},
		{Tipo: domain.VPCompra, Status: domain.VPConcluida, Valor: 100, Permutante: "AB", Oferta: "CD"},
		{Tipo: domain.VPCompra, Status: domain.VPConcluida, Valor: 100, Data: vpTestDate(), Permutante: " ", Oferta: "CD"},
		{Tipo: domain.VPCompra, Status: domain.VPConcluida, Valor: 100, Data: vpTestDate(), Permutante: "AB", Oferta: " "},
	}
	for index, input := range tests {
		if _, err := svc.CreateTransaction(context.Background(), input, 1); !errors.Is(err, domain.ErrValidation) {
			t.Errorf("case %d: err = %v, want validation", index, err)
		}
	}
}

func TestVPOfferAndLimitValidation(t *testing.T) {
	repo := newFakeViaPermutaRepo()
	svc := NewViaPermutaService(repo)
	amount := domain.Money(70000)
	offer, err := svc.CreateOffer(context.Background(), VPOfferInput{
		Titulo: "Tráfego pago", Valor: &amount, Status: domain.VPOfferAberta,
	}, 7)
	if err != nil {
		t.Fatal(err)
	}
	if offer.CreatedBy != 7 || offer.Valor == nil || *offer.Valor != amount {
		t.Fatalf("oferta criada incorretamente: %+v", offer)
	}

	_, err = svc.CreateOffer(context.Background(), VPOfferInput{
		Titulo: "Oferta sem valor", Status: domain.VPOfferAberta,
	}, 7)
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("oferta sem valor err = %v, want validation", err)
	}
	if _, err := svc.CreateOffer(context.Background(), VPOfferInput{
		Titulo: "Oferta negociável", Negociavel: true, Status: domain.VPOfferAberta,
	}, 7); err != nil {
		t.Fatalf("oferta negociável deve aceitar valor vazio: %v", err)
	}

	if _, err := svc.UpdateSettings(context.Background(), -1); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("limite negativo err = %v, want validation", err)
	}
	settings, err := svc.UpdateSettings(context.Background(), 800000)
	if err != nil || settings.CreditLimit != 800000 {
		t.Fatalf("settings = %+v, err = %v", settings, err)
	}
}
