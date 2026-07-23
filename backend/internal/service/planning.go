package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/morfostech/morfos-finance/internal/domain"
)

type PlanningRepo interface {
	CreateMany(context.Context, []domain.PlannedEntry) ([]domain.PlannedEntry, error)
	GetByID(context.Context, int64) (*domain.PlannedEntry, error)
	Update(context.Context, *domain.PlannedEntry) (*domain.PlannedEntry, error)
	SoftDelete(context.Context, int64) error
	List(context.Context, domain.PlanningFilter) ([]domain.PlannedEntry, error)
	Complete(context.Context, int64, int64, domain.Date) (*domain.PlannedEntry, error)
	OpeningBalance(context.Context, time.Time) (domain.Money, error)
	ConfirmedTransactions(context.Context, time.Time, time.Time) ([]domain.CashFlowMovement, error)
	CountOverdue(context.Context, time.Time) (int, error)
	UpsertBudget(context.Context, int64, int, int, domain.Money, int64) (*domain.ExpenseBudget, error)
	ListBudgets(context.Context, int, int) ([]domain.ExpenseBudget, error)
	DeleteBudget(context.Context, int64) error
}

type PlanningService struct {
	repo       PlanningRepo
	recurrence *RecurrenceService
}

func NewPlanningService(repo PlanningRepo, recurrence *RecurrenceService) *PlanningService {
	return &PlanningService{repo: repo, recurrence: recurrence}
}

type PlannedInput struct {
	Tipo         domain.TxType
	Valor        domain.Money
	DueDate      *domain.Date
	ProjectID    *int64
	UserID       *int64
	Origem       *domain.TxOrigem
	CategoryID   *int64
	Descricao    string
	RepeatMonths int
}

func (s *PlanningService) Create(ctx context.Context, in PlannedInput, createdBy int64) ([]domain.PlannedEntry, error) {
	p, err := buildPlanned(in)
	if err != nil {
		return nil, err
	}
	p.CreatedBy = createdBy
	months := in.RepeatMonths
	if months == 0 {
		months = 1
	}
	if months < 1 || months > 24 {
		return nil, fmt.Errorf("%w: repetição deve ficar entre 1 e 24 meses", domain.ErrValidation)
	}
	entries := make([]domain.PlannedEntry, months)
	for i := range entries {
		entries[i] = *p
		entries[i].DueDate = addMonthsClamped(p.DueDate, i)
	}
	return s.repo.CreateMany(ctx, entries)
}

func (s *PlanningService) Update(ctx context.Context, id int64, in PlannedInput) (*domain.PlannedEntry, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing.ActualTransactionID != nil {
		return nil, fmt.Errorf("%w: lançamento realizado não pode ser editado", domain.ErrConflict)
	}
	p, err := buildPlanned(in)
	if err != nil {
		return nil, err
	}
	p.ID, p.CreatedBy = id, existing.CreatedBy
	return s.repo.Update(ctx, p)
}

func (s *PlanningService) Delete(ctx context.Context, id int64) error {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if existing.ActualTransactionID != nil {
		return fmt.Errorf("%w: lançamento realizado não pode ser excluído", domain.ErrConflict)
	}
	return s.repo.SoftDelete(ctx, id)
}

func (s *PlanningService) List(ctx context.Context, f domain.PlanningFilter, today time.Time) ([]domain.PlannedEntryView, error) {
	if f.Status != nil && *f.Status != domain.PlannedOpen && *f.Status != domain.PlannedCompleted {
		return nil, fmt.Errorf("%w: status inválido", domain.ErrValidation)
	}
	entries, err := s.repo.List(ctx, f)
	if err != nil {
		return nil, err
	}
	out := make([]domain.PlannedEntryView, 0, len(entries))
	today = dayStart(today)
	for _, p := range entries {
		out = append(out, domain.PlannedEntryView{
			PlannedEntry: p,
			Status:       p.Status(),
			Overdue:      p.ActualTransactionID == nil && p.DueDate.Time.Before(today),
		})
	}
	return out, nil
}

func (s *PlanningService) Complete(ctx context.Context, id, completedBy int64, paidOn *domain.Date) (*domain.PlannedEntry, error) {
	if paidOn == nil {
		today := domain.NewDate(financeToday())
		paidOn = &today
	}
	return s.repo.Complete(ctx, id, completedBy, *paidOn)
}

func (s *PlanningService) Forecast(ctx context.Context, from, to domain.Date) (*domain.CashFlowForecast, error) {
	if to.Time.Before(from.Time) {
		return nil, fmt.Errorf("%w: período inválido", domain.ErrValidation)
	}
	status := domain.PlannedOpen
	// Overdue open entries are carried into the first day of the projection so
	// they affect the available cash immediately instead of only raising an alert.
	entries, err := s.repo.List(ctx, domain.PlanningFilter{To: &to, Status: &status})
	if err != nil {
		return nil, err
	}
	opening, err := s.repo.OpeningBalance(ctx, from.Time)
	if err != nil {
		return nil, err
	}
	overdue, err := s.repo.CountOverdue(ctx, financeToday())
	if err != nil {
		return nil, err
	}

	byDay := map[string]*domain.CashFlowDay{}
	confirmed, err := s.repo.ConfirmedTransactions(ctx, from.Time, to.Time)
	if err != nil {
		return nil, err
	}
	var confirmedIncome, confirmedExpense domain.Money
	for _, movement := range confirmed {
		key := movement.Data.Format("2006-01-02")
		day := byDay[key]
		if day == nil {
			day = &domain.CashFlowDay{Data: movement.Data, Itens: []domain.CashFlowItem{}}
			byDay[key] = day
		}
		day.Itens = append(day.Itens, movement.Item)
		if movement.Item.Tipo == domain.TxGanho {
			day.Entradas += movement.Item.Valor
			confirmedIncome += movement.Item.Valor
		} else {
			day.Saidas += movement.Item.Valor
			confirmedExpense += movement.Item.Valor
		}
	}

	manualRecurrence := map[string]bool{}
	for _, p := range entries {
		if p.Origem == nil || *p.Origem != domain.OrigemRecorrencia || p.ProjectID == nil {
			continue
		}
		manualRecurrence[recurrencePlanKey(*p.ProjectID, p.DueDate.Time)] = true
	}

	var manualIncome, automaticIncome, manualExpense domain.Money
	for _, p := range entries {
		effectiveDate := p.DueDate
		if effectiveDate.Time.Before(from.Time) {
			effectiveDate = from
		}
		key := effectiveDate.Format("2006-01-02")
		day := byDay[key]
		if day == nil {
			day = &domain.CashFlowDay{Data: effectiveDate, Itens: []domain.CashFlowItem{}}
			byDay[key] = day
		}
		day.Itens = append(day.Itens, domain.CashFlowItem{
			Tipo: p.Tipo, Valor: p.Valor, Descricao: p.Descricao,
			ProjectID: p.ProjectID, Origem: p.Origem,
		})
		if p.Tipo == domain.TxGanho {
			day.Entradas += p.Valor
			manualIncome += p.Valor
		} else {
			day.Saidas += p.Valor
			manualExpense += p.Valor
		}
	}

	if s.recurrence != nil {
		firstMonth := time.Date(from.Year(), from.Month(), 1, 0, 0, 0, 0, time.UTC)
		lastMonth := time.Date(to.Year(), to.Month(), 1, 0, 0, 0, 0, time.UTC)
		for cursor, count := firstMonth, 0; !cursor.After(lastMonth); cursor, count = cursor.AddDate(0, 1, 0), count+1 {
			if count >= 120 {
				return nil, fmt.Errorf("%w: projeção limitada a 120 meses", domain.ErrValidation)
			}
			summary, err := s.recurrence.monthAt(ctx, cursor.Year(), int(cursor.Month()), nil, financeToday())
			if err != nil {
				return nil, err
			}
			for _, project := range summary.Projetos {
				if project.Pendente <= 0 || project.Vencimento == nil || project.Vencimento.Time.After(to.Time) {
					continue
				}
				if manualRecurrence[recurrencePlanKey(project.ProjectID, project.Vencimento.Time)] {
					continue
				}
				effectiveDate := *project.Vencimento
				if effectiveDate.Time.Before(from.Time) {
					effectiveDate = from
				}
				key := effectiveDate.Format("2006-01-02")
				day := byDay[key]
				if day == nil {
					day = &domain.CashFlowDay{Data: effectiveDate, Itens: []domain.CashFlowItem{}}
					byDay[key] = day
				}
				projectID := project.ProjectID
				origin := domain.OrigemRecorrencia
				day.Entradas += project.Pendente
				day.Itens = append(day.Itens, domain.CashFlowItem{
					Tipo: domain.TxGanho, Valor: project.Pendente,
					Descricao: "Mensalidade · " + project.Nome,
					ProjectID: &projectID, Origem: &origin, Automatico: true,
				})
				automaticIncome += project.Pendente
				if project.Vencimento.Time.Before(financeToday()) {
					overdue++
				}
			}
		}
	}
	days := make([]domain.CashFlowDay, 0)
	balance := opening
	for cursor := from.Time; !cursor.After(to.Time); cursor = cursor.AddDate(0, 0, 1) {
		key := cursor.Format("2006-01-02")
		if day := byDay[key]; day != nil {
			balance += day.Entradas - day.Saidas
			day.SaldoProjetado = balance
			days = append(days, *day)
		}
	}
	income := confirmedIncome + manualIncome + automaticIncome
	expense := confirmedExpense + manualExpense
	return &domain.CashFlowForecast{Periodo: domain.Period{From: from, To: to}, SaldoInicial: opening,
		Entradas: income, EntradasAutomaticas: automaticIncome, EntradasManuais: manualIncome,
		EntradasConfirmadas: confirmedIncome, Saidas: expense, SaidasManuais: manualExpense,
		SaidasConfirmadas: confirmedExpense, SaldoFinal: opening + income - expense,
		Vencidos: overdue, Dias: days}, nil
}

func recurrencePlanKey(projectID int64, due time.Time) string {
	return fmt.Sprintf("%d:%04d-%02d", projectID, due.Year(), due.Month())
}

func (s *PlanningService) UpsertBudget(ctx context.Context, categoryID int64, year, month int, value domain.Money, createdBy int64) (*domain.ExpenseBudget, error) {
	if categoryID <= 0 || year < 2000 || year > 2200 || month < 1 || month > 12 || value <= 0 {
		return nil, fmt.Errorf("%w: categoria, mês e valor do orçamento são obrigatórios", domain.ErrValidation)
	}
	return s.repo.UpsertBudget(ctx, categoryID, year, month, value, createdBy)
}

func (s *PlanningService) ListBudgets(ctx context.Context, year, month int) ([]domain.ExpenseBudget, error) {
	if year < 2000 || year > 2200 || month < 1 || month > 12 {
		return nil, fmt.Errorf("%w: mês inválido", domain.ErrValidation)
	}
	items, err := s.repo.ListBudgets(ctx, year, month)
	if items == nil {
		items = []domain.ExpenseBudget{}
	}
	return items, err
}

func (s *PlanningService) DeleteBudget(ctx context.Context, id int64) error {
	return s.repo.DeleteBudget(ctx, id)
}

func buildPlanned(in PlannedInput) (*domain.PlannedEntry, error) {
	if !in.Tipo.Valid() || in.Valor <= 0 || in.DueDate == nil || strings.TrimSpace(in.Descricao) == "" {
		return nil, fmt.Errorf("%w: tipo, valor, vencimento e descrição são obrigatórios", domain.ErrValidation)
	}
	if in.Tipo == domain.TxGanho {
		if in.CategoryID != nil {
			return nil, fmt.Errorf("%w: entrada não tem categoria", domain.ErrValidation)
		}
		if in.Origem != nil && (!in.Origem.Valid() || *in.Origem == domain.OrigemImplementacao) {
			return nil, fmt.Errorf("%w: origem inválida", domain.ErrValidation)
		}
		if in.Origem != nil && *in.Origem == domain.OrigemRecorrencia && in.ProjectID == nil {
			return nil, fmt.Errorf("%w: projeto é obrigatório para recorrência", domain.ErrValidation)
		}
	} else if in.Origem != nil {
		return nil, fmt.Errorf("%w: saída não tem origem", domain.ErrValidation)
	}
	return &domain.PlannedEntry{Tipo: in.Tipo, Valor: in.Valor, DueDate: *in.DueDate,
		ProjectID: in.ProjectID, UserID: in.UserID, Origem: in.Origem,
		CategoryID: in.CategoryID, Descricao: strings.TrimSpace(in.Descricao)}, nil
}

func addMonthsClamped(d domain.Date, months int) domain.Date {
	y, m, day := d.Date()
	m += time.Month(months)
	first := time.Date(y, m, 1, 0, 0, 0, 0, time.UTC)
	lastDay := first.AddDate(0, 1, -1).Day()
	if day > lastDay {
		day = lastDay
	}
	return domain.NewDate(time.Date(first.Year(), first.Month(), day, 0, 0, 0, 0, time.UTC))
}

func dayStart(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func financeToday() time.Time {
	location, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		location = time.FixedZone("BRT", -3*60*60)
	}
	return dayStart(time.Now().In(location))
}
