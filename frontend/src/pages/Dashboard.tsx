import { useState } from "react";
import { Link } from "react-router-dom";
import { api } from "../lib/api";
import { useAuth } from "../lib/auth";
import { useAsync } from "../lib/hooks";
import { currentMonthRange, money, monthLabel } from "../lib/format";
import { canManage, type CompanyDashboard, type MeDashboard } from "../lib/types";
import { Bar, Empty, ErrorBanner, KpiMoney, SectionHead, Spinner } from "../components/ui";
import "./dashboard.css";

export function Dashboard() {
  const { user } = useAuth();
  const hasCompanyView = canManage(user?.role);
  const [view, setView] = useState<"empresa" | "pessoal">("empresa");
  const [range, setRange] = useState(currentMonthRange());

  const showCompany = hasCompanyView && view === "empresa";

  return (
    <div>
      <header className="page-head">
        <span className="kicker">01 / Visão geral</span>
        <h1>{showCompany ? "Painel da empresa" : "Meu painel"}</h1>
        <p>
          {showCompany
            ? "Resultados no período selecionado. Saldo e parcelas de implementação são acumulados."
            : "Seus ganhos e despesas no período, com apenas os projetos em que você está alocado."}
        </p>
      </header>

      {hasCompanyView && (
        <div className="view-toggle">
          <button className={view === "empresa" ? "active" : ""} onClick={() => setView("empresa")}>
            Empresa
          </button>
          <button className={view === "pessoal" ? "active" : ""} onClick={() => setView("pessoal")}>
            Minha visão
          </button>
        </div>
      )}

      <div className="toolbar">
        <div className="field">
          <label>De</label>
          <input type="date" value={range.from} onChange={(e) => setRange({ ...range, from: e.target.value })} />
        </div>
        <div className="field">
          <label>Até</label>
          <input type="date" value={range.to} onChange={(e) => setRange({ ...range, to: e.target.value })} />
        </div>
        <div className="toolbar-spacer" />
        <button className="btn btn-ghost btn-sm" onClick={() => setRange(currentMonthRange())}>
          Mês atual
        </button>
      </div>

      {showCompany ? <CompanyView from={range.from} to={range.to} /> : <MeView from={range.from} to={range.to} />}
    </div>
  );
}

function CompanyView({ from, to }: { from: string; to: string }) {
  const { data, loading, error } = useAsync<CompanyDashboard>(
    () => api.get(`/dashboard/company?from=${from}&to=${to}`),
    [from, to],
  );

  if (loading) return <Spinner />;
  if (error) return <ErrorBanner>{error}</ErrorBanner>;
  if (!data) return null;

  const rec = data.recorrencia_mes;
  const despTotal = data.despesas_por_categoria.reduce((s, c) => s + c.total, 0);

  return (
    <div className="dash">
      <div className="grid grid-4">
        <KpiMoney label="Saldo em caixa" value={data.saldo_em_caixa} accent="teal" hint="acumulado" />
        <KpiMoney label="Ganhos no período" value={data.ganhos} />
        <KpiMoney label="Despesas no período" value={data.despesas} accent="copper" />
        <KpiMoney
          label="Resultado"
          value={data.resultado}
          accent={data.resultado >= 0 ? "teal" : "danger"}
        />
      </div>

      <div className="grid grid-2 dash-block">
        <div className="card panel">
          <SectionHead idx="02" title="Implementação acumulada × recorrência mensal" />
          <div className="split-row">
            <div>
              <div className="split-k mono">Implementação</div>
              <div className="split-v">{money(data.implementacao.a_receber)}</div>
              <div className="split-s muted">
                a receber · {money(data.implementacao.recebido)} recebido
              </div>
            </div>
            <div>
              <div className="split-k mono">Recorrência (mês)</div>
              <div className="split-v">{money(rec.pendente)}</div>
              <div className="split-s muted">
                pendente · {money(rec.recebido)} recebido
              </div>
            </div>
          </div>
          <div className="parcelas-note mono muted">
            acumulado · {data.parcelas_pendentes.quantidade} parcela(s) pendente(s) · {money(data.parcelas_pendentes.total)}
          </div>
        </div>

        <div className="card panel">
          <SectionHead idx="03" title="Ganhos por origem" />
          <Bar label="Implementação" value={data.ganhos_por_origem.implementacao} total={data.ganhos || 1} />
          <Bar label="Recorrência" value={data.ganhos_por_origem.recorrencia} total={data.ganhos || 1} />
          <Bar label="Avulso" value={data.ganhos_por_origem.avulso} total={data.ganhos || 1} />
          {data.ganhos_por_origem.sem_origem > 0 && (
            <Bar label="Sem origem" value={data.ganhos_por_origem.sem_origem} total={data.ganhos || 1} />
          )}
        </div>
      </div>

      <div className="grid grid-2 dash-block">
        <div className="card panel">
          <SectionHead idx="04" title="Despesas por categoria" />
          {data.despesas_por_categoria.length === 0 ? (
            <Empty>Sem despesas no período.</Empty>
          ) : (
            data.despesas_por_categoria.map((c) => (
              <Bar key={c.category_id ?? "none"} label={c.nome} value={c.total} total={despTotal || 1} tone="copper" />
            ))
          )}
        </div>

        <div className="card panel">
          <SectionHead idx="05" title={`Recorrência · ${monthLabel(rec.mes)}/${rec.ano}`} />
          <div className="rec-totals">
            <span>Previsto <b className="num">{money(rec.previsto)}</b></span>
            <span>Recebido <b className="num accent-teal">{money(rec.recebido)}</b></span>
            <span>Pendente <b className="num accent-copper">{money(rec.pendente)}</b></span>
          </div>
          <div className="rec-list">
            {rec.projetos.map((p) => (
              <div key={p.project_id} className="rec-item">
                <span className="rec-name">{p.nome}</span>
                <span className="num muted">{money(p.recebido)} / {money(p.previsto)}</span>
                <span className={`pill ${p.pendente === 0 ? "pill-ok" : "pill-pending"}`}>
                  {p.pendente === 0 ? "quitado" : money(p.pendente)}
                </span>
              </div>
            ))}
          </div>
        </div>
      </div>

      <div className="grid grid-2 dash-block">
        <BreakdownTable title="Por projeto" idx="06" rows={data.por_projeto.map((p) => ({ nome: p.nome, ganhos: p.ganhos, despesas: p.despesas }))} />
        <BreakdownTable title="Por colaborador" idx="07" rows={data.por_colaborador.map((u) => ({ nome: u.nome, ganhos: u.ganhos, despesas: u.despesas }))} />
      </div>
    </div>
  );
}

function BreakdownTable({ title, idx, rows }: { title: string; idx: string; rows: { nome: string; ganhos: number; despesas: number }[] }) {
  return (
    <div className="card panel">
      <SectionHead idx={idx} title={title} />
      {rows.length === 0 ? (
        <Empty>Nada no período.</Empty>
      ) : (
        <table>
          <thead>
            <tr>
              <th>Nome</th>
              <th style={{ textAlign: "right" }}>Ganhos</th>
              <th style={{ textAlign: "right" }}>Despesas</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((r) => (
              <tr key={r.nome}>
                <td>{r.nome}</td>
                <td className="num" style={{ textAlign: "right", color: "var(--teal)" }}>{money(r.ganhos)}</td>
                <td className="num" style={{ textAlign: "right", color: "var(--copper)" }}>{money(r.despesas)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}

function MeView({ from, to }: { from: string; to: string }) {
  const { data, loading, error } = useAsync<MeDashboard>(
    () => api.get(`/dashboard/me?from=${from}&to=${to}`),
    [from, to],
  );

  if (loading) return <Spinner />;
  if (error) return <ErrorBanner>{error}</ErrorBanner>;
  if (!data) return null;

  return (
    <div className="dash">
      <div className="grid grid-3">
        <KpiMoney label="Meus ganhos" value={data.ganhos} accent="teal" />
        <KpiMoney label="Minhas despesas" value={data.despesas} accent="copper" />
        <KpiMoney label="Saldo" value={data.saldo} accent={data.saldo >= 0 ? "teal" : "danger"} />
      </div>

      <div className="card panel dash-block">
        <SectionHead idx="02" title="Meus projetos" />
        {data.projetos.length === 0 ? (
          <Empty>Você ainda não está alocado em projetos.</Empty>
        ) : (
          <div className="chip-grid">
            {data.projetos.map((p) => (
              <Link to={`/projetos/${p.id}`} key={p.id} className="proj-chip">
                <span className="proj-chip-name">{p.nome}</span>
                <span className="proj-chip-meta mono muted">{p.cliente ?? "—"}</span>
              </Link>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
