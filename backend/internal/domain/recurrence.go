package domain

import "time"

// RecurrenceRow is the raw read-model for one recurring project in a month:
// its monthly fee, its active period, and how much recurrence income it
// received during the month.
type RecurrenceRow struct {
	ProjectID     int64
	Nome          string
	ValorMensal   Money
	DiaVencimento *int
	DataInicio    *Date
	DataFim       *Date
	Status        ProjectStatus
	Recebido      Money
}

type RecurrenceStatus string

const (
	RecurrencePaid          RecurrenceStatus = "quitado"
	RecurrencePartiallyPaid RecurrenceStatus = "parcial"
	RecurrenceOverdue       RecurrenceStatus = "vencido"
	RecurrenceDue           RecurrenceStatus = "a_vencer"
	RecurrenceReceivedOnly  RecurrenceStatus = "recebido"
)

// ProjectRecurrence is one project's recurrence status for a given month.
type ProjectRecurrence struct {
	ProjectID  int64            `json:"project_id"`
	Nome       string           `json:"nome"`
	Previsto   Money            `json:"previsto"`
	Recebido   Money            `json:"recebido"`
	Pendente   Money            `json:"pendente"`
	Vencido    Money            `json:"vencido"`
	AVencer    Money            `json:"a_vencer"`
	Vencimento *Date            `json:"vencimento,omitempty"`
	Situacao   RecurrenceStatus `json:"situacao"`
	Ativo      bool             `json:"ativo"`
}

// RecurrenceSummary aggregates a month's recurring revenue across projects.
type RecurrenceSummary struct {
	Ano      int                 `json:"ano"`
	Mes      int                 `json:"mes"`
	Previsto Money               `json:"previsto"`
	Recebido Money               `json:"recebido"`
	Pendente Money               `json:"pendente"`
	Vencido  Money               `json:"vencido"`
	AVencer  Money               `json:"a_vencer"`
	Projetos []ProjectRecurrence `json:"projetos"`
}

// RecurrencePeriodMonth is the realized-versus-expected recurrence position
// for one month inside a dashboard date range.
type RecurrencePeriodMonth struct {
	Ano      int   `json:"ano"`
	Mes      int   `json:"mes"`
	Previsto Money `json:"previsto"`
	Recebido Money `json:"recebido"`
	Pendente Money `json:"pendente"`
	Vencido  Money `json:"vencido"`
	AVencer  Money `json:"a_vencer"`
}

// RecurrencePeriod accumulates all calendar months touched by a selected
// range. Expected revenue remains separate from received cash.
type RecurrencePeriod struct {
	Previsto Money                   `json:"previsto"`
	Recebido Money                   `json:"recebido"`
	Pendente Money                   `json:"pendente"`
	Vencido  Money                   `json:"vencido"`
	AVencer  Money                   `json:"a_vencer"`
	Meses    []RecurrencePeriodMonth `json:"meses"`
}

// RecurrenceForecastMonth is the expected recurring revenue for one future
// month, based only on project fees and active periods.
type RecurrenceForecastMonth struct {
	Ano      int   `json:"ano"`
	Mes      int   `json:"mes"`
	Previsto Money `json:"previsto"`
}

// RecurrenceForecast projects recurring revenue over a bounded future window.
// It does not create receivables or count future transactions as received.
type RecurrenceForecast struct {
	HorizonteMeses int                       `json:"horizonte_meses"`
	Total          Money                     `json:"total"`
	Meses          []RecurrenceForecastMonth `json:"meses"`
}

// MonthBounds returns the first and last calendar day of the month (UTC),
// matching how Postgres DATE values are scanned.
func MonthBounds(ano int, mes time.Month) (start, end time.Time) {
	start = time.Date(ano, mes, 1, 0, 0, 0, 0, time.UTC)
	end = start.AddDate(0, 1, -1)
	return start, end
}

// activeInMonth reports whether a project's [inicio, fim] period overlaps the
// month bounded by [start, end]. Nil bounds are treated as open-ended.
func activeInMonth(inicio, fim *Date, start, end time.Time) bool {
	if inicio != nil && inicio.Time.After(end) {
		return false
	}
	if fim != nil && fim.Time.Before(start) {
		return false
	}
	return true
}

// billingDate returns the actual charge date for the month. The configured
// due day is clamped to the month's final day. When it is absent, the start
// day is used, falling back to the first day of the month.
func billingDate(row RecurrenceRow, ano int, mes time.Month) time.Time {
	day := 1
	if row.DiaVencimento != nil {
		day = *row.DiaVencimento
	} else if row.DataInicio != nil {
		day = row.DataInicio.Day()
	}
	_, end := MonthBounds(ano, mes)
	if day > end.Day() {
		day = end.Day()
	}
	return time.Date(ano, mes, day, 0, 0, 0, 0, time.UTC)
}

// billableInMonth is stricter than a simple month overlap: the project's
// billing date itself must belong to the configured active period.
func billableInMonth(row RecurrenceRow, ano int, mes time.Month) (time.Time, bool) {
	due := billingDate(row, ano, mes)
	if row.DataInicio != nil && due.Before(row.DataInicio.Time) {
		return due, false
	}
	if row.DataFim != nil && due.After(row.DataFim.Time) {
		return due, false
	}
	return due, true
}

// clampPendente is previsto - recebido, floored at zero (overpayment is not
// negative pending).
func clampPendente(previsto, recebido Money) Money {
	if recebido >= previsto {
		return 0
	}
	return previsto - recebido
}

// BuildSummary aggregates raw rows into a month's recurrence summary. A project
// is included when it is active that month or received recurrence income in it.
func BuildSummary(ano int, mes time.Month, rows []RecurrenceRow) *RecurrenceSummary {
	return BuildSummaryAt(ano, mes, rows, time.Now().UTC())
}

// BuildSummaryAt builds the month position relative to a reference date so
// unpaid future charges are not mislabeled as overdue.
func BuildSummaryAt(ano int, mes time.Month, rows []RecurrenceRow, today time.Time) *RecurrenceSummary {
	sum := &RecurrenceSummary{Ano: ano, Mes: int(mes), Projetos: make([]ProjectRecurrence, 0, len(rows))}
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)

	for _, r := range rows {
		due, ativo := billableInMonth(r, ano, mes)
		if !ativo && r.Recebido == 0 {
			continue // nothing to report for this project this month
		}
		previsto := Money(0)
		if ativo {
			previsto = r.ValorMensal
		}
		pr := ProjectRecurrence{
			ProjectID: r.ProjectID,
			Nome:      r.Nome,
			Previsto:  previsto,
			Recebido:  r.Recebido,
			Pendente:  clampPendente(previsto, r.Recebido),
			Ativo:     ativo,
		}
		if ativo {
			date := NewDate(due)
			pr.Vencimento = &date
		}
		switch {
		case previsto == 0 && r.Recebido > 0:
			pr.Situacao = RecurrenceReceivedOnly
		case pr.Pendente == 0:
			pr.Situacao = RecurrencePaid
		case due.Before(today):
			pr.Vencido = pr.Pendente
			if pr.Recebido > 0 {
				pr.Situacao = RecurrencePartiallyPaid
			} else {
				pr.Situacao = RecurrenceOverdue
			}
		default:
			pr.AVencer = pr.Pendente
			if pr.Recebido > 0 {
				pr.Situacao = RecurrencePartiallyPaid
			} else {
				pr.Situacao = RecurrenceDue
			}
		}
		sum.Projetos = append(sum.Projetos, pr)
		sum.Previsto += pr.Previsto
		sum.Recebido += pr.Recebido
		sum.Pendente += pr.Pendente
		sum.Vencido += pr.Vencido
		sum.AVencer += pr.AVencer
	}
	return sum
}

// BuildForecast calculates expected recurring revenue from startMonth forward.
func BuildForecast(startMonth time.Time, months int, rows []RecurrenceRow) *RecurrenceForecast {
	forecast := &RecurrenceForecast{
		HorizonteMeses: months,
		Meses:          make([]RecurrenceForecastMonth, 0, months),
	}
	for offset := 0; offset < months; offset++ {
		month := startMonth.AddDate(0, offset, 0)
		projected := Money(0)
		for _, row := range rows {
			if row.Status != "" && row.Status != StatusAtivo {
				continue
			}
			if _, billable := billableInMonth(row, month.Year(), month.Month()); billable {
				projected += row.ValorMensal
			}
		}
		forecast.Meses = append(forecast.Meses, RecurrenceForecastMonth{
			Ano:      month.Year(),
			Mes:      int(month.Month()),
			Previsto: projected,
		})
		forecast.Total += projected
	}
	return forecast
}
