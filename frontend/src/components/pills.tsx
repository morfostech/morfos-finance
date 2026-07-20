import type { ProjectStatus } from "../lib/types";

const STATUS: Record<ProjectStatus, { label: string; cls: string }> = {
  ativo: { label: "Ativo", cls: "pill-ok" },
  pausado: { label: "Pausado", cls: "pill-pending" },
  concluido: { label: "Concluído", cls: "pill-neutral" },
  cancelado: { label: "Cancelado", cls: "pill-danger" },
};

export function StatusPill({ status }: { status: ProjectStatus }) {
  const s = STATUS[status] ?? { label: status, cls: "pill-neutral" };
  return <span className={`pill ${s.cls}`}>{s.label}</span>;
}

export function PaidPill({ pago }: { pago: boolean }) {
  return <span className={`pill ${pago ? "pill-ok" : "pill-pending"}`}>{pago ? "Pago" : "Pendente"}</span>;
}
