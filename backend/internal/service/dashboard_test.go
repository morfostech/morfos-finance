package service

import (
	"context"
	"testing"
	"time"

	"github.com/morfostech/morfos-finance/internal/domain"
)

// fakeDashboardRepo returns canned aggregates.
type fakeDashboardRepo struct {
	saldo        domain.Money
	ganhos       domain.Money
	despesas     domain.Money
	userGanhos   domain.Money
	userDespesas domain.Money
}

func (f *fakeDashboardRepo) SaldoEmCaixa(context.Context, time.Time) (domain.Money, error) {
	return f.saldo, nil
}
func (f *fakeDashboardRepo) PeriodTotals(context.Context, time.Time, time.Time) (domain.Money, domain.Money, error) {
	return f.ganhos, f.despesas, nil
}
func (f *fakeDashboardRepo) GanhosPorOrigem(context.Context, time.Time, time.Time) (domain.OrigemTotals, error) {
	return domain.OrigemTotals{}, nil
}
func (f *fakeDashboardRepo) DespesasPorCategoria(context.Context, time.Time, time.Time) ([]domain.CategoryTotal, error) {
	return nil, nil
}
func (f *fakeDashboardRepo) ImplementacaoTotals(context.Context) (domain.ImplementacaoTotals, error) {
	return domain.ImplementacaoTotals{}, nil
}
func (f *fakeDashboardRepo) ParcelasPendentes(context.Context) (domain.PendingInstallments, error) {
	return domain.PendingInstallments{}, nil
}
func (f *fakeDashboardRepo) PorProjeto(context.Context, time.Time, time.Time) ([]domain.ProjectTotals, error) {
	return nil, nil
}
func (f *fakeDashboardRepo) PorColaborador(context.Context, time.Time, time.Time) ([]domain.UserTotals, error) {
	return nil, nil
}
func (f *fakeDashboardRepo) UserTotalsFor(context.Context, int64, time.Time, time.Time) (domain.Money, domain.Money, error) {
	return f.userGanhos, f.userDespesas, nil
}

// recurrence repo returning no rows, for the embedded RecurrenceService.
type emptyRecurrenceRepo struct{}

func (emptyRecurrenceRepo) MonthRows(context.Context, time.Time, time.Time, *int64) ([]domain.RecurrenceRow, error) {
	return nil, nil
}

func TestCompanyDashboardMath(t *testing.T) {
	repo := &fakeDashboardRepo{saldo: 1500000, ganhos: 800000, despesas: 300000}
	svc := NewDashboardService(repo, NewRecurrenceService(emptyRecurrenceRepo{}), nil)

	from := domain.MustDate("2026-07-01")
	to := domain.MustDate("2026-07-31")
	dash, err := svc.Company(context.Background(), from, to)
	if err != nil {
		t.Fatal(err)
	}
	if dash.SaldoEmCaixa != 1500000 {
		t.Errorf("saldo = %d, want 1500000", dash.SaldoEmCaixa)
	}
	if dash.Resultado != 500000 {
		t.Errorf("resultado = %d, want 500000 (ganhos - despesas)", dash.Resultado)
	}
	if dash.RecorrenciaMes == nil || dash.RecorrenciaMes.Mes != 7 {
		t.Errorf("recorrência do mês ausente ou mês errado: %+v", dash.RecorrenciaMes)
	}
	if dash.RecorrenciaFutura == nil || dash.RecorrenciaFutura.HorizonteMeses != 12 || len(dash.RecorrenciaFutura.Meses) != 12 {
		t.Errorf("previsão de recorrência ausente ou inválida: %+v", dash.RecorrenciaFutura)
	} else if dash.RecorrenciaFutura.Meses[0].Mes != 8 {
		t.Errorf("previsão deveria começar em agosto: %+v", dash.RecorrenciaFutura.Meses[0])
	}
	// Empty slices, never nil, for stable JSON.
	if dash.DespesasPorCategoria == nil || dash.PorProjeto == nil || dash.PorColaborador == nil {
		t.Error("slices devem ser [] e não nil")
	}
}

func TestMeDashboardSaldo(t *testing.T) {
	repo := &fakeDashboardRepo{userGanhos: 200000, userDespesas: 50000}
	svc := NewDashboardService(repo, NewRecurrenceService(emptyRecurrenceRepo{}), NewProjectService(&fakeProjectRepo{}))

	from := domain.MustDate("2026-07-01")
	to := domain.MustDate("2026-07-31")
	dash, err := svc.Me(context.Background(), Viewer{UserID: 7, Role: domain.RoleColaborador}, from, to)
	if err != nil {
		t.Fatal(err)
	}
	if dash.Saldo != 150000 {
		t.Errorf("saldo = %d, want 150000", dash.Saldo)
	}
	if dash.Projetos == nil {
		t.Error("projetos deve ser [] e não nil")
	}
}
