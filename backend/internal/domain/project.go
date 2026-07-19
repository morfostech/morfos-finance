package domain

import "time"

// ProjectStatus is the lifecycle state of a project.
type ProjectStatus string

const (
	StatusAtivo     ProjectStatus = "ativo"
	StatusPausado   ProjectStatus = "pausado"
	StatusConcluido ProjectStatus = "concluido"
	StatusCancelado ProjectStatus = "cancelado"
)

func (s ProjectStatus) Valid() bool {
	switch s {
	case StatusAtivo, StatusPausado, StatusConcluido, StatusCancelado:
		return true
	default:
		return false
	}
}

// InstallmentType distinguishes the two implementation payments (50% / 50%).
type InstallmentType string

const (
	InstallmentEntrada     InstallmentType = "entrada"
	InstallmentFinalizacao InstallmentType = "finalizacao"
)

// Project is a unit of work with up to two revenue sources: a one-off
// implementation fee (paid in two installments) and/or a fixed monthly fee.
type Project struct {
	ID                 int64         `json:"id"`
	Nome               string        `json:"nome"`
	Cliente            *string       `json:"cliente,omitempty"`
	ValorImplementacao *Money        `json:"valor_implementacao,omitempty"`
	ValorMensal        *Money        `json:"valor_mensal,omitempty"`
	DiaVencimento      *int          `json:"dia_vencimento,omitempty"`
	DataInicio         *Date         `json:"data_inicio,omitempty"`
	DataFim            *Date         `json:"data_fim,omitempty"`
	Status             ProjectStatus `json:"status"`
	CreatedAt          time.Time     `json:"created_at"`
	UpdatedAt          time.Time     `json:"updated_at"`

	// Hydrated on GetByID; omitted on list.
	Installments []Installment `json:"installments,omitempty"`
	MemberIDs    []int64       `json:"member_ids,omitempty"`
}

// Installment is one of the two implementation payments.
type Installment struct {
	ID        int64           `json:"id"`
	ProjectID int64           `json:"project_id"`
	Tipo      InstallmentType `json:"tipo"`
	Valor     Money           `json:"valor"`
	PagoEm    *Date           `json:"pago_em,omitempty"`
	Pago      bool            `json:"pago"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// GenerateInstallments splits the implementation fee into entrada (50%,
// rounded down) and finalizacao (remainder), so the two always sum to total.
func GenerateInstallments(total Money) []Installment {
	entrada := total / 2
	return []Installment{
		{Tipo: InstallmentEntrada, Valor: entrada},
		{Tipo: InstallmentFinalizacao, Valor: total - entrada},
	}
}
