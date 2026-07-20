package service

import (
	"context"
	"errors"
	"testing"

	"github.com/morfostech/morfos-finance/internal/domain"
	"github.com/morfostech/morfos-finance/internal/repository"
)

func mptr(v int64) *domain.Money {
	m := domain.Money(v)
	return &m
}

func TestPlanInstallments(t *testing.T) {
	paidRows := []domain.Installment{
		{Tipo: domain.InstallmentEntrada, Valor: 50000, Pago: true},
		{Tipo: domain.InstallmentFinalizacao, Valor: 50000},
	}
	unpaidRows := []domain.Installment{
		{Tipo: domain.InstallmentEntrada, Valor: 50000},
		{Tipo: domain.InstallmentFinalizacao, Valor: 50000},
	}

	tests := []struct {
		name       string
		oldImpl    *domain.Money
		newImpl    *domain.Money
		existing   []domain.Installment
		wantDelete bool
		wantCreate int
		wantErr    error
	}{
		{"sem implementação em nenhum", nil, nil, nil, false, 0, nil},
		{"valor inalterado", mptr(100000), mptr(100000), unpaidRows, false, 0, nil},
		{"adiciona implementação", nil, mptr(100000), nil, false, 2, nil},
		{"altera valor sem parcela paga", mptr(100000), mptr(200000), unpaidRows, true, 2, nil},
		{"remove implementação sem paga", mptr(100000), nil, unpaidRows, true, 0, nil},
		{"altera valor com parcela paga", mptr(100000), mptr(200000), paidRows, false, 0, domain.ErrPaidInstallment},
		{"remove implementação com paga", mptr(100000), nil, paidRows, false, 0, domain.ErrPaidInstallment},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			plan, err := planInstallments(tc.oldImpl, tc.newImpl, tc.existing)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr != nil {
				return
			}
			if plan.DeleteAll != tc.wantDelete {
				t.Errorf("DeleteAll = %v, want %v", plan.DeleteAll, tc.wantDelete)
			}
			if len(plan.Create) != tc.wantCreate {
				t.Errorf("len(Create) = %d, want %d", len(plan.Create), tc.wantCreate)
			}
		})
	}
}

// fakeProjectRepo is an in-memory ProjectRepo capturing what Create receives.
type fakeProjectRepo struct {
	created *domain.Project
}

func (f *fakeProjectRepo) Create(_ context.Context, p *domain.Project) (*domain.Project, error) {
	f.created = p
	p.ID = 1
	return p, nil
}
func (f *fakeProjectRepo) Update(_ context.Context, p *domain.Project, _ repository.InstallmentPlan) (*domain.Project, error) {
	return p, nil
}
func (f *fakeProjectRepo) GetByID(_ context.Context, _ int64) (*domain.Project, error) {
	return f.created, nil
}
func (f *fakeProjectRepo) List(_ context.Context, _ *int64) ([]domain.Project, error) {
	return nil, nil
}
func (f *fakeProjectRepo) ReplaceMembers(_ context.Context, _ int64, _ []int64) error { return nil }
func (f *fakeProjectRepo) IsMember(_ context.Context, _, _ int64) (bool, error)       { return false, nil }
func (f *fakeProjectRepo) GetInstallment(_ context.Context, _, _ int64) (*domain.Installment, error) {
	return nil, nil
}
func (f *fakeProjectRepo) SetInstallment(_ context.Context, _, _ int64, _ domain.Money, _ *domain.Date) (*domain.Installment, error) {
	return nil, nil
}

func TestCreateProjectValidation(t *testing.T) {
	tests := []struct {
		name    string
		in      ProjectInput
		wantErr error
	}{
		{"só implementação", ProjectInput{Nome: "P", ValorImplementacao: mptr(100000)}, nil},
		{"só mensalidade", ProjectInput{Nome: "P", ValorMensal: mptr(50000)}, nil},
		{"ambas", ProjectInput{Nome: "P", ValorImplementacao: mptr(100000), ValorMensal: mptr(50000)}, nil},
		{"nome vazio", ProjectInput{ValorMensal: mptr(50000)}, domain.ErrValidation},
		{"sem fonte de receita", ProjectInput{Nome: "P"}, domain.ErrValidation},
		{"implementação zero", ProjectInput{Nome: "P", ValorImplementacao: mptr(0)}, domain.ErrValidation},
		{"dia inválido", ProjectInput{Nome: "P", ValorMensal: mptr(50000), DiaVencimento: intPtr(32)}, domain.ErrValidation},
		{"fim antes do início", ProjectInput{Nome: "P", ValorMensal: mptr(50000),
			DataInicio: datePtr2("2024-06-01"), DataFim: datePtr2("2024-01-01")}, domain.ErrValidation},
		{"status inválido", ProjectInput{Nome: "P", ValorMensal: mptr(50000), Status: domain.ProjectStatus("x")}, domain.ErrValidation},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := NewProjectService(&fakeProjectRepo{})
			_, err := svc.Create(context.Background(), tc.in)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestCreateGeneratesInstallments(t *testing.T) {
	repo := &fakeProjectRepo{}
	svc := NewProjectService(repo)
	_, err := svc.Create(context.Background(), ProjectInput{Nome: "P", ValorImplementacao: mptr(100001)})
	if err != nil {
		t.Fatal(err)
	}
	if got := len(repo.created.Installments); got != 2 {
		t.Fatalf("esperava 2 parcelas geradas, veio %d", got)
	}
	// Mensalidade sozinha não gera parcelas.
	repo2 := &fakeProjectRepo{}
	svc2 := NewProjectService(repo2)
	if _, err := svc2.Create(context.Background(), ProjectInput{Nome: "P", ValorMensal: mptr(50000)}); err != nil {
		t.Fatal(err)
	}
	if len(repo2.created.Installments) != 0 {
		t.Fatalf("mensalidade não deveria gerar parcelas")
	}
}

func intPtr(v int) *int { return &v }

func datePtr2(s string) *domain.Date {
	d := domain.MustDate(s)
	return &d
}
