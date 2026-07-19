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
	if err := validateMonth(ano, mes); err != nil {
		return nil, err
	}
	start, end := domain.MonthBounds(ano, time.Month(mes))
	rows, err := s.repo.MonthRows(ctx, start, end, projectID)
	if err != nil {
		return nil, err
	}
	return domain.BuildSummary(ano, time.Month(mes), rows), nil
}

// Year returns the 12 monthly summaries for a year (a recurrence timeline),
// optionally scoped to one project.
func (s *RecurrenceService) Year(ctx context.Context, ano int, projectID *int64) ([]*domain.RecurrenceSummary, error) {
	if err := validateYear(ano); err != nil {
		return nil, err
	}
	out := make([]*domain.RecurrenceSummary, 0, 12)
	for mes := 1; mes <= 12; mes++ {
		summary, err := s.Month(ctx, ano, mes, projectID)
		if err != nil {
			return nil, err
		}
		out = append(out, summary)
	}
	return out, nil
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
