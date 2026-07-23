package service

import (
	"context"
	"fmt"
	"time"

	"github.com/morfostech/morfos-finance/internal/domain"
)

// RecurrenceRepo is the read-model contract for recurring revenue.
type RecurrenceRepo interface {
	MonthRows(ctx context.Context, start, end time.Time, projectID *int64) ([]domain.RecurrenceRow, error)
}

type RecurrenceService struct {
	repo RecurrenceRepo
}

func NewRecurrenceService(repo RecurrenceRepo) *RecurrenceService {
	return &RecurrenceService{repo: repo}
}

// Month returns the recurrence summary (previsto × recebido × pendente) for a
// single month, optionally scoped to one project.
func (s *RecurrenceService) Month(ctx context.Context, ano, mes int, projectID *int64) (*domain.RecurrenceSummary, error) {
	return s.monthAt(ctx, ano, mes, projectID, financeToday())
}

func (s *RecurrenceService) monthAt(ctx context.Context, ano, mes int, projectID *int64, today time.Time) (*domain.RecurrenceSummary, error) {
	if err := validateMonth(ano, mes); err != nil {
		return nil, err
	}
	start, end := domain.MonthBounds(ano, time.Month(mes))
	rows, err := s.repo.MonthRows(ctx, start, end, projectID)
	if err != nil {
		return nil, err
	}
	return domain.BuildSummaryAt(ano, time.Month(mes), rows, today), nil
}

// Year returns the 12 monthly summaries for a year (a recurrence timeline),
// optionally scoped to one project.
func (s *RecurrenceService) Year(ctx context.Context, ano int, projectID *int64) ([]*domain.RecurrenceSummary, error) {
	if err := validateYear(ano); err != nil {
		return nil, err
	}
	out := make([]*domain.RecurrenceSummary, 0, 12)
	today := financeToday()
	for mes := 1; mes <= 12; mes++ {
		summary, err := s.monthAt(ctx, ano, mes, projectID, today)
		if err != nil {
			return nil, err
		}
		out = append(out, summary)
	}
	return out, nil
}

// Period accumulates recurrence for every calendar month touched by [from,
// to]. A monthly fee is counted once for each active month; received values
// continue to come exclusively from transactions in that month.
func (s *RecurrenceService) Period(ctx context.Context, from, to time.Time, projectID *int64) (*domain.RecurrencePeriod, error) {
	if to.Before(from) {
		return nil, fmt.Errorf("%w: período inválido", domain.ErrValidation)
	}
	start, _ := domain.MonthBounds(from.Year(), from.Month())
	last, _ := domain.MonthBounds(to.Year(), to.Month())
	out := &domain.RecurrencePeriod{Meses: []domain.RecurrencePeriodMonth{}}
	today := financeToday()

	for cursor, count := start, 0; !cursor.After(last); cursor, count = cursor.AddDate(0, 1, 0), count+1 {
		if count >= 120 {
			return nil, fmt.Errorf("%w: período de recorrência limitado a 120 meses", domain.ErrValidation)
		}
		summary, err := s.monthAt(ctx, cursor.Year(), int(cursor.Month()), projectID, today)
		if err != nil {
			return nil, err
		}
		out.Meses = append(out.Meses, domain.RecurrencePeriodMonth{
			Ano: summary.Ano, Mes: summary.Mes, Previsto: summary.Previsto,
			Recebido: summary.Recebido, Pendente: summary.Pendente,
			Vencido: summary.Vencido, AVencer: summary.AVencer,
		})
		out.Previsto += summary.Previsto
		out.Recebido += summary.Recebido
		out.Pendente += summary.Pendente
		out.Vencido += summary.Vencido
		out.AVencer += summary.AVencer
	}
	return out, nil
}

// Forecast projects recurring revenue over a bounded number of months,
// starting with the month containing start.
func (s *RecurrenceService) Forecast(ctx context.Context, start time.Time, months int, projectID *int64) (*domain.RecurrenceForecast, error) {
	if months < 1 || months > 60 {
		return nil, fmt.Errorf("%w: horizonte deve estar entre 1 e 60 meses", domain.ErrValidation)
	}
	monthStart, _ := domain.MonthBounds(start.Year(), start.Month())
	lastMonth := monthStart.AddDate(0, months-1, 0)
	_, rangeEnd := domain.MonthBounds(lastMonth.Year(), lastMonth.Month())
	rows, err := s.repo.MonthRows(ctx, monthStart, rangeEnd, projectID)
	if err != nil {
		return nil, err
	}
	return domain.BuildForecast(monthStart, months, rows), nil
}

func validateMonth(ano, mes int) error {
	if err := validateYear(ano); err != nil {
		return err
	}
	if mes < 1 || mes > 12 {
		return fmt.Errorf("%w: mês deve estar entre 1 e 12", domain.ErrValidation)
	}
	return nil
}

func validateYear(ano int) error {
	if ano < 2000 || ano > 2100 {
		return fmt.Errorf("%w: ano fora do intervalo", domain.ErrValidation)
	}
	return nil
}
