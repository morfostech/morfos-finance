package domain

import "time"

// RecurrenceRow is the raw read-model for one recurring project in a month:
// its monthly fee, its active period, and how much recurrence income it
// received during the month.
type RecurrenceRow struct {
	ProjectID   int64
	Nome        string
	ValorMensal Money
	DataInicio  *Date
	DataFim     *Date
	Recebido    Money
}

// ProjectRecurrence is one project's recurrence status for a given month.
type ProjectRecurrence struct {
	ProjectID int64  `json:"project_id"`
	Nome      string `json:"nome"`
	Previsto  Money  `json:"previsto"`
	Recebido  Money  `json:"recebido"`
	Pendente  Money  `json:"pendente"`
	Ativo     bool   `json:"ativo"`
}

// RecurrenceSummary aggregates a month's recurring revenue across projects.
type RecurrenceSummary struct {
	Ano      int                 `json:"ano"`
	Mes      int                 `json:"mes"`
	Previsto Money               `json:"previsto"`
	Recebido Money               `json:"recebido"`
	Pendente Money               `json:"pendente"`
	Projetos []ProjectRecurrence `json:"projetos"`
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
	start, end := MonthBounds(ano, mes)
	sum := &RecurrenceSummary{Ano: ano, Mes: int(mes), Projetos: make([]ProjectRecurrence, 0, len(rows))}

	for _, r := range rows {
		ativo := activeInMonth(r.DataInicio, r.DataFim, start, end)
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
		sum.Projetos = append(sum.Projetos, pr)
		sum.Previsto += pr.Previsto
		sum.Recebido += pr.Recebido
		sum.Pendente += pr.Pendente
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
		start, end := MonthBounds(month.Year(), month.Month())
		projected := Money(0)
		for _, row := range rows {
			if activeInMonth(row.DataInicio, row.DataFim, start, end) {
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
