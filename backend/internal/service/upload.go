package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"path"
	"strings"
	"unicode"

	"github.com/morfostech/morfos-finance/internal/domain"
)

// normalizeUploadFilename keeps the user-facing basename while removing path
// components and control characters supplied by multipart clients.
func normalizeUploadFilename(filename string) (string, error) {
	filename = strings.ReplaceAll(filename, "\\", "/")
	filename = strings.TrimSpace(path.Base(filename))
	filename = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, filename)
	filename = strings.TrimSpace(filename)
	if filename == "" || filename == "." {
		return "", fmt.Errorf("%w: nome do arquivo é obrigatório", domain.ErrValidation)
	}
	if len([]byte(filename)) > 255 {
		return "", fmt.Errorf("%w: nome do arquivo excede 255 bytes", domain.ErrValidation)
	}
	return filename, nil
}

// Upload is an incoming file to be stored.
type Upload struct {
	Filename    string
	ContentType string
	Size        int64
	Data        io.Reader
	Descricao   *string
}

// Allowed extensions per attachment kind.
var (
	comprovanteExts = map[string]string{
		".pdf":  "application/pdf",
		".png":  "image/png",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
	}
	proposalExts = map[string]domain.ProposalType{
		".pdf":  domain.ProposalPDF,
		".docx": domain.ProposalDOCX,
	}
)

// validateComprovante checks a receipt upload (PDF/PNG/JPG/JPEG) and returns the
// canonical content type for storage.
func validateComprovante(u Upload, maxBytes int64) (contentType string, err error) {
	if err := checkSize(u.Size, maxBytes); err != nil {
		return "", err
	}
	ext := strings.ToLower(path.Ext(u.Filename))
	ct, ok := comprovanteExts[ext]
	if !ok {
		return "", fmt.Errorf("%w: comprovante deve ser PDF, PNG, JPG ou JPEG", domain.ErrValidation)
	}
	return ct, nil
}

// validateProposal checks a proposal upload (PDF/DOCX) and returns its type and
// content type for storage.
func validateProposal(u Upload, maxBytes int64) (domain.ProposalType, string, error) {
	if err := checkSize(u.Size, maxBytes); err != nil {
		return "", "", err
	}
	ext := strings.ToLower(path.Ext(u.Filename))
	t, ok := proposalExts[ext]
	if !ok {
		return "", "", fmt.Errorf("%w: proposta deve ser PDF ou DOCX", domain.ErrValidation)
	}
	ct := "application/pdf"
	if t == domain.ProposalDOCX {
		ct = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	}
	return t, ct, nil
}

func checkSize(size, maxBytes int64) error {
	if size <= 0 {
		return fmt.Errorf("%w: arquivo vazio", domain.ErrValidation)
	}
	if size > maxBytes {
		return fmt.Errorf("%w: arquivo excede o limite de %d MB", domain.ErrValidation, maxBytes/(1024*1024))
	}
	return nil
}

// objectKey builds a collision-resistant storage key: prefix/<random><ext>.
func objectKey(prefix, filename string) string {
	buf := make([]byte, 12)
	_, _ = rand.Read(buf)
	ext := strings.ToLower(path.Ext(filename))
	return fmt.Sprintf("%s/%s%s", strings.Trim(prefix, "/"), hex.EncodeToString(buf), ext)
}
