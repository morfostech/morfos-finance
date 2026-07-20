package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/morfostech/morfos-finance/internal/domain"
	"github.com/morfostech/morfos-finance/internal/repository"
)

// ProjectRepo is the persistence contract the project service needs.
type ProjectRepo interface {
	Create(ctx context.Context, p *domain.Project) (*domain.Project, error)
	Update(ctx context.Context, p *domain.Project, plan repository.InstallmentPlan) (*domain.Project, error)
	GetByID(ctx context.Context, id int64) (*domain.Project, error)
	List(ctx context.Context, memberID *int64) ([]domain.Project, error)
	ReplaceMembers(ctx context.Context, projectID int64, userIDs []int64) error
	IsMember(ctx context.Context, projectID, userID int64) (bool, error)
	GetInstallment(ctx context.Context, projectID, installmentID int64) (*domain.Installment, error)
	SetInstallment(ctx context.Context, projectID, installmentID int64, valor domain.Money, pagoEm *domain.Date) (*domain.Installment, error)
}

type ProjectService struct {
	projects ProjectRepo
}

func NewProjectService(projects ProjectRepo) *ProjectService {
	return &ProjectService{projects: projects}
}

// ProjectInput is the create/update payload (full desired state of editable
// fields). MemberIDs is applied only on create; use SetMembers to change it.
type ProjectInput struct {
	Nome               string
	Cliente            *string
	ValorImplementacao *domain.Money
	ValorMensal        *domain.Money
	DiaVencimento      *int
	DataInicio         *domain.Date
	DataFim            *domain.Date
	Status             domain.ProjectStatus
	MemberIDs          []int64
}

func (s *ProjectService) Create(ctx context.Context, in ProjectInput) (*domain.Project, error) {
	p, err := s.build(in)
	if err != nil {
		return nil, err
	}
	if p.ValorImplementacao != nil {
		p.Installments = domain.GenerateInstallments(*p.ValorImplementacao)
	}
	return s.projects.Create(ctx, p)
}

// Update reconciles a project against new input, regenerating implementation
// installments as needed (see planInstallments for the rules).
func (s *ProjectService) Update(ctx context.Context, id int64, in ProjectInput) (*domain.Project, error) {
	existing, err := s.projects.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	p, err := s.build(in)
	if err != nil {
		return nil, err
	}
	p.ID = id

	plan, err := planInstallments(existing.ValorImplementacao, p.ValorImplementacao, existing.Installments)
	if err != nil {
		return nil, err
	}
	return s.projects.Update(ctx, p, plan)
}

func (s *ProjectService) Get(ctx context.Context, id int64, viewer Viewer) (*domain.Project, error) {
	p, err := s.projects.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if viewer.Role == domain.RoleColaborador {
		member, err := s.projects.IsMember(ctx, id, viewer.UserID)
		if err != nil {
			return nil, err
		}
		if !member {
			return nil, domain.ErrForbidden
		}
	}
	return p, nil
}

func (s *ProjectService) List(ctx context.Context, viewer Viewer) ([]domain.Project, error) {
	var filter *int64
	if viewer.Role == domain.RoleColaborador {
		filter = &viewer.UserID // colaborador sees only allocated projects
	}
	return s.projects.List(ctx, filter)
}

func (s *ProjectService) SetMembers(ctx context.Context, projectID int64, userIDs []int64) error {
	return s.projects.ReplaceMembers(ctx, projectID, dedupe(userIDs))
}

// MarkInstallment sets an installment paid (with a date) or pending (nil date).
func (s *ProjectService) MarkInstallment(ctx context.Context, projectID, installmentID int64, pagoEm *domain.Date) (*domain.Installment, error) {
	inst, err := s.projects.GetInstallment(ctx, projectID, installmentID)
	if err != nil {
		return nil, err
	}
	return s.projects.SetInstallment(ctx, projectID, installmentID, inst.Valor, pagoEm)
}

// Viewer is the authenticated principal's identity for authorization decisions.
type Viewer struct {
	UserID int64
	Role   domain.Role
}

// build validates input and constructs a Project (without installments).
func (s *ProjectService) build(in ProjectInput) (*domain.Project, error) {
	nome := strings.TrimSpace(in.Nome)
	if nome == "" {
		return nil, fmt.Errorf("%w: nome é obrigatório", domain.ErrValidation)
	}
	if in.ValorImplementacao == nil && in.ValorMensal == nil {
		return nil, fmt.Errorf("%w: informe implementação e/ou mensalidade", domain.ErrValidation)
	}
	if in.ValorImplementacao != nil && *in.ValorImplementacao <= 0 {
		return nil, fmt.Errorf("%w: valor de implementação deve ser positivo", domain.ErrValidation)
	}
	if in.ValorMensal != nil && *in.ValorMensal <= 0 {
		return nil, fmt.Errorf("%w: valor mensal deve ser positivo", domain.ErrValidation)
	}
	if in.DiaVencimento != nil && (*in.DiaVencimento < 1 || *in.DiaVencimento > 31) {
		return nil, fmt.Errorf("%w: dia de vencimento deve estar entre 1 e 31", domain.ErrValidation)
	}
	if in.DataInicio != nil && in.DataFim != nil && in.DataFim.Before(in.DataInicio.Time) {
		return nil, fmt.Errorf("%w: data de fim anterior ao início", domain.ErrValidation)
	}

	status := in.Status
	if status == "" {
		status = domain.StatusAtivo
	}
	if !status.Valid() {
		return nil, fmt.Errorf("%w: status inválido", domain.ErrValidation)
	}

	return &domain.Project{
		Nome:               nome,
		Cliente:            trimPtr(in.Cliente),
		ValorImplementacao: in.ValorImplementacao,
		ValorMensal:        in.ValorMensal,
		DiaVencimento:      in.DiaVencimento,
		DataInicio:         in.DataInicio,
		DataFim:            in.DataFim,
		Status:             status,
		MemberIDs:          dedupe(in.MemberIDs),
	}, nil
}

// planInstallments decides how implementation installments must change when the
// implementation value goes from oldImpl to newImpl, given the current rows.
//
//   - value unchanged                -> no-op
//   - value removed, none paid       -> delete all
//   - value removed, any paid        -> error
//   - value set, no rows             -> create 50/50
//   - value changed, none paid       -> regenerate (delete + create)
//   - value changed, any paid        -> error
func planInstallments(oldImpl, newImpl *domain.Money, existing []domain.Installment) (repository.InstallmentPlan, error) {
	if moneyEqual(oldImpl, newImpl) {
		return repository.InstallmentPlan{}, nil // nothing changed
	}
	paid := anyPaid(existing)

	if newImpl == nil {
		if paid {
			return repository.InstallmentPlan{}, fmt.Errorf("%w: remova o pagamento antes de excluir a implementação", domain.ErrPaidInstallment)
		}
		return repository.InstallmentPlan{DeleteAll: true}, nil
	}

	// newImpl is set and differs from oldImpl.
	if paid {
		return repository.InstallmentPlan{}, fmt.Errorf("%w: valor de implementação não pode mudar com parcela paga", domain.ErrPaidInstallment)
	}
	return repository.InstallmentPlan{
		DeleteAll: len(existing) > 0,
		Create:    domain.GenerateInstallments(*newImpl),
	}, nil
}

func anyPaid(installments []domain.Installment) bool {
	for _, i := range installments {
		if i.Pago {
			return true
		}
	}
	return false
}

func moneyEqual(a, b *domain.Money) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func trimPtr(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil
	}
	return &t
}

func dedupe(ids []int64) []int64 {
	if len(ids) == 0 {
		return nil
	}
	seen := make(map[int64]bool, len(ids))
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}
