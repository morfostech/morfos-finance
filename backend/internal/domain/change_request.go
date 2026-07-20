package domain

import (
	"encoding/json"
	"time"
)

type ChangeRequestAction string

const (
	ChangeNoteCreate ChangeRequestAction = "note_create"
	ChangeNoteUpdate ChangeRequestAction = "note_update"
	ChangeNoteDelete ChangeRequestAction = "note_delete"
)

func (a ChangeRequestAction) Valid() bool {
	switch a {
	case ChangeNoteCreate, ChangeNoteUpdate, ChangeNoteDelete:
		return true
	default:
		return false
	}
}

type ChangeRequestStatus string

const (
	ChangePending    ChangeRequestStatus = "pending"
	ChangeProcessing ChangeRequestStatus = "processing"
	ChangeApproved   ChangeRequestStatus = "approved"
	ChangeRejected   ChangeRequestStatus = "rejected"
)

type ChangeRequest struct {
	ID            int64               `json:"id"`
	RequesterID   int64               `json:"requester_id"`
	RequesterName string              `json:"requester_name"`
	Action        ChangeRequestAction `json:"action"`
	Payload       json.RawMessage     `json:"payload"`
	Status        ChangeRequestStatus `json:"status"`
	ReviewerID    *int64              `json:"reviewer_id,omitempty"`
	ReviewerName  *string             `json:"reviewer_name,omitempty"`
	ReviewComment *string             `json:"review_comment,omitempty"`
	CreatedAt     time.Time           `json:"created_at"`
	ReviewedAt    *time.Time          `json:"reviewed_at,omitempty"`
}
