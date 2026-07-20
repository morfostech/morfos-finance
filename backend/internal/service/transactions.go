package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/morfostech/morfos-finance/internal/domain"
)

// TransactionRepo is the persistence contract the transaction service needs.
type TransactionRepo interface {
	Create(ctx context.Context, t *domain.Transaction) (*domain.Transaction, error)
	GetByID(ctx context.Context, id int64) (*domain.Transaction, error)
	Update(ctx context.Context, t *domain.Transaction) (*domain.Transaction, error)
	SoftDelete(ctx context.Context, id int64) error
	List(ctx context.Context, f domain.TransactionFilter) ([]domain.Transaction, error)
}

type TransactionService struct {
	txs TransactionRepo
}

func NewTransactionService(txs TransactionRepo) *TransactionService {
	return &TransactionService{txs: txs}
}

// TransactionInput is the create/update payload (full desired state).
type TransactionInput struct {
	Tipo       domain.TxType
	Valor      domain.Money
	Data       *domain.Date
	ProjectID  *int64
	UserID     *int64
	Origem     *domain.TxOrigem
	CategoryID *int64
	Descricao  *string
}

// Create validates and persists a transaction, stamping created_by.
func (s *TransactionService) Create(ctx context.Context, in TransactionInput, createdBy int64) (*domain.Transaction, error) {
	t, err := buildTransaction(in)
	if err != nil {
		return nil, err
	}
	t.CreatedBy = createdBy
	return s.txs.Create(ctx, t)
}

// Update replaces the editable fields of an existing (non-deleted) transaction.
// created_by is preserved.
func (s *TransactionService) Update(ctx context.Context, id int64, in TransactionInput) (*domain.Transaction, error) {
	existing, err := s.txs.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing.InstallmentID != nil {
		return nil, domain.ErrManagedTransaction
	}
	t, err := buildTransaction(in)
	if err != nil {
		return nil, err
	}
	t.ID = id
	return s.txs.Update(ctx, t)
}

func (s *TransactionService) Delete(ctx context.Context, id int64) error {
	existing, err := s.txs.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if existing.InstallmentID != nil {
		return domain.ErrManagedTransaction
	}
	return s.txs.SoftDelete(ctx, id)
}

// Get returns a transaction. A colaborador may only read their own.
func (s *TransactionService) Get(ctx context.Context, id int64, viewer Viewer) (*domain.Transaction, error) {
	t, err := s.txs.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if viewer.Role == domain.RoleColaborador && !ownedBy(t, viewer.UserID) {
		return nil, domain.ErrForbidden
	}
	return t, nil
}

// List applies the filter, forcing a colaborador's scope to their own rows
// regardless of any user_id passed in.
func (s *TransactionService) List(ctx context.Context, f domain.TransactionFilter, viewer Viewer) ([]domain.Transaction, error) {
	if viewer.Role == domain.RoleColaborador {
		uid := viewer.UserID
		f.UserID = &uid
	}
	return s.txs.List(ctx, f)
}

// buildTransaction validates input and constructs a Transaction. Enforces the
// type-specific rules mirrored by the DB CHECK constraints.
func buildTransaction(in TransactionInput) (*domain.Transaction, error) {
	if !in.Tipo.Valid() {
		return nil, fmt.Errorf("%w: tipo inválido", domain.ErrValidation)
	}
	if in.Valor <= 0 {
		return nil, fmt.Errorf("%w: valor deve ser positivo", domain.ErrValidation)
	}
	if in.Data == nil {
		return nil, fmt.Errorf("%w: data é obrigatória", domain.ErrValidation)
	}

	switch in.Tipo {
	case domain.TxGanho:
		if in.CategoryID != nil {
			return nil, fmt.Errorf("%w: ganho não tem categoria", domain.ErrValidation)
		}
		if in.Origem != nil && !in.Origem.Valid() {
			return nil, fmt.Errorf("%w: origem inválida", domain.ErrValidation)
		}
		if in.Origem != nil && *in.Origem == domain.OrigemImplementacao {
			return nil, fmt.Errorf("%w: ganhos de implementação são gerados pelo pagamento da parcela", domain.ErrValidation)
		}
		if in.Origem != nil && *in.Origem == domain.OrigemRecorrencia && in.ProjectID == nil {
			return nil, fmt.Errorf("%w: projeto é obrigatório para ganhos de recorrência", domain.ErrValidation)
		}
	case domain.TxDespesa:
		if in.Origem != nil {
			return nil, fmt.Errorf("%w: despesa não tem origem", domain.ErrValidation)
		}
	}

	return &domain.Transaction{
		Tipo:       in.Tipo,
		Valor:      in.Valor,
		Data:       *in.Data,
		ProjectID:  in.ProjectID,
		UserID:     in.UserID,
		Origem:     in.Origem,
		CategoryID: in.CategoryID,
		Descricao:  trimPtr(in.Descricao),
	}, nil
}

func ownedBy(t *domain.Transaction, userID int64) bool {
	return t.UserID != nil && *t.UserID == userID
}

// --- categories ---

// CategoryRepo is the persistence contract for expense categories.
type CategoryRepo interface {
	List(ctx context.Context) ([]domain.ExpenseCategory, error)
	Create(ctx context.Context, nome string) (*domain.ExpenseCategory, error)
	Delete(ctx context.Context, id int64) error
}

type CategoryService struct {
	cats CategoryRepo
}

func NewCategoryService(cats CategoryRepo) *CategoryService {
	return &CategoryService{cats: cats}
}

func (s *CategoryService) List(ctx context.Context) ([]domain.ExpenseCategory, error) {
	return s.cats.List(ctx)
}

func (s *CategoryService) Create(ctx context.Context, nome string) (*domain.ExpenseCategory, error) {
	nome = strings.TrimSpace(nome)
	if nome == "" {
		return nil, fmt.Errorf("%w: nome é obrigatório", domain.ErrValidation)
	}
	return s.cats.Create(ctx, nome)
}

func (s *CategoryService) Delete(ctx context.Context, id int64) error {
	return s.cats.Delete(ctx, id)
}
