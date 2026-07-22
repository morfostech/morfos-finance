package domain

// Period is the inclusive date range a dashboard reports over.
type Period struct {
	From Date `json:"from"`
	To   Date `json:"to"`
}

// CategoryTotal is expense spending grouped by category (nil = sem categoria).
type CategoryTotal struct {
	CategoryID *int64 `json:"category_id"`
	Nome       string `json:"nome"`
	Total      Money  `json:"total"`
}

// OrigemTotals splits income by its origem within the period.
type OrigemTotals struct {
	Implementacao Money `json:"implementacao"`
	Recorrencia   Money `json:"recorrencia"`
	Avulso        Money `json:"avulso"`
	SemOrigem     Money `json:"sem_origem"`
}

// ImplementacaoTotals is the all-time state of implementation installments.
type ImplementacaoTotals struct {
	Total    Money `json:"total"`
	Recebido Money `json:"recebido"`
	AReceber Money `json:"a_receber"`
}

// PendingInstallments is the count and sum of unpaid installments.
type PendingInstallments struct {
	Quantidade int   `json:"quantidade"`
	Total      Money `json:"total"`
}

// ProjectTotals is income/expense for one project within the period.
type ProjectTotals struct {
	ProjectID int64  `json:"project_id"`
	Nome      string `json:"nome"`
	Ganhos    Money  `json:"ganhos"`
	Despesas  Money  `json:"despesas"`
}

// UserTotals is income/expense attributed to one collaborator within the period.
type UserTotals struct {
	UserID   int64  `json:"user_id"`
	Nome     string `json:"nome"`
	Ganhos   Money  `json:"ganhos"`
	Despesas Money  `json:"despesas"`
}

// CompanyDashboard is the admin/sócio financial overview.
type CompanyDashboard struct {
	Periodo              Period              `json:"periodo"`
	SaldoEmCaixa         Money               `json:"saldo_em_caixa"` // all-time entrou - saiu
	Ganhos               Money               `json:"ganhos"`
	Despesas             Money               `json:"despesas"`
	Resultado            Money               `json:"resultado"`
	GanhosPorOrigem      OrigemTotals        `json:"ganhos_por_origem"`
	DespesasPorCategoria []CategoryTotal     `json:"despesas_por_categoria"`
	Implementacao        ImplementacaoTotals `json:"implementacao"`
	ParcelasPendentes    PendingInstallments `json:"parcelas_pendentes"`
	RecorrenciaMes       *RecurrenceSummary  `json:"recorrencia_mes"`
	RecorrenciaPeriodo   *RecurrencePeriod   `json:"recorrencia_periodo"`
	RecorrenciaFutura    *RecurrenceForecast `json:"recorrencia_futura"`
	PorProjeto           []ProjectTotals     `json:"por_projeto"`
	PorColaborador       []UserTotals        `json:"por_colaborador"`
}

// MeDashboard is a collaborator's personal view.
type MeDashboard struct {
	Periodo  Period    `json:"periodo"`
	Ganhos   Money     `json:"ganhos"`
	Despesas Money     `json:"despesas"`
	Saldo    Money     `json:"saldo"`
	Projetos []Project `json:"projetos"`
}
