package domain

import (
	"fmt"
	"strconv"
	"strings"
)

// Money is an amount in centavos (1/100 BRL). Integer to stay exact — never
// float. JSON-encodes as the integer number of centavos (e.g. 500000 = R$ 5.000,00).
type Money int64

// ParseNumeric converts a Postgres NUMERIC text (e.g. "1234.56") to centavos.
func ParseNumeric(s string) (Money, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("valor numérico vazio")
	}
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")

	intPart, fracPart, _ := strings.Cut(s, ".")
	if intPart == "" {
		intPart = "0"
	}
	frac := (fracPart + "00")[:2] // pad/truncate to 2 decimals

	reais, err := strconv.ParseInt(intPart, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("valor numérico inválido: %q", s)
	}
	cents, err := strconv.ParseInt(frac, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("valor numérico inválido: %q", s)
	}

	total := reais*100 + cents
	if neg {
		total = -total
	}
	return Money(total), nil
}

// Numeric renders centavos as a NUMERIC-compatible string ("1234.56") for SQL.
func (m Money) Numeric() string {
	n := int64(m)
	neg := n < 0
	if neg {
		n = -n
	}
	s := fmt.Sprintf("%d.%02d", n/100, n%100)
	if neg {
		s = "-" + s
	}
	return s
}
