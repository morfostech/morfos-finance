import { useState, type FormEvent } from "react";
import { useNavigate } from "react-router-dom";
import { api } from "../lib/api";
import { useAuth } from "../lib/auth";
import { useAsync } from "../lib/hooks";
import { money, toCentavos } from "../lib/format";
import { canManage, type Project } from "../lib/types";
import { Empty, ErrorBanner, SectionHead, Spinner } from "../components/ui";
import { Modal } from "../components/Modal";
import { StatusPill } from "../components/pills";
import "./pages.css";

export function Projects() {
  const { user } = useAuth();
  const navigate = useNavigate();
  const isAdmin = canManage(user?.role);
  const { data, loading, error, reload } = useAsync<Project[]>(() => api.get("/projects"), []);
  const [creating, setCreating] = useState(false);

  return (
    <div>
      <header className="page-head">
        <span className="kicker">02 / Projetos</span>
        <h1>Projetos</h1>
        <p>Implementação (parcelas 50/50) e/ou mensalidade, com colaboradores alocados.</p>
      </header>

      <SectionHead
        title="Todos os projetos"
        action={isAdmin ? <button className="btn btn-primary btn-sm" onClick={() => setCreating(true)}>+ Novo projeto</button> : undefined}
      />

      {loading ? (
        <Spinner />
      ) : error ? (
        <ErrorBanner>{error}</ErrorBanner>
      ) : !data || data.length === 0 ? (
        <Empty>Nenhum projeto ainda.</Empty>
      ) : (
        <div className="card table-wrap">
          <table>
            <thead>
              <tr>
                <th>Projeto</th>
                <th>Cliente</th>
                <th style={{ textAlign: "right" }}>Implementação</th>
                <th style={{ textAlign: "right" }}>Mensalidade</th>
                <th>Status</th>
              </tr>
            </thead>
            <tbody>
              {data.map((p) => (
                <tr key={p.id} className="clickable" onClick={() => navigate(`/projetos/${p.id}`)}>
                  <td style={{ fontWeight: 600 }}>{p.nome}</td>
                  <td className="muted">{p.cliente ?? "—"}</td>
                  <td className="num" style={{ textAlign: "right" }}>{p.valor_implementacao ? money(p.valor_implementacao) : "—"}</td>
                  <td className="num" style={{ textAlign: "right" }}>{p.valor_mensal ? money(p.valor_mensal) : "—"}</td>
                  <td><StatusPill status={p.status} /></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {creating && (
        <NewProjectModal
          onClose={() => setCreating(false)}
          onCreated={(p) => {
            setCreating(false);
            reload();
            navigate(`/projetos/${p.id}`);
          }}
        />
      )}
    </div>
  );
}

function NewProjectModal({ onClose, onCreated }: { onClose: () => void; onCreated: (p: Project) => void }) {
  const [nome, setNome] = useState("");
  const [cliente, setCliente] = useState("");
  const [impl, setImpl] = useState("");
  const [mensal, setMensal] = useState("");
  const [dia, setDia] = useState("");
  const [inicio, setInicio] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function submit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    const body: Record<string, unknown> = { nome, cliente: cliente || null };
    if (impl) body.valor_implementacao = toCentavos(impl);
    if (mensal) body.valor_mensal = toCentavos(mensal);
    if (dia) body.dia_vencimento = Number(dia);
    if (inicio) body.data_inicio = inicio;
    if (!body.valor_implementacao && !body.valor_mensal) {
      return setError("Informe implementação e/ou mensalidade.");
    }
    setBusy(true);
    try {
      const p = await api.post<Project>("/projects", body);
      onCreated(p);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Falha ao criar");
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal title="Novo projeto" onClose={onClose}>
      <form onSubmit={submit} style={{ display: "flex", flexDirection: "column", gap: 16 }}>
        {error && <ErrorBanner>{error}</ErrorBanner>}
        <div className="field">
          <label>Nome *</label>
          <input value={nome} onChange={(e) => setNome(e.target.value)} required />
        </div>
        <div className="field">
          <label>Cliente</label>
          <input value={cliente} onChange={(e) => setCliente(e.target.value)} />
        </div>
        <div className="form-row">
          <div className="field">
            <label>Implementação (R$)</label>
            <input inputMode="decimal" placeholder="10.000,00" value={impl} onChange={(e) => setImpl(e.target.value)} />
          </div>
          <div className="field">
            <label>Mensalidade (R$)</label>
            <input inputMode="decimal" placeholder="3.000,00" value={mensal} onChange={(e) => setMensal(e.target.value)} />
          </div>
        </div>
        <div className="form-row">
          <div className="field">
            <label>Dia de vencimento</label>
            <input inputMode="numeric" placeholder="10" value={dia} onChange={(e) => setDia(e.target.value)} />
          </div>
          <div className="field">
            <label>Início</label>
            <input type="date" value={inicio} onChange={(e) => setInicio(e.target.value)} />
          </div>
        </div>
        <p className="muted" style={{ fontSize: 12.5 }}>
          Com implementação, o sistema cria automaticamente as parcelas de entrada e finalização (50% cada).
        </p>
        <div className="modal-actions">
          <button type="button" className="btn btn-ghost btn-sm" onClick={onClose}>Cancelar</button>
          <button className="btn btn-primary btn-sm" disabled={busy}>{busy ? "Criando…" : "Criar projeto"}</button>
        </div>
      </form>
    </Modal>
  );
}
