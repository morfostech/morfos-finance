package domain

import "time"

// VPTransactionType identifies whether VP enters or leaves the company's
// Via Permuta account. VP is tracked separately from BRL cash.
type VPTransactionType string

const (
	VPVenda  VPTransactionType = "venda"
	VPCompra VPTransactionType = "compra"
)

func (t VPTransactionType) Valid() bool { return t == VPVenda || t == VPCompra }

type VPTransactionStatus string

const (
	VPNegociando VPTransactionStatus = "negociando"
	VPConcluida  VPTransactionStatus = "concluida"
	VPRecusada   VPTransactionStatus = "recusada"
	VPCancelada  VPTransactionStatus = "cancelada"
)

func (s VPTransactionStatus) Valid() bool {
	switch s {
	case VPNegociando, VPConcluida, VPRecusada, VPCancelada:
		return true
	default:
		return false
	}
}

type VPOfferStatus string

const (
	VPOfferAberta    VPOfferStatus = "aberta"
	VPOfferLiberada  VPOfferStatus = "liberada"
	VPOfferPendente  VPOfferStatus = "pendente"
	VPOfferBloqueada VPOfferStatus = "bloqueada"
	VPOfferEncerrada VPOfferStatus = "encerrada"
)

func (s VPOfferStatus) Valid() bool {
	switch s {
	case VPOfferAberta, VPOfferLiberada, VPOfferPendente, VPOfferBloqueada, VPOfferEncerrada:
		return true
	default:
		return false
	}
}

// VPSettings holds account-level controls. There is one row for the company.
type VPSettings struct {
	CreditLimit Money     `json:"limite_credito"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type VPTransaction struct {
	ID          int64               `json:"id"`
	Tipo        VPTransactionType   `json:"tipo"`
	Status      VPTransactionStatus `json:"status"`
	Valor       Money               `json:"valor"`
	Data        Date                `json:"data"`
	Permutante  string              `json:"permutante"`
	Oferta      string              `json:"oferta"`
	ProjectID   *int64              `json:"project_id,omitempty"`
	VoucherCode *string             `json:"voucher_code,omitempty"`
	Observacoes *string             `json:"observacoes,omitempty"`
	CreatedBy   int64               `json:"created_by"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
}

type VPTransactionFilter struct {
	From   *Date
	To     *Date
	Tipo   *VPTransactionType
	Status *VPTransactionStatus
}

type VPOffer struct {
	ID          int64         `json:"id"`
	Titulo      string        `json:"titulo"`
	Descricao   *string       `json:"descricao,omitempty"`
	Valor       *Money        `json:"valor,omitempty"`
	Negociavel  bool          `json:"negociavel"`
	Status      VPOfferStatus `json:"status"`
	ExternalURL *string       `json:"external_url,omitempty"`
	CreatedBy   int64         `json:"created_by"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// VPSummary is deliberately separate from the BRL dashboard. Saldo is the
// settled VP ledger; Disponivel combines it with the approved VP credit limit.
type VPSummary struct {
	Saldo               Money `json:"saldo"`
	LimiteCredito       Money `json:"limite_credito"`
	Disponivel          Money `json:"disponivel"`
	VendasPeriodo       Money `json:"vendas_periodo"`
	ComprasPeriodo      Money `json:"compras_periodo"`
	TicketMedioVenda    Money `json:"ticket_medio_venda"`
	TicketMedioCompra   Money `json:"ticket_medio_compra"`
	NegociacoesAbertas  int64 `json:"negociacoes_abertas"`
	OfertasAbertas      int64 `json:"ofertas_abertas"`
	OfertasLiberadas    int64 `json:"ofertas_liberadas"`
	OfertasComPendencia int64 `json:"ofertas_com_pendencia"`
}
