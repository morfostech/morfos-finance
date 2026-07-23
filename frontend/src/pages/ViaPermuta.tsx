import { useState, type FormEvent } from "react";
import { api } from "../lib/api";
import { currentMonthRange, date, todayISO, toCentavos, vpMoney } from "../lib/format";
import { useAsync } from "../lib/hooks";
import type {
  Project,
  VPOffer,
  VPOfferStatus,
  VPSettings,
  VPSummary,
  VPTransaction,
  VPTransactionStatus,
  VPTransactionType,
} from "../lib/types";
import { DatePicker } from "../components/DatePicker";
import { Modal } from "../components/Modal";
import { Empty, ErrorBanner, Select, Spinner } from "../components/ui";
import "./pages.css";
import "./via-permuta.css";

type VPView = "overview" | "transactions" | "offers";

const TRANSACTION_STATUS: Record<VPTransactionStatus, string> = {
  negociando: "Negociando",
  concluida: "Concluída",
  recusada: "Recusada",
  cancelada: "Cancelada",
};

const OFFER_STATUS: Record<VPOfferStatus, string> = {
  aberta: "Aberta",
  liberada: "Liberada",
  pendente: "Pendente",
  bloqueada: "Bloqueada",
  encerrada: "Encerrada",
};

export function ViaPermuta() {
  const initialRange = currentMonthRange();
  const [view, setView] = useState<VPView>("overview");
  const [from, setFrom] = useState(initialRange.from);
  const [to, setTo] = useState(initialRange.to);
  const [tipo, setTipo] = useState("");
  const [status, setStatus] = useState("");
  const [editingTransaction, setEditingTransaction] = useState<VPTransaction | null | undefined>(undefined);
  const [editingOffer, setEditingOffer] = useState<VPOffer | null | undefined>(undefined);
  const [editingLimit, setEditingLimit] = useState(false);

  const summaryQuery = new URLSearchParams({ from, to }).toString();
  const transactionParams = new URLSearchParams();
  if (from) transactionParams.set("from", from);
  if (to) transactionParams.set("to", to);
  if (tipo) transactionParams.set("tipo", tipo);
  if (status) transactionParams.set("status", status);
  const transactionQuery = transactionParams.toString();

  const summary = useAsync<VPSummary>(() => api.get(`/via-permuta/summary?${summaryQuery}`), [summaryQuery]);
  const transactions = useAsync<VPTransaction[]>(
    () => api.get(`/via-permuta/transactions${transactionQuery ? `?${transactionQuery}` : ""}`),
    [transactionQuery],
  );
  const offers = useAsync<VPOffer[]>(() => api.get("/via-permuta/offers"), []);
  const settings = useAsync<VPSettings>(() => api.get("/via-permuta/settings"), []);
  const projects = useAsync<Project[]>(() => api.get("/projects"), []);

  const reloadAll = () => {
    summary.reload();
    transactions.reload();
    offers.reload();
    settings.reload();
  };

  function exportCSV() {
    if (!transactions.data?.length) return;
    const escape = (value: unknown) => `"${String(value ?? "").replace(/"/g, '""')}"`;
    const rows = transactions.data.map((item) => [
      item.data,
      item.tipo,
      item.status,
      (item.valor / 100).toFixed(2).replace(".", ","),
      item.permutante,
      item.oferta,
      item.voucher_code ?? "",
      item.observacoes ?? "",
    ]);
    const csv = "\uFEFF" + [["Data", "Tipo", "Status", "Valor VP", "Permutante", "Oferta", "Voucher", "Observações"], ...rows]
      .map((row) => row.map(escape).join(";"))
      .join("\r\n");
    const url = URL.createObjectURL(new Blob([csv], { type: "text/csv;charset=utf-8" }));
    const link = document.createElement("a");
    link.href = url;
    link.download = `via-permuta-${todayISO()}.csv`;
    link.click();
    URL.revokeObjectURL(url);
  }

  return (
    <div className="vp-page">
      <header className="vp-hero">
        <div>
          <span className="kicker vp-kicker">VP / Via Permuta</span>
          <h1>Ecossistema de permuta</h1>
          <p>Controle VP separado do caixa em reais, com rastreabilidade de vendas, compras, ofertas e negociações.</p>
        </div>
        <a className="btn vp-external" href="https://associadovp.laks.net.br/associado/dashboard" target="_blank" rel="noreferrer">
          Abrir Via Permuta <span aria-hidden>↗</span>
        </a>
      </header>

      <div className="vp-separation-note">
        <span className="vp-mark" aria-hidden>VP</span>
        <div>
          <strong>1 VP acompanha R$ 1,00 como referência de valor</strong>
          <span>VP não é conversível em dinheiro e não compõe saldo em caixa, ganhos ou despesas em R$.</span>
        </div>
      </div>

      <nav className="vp-tabs" aria-label="Áreas da Via Permuta">
        <button className={view === "overview" ? "active" : ""} onClick={() => setView("overview")}>Visão geral</button>
        <button className={view === "transactions" ? "active" : ""} onClick={() => setView("transactions")}>Movimentações</button>
        <button className={view === "offers" ? "active" : ""} onClick={() => setView("offers")}>Ofertas</button>
      </nav>

      {(view === "overview" || view === "transactions") && (
        <div className="vp-period">
          <div className="field">
            <label>De</label>
            <DatePicker ariaLabel="Data inicial de VP" value={from} onChange={setFrom} />
          </div>
          <div className="field">
            <label>Até</label>
            <DatePicker ariaLabel="Data final de VP" value={to} onChange={setTo} />
          </div>
          <span className="vp-period-help mono">Os cards de vendas e compras respeitam este período.</span>
        </div>
      )}

      {view === "overview" && (
        <Overview
          summary={summary.data}
          loading={summary.loading}
          error={summary.error}
          transactions={transactions.data ?? []}
          onEditLimit={() => setEditingLimit(true)}
          onNewTransaction={() => setEditingTransaction(null)}
          onShowTransactions={() => setView("transactions")}
        />
      )}

      {view === "transactions" && (
        <TransactionsView
          items={transactions.data}
          loading={transactions.loading}
          error={transactions.error}
          projects={projects.data ?? []}
          tipo={tipo}
          status={status}
          onTipo={setTipo}
          onStatus={setStatus}
          onNew={() => setEditingTransaction(null)}
          onEdit={setEditingTransaction}
          onExport={exportCSV}
          onDeleted={reloadAll}
        />
      )}

      {view === "offers" && (
        <OffersView
          items={offers.data}
          loading={offers.loading}
          error={offers.error}
          onNew={() => setEditingOffer(null)}
          onEdit={setEditingOffer}
          onDeleted={reloadAll}
        />
      )}

      {editingTransaction !== undefined && (
        <TransactionModal
          item={editingTransaction}
          projects={projects.data ?? []}
          onClose={() => setEditingTransaction(undefined)}
          onSaved={() => {
            setEditingTransaction(undefined);
            reloadAll();
          }}
        />
      )}

      {editingOffer !== undefined && (
        <OfferModal
          item={editingOffer}
          onClose={() => setEditingOffer(undefined)}
          onSaved={() => {
            setEditingOffer(undefined);
            reloadAll();
          }}
        />
      )}

      {editingLimit && (
        <LimitModal
          current={settings.data?.limite_credito ?? summary.data?.limite_credito ?? 0}
          onClose={() => setEditingLimit(false)}
          onSaved={() => {
            setEditingLimit(false);
            reloadAll();
          }}
        />
      )}
    </div>
  );
}

function Overview({
  summary,
  loading,
  error,
  transactions,
  onEditLimit,
  onNewTransaction,
  onShowTransactions,
}: {
  summary: VPSummary | null;
  loading: boolean;
  error: string | null;
  transactions: VPTransaction[];
  onEditLimit: () => void;
  onNewTransaction: () => void;
  onShowTransactions: () => void;
}) {
  if (loading) return <Spinner label="Carregando posição VP…" />;
  if (error) return <ErrorBanner>{error}</ErrorBanner>;
  if (!summary) return null;
  const recent = transactions.slice(0, 5);

  return (
    <>
      <section className="vp-kpi-grid" aria-label="Posição da conta Via Permuta">
        <article className="vp-kpi vp-kpi-primary">
          <span>Saldo VP</span>
          <strong>{vpMoney(summary.saldo)}</strong>
          <small>Vendas concluídas − compras concluídas</small>
        </article>
        <article className="vp-kpi">
          <span>Disponível VP</span>
          <strong>{vpMoney(summary.disponivel)}</strong>
          <small>Saldo + limite aprovado</small>
        </article>
        <article className="vp-kpi">
          <span>Limite aprovado</span>
          <strong>{vpMoney(summary.limite_credito)}</strong>
          <button className="vp-inline-action" onClick={onEditLimit}>Ajustar limite</button>
        </article>
        <article className="vp-kpi">
          <span>Em negociação</span>
          <strong>{summary.negociacoes_abertas}</strong>
          <small>Movimentações ainda sem efeito no saldo</small>
        </article>
      </section>

      <section className="vp-flow-grid">
        <article className="card vp-flow-card vp-sale">
          <div><span>Vendas no período</span><small>VP recebido por serviços e produtos</small></div>
          <strong>{vpMoney(summary.vendas_periodo)}</strong>
          <span className="mono">Ticket médio {vpMoney(summary.ticket_medio_venda)}</span>
        </article>
        <article className="card vp-flow-card vp-purchase">
          <div><span>Compras no período</span><small>VP utilizado dentro do ecossistema</small></div>
          <strong>{vpMoney(summary.compras_periodo)}</strong>
          <span className="mono">Ticket médio {vpMoney(summary.ticket_medio_compra)}</span>
        </article>
      </section>

      <section className="vp-overview-grid">
        <div className="card vp-panel">
          <div className="vp-panel-head">
            <div><span className="mono">EXTRATO</span><h2>Últimas movimentações</h2></div>
            <div className="vp-actions"><button className="btn btn-ghost btn-sm" onClick={onShowTransactions}>Ver todas</button><button className="btn vp-btn btn-sm" onClick={onNewTransaction}>+ Nova</button></div>
          </div>
          {recent.length === 0 ? <Empty>Nenhuma movimentação VP registrada neste período.</Empty> : <VPTransactionTable items={recent} compact />}
        </div>
        <aside className="card vp-panel vp-offer-summary">
          <span className="mono">OFERTAS</span>
          <h2>Posição no catálogo</h2>
          <dl>
            <div><dt>Abertas</dt><dd>{summary.ofertas_abertas}</dd></div>
            <div><dt>Liberadas</dt><dd>{summary.ofertas_liberadas}</dd></div>
            <div><dt>Com pendência</dt><dd>{summary.ofertas_com_pendencia}</dd></div>
          </dl>
        </aside>
      </section>
    </>
  );
}

function TransactionsView({
  items, loading, error, projects, tipo, status, onTipo, onStatus, onNew, onEdit, onExport, onDeleted,
}: {
  items: VPTransaction[] | null;
  loading: boolean;
  error: string | null;
  projects: Project[];
  tipo: string;
  status: string;
  onTipo: (value: string) => void;
  onStatus: (value: string) => void;
  onNew: () => void;
  onEdit: (item: VPTransaction) => void;
  onExport: () => void;
  onDeleted: () => void;
}) {
  const projectName = (id?: number) => projects.find((project) => project.id === id)?.nome;
  return (
    <section>
      <div className="vp-list-head">
        <div><span className="mono">LIVRO VP</span><h2>Movimentações</h2><p>Somente itens concluídos alteram o saldo disponível.</p></div>
        <button className="btn vp-btn" onClick={onNew}>+ Nova movimentação</button>
      </div>
      <div className="filters vp-filters">
        <div className="field"><label>Tipo</label><Select ariaLabel="Filtrar tipo VP" value={tipo} onChange={onTipo} options={[{ value: "", label: "Todos" }, { value: "venda", label: "Vendas" }, { value: "compra", label: "Compras" }]} /></div>
        <div className="field"><label>Status</label><Select ariaLabel="Filtrar status VP" value={status} onChange={onStatus} options={[{ value: "", label: "Todos" }, ...Object.entries(TRANSACTION_STATUS).map(([value, label]) => ({ value, label }))]} /></div>
        <div className="toolbar-spacer" />
        <button className="btn btn-ghost btn-sm" disabled={!items?.length} onClick={onExport}>Exportar CSV</button>
      </div>
      {loading ? <Spinner /> : error ? <ErrorBanner>{error}</ErrorBanner> : !items?.length ? <Empty>Nenhuma movimentação encontrada.</Empty> : (
        <div className="card table-wrap vp-table-card">
          <table>
            <thead><tr><th>Data</th><th>Tipo</th><th>Permutante / oferta</th><th>Projeto</th><th>Status</th><th style={{ textAlign: "right" }}>Valor</th><th /></tr></thead>
            <tbody>{items.map((item) => (
              <tr key={item.id}>
                <td className="num muted">{date(item.data)}</td>
                <td><span className={`vp-direction ${item.tipo}`}>{item.tipo === "venda" ? "↗ Venda" : "↙ Compra"}</span></td>
                <td><strong>{item.permutante}</strong><small className="vp-cell-detail">{item.oferta}{item.voucher_code ? ` · Voucher ${item.voucher_code}` : ""}</small></td>
                <td className="muted">{projectName(item.project_id) ?? "—"}</td>
                <td><StatusPill status={item.status} label={TRANSACTION_STATUS[item.status]} /></td>
                <td className={`num vp-value ${item.tipo}`} style={{ textAlign: "right" }}>{item.tipo === "venda" ? "+" : "−"} {vpMoney(item.valor)}</td>
                <td><div className="vp-row-actions"><button className="btn btn-ghost btn-sm" onClick={() => onEdit(item)}>Editar</button><button className="btn btn-danger btn-sm" onClick={async () => { if (confirm("Excluir esta movimentação VP?")) { await api.del(`/via-permuta/transactions/${item.id}`); onDeleted(); } }}>Excluir</button></div></td>
              </tr>
            ))}</tbody>
          </table>
        </div>
      )}
    </section>
  );
}

function VPTransactionTable({ items, compact = false }: { items: VPTransaction[]; compact?: boolean }) {
  return (
    <div className="table-wrap">
      <table className={compact ? "vp-compact-table" : ""}>
        <thead><tr><th>Data</th><th>Permutante</th><th>Status</th><th style={{ textAlign: "right" }}>Valor</th></tr></thead>
        <tbody>{items.map((item) => <tr key={item.id}><td className="num muted">{date(item.data)}</td><td>{item.permutante}<small className="vp-cell-detail">{item.oferta}</small></td><td><StatusPill status={item.status} label={TRANSACTION_STATUS[item.status]} /></td><td className={`num vp-value ${item.tipo}`} style={{ textAlign: "right" }}>{item.tipo === "venda" ? "+" : "−"} {vpMoney(item.valor)}</td></tr>)}</tbody>
      </table>
    </div>
  );
}

function OffersView({ items, loading, error, onNew, onEdit, onDeleted }: { items: VPOffer[] | null; loading: boolean; error: string | null; onNew: () => void; onEdit: (item: VPOffer) => void; onDeleted: () => void }) {
  return (
    <section>
      <div className="vp-list-head">
        <div><span className="mono">CATÁLOGO</span><h2>Ofertas da Morfos</h2><p>Espelho operacional das ofertas publicadas ou em preparação na Via Permuta.</p></div>
        <button className="btn vp-btn" onClick={onNew}>+ Nova oferta</button>
      </div>
      {loading ? <Spinner /> : error ? <ErrorBanner>{error}</ErrorBanner> : !items?.length ? <Empty>Nenhuma oferta cadastrada. Registre aqui as ofertas publicadas na Via Permuta.</Empty> : (
        <div className="vp-offer-grid">{items.map((item) => (
          <article className="card vp-offer-card" key={item.id}>
            <div className="vp-offer-card-top"><StatusPill status={item.status} label={OFFER_STATUS[item.status]} /><span className="mono">{item.negociavel ? "À combinar" : item.valor !== undefined ? vpMoney(item.valor) : "—"}</span></div>
            <h3>{item.titulo}</h3>
            <p>{item.descricao || "Sem descrição complementar."}</p>
            <div className="vp-offer-actions">{item.external_url && <a className="btn btn-ghost btn-sm" href={item.external_url} target="_blank" rel="noreferrer">Ver na Via Permuta ↗</a>}<button className="btn btn-ghost btn-sm" onClick={() => onEdit(item)}>Editar</button><button className="btn btn-danger btn-sm" onClick={async () => { if (confirm("Excluir esta oferta do controle interno?")) { await api.del(`/via-permuta/offers/${item.id}`); onDeleted(); } }}>Excluir</button></div>
          </article>
        ))}</div>
      )}
    </section>
  );
}

function StatusPill({ status, label }: { status: string; label: string }) {
  return <span className={`vp-status vp-status-${status}`}>{label}</span>;
}

function TransactionModal({ item, projects, onClose, onSaved }: { item: VPTransaction | null; projects: Project[]; onClose: () => void; onSaved: () => void }) {
  const [tipo, setTipo] = useState<VPTransactionType>(item?.tipo ?? "venda");
  const [status, setStatus] = useState<VPTransactionStatus>(item?.status ?? "concluida");
  const [valor, setValor] = useState(item ? (item.valor / 100).toFixed(2).replace(".", ",") : "");
  const [data, setData] = useState(item?.data ?? todayISO());
  const [permutante, setPermutante] = useState(item?.permutante ?? "");
  const [oferta, setOferta] = useState(item?.oferta ?? "");
  const [projectId, setProjectId] = useState(item?.project_id ? String(item.project_id) : "");
  const [voucher, setVoucher] = useState(item?.voucher_code ?? "");
  const [observacoes, setObservacoes] = useState(item?.observacoes ?? "");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function submit(event: FormEvent) {
    event.preventDefault();
    const amount = toCentavos(valor);
    if (!amount || amount <= 0) return setError("Informe um valor VP válido.");
    setBusy(true); setError(null);
    const payload = { tipo, status, valor: amount, data, permutante, oferta, project_id: projectId ? Number(projectId) : null, voucher_code: voucher.trim() || null, observacoes: observacoes.trim() || null };
    try { if (item) await api.put(`/via-permuta/transactions/${item.id}`, payload); else await api.post("/via-permuta/transactions", payload); onSaved(); }
    catch (err) { setError(err instanceof Error ? err.message : "Falha ao salvar movimentação VP"); }
    finally { setBusy(false); }
  }

  return <Modal title={item ? "Editar movimentação VP" : "Nova movimentação VP"} onClose={onClose} width={720}><form className="transaction-form" onSubmit={submit}>{error && <ErrorBanner>{error}</ErrorBanner>}<div className="form-row"><div className="field"><label>Tipo *</label><Select ariaLabel="Tipo da movimentação VP" value={tipo} onChange={(value) => setTipo(value as VPTransactionType)} options={[{ value: "venda", label: "Venda — entrada de VP" }, { value: "compra", label: "Compra — saída de VP" }]} /></div><div className="field"><label>Status *</label><Select ariaLabel="Status da movimentação VP" value={status} onChange={(value) => setStatus(value as VPTransactionStatus)} options={Object.entries(TRANSACTION_STATUS).map(([value, label]) => ({ value, label }))} /></div></div><div className="form-row"><div className="field"><label>Valor VP *</label><input inputMode="decimal" placeholder="1.000,00" value={valor} onChange={(event) => setValor(event.target.value)} required /></div><div className="field"><label>Data *</label><DatePicker ariaLabel="Data da movimentação VP" value={data} onChange={setData} required /></div></div><div className="form-row"><div className="field"><label>Associado permutante *</label><input value={permutante} onChange={(event) => setPermutante(event.target.value)} placeholder={tipo === "venda" ? "Cliente que pagou em VP" : "Fornecedor da compra"} required /></div><div className="field"><label>Oferta / serviço *</label><input value={oferta} onChange={(event) => setOferta(event.target.value)} placeholder="Serviço negociado" required /></div></div><div className="form-row"><div className="field"><label>Projeto relacionado</label><Select ariaLabel="Projeto relacionado à movimentação VP" value={projectId} onChange={setProjectId} options={[{ value: "", label: "Nenhum projeto" }, ...projects.map((project) => ({ value: String(project.id), label: project.nome }))]} /></div><div className="field"><label>Código do voucher</label><input value={voucher} onChange={(event) => setVoucher(event.target.value)} placeholder="Opcional" /></div></div><div className="field"><label>Observações</label><textarea rows={3} value={observacoes} onChange={(event) => setObservacoes(event.target.value)} placeholder="Condições, comissão, prazo ou detalhes da negociação" /></div><div className="modal-actions"><button type="button" className="btn btn-ghost" onClick={onClose}>Cancelar</button><button className="btn vp-btn" disabled={busy}>{busy ? "Salvando…" : "Salvar movimentação"}</button></div></form></Modal>;
}

function OfferModal({ item, onClose, onSaved }: { item: VPOffer | null; onClose: () => void; onSaved: () => void }) {
  const [titulo, setTitulo] = useState(item?.titulo ?? "");
  const [descricao, setDescricao] = useState(item?.descricao ?? "");
  const [valor, setValor] = useState(item?.valor !== undefined ? (item.valor / 100).toFixed(2).replace(".", ",") : "");
  const [negociavel, setNegociavel] = useState(item?.negociavel ?? false);
  const [status, setStatus] = useState<VPOfferStatus>(item?.status ?? "aberta");
  const [externalURL, setExternalURL] = useState(item?.external_url ?? "");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function submit(event: FormEvent) {
    event.preventDefault();
    const amount = valor.trim() ? toCentavos(valor) : null;
    if (!negociavel && (!amount || amount <= 0)) return setError("Informe o valor VP ou marque como negociável.");
    if (amount !== null && amount <= 0) return setError("Informe um valor VP válido.");
    setBusy(true); setError(null);
    const payload = { titulo, descricao: descricao.trim() || null, valor: amount, negociavel, status, external_url: externalURL.trim() || null };
    try { if (item) await api.put(`/via-permuta/offers/${item.id}`, payload); else await api.post("/via-permuta/offers", payload); onSaved(); }
    catch (err) { setError(err instanceof Error ? err.message : "Falha ao salvar oferta"); }
    finally { setBusy(false); }
  }

  return <Modal title={item ? "Editar oferta VP" : "Nova oferta VP"} onClose={onClose} width={680}><form className="transaction-form" onSubmit={submit}>{error && <ErrorBanner>{error}</ErrorBanner>}<div className="field"><label>Título *</label><input value={titulo} onChange={(event) => setTitulo(event.target.value)} placeholder="Nome da oferta publicada" required /></div><div className="field"><label>Descrição</label><textarea rows={4} value={descricao} onChange={(event) => setDescricao(event.target.value)} placeholder="Escopo, validade e condições do voucher" /></div><div className="form-row"><div className="field"><label>Valor VP</label><input inputMode="decimal" value={valor} onChange={(event) => setValor(event.target.value)} placeholder="1.000,00" /></div><div className="field"><label>Status</label><Select ariaLabel="Status da oferta VP" value={status} onChange={(value) => setStatus(value as VPOfferStatus)} options={Object.entries(OFFER_STATUS).map(([value, label]) => ({ value, label }))} /></div></div><label className="vp-check"><input type="checkbox" checked={negociavel} onChange={(event) => setNegociavel(event.target.checked)} /><span>Valor a combinar na negociação</span></label><div className="field"><label>Link da oferta na Via Permuta</label><input type="url" value={externalURL} onChange={(event) => setExternalURL(event.target.value)} placeholder="https://…" /></div><div className="modal-actions"><button type="button" className="btn btn-ghost" onClick={onClose}>Cancelar</button><button className="btn vp-btn" disabled={busy}>{busy ? "Salvando…" : "Salvar oferta"}</button></div></form></Modal>;
}

function LimitModal({ current, onClose, onSaved }: { current: number; onClose: () => void; onSaved: () => void }) {
  const [value, setValue] = useState((current / 100).toFixed(2).replace(".", ","));
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  async function submit(event: FormEvent) { event.preventDefault(); const amount = toCentavos(value); if (amount === null || amount < 0) return setError("Informe um limite válido."); setBusy(true); setError(null); try { await api.put("/via-permuta/settings", { limite_credito: amount }); onSaved(); } catch (err) { setError(err instanceof Error ? err.message : "Falha ao atualizar limite"); } finally { setBusy(false); } }
  return <Modal title="Limite de crédito VP" onClose={onClose} width={460}><form className="transaction-form" onSubmit={submit}>{error && <ErrorBanner>{error}</ErrorBanner>}<p className="muted">Informe o limite aprovado exibido no painel da Via Permuta. O valor disponível será calculado como saldo VP + limite.</p><div className="field"><label>Limite aprovado</label><input inputMode="decimal" value={value} onChange={(event) => setValue(event.target.value)} autoFocus /></div><div className="modal-actions"><button type="button" className="btn btn-ghost" onClick={onClose}>Cancelar</button><button className="btn vp-btn" disabled={busy}>{busy ? "Salvando…" : "Atualizar limite"}</button></div></form></Modal>;
}
