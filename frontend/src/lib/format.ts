// Formatting helpers. Money is always integer centavos on the wire.

const brl = new Intl.NumberFormat("pt-BR", {
  style: "currency",
  currency: "BRL",
});

/** Format centavos as BRL, e.g. 500000 -> "R$ 5.000,00". */
export function money(centavos: number): string {
  return brl.format(centavos / 100);
}

/** Parse a "R$ 5.000,00" / "5000,00" / "5000.00" input into centavos. */
export function toCentavos(input: string): number | null {
  const cleaned = input.replace(/[^\d,.-]/g, "").replace(/\.(?=\d{3})/g, "");
  const normalized = cleaned.replace(",", ".");
  const value = Number(normalized);
  if (Number.isNaN(value)) return null;
  return Math.round(value * 100);
}

/** Format an ISO/date string as dd/mm/yyyy. */
export function date(iso?: string): string {
  if (!iso) return "—";
  const d = new Date(iso.length <= 10 ? iso + "T00:00:00" : iso);
  return d.toLocaleDateString("pt-BR");
}

export function todayISO(): string {
  return iso(new Date());
}

/** First and last day of the current month as ISO strings. */
export function currentMonthRange(): { from: string; to: string } {
  const now = new Date();
  const from = new Date(now.getFullYear(), now.getMonth(), 1);
  const to = new Date(now.getFullYear(), now.getMonth() + 1, 0);
  return { from: iso(from), to: iso(to) };
}

function iso(d: Date): string {
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
}
function pad(n: number): string {
  return String(n).padStart(2, "0");
}

const MESES = [
  "JAN", "FEV", "MAR", "ABR", "MAI", "JUN",
  "JUL", "AGO", "SET", "OUT", "NOV", "DEZ",
];
export function monthLabel(mes: number): string {
  return MESES[mes - 1] ?? String(mes);
}

export const ROLE_LABEL: Record<string, string> = {
  admin: "Admin",
  socio: "Sócio",
  colaborador: "Colaborador",
};
