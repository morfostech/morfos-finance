package service

import (
	"errors"
	"strings"
	"testing"

	"github.com/morfostech/morfos-finance/internal/domain"
)

const mb = 1024 * 1024

func TestValidateComprovante(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		size     int64
		wantCT   string
		wantErr  error
	}{
		{"pdf", "recibo.pdf", 1000, "application/pdf", nil},
		{"png", "foto.PNG", 1000, "image/png", nil},
		{"jpg", "scan.jpg", 1000, "image/jpeg", nil},
		{"jpeg", "scan.jpeg", 1000, "image/jpeg", nil},
		{"docx rejeitado", "recibo.docx", 1000, "", domain.ErrValidation},
		{"exe rejeitado", "malware.exe", 1000, "", domain.ErrValidation},
		{"vazio", "recibo.pdf", 0, "", domain.ErrValidation},
		{"grande demais", "recibo.pdf", 11 * mb, "", domain.ErrValidation},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ct, err := validateComprovante(Upload{Filename: tc.filename, Size: tc.size}, 10*mb)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if ct != tc.wantCT {
				t.Errorf("contentType = %q, want %q", ct, tc.wantCT)
			}
		})
	}
}

func TestValidateProposal(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		size     int64
		wantType domain.ProposalType
		wantErr  error
	}{
		{"pdf", "proposta.pdf", 1000, domain.ProposalPDF, nil},
		{"docx", "proposta.docx", 1000, domain.ProposalDOCX, nil},
		{"png rejeitado", "proposta.png", 1000, "", domain.ErrValidation},
		{"grande demais", "proposta.pdf", 11 * mb, "", domain.ErrValidation},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			typ, _, err := validateProposal(Upload{Filename: tc.filename, Size: tc.size}, 10*mb)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if typ != tc.wantType {
				t.Errorf("type = %q, want %q", typ, tc.wantType)
			}
		})
	}
}

func TestObjectKeyUniqueAndScoped(t *testing.T) {
	k1 := objectKey("comprovantes/transaction/5", "recibo.pdf")
	k2 := objectKey("comprovantes/transaction/5", "recibo.pdf")
	if k1 == k2 {
		t.Fatal("keys should be unique per call")
	}
	for _, k := range []string{k1, k2} {
		if !strings.HasPrefix(k, "comprovantes/transaction/5/") {
			t.Errorf("key %q missing scoped prefix", k)
		}
		if !strings.HasSuffix(k, ".pdf") {
			t.Errorf("key %q lost extension", k)
		}
	}
}

func TestNormalizeUploadFilename(t *testing.T) {
	tests := []struct {
		input   string
		want    string
		wantErr error
	}{
		{"Proposta Comercial 2026.pdf", "Proposta Comercial 2026.pdf", nil},
		{"C:\\fakepath\\Contrato Final.DOCX", "Contrato Final.DOCX", nil},
		{"../../recibo julho.pdf", "recibo julho.pdf", nil},
		{"relatorio\nfinal.pdf", "relatoriofinal.pdf", nil},
		{"   ", "", domain.ErrValidation},
	}
	for _, tc := range tests {
		got, err := normalizeUploadFilename(tc.input)
		if !errors.Is(err, tc.wantErr) {
			t.Fatalf("normalizeUploadFilename(%q) err = %v, want %v", tc.input, err, tc.wantErr)
		}
		if got != tc.want {
			t.Errorf("normalizeUploadFilename(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestKeyFromURL(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"/uploads/comprovantes/transaction/5/abc.pdf", "comprovantes/transaction/5/abc.pdf"},
		{"https://cdn.example.com/propostas/3/xyz.docx", "propostas/3/xyz.docx"},
	}
	for _, tc := range tests {
		if got := keyFromURL(tc.url); got != tc.want {
			t.Errorf("keyFromURL(%q) = %q, want %q", tc.url, got, tc.want)
		}
	}
}
