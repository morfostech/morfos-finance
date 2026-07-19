package domain

import "time"

// AttachmentOwner is what an attachment is attached to.
type AttachmentOwner string

const (
	OwnerTransaction AttachmentOwner = "transaction"
	OwnerInstallment AttachmentOwner = "installment"
)

func (o AttachmentOwner) Valid() bool {
	return o == OwnerTransaction || o == OwnerInstallment
}

// Attachment is a payment receipt (comprovante) attached to a transaction or an
// installment.
type Attachment struct {
	ID        int64           `json:"id"`
	OwnerType AttachmentOwner `json:"owner_type"`
	OwnerID   int64           `json:"owner_id"`
	URL       string          `json:"url"`
	Descricao *string         `json:"descricao,omitempty"`
	CreatedBy *int64          `json:"created_by,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
}

// ProposalType is the file format of a commercial proposal.
type ProposalType string

const (
	ProposalPDF  ProposalType = "pdf"
	ProposalDOCX ProposalType = "docx"
)

func (t ProposalType) Valid() bool {
	return t == ProposalPDF || t == ProposalDOCX
}

// Proposal is a commercial proposal document attached to a project. A project
// may hold several versions.
type Proposal struct {
	ID          int64        `json:"id"`
	ProjectID   int64        `json:"project_id"`
	URL         string       `json:"url"`
	ArquivoTipo ProposalType `json:"arquivo_tipo"`
	Descricao   *string      `json:"descricao,omitempty"`
	CreatedBy   *int64       `json:"created_by,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
}
