package handlers

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/morfostech/morfos-finance/internal/domain"
	"github.com/morfostech/morfos-finance/internal/http/middleware"
	"github.com/morfostech/morfos-finance/internal/http/respond"
	"github.com/morfostech/morfos-finance/internal/service"
)

type AttachmentHandler struct {
	attachments *service.AttachmentService
	maxBytes    int64
}

func NewAttachmentHandler(attachments *service.AttachmentService, maxBytes int64) *AttachmentHandler {
	return &AttachmentHandler{attachments: attachments, maxBytes: maxBytes}
}

// AttachToTransaction handles POST /api/transactions/{id}/attachments (multipart).
func (h *AttachmentHandler) AttachToTransaction(w http.ResponseWriter, r *http.Request) {
	txID, ok := pathID(w, r)
	if !ok {
		return
	}
	up, ok := h.readUpload(w, r)
	if !ok {
		return
	}
	defer up.close()

	p, _ := middleware.PrincipalFrom(r.Context())
	a, err := h.attachments.AttachToTransaction(r.Context(), txID, up.Upload, p.UserID)
	if err != nil {
		respond.DomainError(w, err)
		return
	}
	respond.JSON(w, http.StatusCreated, a)
}

// AttachToInstallment handles POST /api/projects/{id}/installments/{iid}/attachments.
func (h *AttachmentHandler) AttachToInstallment(w http.ResponseWriter, r *http.Request) {
	projectID, ok := pathID(w, r)
	if !ok {
		return
	}
	installmentID, err := strconv.ParseInt(chi.URLParam(r, "iid"), 10, 64)
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "id da parcela inválido")
		return
	}
	up, ok := h.readUpload(w, r)
	if !ok {
		return
	}
	defer up.close()

	p, _ := middleware.PrincipalFrom(r.Context())
	a, err := h.attachments.AttachToInstallment(r.Context(), projectID, installmentID, up.Upload, p.UserID)
	if err != nil {
		respond.DomainError(w, err)
		return
	}
	respond.JSON(w, http.StatusCreated, a)
}

// AttachProposal handles POST /api/projects/{id}/proposals.
func (h *AttachmentHandler) AttachProposal(w http.ResponseWriter, r *http.Request) {
	projectID, ok := pathID(w, r)
	if !ok {
		return
	}
	up, ok := h.readUpload(w, r)
	if !ok {
		return
	}
	defer up.close()

	p, _ := middleware.PrincipalFrom(r.Context())
	proposal, err := h.attachments.AttachProposal(r.Context(), projectID, up.Upload, p.UserID)
	if err != nil {
		respond.DomainError(w, err)
		return
	}
	respond.JSON(w, http.StatusCreated, proposal)
}

func (h *AttachmentHandler) ListTransactionAttachments(w http.ResponseWriter, r *http.Request) {
	txID, ok := pathID(w, r)
	if !ok {
		return
	}
	list, err := h.attachments.ListTransactionAttachments(r.Context(), txID)
	if err != nil {
		respond.DomainError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, orEmptyAttachments(list))
}

func (h *AttachmentHandler) ListProposals(w http.ResponseWriter, r *http.Request) {
	projectID, ok := pathID(w, r)
	if !ok {
		return
	}
	list, err := h.attachments.ListProposals(r.Context(), projectID)
	if err != nil {
		respond.DomainError(w, err)
		return
	}
	if list == nil {
		list = []domain.Proposal{}
	}
	respond.JSON(w, http.StatusOK, list)
}

func (h *AttachmentHandler) DeleteAttachment(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if err := h.attachments.DeleteAttachment(r.Context(), id); err != nil {
		respond.DomainError(w, err)
		return
	}
	respond.JSON(w, http.StatusNoContent, nil)
}

func (h *AttachmentHandler) DeleteProposal(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if err := h.attachments.DeleteProposal(r.Context(), id); err != nil {
		respond.DomainError(w, err)
		return
	}
	respond.JSON(w, http.StatusNoContent, nil)
}

// uploadWithCloser bundles the parsed Upload with the multipart file to close.
type uploadWithCloser struct {
	service.Upload
	closeFn func()
}

func (u uploadWithCloser) close() {
	if u.closeFn != nil {
		u.closeFn()
	}
}

// readUpload parses a multipart form with a "file" part and optional "descricao".
func (h *AttachmentHandler) readUpload(w http.ResponseWriter, r *http.Request) (uploadWithCloser, bool) {
	// Cap the request body to the configured max plus a small headroom for the
	// multipart envelope.
	r.Body = http.MaxBytesReader(w, r.Body, h.maxBytes+1<<20)
	if err := r.ParseMultipartForm(h.maxBytes + 1<<20); err != nil {
		respond.Error(w, http.StatusBadRequest, "upload inválido ou grande demais")
		return uploadWithCloser{}, false
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "campo 'file' é obrigatório")
		return uploadWithCloser{}, false
	}

	up := uploadWithCloser{
		Upload: service.Upload{
			Filename:    header.Filename,
			ContentType: header.Header.Get("Content-Type"),
			Size:        header.Size,
			Data:        file,
			Descricao:   optionalForm(r, "descricao"),
		},
		closeFn: func() { _ = file.Close() },
	}
	return up, true
}

func optionalForm(r *http.Request, key string) *string {
	if v := r.FormValue(key); v != "" {
		return &v
	}
	return nil
}

func orEmptyAttachments(list []domain.Attachment) []domain.Attachment {
	if list == nil {
		return []domain.Attachment{}
	}
	return list
}
