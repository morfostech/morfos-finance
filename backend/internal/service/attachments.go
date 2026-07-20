package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/morfostech/morfos-finance/internal/domain"
	"github.com/morfostech/morfos-finance/internal/storage"
)

// AttachmentRepo is the persistence contract for receipts and proposals.
type AttachmentRepo interface {
	Create(ctx context.Context, a *domain.Attachment) (*domain.Attachment, error)
	ListByOwner(ctx context.Context, ownerType domain.AttachmentOwner, ownerID int64) ([]domain.Attachment, error)
	Get(ctx context.Context, id int64) (*domain.Attachment, error)
	Delete(ctx context.Context, id int64) error

	CreateProposal(ctx context.Context, p *domain.Proposal) (*domain.Proposal, error)
	ListProposals(ctx context.Context, projectID int64) ([]domain.Proposal, error)
	GetProposal(ctx context.Context, id int64) (*domain.Proposal, error)
	DeleteProposal(ctx context.Context, id int64) error
}

// Narrow existence checks against owning resources, satisfied by the existing
// transaction and project repositories.
type transactionExister interface {
	GetByID(ctx context.Context, id int64) (*domain.Transaction, error)
}
type installmentExister interface {
	GetInstallment(ctx context.Context, projectID, installmentID int64) (*domain.Installment, error)
}
type projectExister interface {
	GetByID(ctx context.Context, id int64) (*domain.Project, error)
}

type AttachmentService struct {
	repo         AttachmentRepo
	store        storage.Storage
	txs          transactionExister
	installments installmentExister
	projects     projectExister
	maxBytes     int64
}

func NewAttachmentService(
	repo AttachmentRepo,
	store storage.Storage,
	txs transactionExister,
	installments installmentExister,
	projects projectExister,
	maxBytes int64,
) *AttachmentService {
	return &AttachmentService{
		repo: repo, store: store, txs: txs, installments: installments,
		projects: projects, maxBytes: maxBytes,
	}
}

// AttachToTransaction stores a receipt for a transaction.
func (s *AttachmentService) AttachToTransaction(ctx context.Context, txID int64, u Upload, createdBy int64) (*domain.Attachment, error) {
	if _, err := s.txs.GetByID(ctx, txID); err != nil {
		return nil, err
	}
	return s.attachComprovante(ctx, domain.OwnerTransaction, txID, u, createdBy)
}

// AttachToInstallment stores a receipt for a project installment.
func (s *AttachmentService) AttachToInstallment(ctx context.Context, projectID, installmentID int64, u Upload, createdBy int64) (*domain.Attachment, error) {
	if _, err := s.installments.GetInstallment(ctx, projectID, installmentID); err != nil {
		return nil, err
	}
	return s.attachComprovante(ctx, domain.OwnerInstallment, installmentID, u, createdBy)
}

func (s *AttachmentService) attachComprovante(ctx context.Context, ownerType domain.AttachmentOwner, ownerID int64, u Upload, createdBy int64) (*domain.Attachment, error) {
	filename, err := normalizeUploadFilename(u.Filename)
	if err != nil {
		return nil, err
	}
	u.Filename = filename
	contentType, err := validateComprovante(u, s.maxBytes)
	if err != nil {
		return nil, err
	}
	key := objectKey(fmt.Sprintf("comprovantes/%s/%d", ownerType, ownerID), u.Filename)
	url, err := s.store.Put(ctx, key, contentType, u.Data, u.Size)
	if err != nil {
		return nil, err
	}
	return s.repo.Create(ctx, &domain.Attachment{
		OwnerType:   ownerType,
		OwnerID:     ownerID,
		URL:         url,
		NomeArquivo: &filename,
		Descricao:   u.Descricao,
		CreatedBy:   &createdBy,
	})
}

func (s *AttachmentService) ListTransactionAttachments(ctx context.Context, txID int64) ([]domain.Attachment, error) {
	return s.repo.ListByOwner(ctx, domain.OwnerTransaction, txID)
}

func (s *AttachmentService) ListInstallmentAttachments(ctx context.Context, installmentID int64) ([]domain.Attachment, error) {
	return s.repo.ListByOwner(ctx, domain.OwnerInstallment, installmentID)
}

// DeleteAttachment removes the DB row and the stored object (best-effort on the
// object, since the row is the source of truth).
func (s *AttachmentService) DeleteAttachment(ctx context.Context, id int64) error {
	a, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	_ = s.store.Delete(ctx, keyFromURL(a.URL))
	return nil
}

// --- proposals ---

// AttachProposal stores a commercial proposal (PDF/DOCX) for a project.
func (s *AttachmentService) AttachProposal(ctx context.Context, projectID int64, u Upload, createdBy int64) (*domain.Proposal, error) {
	if _, err := s.projects.GetByID(ctx, projectID); err != nil {
		return nil, err
	}
	filename, err := normalizeUploadFilename(u.Filename)
	if err != nil {
		return nil, err
	}
	u.Filename = filename
	tipo, contentType, err := validateProposal(u, s.maxBytes)
	if err != nil {
		return nil, err
	}
	key := objectKey(fmt.Sprintf("propostas/%d", projectID), u.Filename)
	url, err := s.store.Put(ctx, key, contentType, u.Data, u.Size)
	if err != nil {
		return nil, err
	}
	return s.repo.CreateProposal(ctx, &domain.Proposal{
		ProjectID:   projectID,
		URL:         url,
		ArquivoTipo: tipo,
		NomeArquivo: &filename,
		Descricao:   u.Descricao,
		CreatedBy:   &createdBy,
	})
}

func (s *AttachmentService) ListProposals(ctx context.Context, projectID int64) ([]domain.Proposal, error) {
	return s.repo.ListProposals(ctx, projectID)
}

func (s *AttachmentService) DeleteProposal(ctx context.Context, id int64) error {
	p, err := s.repo.GetProposal(ctx, id)
	if err != nil {
		return err
	}
	if err := s.repo.DeleteProposal(ctx, id); err != nil {
		return err
	}
	_ = s.store.Delete(ctx, keyFromURL(p.URL))
	return nil
}

// keyFromURL recovers the storage key from a stored URL. Every key starts with
// a known prefix ("comprovantes/" or "propostas/"), so we slice from there —
// works for both the local ("/uploads/<key>") and S3 ("<base>/<key>") forms.
func keyFromURL(url string) string {
	for _, prefix := range []string{"comprovantes/", "propostas/"} {
		if i := strings.Index(url, prefix); i >= 0 {
			return url[i:]
		}
	}
	return url // fall back (Delete is best-effort)
}
