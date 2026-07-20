package service

import (
	"context"
	"errors"
	"testing"

	"github.com/morfostech/morfos-finance/internal/domain"
)

func origemPtr(o domain.TxOrigem) *domain.TxOrigem { return &o }
func i64(v int64) *int64                           { return &v }

func txDate() *domain.Date {
	d := domain.MustDate("2026-07-10")
	return &d
}

// fakeTxRepo records writes and serves canned reads.
type fakeTxRepo struct {
	store      map[int64]*domain.Transaction
	lastFilter domain.TransactionFilter
	nextID     int64
}

func newFakeTxRepo() *fakeTxRepo {
	return &fakeTxRepo{store: map[int64]*domain.Transaction{}, nextID: 1}
}

func (f *fakeTxRepo) Create(_ context.Context, t *domain.Transaction) (*domain.Transaction, error) {
	t.ID = f.nextID
	f.nextID++
	f.store[t.ID] = t
	return t, nil
}
func (f *fakeTxRepo) GetByID(_ context.Context, id int64) (*domain.Transaction, error) {
	t, ok := f.store[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return t, nil
}
func (f *fakeTxRepo) Update(_ context.Context, t *domain.Transaction) (*domain.Transaction, error) {
	f.store[t.ID] = t
	return t, nil
}
func (f *fakeTxRepo) SoftDelete(_ context.Context, id int64) error {
	delete(f.store, id)
	return nil
}
func (f *fakeTxRepo) List(_ context.Context, filter domain.TransactionFilter) ([]domain.Transaction, error) {
	f.lastFilter = filter
	return nil, nil
}

func TestBuildTransactionValidation(t *testing.T) {
	tests := []struct {
		name    string
		in      TransactionInput
		wantErr error
	}{
		{"ganho válido com origem", TransactionInput{Tipo: domain.TxGanho, Valor: 1000, Data: txDate(), Origem: origemPtr(domain.OrigemRecorrencia), ProjectID: i64(1)}, nil},
		{"despesa válida com categoria", TransactionInput{Tipo: domain.TxDespesa, Valor: 1000, Data: txDate(), CategoryID: i64(3)}, nil},
		{"tipo inválido", TransactionInput{Tipo: domain.TxType("x"), Valor: 1000, Data: txDate()}, domain.ErrValidation},
		{"valor zero", TransactionInput{Tipo: domain.TxGanho, Valor: 0, Data: txDate()}, domain.ErrValidation},
		{"valor negativo", TransactionInput{Tipo: domain.TxGanho, Valor: -5, Data: txDate()}, domain.ErrValidation},
		{"sem data", TransactionInput{Tipo: domain.TxGanho, Valor: 1000}, domain.ErrValidation},
		{"ganho com categoria", TransactionInput{Tipo: domain.TxGanho, Valor: 1000, Data: txDate(), CategoryID: i64(3)}, domain.ErrValidation},
		{"despesa com origem", TransactionInput{Tipo: domain.TxDespesa, Valor: 1000, Data: txDate(), Origem: origemPtr(domain.OrigemAvulso)}, domain.ErrValidation},
		{"origem inválida", TransactionInput{Tipo: domain.TxGanho, Valor: 1000, Data: txDate(), Origem: origemPtr(domain.TxOrigem("x"))}, domain.ErrValidation},
		{"recorrência sem projeto", TransactionInput{Tipo: domain.TxGanho, Valor: 1000, Data: txDate(), Origem: origemPtr(domain.OrigemRecorrencia)}, domain.ErrValidation},
		{"implementação manual", TransactionInput{Tipo: domain.TxGanho, Valor: 1000, Data: txDate(), Origem: origemPtr(domain.OrigemImplementacao), ProjectID: i64(1)}, domain.ErrValidation},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := NewTransactionService(newFakeTxRepo())
			_, err := svc.Create(context.Background(), tc.in, 1)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestManagedTransactionCannotBeChangedDirectly(t *testing.T) {
	repo := newFakeTxRepo()
	installmentID := int64(9)
	repo.store[1] = &domain.Transaction{ID: 1, InstallmentID: &installmentID}
	svc := NewTransactionService(repo)

	_, err := svc.Update(context.Background(), 1, TransactionInput{Tipo: domain.TxGanho, Valor: 1000, Data: txDate()})
	if !errors.Is(err, domain.ErrManagedTransaction) {
		t.Fatalf("update err = %v, want ErrManagedTransaction", err)
	}
	if err := svc.Delete(context.Background(), 1); !errors.Is(err, domain.ErrManagedTransaction) {
		t.Fatalf("delete err = %v, want ErrManagedTransaction", err)
	}
}

func TestCreateStampsCreatedBy(t *testing.T) {
	repo := newFakeTxRepo()
	svc := NewTransactionService(repo)
	tx, err := svc.Create(context.Background(),
		TransactionInput{Tipo: domain.TxGanho, Valor: 1000, Data: txDate()}, 42)
	if err != nil {
		t.Fatal(err)
	}
	if tx.CreatedBy != 42 {
		t.Errorf("created_by = %d, want 42", tx.CreatedBy)
	}
}

func TestListForcesColaboradorScope(t *testing.T) {
	repo := newFakeTxRepo()
	svc := NewTransactionService(repo)

	// Even if the caller passes another user_id, a colaborador is pinned to self.
	other := int64(99)
	_, err := svc.List(context.Background(),
		domain.TransactionFilter{UserID: &other},
		Viewer{UserID: 7, Role: domain.RoleColaborador})
	if err != nil {
		t.Fatal(err)
	}
	if repo.lastFilter.UserID == nil || *repo.lastFilter.UserID != 7 {
		t.Fatalf("colaborador scope não aplicado: %v", repo.lastFilter.UserID)
	}

	// Admin keeps the requested filter.
	_, _ = svc.List(context.Background(),
		domain.TransactionFilter{UserID: &other},
		Viewer{UserID: 1, Role: domain.RoleAdmin})
	if repo.lastFilter.UserID == nil || *repo.lastFilter.UserID != 99 {
		t.Fatalf("admin filter alterado indevidamente: %v", repo.lastFilter.UserID)
	}
}

func TestGetColaboradorOwnershipGuard(t *testing.T) {
	repo := newFakeTxRepo()
	svc := NewTransactionService(repo)
	ctx := context.Background()

	owner := int64(7)
	own, _ := svc.Create(ctx, TransactionInput{Tipo: domain.TxDespesa, Valor: 500, Data: txDate(), UserID: &owner}, 1)
	otherUser := int64(8)
	foreign, _ := svc.Create(ctx, TransactionInput{Tipo: domain.TxDespesa, Valor: 500, Data: txDate(), UserID: &otherUser}, 1)

	colaborador := Viewer{UserID: 7, Role: domain.RoleColaborador}

	if _, err := svc.Get(ctx, own.ID, colaborador); err != nil {
		t.Fatalf("colaborador deve ver a própria transação: %v", err)
	}
	if _, err := svc.Get(ctx, foreign.ID, colaborador); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("err = %v, want ErrForbidden", err)
	}
	// Admin sees any.
	if _, err := svc.Get(ctx, foreign.ID, Viewer{UserID: 1, Role: domain.RoleAdmin}); err != nil {
		t.Fatalf("admin deve ver qualquer transação: %v", err)
	}
}
