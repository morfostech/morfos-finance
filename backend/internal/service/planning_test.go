package service

import (
	"errors"
	"testing"

	"github.com/morfostech/morfos-finance/internal/domain"
)

func planningDate(value string) *domain.Date {
	d := domain.MustDate(value)
	return &d
}

func TestBuildPlannedValidation(t *testing.T) {
	recurrence := domain.OrigemRecorrencia
	category := int64(2)
	tests := []struct {
		name  string
		input PlannedInput
		valid bool
	}{
		{"despesa válida", PlannedInput{Tipo: domain.TxDespesa, Valor: 1000, DueDate: planningDate("2026-07-31"), Descricao: "Servidor", CategoryID: &category}, true},
		{"entrada válida", PlannedInput{Tipo: domain.TxGanho, Valor: 2000, DueDate: planningDate("2026-07-31"), Descricao: "Projeto"}, true},
		{"sem descrição", PlannedInput{Tipo: domain.TxDespesa, Valor: 1000, DueDate: planningDate("2026-07-31")}, false},
		{"sem vencimento", PlannedInput{Tipo: domain.TxDespesa, Valor: 1000, Descricao: "Servidor"}, false},
		{"entrada com categoria", PlannedInput{Tipo: domain.TxGanho, Valor: 1000, DueDate: planningDate("2026-07-31"), Descricao: "Projeto", CategoryID: &category}, false},
		{"recorrência sem projeto", PlannedInput{Tipo: domain.TxGanho, Valor: 1000, DueDate: planningDate("2026-07-31"), Descricao: "Mensalidade", Origem: &recurrence}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := buildPlanned(tc.input)
			if tc.valid && err != nil {
				t.Fatalf("erro inesperado: %v", err)
			}
			if !tc.valid && !errors.Is(err, domain.ErrValidation) {
				t.Fatalf("erro = %v, esperado validação", err)
			}
		})
	}
}

func TestAddMonthsClamped(t *testing.T) {
	start := domain.MustDate("2026-01-31")
	if got := addMonthsClamped(start, 1).Format("2006-01-02"); got != "2026-02-28" {
		t.Fatalf("fevereiro = %s", got)
	}
	if got := addMonthsClamped(start, 2).Format("2006-01-02"); got != "2026-03-31" {
		t.Fatalf("março = %s", got)
	}
}
