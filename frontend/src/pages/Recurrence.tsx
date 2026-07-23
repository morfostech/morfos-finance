import { useState } from "react";
import { useSearchParams } from "react-router-dom";
import { api } from "../lib/api";
import { useAsync } from "../lib/hooks";
import { date, money, monthLabel } from "../lib/format";
import type { ProjectRecurrence, RecurrenceSummary } from "../lib/types";
import { BackButton, ChartTooltip, Empty, ErrorBanner, KpiMoney, SectionHead, Select, Spinner } from "../components/ui";
import "./pages.css";
import "./dashboard.css";

const now = new Date();

function statusLabel(project: ProjectRecurrence) {
  if (project.situacao === "parcial") return project.vencido > 0 ? "PARCIAL · VENCIDO" : "PARCIAL";
  return project.situacao.replace("_", " ").toUpperCase();
}

function statusClass(project: ProjectRecurrence) {
  if (project.situacao === "quitado" || project.situacao === "recebido") return "pill-ok";
  if (project.vencido > 0 || project.situacao === "vencido") return "pill-danger";
  return "pill-pending";
}

export function Recurrence() {
  const [searchParams, setSearchParams] = useSearchParams();
  const initialYear = Number(searchParams.get("ano")) || now.getFullYear();
  const initialMonth = Number(searchParams.get("mes")) || now.getMonth() + 1;
  const [ano, setAno] = useState(initialYear);
  const [mes, setMes] = useState(initialMonth >= 1 && initialMonth <= 12 ? initialMonth : now.getMonth() + 1);

  const updatePeriod = (nextYear: number, nextMonth: number) => {
    setAno(nextYear);
    setMes(nextMonth);
    setSearchParams({ ano: String(nextYear), mes: String(nextMonth) });
  };

  const month = useAsync<RecurrenceSummary>(() => api.get(`/recurrence?ano=${ano}&mes=${mes}`), [ano, mes]);
  const timeline = useAsync<RecurrenceSummary[]>(() => api.get(`/recurrence/timeline?ano=${ano}`), [ano]);
  const maxValue = Math.max(1, ...(timeline.data ?? []).flatMap((item) => [item.previsto, item.recebido]));

  return (
    <div>
      <BackButton />
      <header className="page-head">
        <span className="kicker">04 / Recorrência</span>
        <h1>Receita recorrente</h1>
        <p>Mensalidades por vencimento: recebido, vencido e a vencer, respeitando as datas de cada projeto.</p>
      </header>

      <div className="toolbar">
        <div className="field">
          <label>Mês</label>
          <Select
            ariaLabel="Mês da recorrência"
            value={String(mes)}
            onChange={(value) => updatePeriod(ano, Number(value))}
            options={Array.from({ length: 12 }, (_, index) => ({ value: String(index + 1), label: monthLabel(index + 1) }))}
          />
        </div>
        <div className="field">
          <label>Ano</label>
          <input type="number" value={ano} onChange={(event) => updatePeriod(Number(event.target.value), mes)} style={{ width: 110 }} />
        </div>
      </div>

      {month.loading ? <Spinner /> : month.error ? <ErrorBanner>{month.error}</ErrorBanner> : month.data ? (
        <>
          <div className="grid grid-4">
            <KpiMoney label="Previsto no mês" value={month.data.previsto} />
            <KpiMoney label="Recebido no mês" value={month.data.recebido} accent="teal" />
            <KpiMoney label="Em atraso" value={month.data.vencido} accent="danger" />
            <KpiMoney label="A vencer" value={month.data.a_vencer} accent="copper" />
          </div>

          <div className="card panel dash-block">
            <SectionHead title={`Projetos · ${monthLabel(mes)}/${ano}`} />
            {month.data.projetos.length === 0 ? <Empty>Nenhuma mensalidade com vencimento neste mês.</Empty> : (
              <div className="table-wrap">
                <table>
                  <thead><tr><th>Projeto</th><th>Vencimento</th><th style={{ textAlign: "right" }}>Previsto</th><th style={{ textAlign: "right" }}>Recebido</th><th style={{ textAlign: "right" }}>Em aberto</th><th>Situação</th></tr></thead>
                  <tbody>{month.data.projetos.map((project) => (
                    <tr key={project.project_id}>
                      <td style={{ fontWeight: 600 }}>{project.nome}</td>
                      <td className="mono muted">{date(project.vencimento)}</td>
                      <td className="num" style={{ textAlign: "right" }}>{money(project.previsto)}</td>
                      <td className="num accent-teal" style={{ textAlign: "right" }}>{money(project.recebido)}</td>
                      <td className={`num ${project.vencido > 0 ? "accent-danger" : "accent-copper"}`} style={{ textAlign: "right" }}>{money(project.pendente)}</td>
                      <td><span className={`pill ${statusClass(project)}`}>{statusLabel(project)}</span></td>
                    </tr>
                  ))}</tbody>
                </table>
              </div>
            )}
          </div>
        </>
      ) : null}

      <div className="card panel dash-block">
        <SectionHead title={`Linha do tempo por vencimento · ${ano}`} />
        <p className="panel-explainer">As barras usam a mesma escala. O valor planejado e o valor recebido são mostrados separadamente, sem dupla proporcionalidade.</p>
        {timeline.loading ? <Spinner /> : timeline.error ? <ErrorBanner>{timeline.error}</ErrorBanner> : (
          <div className="timeline-scroll">
            <div className="timeline">
              {(timeline.data ?? []).map((item) => {
                const plannedHeight = (item.previsto / maxValue) * 100;
                const receivedHeight = (item.recebido / maxValue) * 100;
                const state = item.vencido > 0 ? "VENCIDO" : item.a_vencer > 0 ? "A VENCER" : item.previsto > 0 ? "QUITADO" : "SEM COBRANÇA";
                return (
                  <ChartTooltip
                    key={item.mes}
                    className="tl-col"
                    label={`${monthLabel(item.mes)}/${ano}`}
                    value={`Previsto ${money(item.previsto)}`}
                    description={`Recebido ${money(item.recebido)} · vencido ${money(item.vencido)} · a vencer ${money(item.a_vencer)}.`}
                  >
                    <div className="tl-values"><strong>{money(item.previsto)}</strong><span>{money(item.recebido)} recebido</span></div>
                    <div className="tl-plot"><span className="tl-planned" style={{ height: `${plannedHeight}%` }} /><span className="tl-received" style={{ height: `${receivedHeight}%` }} /></div>
                    <span className="tl-label mono">{monthLabel(item.mes)}</span>
                    <span className={`tl-state mono ${item.vencido > 0 ? "is-overdue" : ""}`}>{state}</span>
                  </ChartTooltip>
                );
              })}
            </div>
          </div>
        )}
        <div className="tl-legend mono muted"><span><i className="sw sw-track" /> previsto</span><span><i className="sw sw-fill" /> recebido</span></div>
      </div>
    </div>
  );
}
