package domain

import "time"

type PlannedStatus string

const (
	PlannedOpen      PlannedStatus = "aberto"
	PlannedCompleted PlannedStatus = "realizado"
)

type PlannedEntry struct {
	ID                  int64     `json:"id"`
	Tipo                TxType    `json:"tipo"`
	Valor               Money     `json:"valor"`
	DueDate             Date      `json:"due_date"`
	ProjectID           *int64    `json:"project_id,omitempty"`
	UserID              *int64    `json:"user_id,omitempty"`
	Origem              *TxOrigem `json:"origem,omitempty"`
	CategoryID          *int64    `json:"category_id,omitempty"`
	Descricao           string    `json:"descricao"`
	ActualTransactionID *int64    `json:"actual_transaction_id,omitempty"`
	CreatedBy           int64     `json:"created_by"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

func (p PlannedEntry) Status() PlannedStatus {
	if p.ActualTransactionID != nil {
		return PlannedCompleted
	}
	return PlannedOpen
}

type PlannedEntryView struct {
	PlannedEntry
	Status  PlannedStatus `json:"status"`
	Overdue bool          `json:"overdue"`
}

type PlanningFilter struct {
	From   *Date
	To     *Date
	Status *PlannedStatus
}

type CashFlowDay struct {
	Data           Date           `json:"data"`
	Entradas       Money          `json:"entradas"`
	Saidas         Money          `json:"saidas"`
	SaldoProjetado Money          `json:"saldo_projetado"`
	Itens          []CashFlowItem `json:"itens"`
}

type CashFlowItem struct {
	Tipo       TxType    `json:"tipo"`
	Valor      Money     `json:"valor"`
	Descricao  string    `json:"descricao"`
	ProjectID  *int64    `json:"project_id,omitempty"`
	Origem     *TxOrigem `json:"origem,omitempty"`
	Automatico bool      `json:"automatico"`
	Confirmado bool      `json:"confirmado"`
}

type CashFlowMovement struct {
	Data Date         `json:"data"`
	Item CashFlowItem `json:"item"`
}

type CashFlowForecast struct {
	Periodo             Period        `json:"periodo"`
	SaldoInicial        Money         `json:"saldo_inicial"`
	Entradas            Money         `json:"entradas"`
	EntradasAutomaticas Money         `json:"entradas_automaticas"`
	EntradasManuais     Money         `json:"entradas_manuais"`
	EntradasConfirmadas Money         `json:"entradas_confirmadas"`
	Saidas              Money         `json:"saidas"`
	SaidasManuais       Money         `json:"saidas_manuais"`
	SaidasConfirmadas   Money         `json:"saidas_confirmadas"`
	SaldoFinal          Money         `json:"saldo_final"`
	Vencidos            int           `json:"vencidos"`
	Dias                []CashFlowDay `json:"dias"`
}

type ExpenseBudget struct {
	ID         int64  `json:"id"`
	CategoryID int64  `json:"category_id"`
	Category   string `json:"category"`
	Ano        int    `json:"ano"`
	Mes        int    `json:"mes"`
	Valor      Money  `json:"valor"`
	Realizado  Money  `json:"realizado"`
	Restante   Money  `json:"restante"`
	Percentual int    `json:"percentual"`
}
