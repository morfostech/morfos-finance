package domain

import "time"

// TxType is the direction of a transaction.
type TxType string

const (
	TxGanho   TxType = "ganho"
	TxDespesa TxType = "despesa"
)

func (t TxType) Valid() bool { return t == TxGanho || t == TxDespesa }

// TxOrigem classifies the source of a ganho.
type TxOrigem string

const (
	OrigemImplementacao TxOrigem = "implementacao"
	OrigemRecorrencia   TxOrigem = "recorrencia"
	OrigemAvulso        TxOrigem = "avulso"
)

func (o TxOrigem) Valid() bool {
	switch o {
	case OrigemImplementacao, OrigemRecorrencia, OrigemAvulso:
		return true
	default:
		return false
	}
}

// Transaction is a single financial movement (ganho or despesa). Financial data
// is never hard-deleted: DeletedAt marks a soft delete.
type Transaction struct {
	ID         int64     `json:"id"`
	Tipo       TxType    `json:"tipo"`
	Valor      Money     `json:"valor"`
	Data       Date      `json:"data"`
	ProjectID  *int64    `json:"project_id,omitempty"`
	UserID     *int64    `json:"user_id,omitempty"`
	Origem     *TxOrigem `json:"origem,omitempty"`      // ganho only
	CategoryID *int64    `json:"category_id,omitempty"` // despesa only
	Descricao  *string   `json:"descricao,omitempty"`
	CreatedBy  int64     `json:"created_by"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// ExpenseCategory is an admin-managed label for despesas.
type ExpenseCategory struct {
	ID   int64  `json:"id"`
	Nome string `json:"nome"`
}

// TransactionFilter narrows a transaction listing. Nil fields are ignored.
// Soft-deleted rows are always excluded.
type TransactionFilter struct {
	From       *Date
	To         *Date
	Tipo       *TxType
	ProjectID  *int64
	UserID     *int64
	CategoryID *int64
	Origem     *TxOrigem
}
