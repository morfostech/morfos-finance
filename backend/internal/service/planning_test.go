package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/morfostech/morfos-finance/internal/domain"
)

type forecastPlanningRepo struct{}

func (forecastPlanningRepo) CreateMany(context.Context, []domain.PlannedEntry) ([]domain.PlannedEntry, error) {
	return nil, nil
}
func (forecastPlanningRepo) GetByID(context.Context, int64) (*domain.PlannedEntry, error) {
	return nil, domain.ErrNotFound
}
func (forecastPlanningRepo) Update(context.Context, *domain.PlannedEntry) (*domain.PlannedEntry, error) {
	return nil, nil
}
func (forecastPlanningRepo) SoftDelete(context.Context, int64) error { return nil }
func (forecastPlanningRepo) List(context.Context, domain.PlanningFilter) ([]domain.PlannedEntry, error) {
	return []domain.PlannedEntry{}, nil
}
func (forecastPlanningRepo) Complete(context.Context, int64, int64, domain.Date) (*domain.PlannedEntry, error) {
	return nil, nil
}
func (forecastPlanningRepo) OpeningBalance(context.Context, time.Time) (domain.Money, error) {
	return 135000, nil
}
func (forecastPlanningRepo) ConfirmedTransactions(context.Context, time.Time, time.Time) ([]domain.CashFlowMovement, error) {
	return []domain.CashFlowMovement{}, nil
}
func (forecastPlanningRepo) CountOverdue(context.Context, time.Time) (int, error) { return 0, nil }
func (forecastPlanningRepo) UpsertBudget(context.Context, int64, int, int, domain.Money, int64) (*domain.ExpenseBudget, error) {
	return nil, nil
}
func (forecastPlanningRepo) ListBudgets(context.Context, int, int) ([]domain.ExpenseBudget, error) {
	return nil, nil
}
func (forecastPlanningRepo) DeleteBudget(context.Context, int64) error { return nil }

type forecastRecurrenceRepo struct{}

func (forecastRecurrenceRepo) MonthRows(context.Context, time.Time, time.Time, *int64) ([]domain.RecurrenceRow, error) {
	start := domain.MustDate("2026-08-01")
	due := 10
	return []domain.RecurrenceRow{{
		ProjectID: 1, Nome: "Contract", ValorMensal: 85000,
		DiaVencimento: &due, DataInicio: &start, Status: domain.StatusAtivo,
	}}, nil
}

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

func TestForecastIncludesAutomaticRecurringIncomeByDueDate(t *testing.T) {
	svc := NewPlanningService(forecastPlanningRepo{}, NewRecurrenceService(forecastRecurrenceRepo{}))
	from := domain.MustDate("2026-07-22")
	to := domain.MustDate("2026-10-22")

	forecast, err := svc.Forecast(context.Background(), from, to)
	if err != nil {
		t.Fatal(err)
	}
	if forecast.EntradasAutomaticas != 255000 || forecast.Entradas != 255000 {
		t.Fatalf("automatic/total income = %d/%d, want 255000", forecast.EntradasAutomaticas, forecast.Entradas)
	}
	if forecast.SaldoFinal != 390000 {
		t.Fatalf("final balance = %d, want 390000", forecast.SaldoFinal)
	}
	if len(forecast.Dias) != 3 {
		t.Fatalf("forecast days = %d, want 3", len(forecast.Dias))
	}
	if got := forecast.Dias[0].Data.Format("2006-01-02"); got != "2026-08-10" {
		t.Fatalf("first automatic due date = %s", got)
	}
	if len(forecast.Dias[0].Itens) != 1 || !forecast.Dias[0].Itens[0].Automatico {
		t.Fatalf("automatic forecast detail missing: %+v", forecast.Dias[0].Itens)
	}
}
