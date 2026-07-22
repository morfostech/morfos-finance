import { useState } from "react";
import { useSearchParams } from "react-router-dom";
import { api } from "../lib/api";
import { useAsync } from "../lib/hooks";
import { money, monthLabel } from "../lib/format";
import type { RecurrenceSummary } from "../lib/types";
import { Empty, ErrorBanner, KpiMoney, SectionHead, Select, Spinner } from "../components/ui";
import "./pages.css";
import "./dashboard.css";

const now = new Date();

export function Recurrence() {
  const [searchParams] = useSearchParams();
  const initialYear = Number(searchParams.get("ano")) || now.getFullYear();
  const initialMonth = Number(searchParams.get("mes")) || now.getMonth() + 1;
  const [ano, setAno] = useState(initialYear);
  const [mes, setMes] = useState(initialMonth >= 1 && initialMonth <= 12 ? initialMonth : now.getMonth() + 1);

  const month = useAsync<RecurrenceSummary>(() => api.get(`/recurrence?ano=${ano}&mes=${mes}`), [ano, mes]);
  const timeline = useAsync<RecurrenceSummary[]>(() => api.get(`/recurrence/timeline?ano=${ano}`), [ano]);

  const maxPrev = Math.max(1, ...(timeline.data ?? []).map((m) => m.previsto));

  return (
    <div>
      <header className="page-head">
        <span className="kicker">04 / Recorrência</span>
        <h1>Receita recorrente</h1>
        <p>Previsto × recebido × pendente, calculado da mensalidade e do período de cada projeto.</p>
      </header>

      <div className="toolbar">
        <div className="field">
          <label>Mês</label>
          <Select
            ariaLabel="Mês da recorrência"
            value={String(mes)}
            onChange={(value) => setMes(Number(value))}
            options={Array.from({ length: 12 }, (_, index) => ({
              value: String(index + 1),
              label: monthLabel(index + 1),
            }))}
          />
        </div>
        <div className="field">
          <label>Ano</label>
          <input type="number" value={ano} onChange={(e) => setAno(Number(e.target.value))} style={{ width: 110 }} />
        </div>
      </div>

      {month.loading ? (
        <Spinner />
      ) : month.error ? (
        <ErrorBanner>{month.error}</ErrorBanner>
      ) : month.data ? (
        <>
          <div className="grid grid-3">
            <KpiMoney label="Previsto" value={month.data.previsto} />
            <KpiMoney label="Recebido" value={month.data.recebido} accent="teal" />
            <KpiMoney label="Pendente" value={month.data.pendente} accent="copper" />
          </div>

          <div className="card panel dash-block">
            <SectionHead title={`Projetos · ${monthLabel(mes)}/${ano}`} />
            {month.data.projetos.length === 0 ? (
              <Empty>Nenhum projeto recorrente ativo neste mês.</Empty>
            ) : (
              <table>
                <thead>
                  <tr>
                    <th>Projeto</th>
                    <th style={{ textAlign: "right" }}>Previsto</th>
                    <th style={{ textAlign: "right" }}>Recebido</th>
                    <th style={{ textAlign: "right" }}>Pendente</th>
                    <th></th>
                  </tr>
                </thead>
                <tbody>
                  {month.data.projetos.map((p) => (
                    <tr key={p.project_id}>
                      <td style={{ fontWeight: 600 }}>{p.nome}</td>
                      <td className="num" style={{ textAlign: "right" }}>{money(p.previsto)}</td>
                      <td className="num" style={{ textAlign: "right", color: "var(--teal)" }}>{money(p.recebido)}</td>
                      <td className="num" style={{ textAlign: "right", color: "var(--copper)" }}>{money(p.pendente)}</td>
                      <td style={{ textAlign: "right" }}>
                        <span className={`pill ${p.pendente === 0 ? "pill-ok" : "pill-pending"}`}>
                          {p.pendente === 0 ? "quitado" : "pendente"}
                        </span>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        </>
      ) : null}

      <div className="card panel dash-block">
        <SectionHead title={`Linha do tempo · ${ano}`} />
        {timeline.loading ? (
          <Spinner />
        ) : (
          <div className="timeline">
            {(timeline.data ?? []).map((m) => {
              const h = Math.round((m.previsto / maxPrev) * 100);
              const recH = m.previsto > 0 ? Math.round((m.recebido / m.previsto) * h) : 0;
              return (
                <div key={m.mes} className="tl-col" title={`Recebido ${money(m.recebido)} / Previsto ${money(m.previsto)}`}>
                  <div className="tl-bar" style={{ height: `${Math.max(h, 2)}%` }}>
                    <div className="tl-fill" style={{ height: `${recH}%` }} />
                  </div>
                  <span className="tl-label mono">{monthLabel(m.mes)}</span>
                </div>
              );
            })}
          </div>
        )}
        <div className="tl-legend mono muted">
          <span><i className="sw sw-track" /> previsto</span>
          <span><i className="sw sw-fill" /> recebido</span>
        </div>
      </div>
    </div>
  );
}
