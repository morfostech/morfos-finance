import { useEffect, useId, useLayoutEffect, useMemo, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { DayPicker } from "@daypicker/react";
import { ptBR } from "@daypicker/react/locale";
import "@daypicker/react/style.css";
import "./datepicker.css";

function parseDate(value: string): Date | undefined {
  const [year, month, day] = value.split("-").map(Number);
  if (!year || !month || !day) return undefined;
  const parsed = new Date(year, month - 1, day, 12);
  if (parsed.getFullYear() !== year || parsed.getMonth() !== month - 1 || parsed.getDate() !== day) {
    return undefined;
  }
  return parsed;
}

function toISO(value: Date): string {
  const pad = (part: number) => String(part).padStart(2, "0");
  return `${value.getFullYear()}-${pad(value.getMonth() + 1)}-${pad(value.getDate())}`;
}

function displayDate(value?: Date): string {
  return value?.toLocaleDateString("pt-BR") ?? "dd/mm/aaaa";
}

export function DatePicker({
  value,
  onChange,
  ariaLabel,
  required = false,
}: {
  value: string;
  onChange: (value: string) => void;
  ariaLabel: string;
  required?: boolean;
}) {
  const selected = useMemo(() => parseDate(value), [value]);
  const [open, setOpen] = useState(false);
  const [month, setMonth] = useState(selected ?? new Date());
  const [position, setPosition] = useState({ left: 0, top: 0 });
  const rootRef = useRef<HTMLDivElement>(null);
  const popoverRef = useRef<HTMLDivElement>(null);
  const dialogId = useId();

  useEffect(() => {
    if (selected) setMonth(selected);
  }, [selected]);

  useEffect(() => {
    if (!open) return;
    const closeOutside = (event: PointerEvent) => {
      const target = event.target as Node;
      if (!rootRef.current?.contains(target) && !popoverRef.current?.contains(target)) setOpen(false);
    };
    document.addEventListener("pointerdown", closeOutside);
    return () => document.removeEventListener("pointerdown", closeOutside);
  }, [open]);

  useLayoutEffect(() => {
    if (!open) return;
    const placePopover = () => {
      const trigger = rootRef.current?.getBoundingClientRect();
      const popover = popoverRef.current?.getBoundingClientRect();
      if (!trigger || !popover) return;
      const gutter = 14;
      const gap = 7;
      const maxLeft = window.innerWidth - popover.width - gutter;
      const below = trigger.bottom + gap;
      const above = trigger.top - gap - popover.height;
      const maxTop = window.innerHeight - popover.height - gutter;
      const top = below <= maxTop ? below : above >= gutter ? above : Math.max(gutter, maxTop);
      setPosition({
        left: Math.max(gutter, Math.min(trigger.left, maxLeft)),
        top,
      });
    };
    placePopover();
    window.addEventListener("resize", placePopover);
    window.addEventListener("scroll", placePopover, true);
    return () => {
      window.removeEventListener("resize", placePopover);
      window.removeEventListener("scroll", placePopover, true);
    };
  }, [open]);

  const choose = (date?: Date) => {
    onChange(date ? toISO(date) : "");
    setOpen(false);
  };

  return (
    <div className={`date-picker ${open ? "is-open" : ""}`} ref={rootRef}>
      <button
        type="button"
        className="date-picker-trigger"
        aria-label={ariaLabel}
        aria-haspopup="dialog"
        aria-expanded={open}
        aria-controls={open ? dialogId : undefined}
        aria-required={required}
        onClick={() => setOpen((current) => !current)}
        onKeyDown={(event) => {
          if (event.key === "Escape" && open) {
            event.preventDefault();
            event.stopPropagation();
            setOpen(false);
          }
        }}
      >
        <span className={selected ? "" : "date-picker-placeholder"}>{displayDate(selected)}</span>
        <span className="calendar-icon" aria-hidden />
      </button>

      {open && createPortal(
        <div
          className="date-picker-popover"
          id={dialogId}
          ref={popoverRef}
          role="dialog"
          aria-label={ariaLabel}
          style={position}
        >
          <DayPicker
            mode="single"
            locale={ptBR}
            selected={selected}
            month={month}
            onMonthChange={setMonth}
            onSelect={choose}
            required={required}
            showOutsideDays
            fixedWeeks
            autoFocus
          />
          <div className="date-picker-actions">
            {!required && value && (
              <button type="button" onClick={() => choose(undefined)}>Limpar</button>
            )}
            <button
              type="button"
              onClick={() => {
                const today = new Date();
                setMonth(today);
                choose(today);
              }}
            >
              Hoje
            </button>
          </div>
        </div>,
        document.body,
      )}
    </div>
  );
}
