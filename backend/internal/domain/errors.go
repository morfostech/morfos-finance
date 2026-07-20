package domain

import "errors"

// Sentinel domain errors. The HTTP layer maps these to status codes; services
// and repositories return them so transport concerns stay out of the core.
var (
	ErrNotFound           = errors.New("recurso não encontrado")
	ErrInvalidCredentials = errors.New("e-mail ou senha inválidos")
	ErrUserInactive       = errors.New("usuário inativo")
	ErrEmailTaken         = errors.New("e-mail já cadastrado")
	ErrValidation         = errors.New("dados inválidos")
	ErrForbidden          = errors.New("acesso negado")
	ErrWrongPassword      = errors.New("senha atual incorreta")

	// ErrPaidInstallment guards against mutating the implementation value once a
	// parcela has been paid.
	ErrPaidInstallment = errors.New("parcela já paga não pode ser alterada")

	// ErrConflict is a generic state conflict (e.g. deleting a resource still
	// referenced elsewhere).
	ErrConflict = errors.New("conflito de estado")
)
