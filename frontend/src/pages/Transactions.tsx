import { useState, type FormEvent } from "react";
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

  const [tipo, setTipo] = useState("");
  const [from, setFrom] = useState("");
  const [to, setTo] = useState("");
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

  return (
    <div>
      <header className="page-head">
        <span className="kicker">03 / Transações</span>
        <h1>Transações</h1>
        <p>Ganhos e despesas, com origem, categoria e projeto. Exclusão é soft delete.</p>
      </header>

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
        {isAdmin && (
          <button className="btn btn-primary btn-sm" onClick={() => setCreating(true)}>+ Nova transação</button>
        )}
      </div>

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
    if (descricao) body.descricao = descricao;
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
    <Modal title="Nova transação" onClose={onClose} width={520}>
      <form onSubmit={submit} style={{ display: "flex", flexDirection: "column", gap: 16 }}>
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
                options={[{ value: "avulso", label: "Avulso" }, { value: "recorrencia", label: "Recorrência" }]}
              />
            </div>
          ) : (
            <div className="field">
              <label>Categoria</label>
              <Select
                ariaLabel="Categoria da despesa"
                value={categoryId}
                onChange={setCategoryId}
                options={[
                  { value: "", label: "Sem categoria" },
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
        <div className="field">
          <label>Descrição</label>
          <input value={descricao} onChange={(e) => setDescricao(e.target.value)} />
        </div>
        <div className="modal-actions">
          <button type="button" className="btn btn-ghost btn-sm" onClick={onClose}>Cancelar</button>
          <button className="btn btn-primary btn-sm" disabled={busy}>{busy ? "Salvando…" : "Adicionar"}</button>
        </div>
      </form>
    </Modal>
  );
}
