package domain

import "testing"

func TestGenerateInstallments(t *testing.T) {
	tests := []struct {
		name            string
		total           Money
		wantEntrada     Money
		wantFinalizacao Money
	}{
		{"par exato", 1000000, 500000, 500000},      // R$ 10.000,00 -> 5k / 5k
		{"ímpar em centavos", 100001, 50000, 50001}, // resto vai para a finalização
		{"valor pequeno", 1, 0, 1},                  // 1 centavo
		{"zero", 0, 0, 0},
		{"ímpar em reais", 300000, 150000, 150000}, // R$ 3.000,00
		{"quebra", 333333, 166666, 166667},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := GenerateInstallments(tc.total)
			if len(got) != 2 {
				t.Fatalf("esperava 2 parcelas, veio %d", len(got))
			}
			if got[0].Tipo != InstallmentEntrada || got[1].Tipo != InstallmentFinalizacao {
				t.Fatalf("tipos errados: %s / %s", got[0].Tipo, got[1].Tipo)
			}
			if got[0].Valor != tc.wantEntrada {
				t.Errorf("entrada = %d, want %d", got[0].Valor, tc.wantEntrada)
			}
			if got[1].Valor != tc.wantFinalizacao {
				t.Errorf("finalização = %d, want %d", got[1].Valor, tc.wantFinalizacao)
			}
			if sum := got[0].Valor + got[1].Valor; sum != tc.total {
				t.Errorf("soma das parcelas = %d, want %d (total)", sum, tc.total)
			}
		})
	}
}

func TestParseNumericRoundTrip(t *testing.T) {
	tests := []struct {
		text string
		want Money
	}{
		{"1234.56", 123456},
		{"0.00", 0},
		{"10000.00", 1000000},
		{"5.5", 550},
		{"7", 700},
		{"1234.567", 123456}, // trunca para 2 casas
	}
	for _, tc := range tests {
		t.Run(tc.text, func(t *testing.T) {
			got, err := ParseNumeric(tc.text)
			if err != nil {
				t.Fatalf("ParseNumeric(%q): %v", tc.text, err)
			}
			if got != tc.want {
				t.Errorf("ParseNumeric(%q) = %d, want %d", tc.text, got, tc.want)
			}
		})
	}

	// Numeric() deve produzir texto reparseável.
	for _, m := range []Money{0, 1, 99, 100, 123456, 1000000} {
		if back, _ := ParseNumeric(m.Numeric()); back != m {
			t.Errorf("round-trip falhou para %d: Numeric=%q -> %d", m, m.Numeric(), back)
		}
	}
}
