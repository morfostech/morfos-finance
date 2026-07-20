import { useRef, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { api } from "../lib/api";
import { useAuth } from "../lib/auth";
import { useAsync } from "../lib/hooks";
import { date, money, todayISO } from "../lib/format";
import { canManage, type Installment, type Project, type Proposal, type User } from "../lib/types";
import { Empty, ErrorBanner, SectionHead, Spinner } from "../components/ui";
import { PaidPill, StatusPill } from "../components/pills";
import { NotesPanel } from "../components/NotesPanel";
import "./pages.css";

export function ProjectDetail() {
  const { id } = useParams();
  const { user } = useAuth();
  const isAdmin = canManage(user?.role);
  const { data: project, loading, error, reload } = useAsync<Project>(() => api.get(`/projects/${id}`), [id]);
  const proposals = useAsync<Proposal[]>(() => api.get(`/projects/${id}/proposals`), [id]);
  const users = useAsync<User[]>(() => (isAdmin ? api.get("/users") : Promise.resolve([])), [id, isAdmin]);

  if (loading) return <Spinner />;
  if (error) return <ErrorBanner>{error}</ErrorBanner>;
  if (!project) return null;

  const userName = (uid: number) => users.data?.find((u) => u.id === uid)?.nome ?? `#${uid}`;

  return (
    <div>
      <Link to="/projetos" className="back-link">← Projetos</Link>
      <header className="page-head">
        <span className="kicker">Projeto</span>
        <h1>{project.nome}</h1>
        <p>{project.cliente ?? "Sem cliente"} · <StatusPill status={project.status} /></p>
      </header>

      <div className="detail-grid">
        <div style={{ display: "flex", flexDirection: "column", gap: 16 }}>
          {project.installments && project.installments.length > 0 && (
            <div className="card panel">
              <SectionHead title="Implementação" />
              {project.installments.map((inst) => (
                <InstallmentRow key={inst.id} projectId={project.id} inst={inst} isAdmin={isAdmin} onChange={reload} />
              ))}
            </div>
          )}

          <div className="card panel">
            <SectionHead
              title="Propostas comerciais"
              action={isAdmin ? <UploadButton projectId={project.id} onDone={proposals.reload} /> : undefined}
            />
            {proposals.loading ? (
              <Spinner />
            ) : !proposals.data || proposals.data.length === 0 ? (
              <Empty>Nenhuma proposta anexada.</Empty>
            ) : (
              proposals.data.map((p) => (
                <div key={p.id} className="list-file">
                  <span className="file-ext">{p.arquivo_tipo}</span>
                  <a href={p.url} target="_blank" rel="noreferrer" title={p.descricao}>
                    {p.nome_arquivo ?? p.descricao ?? `Proposta.${p.arquivo_tipo}`}
                  </a>
                  {isAdmin && (
                    <button
                      className="btn btn-danger btn-sm"
                      style={{ marginLeft: "auto" }}
                      onClick={async () => {
                        await api.del(`/proposals/${p.id}`);
                        proposals.reload();
                      }}
                    >
                      Excluir
                    </button>
                  )}
                </div>
              ))
            )}
          </div>
        </div>

        <div style={{ display: "flex", flexDirection: "column", gap: 16 }}>
          <div className="card panel">
            <SectionHead title="Resumo" />
            <div className="meta-list">
              <Meta k="Implementação" v={project.valor_implementacao ? money(project.valor_implementacao) : "—"} />
              <Meta k="Mensalidade" v={project.valor_mensal ? money(project.valor_mensal) : "—"} />
              <Meta k="Vencimento" v={project.dia_vencimento ? `dia ${project.dia_vencimento}` : "—"} />
              <Meta k="Início" v={date(project.data_inicio)} />
              <Meta k="Fim" v={project.data_fim ? date(project.data_fim) : "em aberto"} />
            </div>
          </div>

          <div className="card panel">
            <SectionHead
              title="Colaboradores"
              action={isAdmin ? <MembersButton project={project} users={users.data ?? []} onDone={reload} /> : undefined}
            />
            {!project.member_ids || project.member_ids.length === 0 ? (
              <Empty>Ninguém alocado.</Empty>
            ) : (
              <div style={{ display: "flex", flexWrap: "wrap", gap: 8 }}>
                {project.member_ids.map((uid) => (
                  <span key={uid} className="pill pill-neutral">{userName(uid)}</span>
                ))}
              </div>
            )}
          </div>

          <NotesPanel ownerType="project" ownerId={project.id} title="Minhas anotações sobre este projeto" />
        </div>
      </div>
    </div>
  );
}

function Meta({ k, v }: { k: string; v: string }) {
  return (
    <div className="meta-row">
      <span className="k">{k}</span>
      <span className="num">{v}</span>
    </div>
  );
}

function InstallmentRow({ projectId, inst, isAdmin, onChange }: { projectId: number; inst: Installment; isAdmin: boolean; onChange: () => void }) {
  const [busy, setBusy] = useState(false);
  async function toggle() {
    setBusy(true);
    try {
      await api.patch(`/projects/${projectId}/installments/${inst.id}`, {
        pago_em: inst.pago ? null : todayISO(),
      });
      onChange();
    } finally {
      setBusy(false);
    }
  }
  return (
    <div className="inst-row">
      <div className="inst-left">
        <span className="inst-tipo">{inst.tipo}</span>
        <span className="inst-val">{money(inst.valor)}{inst.pago_em ? ` · pago em ${date(inst.pago_em)}` : ""}</span>
      </div>
      <div className="inst-right">
        <PaidPill pago={inst.pago} />
        {isAdmin && (
          <button className="btn btn-ghost btn-sm" disabled={busy} onClick={toggle}>
            {inst.pago ? "Marcar pendente" : "Marcar paga"}
          </button>
        )}
      </div>
    </div>
  );
}

function UploadButton({ projectId, onDone }: { projectId: number; onDone: () => void }) {
  const ref = useRef<HTMLInputElement>(null);
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  async function onFile(file: File) {
    setErr(null);
    setBusy(true);
    try {
      const form = new FormData();
      form.append("file", file);
      await api.upload(`/projects/${projectId}/proposals`, form);
      onDone();
    } catch (e) {
      setErr(e instanceof Error ? e.message : "Falha no upload");
    } finally {
      setBusy(false);
      if (ref.current) ref.current.value = "";
    }
  }

  return (
    <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
      {err && <span className="muted" style={{ fontSize: 12, color: "var(--danger)" }}>{err}</span>}
      <input
        ref={ref}
        type="file"
        accept=".pdf,.docx"
        style={{ display: "none" }}
        onChange={(e) => e.target.files?.[0] && onFile(e.target.files[0])}
      />
      <button className="btn btn-ghost btn-sm" disabled={busy} onClick={() => ref.current?.click()}>
        {busy ? "Enviando…" : "+ Anexar (PDF/DOCX)"}
      </button>
    </div>
  );
}

function MembersButton({ project, users, onDone }: { project: Project; users: User[]; onDone: () => void }) {
  const [open, setOpen] = useState(false);
  const [selected, setSelected] = useState<number[]>(project.member_ids ?? []);
  const [busy, setBusy] = useState(false);

  if (!open) return <button className="btn btn-ghost btn-sm" onClick={() => setOpen(true)}>Gerenciar</button>;

  const toggle = (uid: number) =>
    setSelected((s) => (s.includes(uid) ? s.filter((x) => x !== uid) : [...s, uid]));

  async function save() {
    setBusy(true);
    try {
      await api.put(`/projects/${project.id}/members`, { member_ids: selected });
      onDone();
      setOpen(false);
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="member-picker">
      {users.map((u) => (
        <label key={u.id} className="member-opt">
          <input type="checkbox" checked={selected.includes(u.id)} onChange={() => toggle(u.id)} />
          {u.nome}
        </label>
      ))}
      <div style={{ display: "flex", gap: 8, marginTop: 8 }}>
        <button className="btn btn-ghost btn-sm" onClick={() => setOpen(false)}>Fechar</button>
        <button className="btn btn-primary btn-sm" disabled={busy} onClick={save}>Salvar</button>
      </div>
    </div>
  );
}
