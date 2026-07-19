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
