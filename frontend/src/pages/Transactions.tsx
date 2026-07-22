import { useState, type FormEvent } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { api } from "../lib/api";
import { useAuth } from "../lib/auth";
import { useAsync } from "../lib/hooks";
import { date, money, todayISO, toCentavos } from "../lib/format";
import { canManage, type Category, type Project, type Transaction, type TxType, type User } from "../lib/types";
import { Empty, ErrorBanner, Select, Spinner } from "../components/ui";
import { DatePicker } from "../components/DatePicker";
import { Modal } from "../components/Modal";
import { NotesPanel } from "../components/NotesPanel";
import "./pages.css";

export function Transactions() {
  const { user } = useAuth();
  const isAdmin = canManage(user?.role);
  const [searchParams] = useSearchParams();

  const [tipo, setTipo] = useState(searchParams.get("tipo") ?? "");
  const [from, setFrom] = useState(searchParams.get("from") ?? "");
  const [to, setTo] = useState(searchParams.get("to") ?? "");
  const [creating, setCreating] = useState(false);
  const [notesFor, setNotesFor] = useState<number | null>(null);

  const qs = new URLSearchParams();
  if (tipo) qs.set("tipo", tipo);
  if (from) qs.set("from", from);
  if (to) qs.set("to", to);
  const query = qs.toString();

  const { data, loading, error, reload } = useAsync<Transaction[]>(
    () => api.get(`/transactions${query ? `?${query}` : ""}`),
    [query],
  );
  const categories = useAsync<Category[]>(() => api.get("/categories"), []);
  const projects = useAsync<Project[]>(() => api.get("/projects"), []);

  const catName = (cid?: number) => categories.data?.find((c) => c.id === cid)?.nome;
  const projName = (pid?: number) => projects.data?.find((p) => p.id === pid)?.nome;
  const contexto = searchParams.get("contexto");
  const totals = (data ?? []).reduce((acc, item) => {
    if (item.tipo === "ganho") acc.entradas += item.valor;
    else acc.saidas += item.valor;
    return acc;
  }, { entradas: 0, saidas: 0 });
  const contextTitle = contexto === "saldo" ? "Composição do saldo em caixa"
    : contexto === "ganhos" ? "Entradas realizadas no período"
    : contexto === "despesas" ? "Saídas realizadas no período"
    : contexto === "periodo" ? "Composição do resultado no período"
    : "Transações";

  function exportCSV() {
    if (!data?.length) return;
    const escape = (value: unknown) => `"${String(value ?? "").replace(/"/g, '""')}"`;
    const rows = data.map((t) => [
      t.data,
      t.tipo,
      (t.valor / 100).toFixed(2).replace(".", ","),
      t.descricao ?? "",
      projName(t.project_id) ?? "",
      t.tipo === "ganho" ? (t.origem ?? "") : (catName(t.category_id) ?? ""),
    ]);
    const csv = "\uFEFF" + [["Data", "Tipo", "Valor (R$)", "Descrição", "Projeto", "Origem/Categoria"], ...rows]
      .map((row) => row.map(escape).join(";"))
      .join("\r\n");
    const url = URL.createObjectURL(new Blob([csv], { type: "text/csv;charset=utf-8" }));
    const link = document.createElement("a");
    link.href = url;
    link.download = `transacoes-${todayISO()}.csv`;
    link.click();
    URL.revokeObjectURL(url);
  }

  return (
    <div>
      <header className="page-head">
        <span className="kicker">03 / Transações</span>
        <h1>{contextTitle}</h1>
        <p>{contexto ? "Relação dos lançamentos que formam o valor selecionado no dashboard." : "Ganhos e despesas, com origem, categoria e projeto."}</p>
      </header>

      {contexto && <Link className="back-link" to="/">← Voltar ao dashboard</Link>}

      <div className="filters">
        <div className="field">
          <label>Tipo</label>
          <Select
            ariaLabel="Filtrar por tipo"
            value={tipo}
            onChange={setTipo}
            options={[
              { value: "", label: "Todos" },
              { value: "ganho", label: "Ganhos" },
              { value: "despesa", label: "Despesas" },
            ]}
          />
        </div>
        <div className="field">
          <label>De</label>
          <DatePicker ariaLabel="Data inicial" value={from} onChange={setFrom} />
        </div>
        <div className="field">
          <label>Até</label>
          <DatePicker ariaLabel="Data final" value={to} onChange={setTo} />
        </div>
        <div className="toolbar-spacer" />
        <button className="btn btn-ghost btn-sm" disabled={!data?.length} onClick={exportCSV}>Exportar CSV</button>
        {isAdmin && (
          <button className="btn btn-primary btn-sm" onClick={() => setCreating(true)}>+ Nova transação</button>
        )}
      </div>

      {!loading && !error && data && (
        <div className="transaction-summary" aria-label="Resumo dos lançamentos filtrados">
          <div><span className="mono muted">Lançamentos</span><strong>{data.length}</strong></div>
          <div><span className="mono muted">Entradas</span><strong className="num accent-teal">{money(totals.entradas)}</strong></div>
          <div><span className="mono muted">Saídas</span><strong className="num accent-copper">{money(totals.saidas)}</strong></div>
          <div><span className="mono muted">Resultado</span><strong className={`num ${totals.entradas - totals.saidas >= 0 ? "accent-teal" : "accent-danger"}`}>{money(totals.entradas - totals.saidas)}</strong></div>
        </div>
      )}

      {loading ? (
        <Spinner />
      ) : error ? (
        <ErrorBanner>{error}</ErrorBanner>
      ) : !data || data.length === 0 ? (
        <Empty>Nenhuma transação no filtro atual.</Empty>
      ) : (
        <div className="card table-wrap">
          <table>
            <thead>
              <tr>
                <th>Data</th>
                <th>Descrição</th>
                <th>Projeto</th>
                <th>Origem / Categoria</th>
                <th style={{ textAlign: "right" }}>Valor</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {data.map((t) => (
                <tr key={t.id}>
                  <td className="num muted">{date(t.data)}</td>
                  <td>{t.descricao ?? "—"}</td>
                  <td className="muted">{projName(t.project_id) ?? "—"}</td>
                  <td className="muted mono" style={{ fontSize: 12 }}>
                    {t.tipo === "ganho" ? (t.origem ?? "—") : (catName(t.category_id) ?? "sem categoria")}
                  </td>
                  <td className={`num ${t.tipo === "ganho" ? "tx-ganho" : "tx-despesa"}`} style={{ textAlign: "right" }}>
                    {t.tipo === "ganho" ? "+" : "−"} {money(t.valor)}
                  </td>
                  <td style={{ textAlign: "right" }}>
                    <div style={{ display: "flex", gap: 8, justifyContent: "flex-end" }}>
                      <button className="btn btn-ghost btn-sm" onClick={() => setNotesFor(t.id)}>Notas</button>
                      {t.installment_id ? (
                        <span className="pill pill-ok" title="Gerenciada pelo pagamento da parcela no projeto">Automática</span>
                      ) : isAdmin && (
                        <button
                          className="btn btn-danger btn-sm"
                          onClick={async () => {
                            if (confirm("Excluir esta transação? (soft delete)")) {
                              await api.del(`/transactions/${t.id}`);
                              reload();
                            }
                          }}
                        >
                          Excluir
                        </button>
                      )}
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {creating && (
        <NewTransactionModal
          categories={categories.data ?? []}
          projects={projects.data ?? []}
          onClose={() => setCreating(false)}
          onCreated={() => {
            setCreating(false);
            reload();
          }}
        />
      )}

      {notesFor !== null && (
        <Modal title="Notas da transação" onClose={() => setNotesFor(null)} width={480}>
          <NotesPanel ownerType="transaction" ownerId={notesFor} title="" bare />
        </Modal>
      )}
    </div>
  );
}

function NewTransactionModal({
  categories,
  projects,
  onClose,
  onCreated,
}: {
  categories: Category[];
  projects: Project[];
  onClose: () => void;
  onCreated: () => void;
}) {
  const users = useAsync<User[]>(() => api.get("/users"), []);
  const [tipo, setTipo] = useState<TxType>("ganho");
  const [valor, setValor] = useState("");
  const [data, setData] = useState(todayISO());
  const [projectId, setProjectId] = useState("");
  const [userId, setUserId] = useState("");
  const [origem, setOrigem] = useState("avulso");
  const [categoryId, setCategoryId] = useState("");
  const [descricao, setDescricao] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function submit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    const cents = toCentavos(valor);
    if (!cents || cents <= 0) return setError("Informe um valor válido.");

    const body: Record<string, unknown> = { tipo, valor: cents, data };
    if (projectId) body.project_id = Number(projectId);
    if (userId) body.user_id = Number(userId);
    if (tipo === "despesa" && descricao.trim().length < 3) {
      return setError("Informe o motivo ou a justificativa da despesa.");
    }
    if (tipo === "despesa" && !categoryId) {
      return setError("Selecione a categoria da despesa.");
    }
    if (descricao.trim()) body.descricao = descricao.trim();
    if (tipo === "ganho") {
      if (origem === "recorrencia" && !projectId) {
        return setError("Selecione o projeto para ganhos de recorrência.");
      }
      body.origem = origem;
    }
    else if (categoryId) body.category_id = Number(categoryId);

    setBusy(true);
    try {
      await api.post("/transactions", body);
      onCreated();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Falha ao criar");
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal title="Nova transação" onClose={onClose} width={640}>
      <form onSubmit={submit} className="transaction-form">
        {error && <ErrorBanner>{error}</ErrorBanner>}
        <div className="form-row">
          <div className="field">
            <label>Tipo *</label>
            <Select
              ariaLabel="Tipo da transação"
              value={tipo}
              onChange={(value) => setTipo(value as TxType)}
              options={[{ value: "ganho", label: "Ganho" }, { value: "despesa", label: "Despesa" }]}
            />
          </div>
          <div className="field">
            <label>Valor (R$) *</label>
            <input inputMode="decimal" placeholder="1.500,00" value={valor} onChange={(e) => setValor(e.target.value)} required />
          </div>
        </div>
        <div className="form-row">
          <div className="field">
            <label>Data *</label>
            <DatePicker ariaLabel="Data da transação" value={data} onChange={setData} required />
          </div>
          {tipo === "ganho" ? (
            <div className="field">
              <label>Origem</label>
              <Select
                ariaLabel="Origem do ganho"
                value={origem}
                onChange={setOrigem}
                options={[{ value: "avulso", label: "Receita avulsa" }, { value: "recorrencia", label: "Mensalidade recorrente" }]}
              />
            </div>
          ) : (
            <div className="field">
              <label>Categoria da despesa *</label>
              <Select
                ariaLabel="Categoria da despesa"
                value={categoryId}
                onChange={setCategoryId}
                options={[
                  { value: "", label: "Selecione o motivo" },
                  ...categories.map((category) => ({ value: String(category.id), label: category.nome })),
                ]}
              />
            </div>
          )}
        </div>
        <div className="form-row">
          <div className="field">
            <label>Projeto{tipo === "ganho" && origem === "recorrencia" ? " *" : ""}</label>
            <Select
              ariaLabel="Projeto da transação"
              value={projectId}
              onChange={setProjectId}
              options={[
                { value: "", label: "Nenhum projeto" },
                ...projects.map((project) => ({ value: String(project.id), label: project.nome })),
              ]}
            />
          </div>
          <div className="field">
            <label>Colaborador</label>
            <Select
              ariaLabel="Colaborador da transação"
              value={userId}
              onChange={setUserId}
              options={[
                { value: "", label: "Nenhum colaborador" },
                ...(users.data ?? []).map((item) => ({ value: String(item.id), label: item.nome })),
              ]}
            />
          </div>
        </div>
        {tipo === "despesa" && (
          <div className="recurring-guidance">
            <div><strong>Esta despesa se repete?</strong><span>Cadastre a sequência no planejamento para projetar o caixa e acompanhar cada vencimento.</span></div>
            <Link to="/planejamento">Planejar recorrência →</Link>
          </div>
        )}
        <div className="field">
          <label>{tipo === "despesa" ? "Motivo / justificativa *" : "Descrição / identificação"}</label>
          <textarea
            rows={3}
            value={descricao}
            onChange={(e) => setDescricao(e.target.value)}
            placeholder={tipo === "despesa" ? "Ex.: renovação anual da ferramenta de atendimento" : "Ex.: mensalidade de julho"}
            required={tipo === "despesa"}
          />
          <span className="field-help">{tipo === "despesa" ? "Explique por que a saída foi necessária para facilitar conferência e decisão futura." : "Use uma identificação que facilite localizar este recebimento depois."}</span>
        </div>
        <div className="modal-actions">
          <button type="button" className="btn btn-ghost btn-sm" onClick={onClose}>Cancelar</button>
          <button className="btn btn-primary btn-sm" disabled={busy}>{busy ? "Salvando…" : "Adicionar"}</button>
        </div>
      </form>
    </Modal>
  );
}
