package service

import (
	"context"
	"testing"
	"time"

	"github.com/morfostech/morfos-finance/internal/domain"
)

type periodRecurrenceRepo struct{}

func (periodRecurrenceRepo) MonthRows(_ context.Context, start, _ time.Time, _ *int64) ([]domain.RecurrenceRow, error) {
	inicio := domain.MustDate("2026-08-01")
	recebido := domain.Money(0)
	if start.Month() == time.August {
		recebido = 85000
	}
	return []domain.RecurrenceRow{{
		ProjectID: 1, Nome: "Contrato recorrente", ValorMensal: 85000,
		DataInicio: &inicio, Recebido: recebido,
	}}, nil
}

func TestRecurrencePeriodAccumulatesOnlyActiveMonths(t *testing.T) {
	svc := NewRecurrenceService(periodRecurrenceRepo{})
	period, err := svc.Period(
		context.Background(),
		time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, time.December, 31, 0, 0, 0, 0, time.UTC),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(period.Meses) != 6 {
		t.Fatalf("meses = %d, want 6", len(period.Meses))
	}
	if period.Meses[0].Previsto != 0 {
		t.Errorf("julho previsto = %d, want 0", period.Meses[0].Previsto)
	}
	if period.Previsto != 425000 { // agosto a dezembro: 5 × R$ 850
		t.Errorf("previsto = %d, want 425000", period.Previsto)
	}
	if period.Recebido != 85000 || period.Pendente != 340000 {
		t.Errorf("recebido/pendente = %d/%d, want 85000/340000", period.Recebido, period.Pendente)
	}
}
