import { useEffect, useId, useLayoutEffect, useRef, useState, type FocusEvent, type KeyboardEvent, type PointerEvent as ReactPointerEvent, type ReactNode } from "react";
import { useNavigate } from "react-router-dom";
import { money } from "../lib/format";

export interface SelectOption {
  value: string;
  label: string;
  disabled?: boolean;
}

export function Select({
  value,
  options,
  onChange,
  ariaLabel,
  disabled = false,
}: {
  value: string;
  options: SelectOption[];
  onChange: (value: string) => void;
  ariaLabel: string;
  disabled?: boolean;
}) {
  const [open, setOpen] = useState(false);
  const selectedIndex = Math.max(0, options.findIndex((option) => option.value === value));
  const [activeIndex, setActiveIndex] = useState(selectedIndex);
  const rootRef = useRef<HTMLDivElement>(null);
  const menuRef = useRef<HTMLDivElement>(null);
  const listboxId = useId();
  const selected = options[selectedIndex];

  useEffect(() => {
    if (!open) setActiveIndex(selectedIndex);
  }, [open, selectedIndex]);

  useEffect(() => {
    if (!open) return;
    const closeOutside = (event: PointerEvent) => {
      if (!rootRef.current?.contains(event.target as Node)) setOpen(false);
    };
    document.addEventListener("pointerdown", closeOutside);
    return () => document.removeEventListener("pointerdown", closeOutside);
  }, [open]);

  useLayoutEffect(() => {
    if (!open) return;
    const menu = menuRef.current;
    const option = menu?.querySelector<HTMLElement>('[aria-selected="true"]');
    if (!menu || !option) return;
    menu.scrollTop = Math.max(0, option.offsetTop - (menu.clientHeight - option.offsetHeight) / 2);
  }, [open]);

  const move = (direction: 1 | -1) => {
    let next = activeIndex;
    do next = (next + direction + options.length) % options.length;
    while (options[next]?.disabled && next !== activeIndex);
    setActiveIndex(next);
  };

  const choose = (index: number) => {
    const option = options[index];
    if (!option || option.disabled) return;
    onChange(option.value);
    setOpen(false);
  };

  const handleKeyDown = (event: KeyboardEvent<HTMLButtonElement>) => {
    if (event.key === "ArrowDown" || event.key === "ArrowUp") {
      event.preventDefault();
      if (!open) setOpen(true);
      else move(event.key === "ArrowDown" ? 1 : -1);
      return;
    }
    if (event.key === "Home" || event.key === "End") {
      if (!open) return;
      event.preventDefault();
      const ordered = event.key === "Home" ? options : [...options].reverse();
      const found = ordered.findIndex((option) => !option.disabled);
      const next = event.key === "Home" ? found : options.length - found - 1;
      if (next >= 0) setActiveIndex(next);
      return;
    }
    if (event.key === "Enter" || event.key === " ") {
      event.preventDefault();
      if (open) choose(activeIndex);
      else setOpen(true);
      return;
    }
    if (event.key === "Escape" && open) {
      event.preventDefault();
      event.stopPropagation();
      setOpen(false);
    }
  };

  return (
    <div className={`select ${open ? "is-open" : ""}`} ref={rootRef}>
      <button
        type="button"
        className="select-trigger"
        aria-label={ariaLabel}
        aria-haspopup="listbox"
        aria-expanded={open}
        aria-controls={open ? listboxId : undefined}
        aria-activedescendant={open ? `${listboxId}-${activeIndex}` : undefined}
        disabled={disabled}
        onClick={() => setOpen((current) => !current)}
        onKeyDown={handleKeyDown}
      >
        <span>{selected?.label ?? "Selecione"}</span>
        <span className="select-chevron" aria-hidden />
      </button>
      {open && (
        <div className="select-menu" id={listboxId} role="listbox" aria-label={ariaLabel} ref={menuRef}>
          {options.map((option, index) => (
            <button
              type="button"
              id={`${listboxId}-${index}`}
              role="option"
              aria-selected={option.value === value}
              className={`select-option ${index === activeIndex ? "is-active" : ""}`}
              key={option.value}
              disabled={option.disabled}
              onPointerMove={() => setActiveIndex(index)}
              onClick={() => choose(index)}
            >
              <span className="select-check" aria-hidden />
              <span>{option.label}</span>
            </button>
          ))}
        </div>
      )}
    </div>
  );
}

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

export function BackButton({ fallback = "/", label = "Voltar" }: { fallback?: string; label?: string }) {
  const navigate = useNavigate();
  return (
    <button
      type="button"
      className="back-link"
      onClick={() => Number(window.history.state?.idx ?? 0) > 0 ? navigate(-1) : navigate(fallback)}
      aria-label={`${label} para a página anterior`}
    >
      <span aria-hidden>←</span> {label}
    </button>
  );
}

export function ChartTooltip({
  label,
  value,
  description,
  className,
  children,
}: {
  label: string;
  value: string;
  description: string;
  className?: string;
  children: ReactNode;
}) {
  const tooltipId = useId();
  const [position, setPosition] = useState<{ x: number; y: number } | null>(null);

  const place = (x: number, y: number) => {
    const tooltipWidth = 260;
    const tooltipHeight = 96;
    setPosition({
      x: Math.max(8, Math.min(x + 14, window.innerWidth - tooltipWidth - 8)),
      y: Math.max(8, Math.min(y + 14, window.innerHeight - tooltipHeight - 8)),
    });
  };
  const followPointer = (event: ReactPointerEvent<HTMLDivElement>) => place(event.clientX, event.clientY);
  const placeByElement = (event: FocusEvent<HTMLDivElement>) => {
    const rect = event.currentTarget.getBoundingClientRect();
    place(rect.left + rect.width / 2, rect.top + rect.height / 2);
  };

  return (
    <div
      className={`chart-tooltip-target ${className ?? ""}`}
      tabIndex={0}
      aria-label={`${label}. ${value}. ${description}`}
      aria-describedby={position ? tooltipId : undefined}
      onPointerEnter={followPointer}
      onPointerMove={followPointer}
      onPointerLeave={() => setPosition(null)}
      onFocus={placeByElement}
      onBlur={() => setPosition(null)}
    >
      {children}
      {position && (
        <div id={tooltipId} role="tooltip" className="chart-tooltip-bubble" style={{ left: position.x, top: position.y }}>
          <span className="chart-tooltip-label mono">{label}</span>
          <strong className="chart-tooltip-value num">{value}</strong>
          <span className="chart-tooltip-description">{description}</span>
        </div>
      )}
    </div>
  );
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
