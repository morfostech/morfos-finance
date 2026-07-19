// Package respond centralizes JSON responses and domain-error mapping so every
// handler returns a consistent shape.
package respond

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/morfostech/morfos-finance/internal/domain"
)

// JSON writes v as JSON with the given status.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v != nil {
		if err := json.NewEncoder(w).Encode(v); err != nil {
			slog.Error("encode response", "err", err)
		}
	}
}

// errorBody is the uniform error envelope.
type errorBody struct {
	Error string `json:"error"`
}

// Error writes a message with an explicit status.
func Error(w http.ResponseWriter, status int, msg string) {
	JSON(w, status, errorBody{Error: msg})
}

// DomainError maps a domain error to the right status. Unknown errors become 500
// and are logged; their message is not leaked to the client.
func DomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		Error(w, http.StatusNotFound, err.Error())
	case errors.Is(err, domain.ErrInvalidCredentials):
		Error(w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, domain.ErrUserInactive):
		Error(w, http.StatusForbidden, err.Error())
	case errors.Is(err, domain.ErrForbidden):
		Error(w, http.StatusForbidden, err.Error())
	case errors.Is(err, domain.ErrEmailTaken):
		Error(w, http.StatusConflict, err.Error())
	case errors.Is(err, domain.ErrPaidInstallment):
		Error(w, http.StatusConflict, err.Error())
	case errors.Is(err, domain.ErrWrongPassword):
		Error(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, domain.ErrValidation):
		Error(w, http.StatusUnprocessableEntity, err.Error())
	default:
		slog.Error("unhandled error", "err", err)
		Error(w, http.StatusInternalServerError, "erro interno")
	}
}
