package domain

import "time"

// NoteOwner is what a note is attached to. Geral notes are free-standing
// (owner_id is nil).
type NoteOwner string

const (
	NoteOwnerProject     NoteOwner = "project"
	NoteOwnerTransaction NoteOwner = "transaction"
	NoteOwnerInstallment NoteOwner = "installment"
	NoteOwnerGeral       NoteOwner = "geral"
)

func (o NoteOwner) Valid() bool {
	switch o {
	case NoteOwnerProject, NoteOwnerTransaction, NoteOwnerInstallment, NoteOwnerGeral:
		return true
	default:
		return false
	}
}

// Note is scoped to UserID. Collaborator mutations are reviewed by an admin or
// partner before the requested change is applied.
type Note struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	OwnerType NoteOwner `json:"owner_type"`
	OwnerID   *int64    `json:"owner_id,omitempty"`
	Texto     string    `json:"texto"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
