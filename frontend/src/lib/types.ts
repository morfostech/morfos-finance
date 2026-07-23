// API types. Monetary values are integer centavos.

export type Role = "admin" | "socio" | "colaborador";

/** Admin and sócio share full management access; only colaborador is scoped down. */
export function canManage(role?: Role): boolean {
  return role === "admin" || role === "socio";
}

export interface User {
  id: number;
  nome: string;
  email: string;
  role: Role;
  must_change_password: boolean;
  ativo: boolean;
}

export interface LoginResponse {
  token: string;
  expires_at: string;
  user: User;
}

export type ProjectStatus = "ativo" | "pausado" | "concluido" | "cancelado";
export type InstallmentType = "entrada" | "finalizacao";

export interface Installment {
  id: number;
  project_id: number;
  tipo: InstallmentType;
  valor: number;
  pago_em?: string;
  pago: boolean;
}

export interface Project {
  id: number;
  nome: string;
  cliente?: string;
  valor_implementacao?: number;
  valor_mensal?: number;
  dia_vencimento?: number;
  data_inicio?: string;
  data_fim?: string;
  status: ProjectStatus;
  installments?: Installment[];
  member_ids?: number[];
}

export type TxType = "ganho" | "despesa";
export type TxOrigem = "implementacao" | "recorrencia" | "avulso";

export interface Transaction {
  id: number;
  tipo: TxType;
  valor: number;
  data: string;
  project_id?: number;
  user_id?: number;
  origem?: TxOrigem;
  category_id?: number;
  descricao?: string;
  installment_id?: number;
  created_by: number;
}

export interface Category {
  id: number;
  nome: string;
}

export type VPTransactionType = "venda" | "compra";
export type VPTransactionStatus = "negociando" | "concluida" | "recusada" | "cancelada";
export type VPOfferStatus = "aberta" | "liberada" | "pendente" | "bloqueada" | "encerrada";

export interface VPTransaction {
  id: number;
  tipo: VPTransactionType;
  status: VPTransactionStatus;
  valor: number;
  data: string;
  permutante: string;
  oferta: string;
  project_id?: number;
  voucher_code?: string;
  observacoes?: string;
  created_by: number;
}

export interface VPOffer {
  id: number;
  titulo: string;
  descricao?: string;
  valor?: number;
  negociavel: boolean;
  status: VPOfferStatus;
  external_url?: string;
  created_by: number;
}

export interface VPSummary {
  saldo: number;
  limite_credito: number;
  disponivel: number;
  vendas_periodo: number;
  compras_periodo: number;
  ticket_medio_venda: number;
  ticket_medio_compra: number;
  negociacoes_abertas: number;
  ofertas_abertas: number;
  ofertas_liberadas: number;
  ofertas_com_pendencia: number;
}

export interface VPSettings {
  limite_credito: number;
  updated_at: string;
}

export interface ProjectRecurrence {
  project_id: number;
  nome: string;
  previsto: number;
  recebido: number;
  pendente: number;
  vencido: number;
  a_vencer: number;
  vencimento?: string;
  situacao: "quitado" | "parcial" | "vencido" | "a_vencer" | "recebido";
  ativo: boolean;
}

export interface RecurrenceSummary {
  ano: number;
  mes: number;
  previsto: number;
  recebido: number;
  pendente: number;
  vencido: number;
  a_vencer: number;
  projetos: ProjectRecurrence[];
}

export interface RecurrenceForecast {
  horizonte_meses: number;
  total: number;
  meses: { ano: number; mes: number; previsto: number }[];
}

export interface RecurrencePeriod {
  previsto: number;
  recebido: number;
  pendente: number;
  vencido: number;
  a_vencer: number;
  meses: { ano: number; mes: number; previsto: number; recebido: number; pendente: number; vencido: number; a_vencer: number }[];
}

export interface CategoryTotal {
  category_id: number | null;
  nome: string;
  total: number;
}

export interface CompanyDashboard {
  periodo: { from: string; to: string };
  saldo_em_caixa: number;
  ganhos: number;
  despesas: number;
  resultado: number;
  ganhos_por_origem: {
    implementacao: number;
    recorrencia: number;
    avulso: number;
    sem_origem: number;
  };
  despesas_por_categoria: CategoryTotal[];
  implementacao: { total: number; recebido: number; a_receber: number };
  parcelas_pendentes: { quantidade: number; total: number };
  recorrencia_mes: RecurrenceSummary;
  recorrencia_periodo: RecurrencePeriod;
  recorrencia_futura?: RecurrenceForecast;
  por_projeto: { project_id: number; nome: string; ganhos: number; despesas: number }[];
  por_colaborador: { user_id: number; nome: string; ganhos: number; despesas: number }[];
}

export interface MeDashboard {
  periodo: { from: string; to: string };
  ganhos: number;
  despesas: number;
  saldo: number;
  projetos: Project[];
}

export type PlannedStatus = "aberto" | "realizado";

export interface PlannedEntry {
  id: number;
  tipo: TxType;
  valor: number;
  due_date: string;
  project_id?: number;
  user_id?: number;
  origem?: TxOrigem;
  category_id?: number;
  descricao: string;
  actual_transaction_id?: number;
  status: PlannedStatus;
  overdue: boolean;
}

export interface CashFlowForecast {
  periodo: { from: string; to: string };
  saldo_inicial: number;
  entradas: number;
  entradas_automaticas: number;
  entradas_manuais: number;
  entradas_confirmadas: number;
  saidas: number;
  saidas_manuais: number;
  saidas_confirmadas: number;
  saldo_final: number;
  vencidos: number;
  dias: {
    data: string;
    entradas: number;
    saidas: number;
    saldo_projetado: number;
    itens: { tipo: TxType; valor: number; descricao: string; project_id?: number; origem?: TxOrigem; automatico: boolean; confirmado: boolean }[];
  }[];
}

export interface ExpenseBudget {
  id: number;
  category_id: number;
  category: string;
  ano: number;
  mes: number;
  valor: number;
  realizado: number;
  restante: number;
  percentual: number;
}

export interface Attachment {
  id: number;
  owner_type: "transaction" | "installment";
  owner_id: number;
  url: string;
  nome_arquivo?: string;
  descricao?: string;
}

export interface Proposal {
  id: number;
  project_id: number;
  url: string;
  arquivo_tipo: "pdf" | "docx";
  nome_arquivo?: string;
  descricao?: string;
}

export type NoteOwner = "project" | "transaction" | "installment" | "geral";

export interface Note {
  id: number;
  user_id: number;
  owner_type: NoteOwner;
  owner_id?: number;
  texto: string;
  created_at: string;
  updated_at: string;
}

export type ChangeRequestAction = "note_create" | "note_update" | "note_delete";
export type ChangeRequestStatus = "pending" | "approved" | "rejected";

export interface NoteChangePayload {
  note_id?: number;
  owner_type?: NoteOwner;
  owner_id?: number;
  texto?: string;
}

export interface ChangeRequest {
  id: number;
  requester_id: number;
  requester_name: string;
  action: ChangeRequestAction;
  payload: NoteChangePayload;
  status: ChangeRequestStatus;
  reviewer_id?: number;
  reviewer_name?: string;
  review_comment?: string;
  created_at: string;
  reviewed_at?: string;
}
