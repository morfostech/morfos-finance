package domain

import (
	"testing"
	"time"
)

func dptr(s string) *Date {
	d := MustDate(s)
	return &d
}

func TestActiveInMonth(t *testing.T) {
	start, end := MonthBounds(2026, time.July) // 2026-07-01 .. 2026-07-31

	tests := []struct {
		name   string
		inicio *Date
		fim    *Date
		want   bool
	}{
		{"aberto dos dois lados", nil, nil, true},
		{"início antes, sem fim", dptr("2026-01-01"), nil, true},
		{"início no mês", dptr("2026-07-15"), nil, true},
		{"início depois do mês", dptr("2026-08-01"), nil, false},
		{"fim antes do mês", dptr("2026-01-01"), dptr("2026-06-30"), false},
		{"fim dentro do mês", dptr("2026-01-01"), dptr("2026-07-10"), true},
		{"início no último dia", dptr("2026-07-31"), nil, true},
		{"fim no primeiro dia", nil, dptr("2026-07-01"), true},
		{"período engloba o mês", dptr("2026-01-01"), dptr("2026-12-31"), true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := activeInMonth(tc.inicio, tc.fim, start, end); got != tc.want {
				t.Errorf("activeInMonth = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestClampPendente(t *testing.T) {
	tests := []struct {
		previsto, recebido, want Money
	}{
		{300000, 0, 300000},      // nada recebido
		{300000, 100000, 200000}, // parcial
		{300000, 300000, 0},      // quitado
		{300000, 400000, 0},      // pago a mais -> não fica negativo
		{0, 0, 0},
	}
	for _, tc := range tests {
		if got := clampPendente(tc.previsto, tc.recebido); got != tc.want {
			t.Errorf("clampPendente(%d,%d) = %d, want %d", tc.previsto, tc.recebido, got, tc.want)
		}
	}
}

func TestBuildSummary(t *testing.T) {
	rows := []RecurrenceRow{
		// Ativo, recebido parcial.
		{ProjectID: 1, Nome: "A", ValorMensal: 300000, DataInicio: dptr("2026-01-01"), Recebido: 100000},
		// Ativo, quitado.
		{ProjectID: 2, Nome: "B", ValorMensal: 250000, DataInicio: dptr("2026-07-01"), Recebido: 250000},
		// Inativo (terminou antes) e sem recebimento -> deve sair do resultado.
		{ProjectID: 3, Nome: "C", ValorMensal: 100000, DataInicio: dptr("2025-01-01"), DataFim: dptr("2026-06-30"), Recebido: 0},
		// Inativo mas com recebimento no mês -> aparece com previsto 0.
		{ProjectID: 4, Nome: "D", ValorMensal: 500000, DataInicio: dptr("2026-08-01"), Recebido: 500000},
	}

	sum := BuildSummary(2026, time.July, rows)

	if sum.Ano != 2026 || sum.Mes != 7 {
		t.Fatalf("ano/mes = %d/%d", sum.Ano, sum.Mes)
	}
	if len(sum.Projetos) != 3 {
		t.Fatalf("esperava 3 projetos (C excluído), veio %d", len(sum.Projetos))
	}
	// Previsto = A(300k) + B(250k); D inativo não soma previsto.
	if sum.Previsto != 550000 {
		t.Errorf("previsto total = %d, want 550000", sum.Previsto)
	}
	// Recebido = 100k + 250k + 500k (D também recebeu).
	if sum.Recebido != 850000 {
		t.Errorf("recebido total = %d, want 850000", sum.Recebido)
	}
	// Pendente = A(200k) + B(0) + D(0, previsto 0).
	if sum.Pendente != 200000 {
		t.Errorf("pendente total = %d, want 200000", sum.Pendente)
	}

	// Projeto D: inativo, previsto 0, pendente 0, recebido preservado.
	var d ProjectRecurrence
	for _, p := range sum.Projetos {
		if p.ProjectID == 4 {
			d = p
		}
	}
	if d.Ativo || d.Previsto != 0 || d.Pendente != 0 || d.Recebido != 500000 {
		t.Errorf("projeto D inesperado: %+v", d)
	}
}

func TestBuildForecastRespectsProjectPeriods(t *testing.T) {
	rows := []RecurrenceRow{
		{ProjectID: 1, Nome: "Open", ValorMensal: 5000, DataInicio: dptr("2026-08-10")},
		{ProjectID: 2, Nome: "Limited", ValorMensal: 8000, DataInicio: dptr("2026-09-01"), DataFim: dptr("2026-10-31")},
		{ProjectID: 3, Nome: "Finished", ValorMensal: 10000, DataFim: dptr("2026-07-31")},
	}

	forecast := BuildForecast(time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC), 12, rows)

	if forecast.HorizonteMeses != 12 || len(forecast.Meses) != 12 {
		t.Fatalf("horizonte inesperado: %+v", forecast)
	}
	if forecast.Meses[0].Ano != 2026 || forecast.Meses[0].Mes != 8 || forecast.Meses[0].Previsto != 5000 {
		t.Errorf("agosto inesperado: %+v", forecast.Meses[0])
	}
	if forecast.Meses[1].Previsto != 13000 || forecast.Meses[2].Previsto != 13000 {
		t.Errorf("setembro/outubro inesperados: %+v", forecast.Meses[1:3])
	}
	if forecast.Meses[3].Previsto != 5000 {
		t.Errorf("novembro inesperado: %+v", forecast.Meses[3])
	}
	if forecast.Total != 76000 {
		t.Errorf("total = %d, want 76000", forecast.Total)
	}
}

func TestBuildSummaryUsesBillingDateAndSeparatesOverdueFromFuture(t *testing.T) {
	dueDay := 10
	rows := []RecurrenceRow{
		{ProjectID: 1, Nome: "Monthly", ValorMensal: 85000, DiaVencimento: &dueDay, DataInicio: dptr("2026-08-01")},
	}

	augustBeforeDue := BuildSummaryAt(2026, time.August, rows, time.Date(2026, time.August, 5, 0, 0, 0, 0, time.UTC))
	if augustBeforeDue.AVencer != 85000 || augustBeforeDue.Vencido != 0 {
		t.Fatalf("before due a_vencer/vencido = %d/%d", augustBeforeDue.AVencer, augustBeforeDue.Vencido)
	}
	if got := augustBeforeDue.Projetos[0].Vencimento.Format("2006-01-02"); got != "2026-08-10" {
		t.Fatalf("vencimento = %s", got)
	}

	augustAfterDue := BuildSummaryAt(2026, time.August, rows, time.Date(2026, time.August, 11, 0, 0, 0, 0, time.UTC))
	if augustAfterDue.Vencido != 85000 || augustAfterDue.AVencer != 0 {
		t.Fatalf("after due vencido/a_vencer = %d/%d", augustAfterDue.Vencido, augustAfterDue.AVencer)
	}

	startsAfterDue := dptr("2026-08-20")
	rows[0].DataInicio = startsAfterDue
	if got := BuildSummaryAt(2026, time.August, rows, time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC)).Previsto; got != 0 {
		t.Fatalf("monthly charge before project start = %d", got)
	}
}
