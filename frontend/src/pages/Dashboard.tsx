import { useState } from "react";
import { Link } from "react-router-dom";
import { api } from "../lib/api";
import { useAuth } from "../lib/auth";
import { useAsync } from "../lib/hooks";
import { currentMonthRange, money, monthLabel } from "../lib/format";
import { canManage, type CompanyDashboard, type MeDashboard } from "../lib/types";
import { Bar, ChartTooltip, Empty, ErrorBanner, KpiMoney, SectionHead, Spinner } from "../components/ui";
import { DatePicker } from "../components/DatePicker";
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
          <DatePicker ariaLabel="Data inicial" value={range.from} onChange={(value) => setRange({ ...range, from: value })} />
        </div>
        <div className="field">
          <label>Até</label>
          <DatePicker ariaLabel="Data final" value={range.to} onChange={(value) => setRange({ ...range, to: value })} />
        </div>
        <div className="toolbar-spacer" />
        <div className="period-shortcuts" aria-label="Atalhos de período">
          <button className="btn btn-ghost btn-sm" onClick={() => setRange(monthRange(3))}>3 meses</button>
          <button className="btn btn-ghost btn-sm" onClick={() => setRange(monthRange(6))}>6 meses</button>
          <button className="btn btn-ghost btn-sm" onClick={() => setRange(yearRange())}>Ano atual</button>
        </div>
        <button className="btn btn-ghost btn-sm" onClick={() => setRange(currentMonthRange())}>
          Mês atual
        </button>
      </div>

      {showCompany ? <CompanyView from={range.from} to={range.to} /> : <MeView from={range.from} to={range.to} />}
    </div>
  );
}

function localISO(value: Date) {
  const pad = (part: number) => String(part).padStart(2, "0");
  return `${value.getFullYear()}-${pad(value.getMonth() + 1)}-${pad(value.getDate())}`;
}

function monthRange(months: number) {
  const now = new Date();
  return {
    from: localISO(new Date(now.getFullYear(), now.getMonth() - months + 1, 1)),
    to: localISO(new Date(now.getFullYear(), now.getMonth() + 1, 0)),
  };
}

function yearRange() {
  const now = new Date();
  return { from: `${now.getFullYear()}-01-01`, to: `${now.getFullYear()}-12-31` };
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
  const recPeriod = data.recorrencia_periodo;
  const forecast = data.recorrencia_futura;
  const forecastMonths = forecast?.meses ?? [];
  const maxForecast = Math.max(1, ...forecastMonths.map((month) => month.previsto));
  const despTotal = data.despesas_por_categoria.reduce((s, c) => s + c.total, 0);

  const transactionsUrl = (params: Record<string, string | undefined>) => {
    const query = new URLSearchParams();
    Object.entries(params).forEach(([key, value]) => value && query.set(key, value));
    return `/transacoes?${query.toString()}`;
  };
  const periodUrl = transactionsUrl({ from, to, contexto: "periodo" });
  const gainsUrl = transactionsUrl({ tipo: "ganho", from, to, contexto: "ganhos" });
  const expensesUrl = transactionsUrl({ tipo: "despesa", from, to, contexto: "despesas" });

  return (
    <div className="dash">
      <div className="grid grid-4">
        <DashboardKpi to="/transacoes?contexto=saldo" label="Saldo em caixa" value={data.saldo_em_caixa} accent="teal" hint="realizado · acumulado até hoje" />
        <DashboardKpi to={gainsUrl} label="Entradas realizadas" value={data.ganhos} hint="no período selecionado" />
        <DashboardKpi to={expensesUrl} label="Saídas realizadas" value={data.despesas} accent="copper" hint="no período selecionado" />
        <DashboardKpi to={periodUrl} label="Resultado realizado" value={data.resultado} accent={data.resultado >= 0 ? "teal" : "danger"} hint="entradas menos saídas" />
      </div>

      <div className="grid grid-2 dash-block">
        <div className="card panel">
          <SectionHead idx="02" title="Realizado × valores a receber" action={<Link className="panel-link" to="/planejamento">Ver planejamento →</Link>} />
          <div className="decision-summary">
            <div className="decision-row">
              <div><span className="status-dot status-realized" /><span>Receita realizada no período</span></div>
              <strong className="num accent-teal">{money(data.ganhos)}</strong>
            </div>
            <div className="decision-row">
              <div><span className="status-dot status-overdue" /><span>Recorrência vencida no período</span></div>
              <strong className="num accent-danger">{money(recPeriod.vencido)}</strong>
            </div>
            <div className="decision-row">
              <div><span className="status-dot status-pending" /><span>Recorrência a vencer no período</span></div>
              <strong className="num accent-copper">{money(recPeriod.a_vencer)}</strong>
            </div>
            <div className="decision-row">
              <div><span className="status-dot status-future" /><span>Implementação a receber</span></div>
              <strong className="num">{money(data.implementacao.a_receber)}</strong>
            </div>
          </div>
          <div className="parcelas-note mono muted">
            implementação acumulada · {data.parcelas_pendentes.quantidade} parcela(s) pendente(s) · {money(data.parcelas_pendentes.total)}
          </div>
        </div>

        <div className="card panel">
          <SectionHead idx="03" title="Entradas realizadas por origem" action={<Link className="panel-link" to={gainsUrl}>Ver lançamentos →</Link>} />
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
          <SectionHead idx="04" title="Saídas realizadas por categoria" action={<Link className="panel-link" to={expensesUrl}>Ver lançamentos →</Link>} />
          {data.despesas_por_categoria.length === 0 ? (
            <Empty>Sem despesas no período.</Empty>
          ) : (
            data.despesas_por_categoria.map((c) => (
              <Bar key={c.category_id ?? "none"} label={c.nome} value={c.total} total={despTotal || 1} tone="copper" />
            ))
          )}
        </div>

        <div className="card panel">
          <SectionHead idx="05" title="Recorrência no período" action={<Link className="panel-link" to={`/recorrencia?ano=${rec.ano}&mes=${rec.mes}`}>Abrir recorrência →</Link>} />
          <p className="panel-explainer">Cada mensalidade entra na data de vencimento quando essa data pertence ao período ativo do projeto. Recebido considera apenas transações efetivamente lançadas.</p>
          <div className="rec-totals rec-totals-featured">
            <span>Previsto acumulado <b className="num">{money(recPeriod.previsto)}</b></span>
            <span>Recebido <b className="num accent-teal">{money(recPeriod.recebido)}</b></span>
            <span>Vencido <b className="num accent-danger">{money(recPeriod.vencido)}</b></span>
            <span>A vencer <b className="num accent-copper">{money(recPeriod.a_vencer)}</b></span>
          </div>
          <div className="period-months" aria-label="Recorrência por mês do período">
            {recPeriod.meses.map((item) => (
              <ChartTooltip
                className="period-month"
                key={`${item.ano}-${item.mes}`}
                label={`${monthLabel(item.mes)}/${item.ano}`}
                value={`Previsto ${money(item.previsto)}`}
                description={`Recebido ${money(item.recebido)} · vencido ${money(item.vencido)} · a vencer ${money(item.a_vencer)}.`}
              >
                <div className="period-month-head"><span className="mono">{monthLabel(item.mes)}/{String(item.ano).slice(-2)}</span><strong className="num">{money(item.previsto)}</strong></div>
                <div className="period-month-bar"><span style={{ width: `${item.previsto > 0 ? Math.min(100, (item.recebido / item.previsto) * 100) : 0}%` }} /></div>
                <div className="period-month-meta"><span>recebido {money(item.recebido)}</span><span>vencido {money(item.vencido)}</span><span>a vencer {money(item.a_vencer)}</span></div>
              </ChartTooltip>
            ))}
          </div>
          {forecast ? (
            <div className="forecast">
              <div className="forecast-head">
                <div>
                  <span className="split-k mono">Projeção após o período · {forecast.horizonte_meses} meses</span>
                  <span className="forecast-note muted">não compõe o caixa realizado</span>
                </div>
                <strong className="forecast-total num">{money(forecast.total)}</strong>
              </div>
              <div className="forecast-chart" aria-label="Previsão mensal de recorrência">
                {forecastMonths.map((month) => (
                  <ChartTooltip
                    className="forecast-month"
                    key={`${month.ano}-${month.mes}`}
                    label={`${monthLabel(month.mes)}/${month.ano}`}
                    value={`Previsto ${money(month.previsto)}`}
                    description="Projeção futura; ainda não compõe o caixa realizado."
                  >
                    <div className="forecast-track">
                      <span
                        style={{
                          height: `${month.previsto === 0 ? 0 : Math.max(4, (month.previsto / maxForecast) * 100)}%`,
                        }}
                      />
                    </div>
                    <span className="forecast-label mono">{monthLabel(month.mes)}</span>
                    <span className="forecast-year mono">{String(month.ano).slice(-2)}</span>
                  </ChartTooltip>
                ))}
              </div>
            </div>
          ) : (
            <p className="forecast-unavailable mono">Previsão futura indisponível.</p>
          )}
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

function DashboardKpi({ to, label, value, hint, accent }: { to: string; label: string; value: number; hint?: string; accent?: "teal" | "copper" | "danger" | "bone" }) {
  return (
    <Link className="kpi-link" to={to} aria-label={`${label}: ${money(value)}. Abrir detalhes`}>
      <KpiMoney label={label} value={value} hint={hint} accent={accent} />
      <span className="kpi-open" aria-hidden>Ver detalhes <b>→</b></span>
    </Link>
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
        <DashboardKpi to={`/transacoes?tipo=ganho&from=${from}&to=${to}&contexto=ganhos`} label="Meus ganhos" value={data.ganhos} accent="teal" />
        <DashboardKpi to={`/transacoes?tipo=despesa&from=${from}&to=${to}&contexto=despesas`} label="Minhas despesas" value={data.despesas} accent="copper" />
        <DashboardKpi to={`/transacoes?from=${from}&to=${to}&contexto=periodo`} label="Saldo" value={data.saldo} accent={data.saldo >= 0 ? "teal" : "danger"} />
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
