package service

import (
	"context"
	"time"

	"github.com/morfostech/morfos-finance/internal/domain"
)

// DashboardRepo is the aggregate read-model the dashboards need.
type DashboardRepo interface {
	SaldoEmCaixa(ctx context.Context, asOf time.Time) (domain.Money, error)
	PeriodTotals(ctx context.Context, from, to time.Time) (ganhos, despesas domain.Money, err error)
	GanhosPorOrigem(ctx context.Context, from, to time.Time) (domain.OrigemTotals, error)
	DespesasPorCategoria(ctx context.Context, from, to time.Time) ([]domain.CategoryTotal, error)
	ImplementacaoTotals(ctx context.Context) (domain.ImplementacaoTotals, error)
	ParcelasPendentes(ctx context.Context) (domain.PendingInstallments, error)
	PorProjeto(ctx context.Context, from, to time.Time) ([]domain.ProjectTotals, error)
	PorColaborador(ctx context.Context, from, to time.Time) ([]domain.UserTotals, error)
	UserTotalsFor(ctx context.Context, userID int64, from, to time.Time) (ganhos, despesas domain.Money, err error)
}

type DashboardService struct {
	repo       DashboardRepo
	recurrence *RecurrenceService
	projects   *ProjectService
}

func NewDashboardService(repo DashboardRepo, recurrence *RecurrenceService, projects *ProjectService) *DashboardService {
	return &DashboardService{repo: repo, recurrence: recurrence, projects: projects}
}

// Company assembles the admin/sócio financial overview for [from, to].
// Realized aggregates are capped at today; recurrence keeps the entire chosen
// period so future expected values remain visible as amounts due.
func (s *DashboardService) Company(ctx context.Context, from, to domain.Date) (*domain.CompanyDashboard, error) {
	fromT, toT := from.Time, to.Time
	realizedTo := toT
	if today := financeToday(); realizedTo.After(today) {
		realizedTo = today
	}

	saldo, err := s.repo.SaldoEmCaixa(ctx, financeToday())
	if err != nil {
		return nil, err
	}
	ganhos, despesas, err := s.repo.PeriodTotals(ctx, fromT, realizedTo)
	if err != nil {
		return nil, err
	}
	porOrigem, err := s.repo.GanhosPorOrigem(ctx, fromT, realizedTo)
	if err != nil {
		return nil, err
	}
	porCategoria, err := s.repo.DespesasPorCategoria(ctx, fromT, realizedTo)
	if err != nil {
		return nil, err
	}
	impl, err := s.repo.ImplementacaoTotals(ctx)
	if err != nil {
		return nil, err
	}
	pend, err := s.repo.ParcelasPendentes(ctx)
	if err != nil {
		return nil, err
	}
	porProjeto, err := s.repo.PorProjeto(ctx, fromT, realizedTo)
	if err != nil {
		return nil, err
	}
	porColaborador, err := s.repo.PorColaborador(ctx, fromT, realizedTo)
	if err != nil {
		return nil, err
	}
	recorrencia, err := s.recurrence.Month(ctx, toT.Year(), int(toT.Month()), nil)
	if err != nil {
		return nil, err
	}
	recorrenciaPeriodo, err := s.recurrence.Period(ctx, fromT, toT, nil)
	if err != nil {
		return nil, err
	}
	forecastStart := time.Date(toT.Year(), toT.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, 1, 0)
	recorrenciaFutura, err := s.recurrence.Forecast(ctx, forecastStart, 12, nil)
	if err != nil {
		return nil, err
	}

	return &domain.CompanyDashboard{
		Periodo:              domain.Period{From: from, To: to},
		SaldoEmCaixa:         saldo,
		Ganhos:               ganhos,
		Despesas:             despesas,
		Resultado:            ganhos - despesas,
		GanhosPorOrigem:      porOrigem,
		DespesasPorCategoria: emptyIfNil(porCategoria),
		Implementacao:        impl,
		ParcelasPendentes:    pend,
		RecorrenciaMes:       recorrencia,
		RecorrenciaPeriodo:   recorrenciaPeriodo,
		RecorrenciaFutura:    recorrenciaFutura,
		PorProjeto:           emptyProjectTotals(porProjeto),
		PorColaborador:       emptyUserTotals(porColaborador),
	}, nil
}

// Me assembles a collaborator's personal view for [from, to].
func (s *DashboardService) Me(ctx context.Context, viewer Viewer, from, to domain.Date) (*domain.MeDashboard, error) {
	ganhos, despesas, err := s.repo.UserTotalsFor(ctx, viewer.UserID, from.Time, to.Time)
	if err != nil {
		return nil, err
	}
	projetos, err := s.projects.ListPersonal(ctx, viewer.UserID)
	if err != nil {
		return nil, err
	}
	if projetos == nil {
		projetos = []domain.Project{}
	}
	return &domain.MeDashboard{
		Periodo:  domain.Period{From: from, To: to},
		Ganhos:   ganhos,
		Despesas: despesas,
		Saldo:    ganhos - despesas,
		Projetos: projetos,
	}, nil
}

func emptyIfNil(v []domain.CategoryTotal) []domain.CategoryTotal {
	if v == nil {
		return []domain.CategoryTotal{}
	}
	return v
}

func emptyProjectTotals(v []domain.ProjectTotals) []domain.ProjectTotals {
	if v == nil {
		return []domain.ProjectTotals{}
	}
	return v
}

func emptyUserTotals(v []domain.UserTotals) []domain.UserTotals {
	if v == nil {
		return []domain.UserTotals{}
	}
	return v
}
