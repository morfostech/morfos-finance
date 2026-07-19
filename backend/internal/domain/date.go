package domain

import (
	"strings"
	"time"
)

const dateLayout = "2006-01-02"

// Date is a calendar date (no time-of-day) that JSON-encodes as YYYY-MM-DD,
// matching Postgres DATE columns.
type Date struct{ time.Time }

// NewDate wraps a time.Time as a Date (time-of-day is ignored on encode).
func NewDate(t time.Time) Date { return Date{Time: t} }

// MustDate parses YYYY-MM-DD, panicking on error. For tests and constants.
func MustDate(s string) Date {
	t, err := time.Parse(dateLayout, s)
	if err != nil {
		panic(err)
	}
	return Date{Time: t}
}

func (d Date) MarshalJSON() ([]byte, error) {
	return []byte(`"` + d.Format(dateLayout) + `"`), nil
}

func (d *Date) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	if s == "" || s == "null" {
		return nil
	}
	t, err := time.Parse(dateLayout, s)
	if err != nil {
		return err
	}
	d.Time = t
	return nil
}
