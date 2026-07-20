package service

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/morfostech/morfos-finance/internal/domain"
)

type filenameAttachmentRepo struct {
	attachment *domain.Attachment
	proposal   *domain.Proposal
}

func (r *filenameAttachmentRepo) Create(_ context.Context, a *domain.Attachment) (*domain.Attachment, error) {
	r.attachment = a
	return a, nil
}
func (*filenameAttachmentRepo) ListByOwner(context.Context, domain.AttachmentOwner, int64) ([]domain.Attachment, error) {
	return nil, nil
}
func (*filenameAttachmentRepo) Get(context.Context, int64) (*domain.Attachment, error) {
	return nil, domain.ErrNotFound
}
func (*filenameAttachmentRepo) Delete(context.Context, int64) error { return nil }
func (r *filenameAttachmentRepo) CreateProposal(_ context.Context, p *domain.Proposal) (*domain.Proposal, error) {
	r.proposal = p
	return p, nil
}
func (*filenameAttachmentRepo) ListProposals(context.Context, int64) ([]domain.Proposal, error) {
	return nil, nil
}
func (*filenameAttachmentRepo) GetProposal(context.Context, int64) (*domain.Proposal, error) {
	return nil, domain.ErrNotFound
}
func (*filenameAttachmentRepo) DeleteProposal(context.Context, int64) error { return nil }

type filenameStore struct{}

func (filenameStore) Put(_ context.Context, key, _ string, _ io.Reader, _ int64) (string, error) {
	return "/uploads/" + key, nil
}
func (filenameStore) Delete(context.Context, string) error { return nil }

type existingTransaction struct{}

func (existingTransaction) GetByID(context.Context, int64) (*domain.Transaction, error) {
	return &domain.Transaction{ID: 1}, nil
}

type existingProject struct{}

func (existingProject) GetByID(context.Context, int64) (*domain.Project, error) {
	return &domain.Project{ID: 1}, nil
}

func TestAttachmentServicePreservesOriginalBasename(t *testing.T) {
	repo := &filenameAttachmentRepo{}
	svc := NewAttachmentService(repo, filenameStore{}, existingTransaction{}, nil, existingProject{}, 10*mb)

	proposal, err := svc.AttachProposal(context.Background(), 1, Upload{
		Filename: `C:\fakepath\Proposta Comercial Final.PDF`,
		Size:     4,
		Data:     strings.NewReader("test"),
	}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if proposal.NomeArquivo == nil || *proposal.NomeArquivo != "Proposta Comercial Final.PDF" {
		t.Fatalf("proposal nome_arquivo = %v", proposal.NomeArquivo)
	}

	attachment, err := svc.AttachToTransaction(context.Background(), 1, Upload{
		Filename: "Comprovante Cliente Julho.pdf",
		Size:     4,
		Data:     strings.NewReader("test"),
	}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if attachment.NomeArquivo == nil || *attachment.NomeArquivo != "Comprovante Cliente Julho.pdf" {
		t.Fatalf("attachment nome_arquivo = %v", attachment.NomeArquivo)
	}
}
