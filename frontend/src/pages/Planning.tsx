import { useMemo, useState, type FormEvent } from "react";
import { Bar, Empty, ErrorBanner, KpiMoney, SectionHead, Select, Spinner } from "../components/ui";
import { Modal } from "../components/Modal";
import { DatePicker } from "../components/DatePicker";
import { api } from "../lib/api";
import { date, money, monthLabel, todayISO, toCentavos } from "../lib/format";
import { useAsync } from "../lib/hooks";
import type { CashFlowForecast, Category, ExpenseBudget, PlannedEntry, Project, TxType } from "../lib/types";
import "./pages.css";
import "./planning.css";

function addMonthsISO(months: number) {
  const value = new Date();
  value.setMonth(value.getMonth() + months);
  return `${value.getFullYear()}-${String(value.getMonth() + 1).padStart(2, "0")}-${String(value.getDate()).padStart(2, "0")}`;
}

export function Planning() {
  const [from, setFrom] = useState(todayISO());
  const [to, setTo] = useState(addMonthsISO(3));
  const [status, setStatus] = useState("aberto");
  const [creating, setCreating] = useState(false);
  const [editing, setEditing] = useState<PlannedEntry | null>(null);
  const now = new Date();
  const [budgetMonth, setBudgetMonth] = useState(`${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, "0")}`);

  const categories = useAsync<Category[]>(() => api.get("/categories"), []);
  const projects = useAsync<Project[]>(() => api.get("/projects"), []);
  const entryQuery = useMemo(() => {
    const q = new URLSearchParams({ status });
    if (from) q.set("from", from);
    if (to) q.set("to", to);
    return q.toString();
  }, [from, to, status]);
  const entries = useAsync<PlannedEntry[]>(() => api.get(`/planning?${entryQuery}`), [entryQuery]);
  const forecast = useAsync<CashFlowForecast>(() => api.get(`/planning/cash-flow?from=${from}&to=${to}`), [from, to]);
  const [year, month] = budgetMonth.split("-").map(Number);
  const budgets = useAsync<ExpenseBudget[]>(() => api.get(`/budgets?ano=${year}&mes=${month}`), [year, month]);

  const reloadAll = () => { entries.reload(); forecast.reload(); budgets.reload(); };
  const catName = (id?: number) => categories.data?.find((item) => item.id === id)?.nome;
  const projectName = (id?: number) => projects.data?.find((item) => item.id === id)?.nome;

  return (
    <div>
      <header className="page-head">
        <span className="kicker">04 / Planejamento</span>
        <h1>Fluxo de caixa futuro</h1>
        <p>Provisione entradas e saídas sem alterar o realizado. Ao dar baixa, o sistema cria a transação automaticamente.</p>
      </header>

      <div className="filters">
        <div className="field"><label>De</label><DatePicker ariaLabel="Início da projeção" value={from} onChange={setFrom} /></div>
        <div className="field"><label>Até</label><DatePicker ariaLabel="Fim da projeção" value={to} onChange={setTo} /></div>
        <div className="field"><label>Situação</label><Select ariaLabel="Situação" value={status} onChange={setStatus} options={[{ value: "aberto", label: "Em aberto" }, { value: "realizado", label: "Realizados" }]} /></div>
        <div className="toolbar-spacer" />
        <button className="btn btn-primary btn-sm" onClick={() => setCreating(true)}>+ Novo planejamento</button>
      </div>

      {forecast.loading ? <Spinner /> : forecast.error ? <ErrorBanner>{forecast.error}</ErrorBanner> : forecast.data && (
        <>
          {forecast.data.vencidos > 0 && <div className="planning-alert">{forecast.data.vencidos} lançamento(s) vencido(s) aguardando baixa.</div>}
          <div className="grid grid-4">
            <KpiMoney label="Saldo antes do período" value={forecast.data.saldo_inicial} />
            <KpiMoney label="Entradas previstas" value={forecast.data.entradas} accent="teal" />
            <KpiMoney label="Saídas previstas" value={forecast.data.saidas} accent="copper" />
            <KpiMoney label="Saldo projetado" value={forecast.data.saldo_final} accent={forecast.data.saldo_final >= 0 ? "teal" : "danger"} />
          </div>
          <div className="card panel dash-block">
            <SectionHead idx="02" title="Movimento projetado por data" />
            {forecast.data.dias.length === 0 ? <Empty>Nenhum movimento previsto no período.</Empty> : (
              <div className="forecast-ledger">
                {forecast.data.dias.map((day) => (
                  <div className="forecast-ledger-row" key={day.data}>
                    <span className="mono muted">{date(day.data)}</span>
                    <span className="tx-ganho num">+ {money(day.entradas)}</span>
                    <span className="tx-despesa num">− {money(day.saidas)}</span>
                    <strong className="num">{money(day.saldo_projetado)}</strong>
                  </div>
                ))}
              </div>
            )}
          </div>
        </>
      )}

      <div className="card table-wrap dash-block">
        <div className="panel"><SectionHead idx="03" title={status === "aberto" ? "Contas em aberto" : "Planejamentos realizados"} /></div>
        {entries.loading ? <Spinner /> : entries.error ? <ErrorBanner>{entries.error}</ErrorBanner> : !entries.data?.length ? <Empty>Nenhum lançamento no período.</Empty> : (
          <table><thead><tr><th>Vencimento</th><th>Descrição</th><th>Projeto / categoria</th><th>Situação</th><th style={{ textAlign: "right" }}>Valor</th><th /></tr></thead>
          <tbody>{entries.data.map((item) => <tr key={item.id}>
            <td className="mono muted">{date(item.due_date)}</td><td>{item.descricao}</td>
            <td className="muted">{projectName(item.project_id) ?? catName(item.category_id) ?? "—"}</td>
            <td><span className={`pill ${item.overdue ? "pill-danger" : item.status === "realizado" ? "pill-ok" : "pill-pending"}`}>{item.overdue ? "vencido" : item.status}</span></td>
            <td className={`num ${item.tipo === "ganho" ? "tx-ganho" : "tx-despesa"}`} style={{ textAlign: "right" }}>{item.tipo === "ganho" ? "+" : "−"} {money(item.valor)}</td>
            <td style={{ textAlign: "right" }}>{item.status === "aberto" && <div className="row-actions">
              <button className="btn btn-ghost btn-sm" onClick={() => setEditing(item)}>Editar</button>
              <button className="btn btn-primary btn-sm" onClick={async () => { if (confirm("Dar baixa e criar a transação realizada?")) { await api.post(`/planning/${item.id}/complete`, { data: todayISO() }); reloadAll(); } }}>Dar baixa</button>
              <button className="btn btn-danger btn-sm" onClick={async () => { if (confirm("Excluir este planejamento?")) { await api.del(`/planning/${item.id}`); reloadAll(); } }}>Excluir</button>
            </div>}</td>
          </tr>)}</tbody></table>
        )}
      </div>

      <BudgetPanel month={budgetMonth} onMonth={setBudgetMonth} categories={categories.data ?? []} data={budgets.data ?? []} loading={budgets.loading} error={budgets.error} reload={budgets.reload} />

      {creating && <NewPlanningModal categories={categories.data ?? []} projects={projects.data ?? []} onClose={() => setCreating(false)} onCreated={() => { setCreating(false); reloadAll(); }} />}
      {editing && <NewPlanningModal initial={editing} categories={categories.data ?? []} projects={projects.data ?? []} onClose={() => setEditing(null)} onCreated={() => { setEditing(null); reloadAll(); }} />}
    </div>
  );
}

function BudgetPanel({ month, onMonth, categories, data, loading, error, reload }: { month: string; onMonth: (v: string) => void; categories: Category[]; data: ExpenseBudget[]; loading: boolean; error: string | null; reload: () => void }) {
  const [category, setCategory] = useState(""); const [value, setValue] = useState(""); const [formError, setFormError] = useState<string | null>(null);
  async function submit(e: FormEvent) { e.preventDefault(); const cents = toCentavos(value); if (!category || !cents || cents <= 0) return setFormError("Selecione uma categoria e informe um valor válido."); const [ano, mes] = month.split("-").map(Number); await api.put("/budgets", { category_id: Number(category), ano, mes, valor: cents }); setValue(""); setFormError(null); reload(); }
  return <div className="card panel dash-block"><SectionHead idx="04" title="Orçamento por categoria" />
    <form className="budget-form" onSubmit={submit}><div className="field"><label>Mês</label><input type="month" value={month} onChange={(e) => onMonth(e.target.value)} /></div><div className="field"><label>Categoria</label><Select ariaLabel="Categoria do orçamento" value={category} onChange={setCategory} options={[{ value: "", label: "Selecione" }, ...categories.map((c) => ({ value: String(c.id), label: c.nome }))]} /></div><div className="field"><label>Limite (R$)</label><input inputMode="decimal" value={value} onChange={(e) => setValue(e.target.value)} placeholder="5.000,00" /></div><button className="btn btn-primary btn-sm">Salvar limite</button></form>
    {formError && <ErrorBanner>{formError}</ErrorBanner>}{loading ? <Spinner /> : error ? <ErrorBanner>{error}</ErrorBanner> : data.length === 0 ? <Empty>Sem limites definidos para {monthLabel(Number(month.slice(5)))}/{month.slice(0, 4)}.</Empty> : <div className="budget-list">{data.map((b) => <div className="budget-item" key={b.id}><div className="budget-head"><strong>{b.category}</strong><span className={b.percentual > 100 ? "danger-text num" : "num"}>{money(b.realizado)} / {money(b.valor)}</span></div><Bar label={`${b.percentual}% utilizado`} value={Math.min(b.realizado, b.valor)} total={b.valor} tone={b.percentual > 100 ? "copper" : undefined} /><button className="btn btn-ghost btn-sm" onClick={async () => { await api.del(`/budgets/${b.id}`); reload(); }}>Remover</button></div>)}</div>}
  </div>;
}

function NewPlanningModal({ initial, categories, projects, onClose, onCreated }: { initial?: PlannedEntry; categories: Category[]; projects: Project[]; onClose: () => void; onCreated: () => void }) {
  const [tipo, setTipo] = useState<TxType>(initial?.tipo ?? "despesa"); const [valor, setValor] = useState(initial ? String(initial.valor / 100).replace(".", ",") : ""); const [dueDate, setDueDate] = useState(initial?.due_date ?? todayISO()); const [descricao, setDescricao] = useState(initial?.descricao ?? ""); const [category, setCategory] = useState(initial?.category_id ? String(initial.category_id) : ""); const [project, setProject] = useState(initial?.project_id ? String(initial.project_id) : ""); const [repeat, setRepeat] = useState("1"); const [error, setError] = useState<string | null>(null); const [busy, setBusy] = useState(false);
  async function submit(e: FormEvent) { e.preventDefault(); const cents = toCentavos(valor); if (!cents || cents <= 0 || !descricao.trim()) return setError("Informe descrição e valor válidos."); const body: Record<string, unknown> = { tipo, valor: cents, due_date: dueDate, descricao, repeat_months: Number(repeat) }; if (project) body.project_id = Number(project); if (tipo === "despesa" && category) body.category_id = Number(category); if (tipo === "ganho") body.origem = "avulso"; setBusy(true); try { if (initial) await api.put(`/planning/${initial.id}`, body); else await api.post("/planning", body); onCreated(); } catch (err) { setError(err instanceof Error ? err.message : "Falha ao salvar"); } finally { setBusy(false); } }
  return <Modal title={initial ? "Editar planejamento" : "Novo planejamento"} onClose={onClose} width={540}><form onSubmit={submit} className="planning-form">{error && <ErrorBanner>{error}</ErrorBanner>}<div className="form-row"><div className="field"><label>Tipo</label><Select ariaLabel="Tipo" value={tipo} onChange={(v) => setTipo(v as TxType)} options={[{ value: "despesa", label: "Saída" }, { value: "ganho", label: "Entrada" }]} /></div><div className="field"><label>Valor (R$)</label><input inputMode="decimal" value={valor} onChange={(e) => setValor(e.target.value)} required /></div></div><div className="field"><label>Descrição</label><input value={descricao} onChange={(e) => setDescricao(e.target.value)} required /></div><div className="form-row"><div className="field"><label>Vencimento</label><DatePicker ariaLabel="Vencimento" value={dueDate} onChange={setDueDate} required /></div>{!initial && <div className="field"><label>Repetir</label><Select ariaLabel="Repetição mensal" value={repeat} onChange={setRepeat} options={[{ value: "1", label: "Não repetir" }, { value: "3", label: "3 meses" }, { value: "6", label: "6 meses" }, { value: "12", label: "12 meses" }, { value: "24", label: "24 meses" }]} /></div>}</div><div className="form-row"><div className="field"><label>Projeto</label><Select ariaLabel="Projeto" value={project} onChange={setProject} options={[{ value: "", label: "Nenhum" }, ...projects.map((p) => ({ value: String(p.id), label: p.nome }))]} /></div>{tipo === "despesa" && <div className="field"><label>Categoria</label><Select ariaLabel="Categoria" value={category} onChange={setCategory} options={[{ value: "", label: "Sem categoria" }, ...categories.map((c) => ({ value: String(c.id), label: c.nome }))]} /></div>}</div><div className="modal-actions"><button type="button" className="btn btn-ghost btn-sm" onClick={onClose}>Cancelar</button><button className="btn btn-primary btn-sm" disabled={busy}>{busy ? "Salvando…" : initial ? "Salvar" : "Planejar"}</button></div></form></Modal>;
}
