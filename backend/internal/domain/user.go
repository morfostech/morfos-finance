package domain

import "time"

// Role is the user's authorization level.
type Role string

const (
	RoleAdmin       Role = "admin"       // vê e edita tudo; gerencia usuários
	RoleSocio       Role = "socio"       // visão financeira completa, somente leitura
	RoleColaborador Role = "colaborador" // vê apenas a própria área
)

// Valid reports whether r is a known role.
func (r Role) Valid() bool {
	switch r {
	case RoleAdmin, RoleSocio, RoleColaborador:
		return true
	default:
		return false
	}
}

// User is an authenticated principal. SenhaHash never leaves the backend.
type User struct {
	ID                 int64     `json:"id"`
	Nome               string    `json:"nome"`
	Email              string    `json:"email"`
	SenhaHash          string    `json:"-"`
	Role               Role      `json:"role"`
	MustChangePassword bool      `json:"must_change_password"`
	Ativo              bool      `json:"ativo"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}
