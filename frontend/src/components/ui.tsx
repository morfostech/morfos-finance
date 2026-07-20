import type { ReactNode } from "react";
import { money } from "../lib/format";

/** Numbered section header in the Morfos style (mono kicker + rule). */
export function SectionHead({ idx, title, action }: { idx?: string; title: string; action?: ReactNode }) {
  return (
    <div className="section-head">
      <div className="section-head-l">
        {idx && <span className="section-idx mono">{idx}</span>}
        <h2>{title}</h2>
      </div>
      {action}
    </div>
  );
}

/** KPI tile — big value in display font, mono label. Accent tints the value. */
export function Kpi({
  label,
  value,
  hint,
  accent,
}: {
  label: string;
  value: ReactNode;
  hint?: string;
  accent?: "teal" | "copper" | "danger" | "bone";
}) {
  return (
    <div className="kpi card">
      <div className="kpi-label mono">{label}</div>
      <div className={`kpi-value ${accent ? `accent-${accent}` : ""}`}>{value}</div>
      {hint && <div className="kpi-hint mono">{hint}</div>}
    </div>
  );
}

/** Money KPI convenience. */
export function KpiMoney(props: { label: string; value: number; hint?: string; accent?: "teal" | "copper" | "danger" | "bone" }) {
  return <Kpi {...props} value={money(props.value)} />;
}

export function Spinner({ label }: { label?: string }) {
  return (
    <div className="spinner">
      <span className="spinner-dot" />
      {label ?? "Carregando…"}
    </div>
  );
}

export function Empty({ children }: { children: ReactNode }) {
  return <div className="empty muted">{children}</div>;
}

export function ErrorBanner({ children }: { children: ReactNode }) {
  return <div className="error-banner">{children}</div>;
}

/** A labeled horizontal bar for simple distributions (expenses, recurrence). */
export function Bar({ label, value, total, tone = "teal" }: { label: string; value: number; total: number; tone?: "teal" | "copper" }) {
  const pct = total > 0 ? Math.round((value / total) * 100) : 0;
  return (
    <div className="bar-row">
      <div className="bar-top">
        <span>{label}</span>
        <span className="num muted">{money(value)}</span>
      </div>
      <div className="bar-track">
        <div className={`bar-fill bar-${tone}`} style={{ width: `${pct}%` }} />
      </div>
    </div>
  );
}
