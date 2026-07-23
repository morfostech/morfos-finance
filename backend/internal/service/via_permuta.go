package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/morfostech/morfos-finance/internal/domain"
)

type ViaPermutaRepo interface {
	CreateTransaction(context.Context, *domain.VPTransaction) (*domain.VPTransaction, error)
	GetTransaction(context.Context, int64) (*domain.VPTransaction, error)
	UpdateTransaction(context.Context, *domain.VPTransaction) (*domain.VPTransaction, error)
	DeleteTransaction(context.Context, int64) error
	ListTransactions(context.Context, domain.VPTransactionFilter) ([]domain.VPTransaction, error)
	GetSettings(context.Context) (*domain.VPSettings, error)
	UpdateSettings(context.Context, domain.Money) (*domain.VPSettings, error)
	Summary(context.Context, domain.VPTransactionFilter) (*domain.VPSummary, error)
	CreateOffer(context.Context, *domain.VPOffer) (*domain.VPOffer, error)
	GetOffer(context.Context, int64) (*domain.VPOffer, error)
	UpdateOffer(context.Context, *domain.VPOffer) (*domain.VPOffer, error)
	DeleteOffer(context.Context, int64) error
	ListOffers(context.Context) ([]domain.VPOffer, error)
}

type ViaPermutaService struct {
	repo ViaPermutaRepo
}

func NewViaPermutaService(repo ViaPermutaRepo) *ViaPermutaService {
	return &ViaPermutaService{repo: repo}
}

type VPTransactionInput struct {
	Tipo        domain.VPTransactionType
	Status      domain.VPTransactionStatus
	Valor       domain.Money
	Data        *domain.Date
	Permutante  string
	Oferta      string
	ProjectID   *int64
	VoucherCode *string
	Observacoes *string
}

func buildVPTransaction(in VPTransactionInput) (*domain.VPTransaction, error) {
	if !in.Tipo.Valid() {
		return nil, fmt.Errorf("%w: tipo de movimentação VP inválido", domain.ErrValidation)
	}
	if !in.Status.Valid() {
		return nil, fmt.Errorf("%w: status de movimentação VP inválido", domain.ErrValidation)
	}
	if in.Valor <= 0 {
		return nil, fmt.Errorf("%w: valor VP deve ser positivo", domain.ErrValidation)
	}
	if in.Data == nil {
		return nil, fmt.Errorf("%w: data é obrigatória", domain.ErrValidation)
	}
	permutante := strings.TrimSpace(in.Permutante)
	if len(permutante) < 2 {
		return nil, fmt.Errorf("%w: informe o associado permutante", domain.ErrValidation)
	}
	oferta := strings.TrimSpace(in.Oferta)
	if len(oferta) < 2 {
		return nil, fmt.Errorf("%w: informe a oferta ou serviço", domain.ErrValidation)
	}
	return &domain.VPTransaction{
		Tipo: in.Tipo, Status: in.Status, Valor: in.Valor, Data: *in.Data,
		Permutante: permutante, Oferta: oferta, ProjectID: in.ProjectID,
		VoucherCode: trimPtr(in.VoucherCode), Observacoes: trimPtr(in.Observacoes),
	}, nil
}

func (s *ViaPermutaService) CreateTransaction(ctx context.Context, in VPTransactionInput, createdBy int64) (*domain.VPTransaction, error) {
	t, err := buildVPTransaction(in)
	if err != nil {
		return nil, err
	}
	t.CreatedBy = createdBy
	return s.repo.CreateTransaction(ctx, t)
}

func (s *ViaPermutaService) UpdateTransaction(ctx context.Context, id int64, in VPTransactionInput) (*domain.VPTransaction, error) {
	if _, err := s.repo.GetTransaction(ctx, id); err != nil {
		return nil, err
	}
	t, err := buildVPTransaction(in)
	if err != nil {
		return nil, err
	}
	t.ID = id
	return s.repo.UpdateTransaction(ctx, t)
}

func (s *ViaPermutaService) DeleteTransaction(ctx context.Context, id int64) error {
	return s.repo.DeleteTransaction(ctx, id)
}

func (s *ViaPermutaService) ListTransactions(ctx context.Context, f domain.VPTransactionFilter) ([]domain.VPTransaction, error) {
	if f.Tipo != nil && !f.Tipo.Valid() {
		return nil, fmt.Errorf("%w: tipo VP inválido", domain.ErrValidation)
	}
	if f.Status != nil && !f.Status.Valid() {
		return nil, fmt.Errorf("%w: status VP inválido", domain.ErrValidation)
	}
	return s.repo.ListTransactions(ctx, f)
}

func (s *ViaPermutaService) Summary(ctx context.Context, f domain.VPTransactionFilter) (*domain.VPSummary, error) {
	return s.repo.Summary(ctx, f)
}

func (s *ViaPermutaService) GetSettings(ctx context.Context) (*domain.VPSettings, error) {
	return s.repo.GetSettings(ctx)
}

func (s *ViaPermutaService) UpdateSettings(ctx context.Context, creditLimit domain.Money) (*domain.VPSettings, error) {
	if creditLimit < 0 {
		return nil, fmt.Errorf("%w: limite de crédito não pode ser negativo", domain.ErrValidation)
	}
	return s.repo.UpdateSettings(ctx, creditLimit)
}

type VPOfferInput struct {
	Titulo      string
	Descricao   *string
	Valor       *domain.Money
	Negociavel  bool
	Status      domain.VPOfferStatus
	ExternalURL *string
}

func buildVPOffer(in VPOfferInput) (*domain.VPOffer, error) {
	titulo := strings.TrimSpace(in.Titulo)
	if len(titulo) < 3 {
		return nil, fmt.Errorf("%w: título da oferta é obrigatório", domain.ErrValidation)
	}
	if !in.Status.Valid() {
		return nil, fmt.Errorf("%w: status da oferta inválido", domain.ErrValidation)
	}
	if !in.Negociavel && in.Valor == nil {
		return nil, fmt.Errorf("%w: informe o valor ou marque a oferta como negociável", domain.ErrValidation)
	}
	if in.Valor != nil && *in.Valor <= 0 {
		return nil, fmt.Errorf("%w: valor da oferta deve ser positivo", domain.ErrValidation)
	}
	return &domain.VPOffer{
		Titulo: titulo, Descricao: trimPtr(in.Descricao), Valor: in.Valor,
		Negociavel: in.Negociavel, Status: in.Status, ExternalURL: trimPtr(in.ExternalURL),
	}, nil
}

func (s *ViaPermutaService) CreateOffer(ctx context.Context, in VPOfferInput, createdBy int64) (*domain.VPOffer, error) {
	o, err := buildVPOffer(in)
	if err != nil {
		return nil, err
	}
	o.CreatedBy = createdBy
	return s.repo.CreateOffer(ctx, o)
}

func (s *ViaPermutaService) UpdateOffer(ctx context.Context, id int64, in VPOfferInput) (*domain.VPOffer, error) {
	if _, err := s.repo.GetOffer(ctx, id); err != nil {
		return nil, err
	}
	o, err := buildVPOffer(in)
	if err != nil {
		return nil, err
	}
	o.ID = id
	return s.repo.UpdateOffer(ctx, o)
}

func (s *ViaPermutaService) DeleteOffer(ctx context.Context, id int64) error {
	return s.repo.DeleteOffer(ctx, id)
}

func (s *ViaPermutaService) ListOffers(ctx context.Context) ([]domain.VPOffer, error) {
	return s.repo.ListOffers(ctx)
}
