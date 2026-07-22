import { NavLink, Outlet, useNavigate } from "react-router-dom";
import { useAuth } from "../lib/auth";
import { ROLE_LABEL } from "../lib/format";
import "./shell.css";

const MARK = (
  <svg viewBox="0 0 100 100" className="mk" aria-hidden>
    <defs>
      <linearGradient id="mg" x1="8" y1="16" x2="92" y2="86" gradientUnits="userSpaceOnUse">
        <stop offset="0" stopColor="#16B0A0" />
        <stop offset="0.42" stopColor="#0C7C75" />
        <stop offset="1" stopColor="#D08B57" />
      </linearGradient>
    </defs>
    <path d="M15 82 V27 L50 48 L85 27 V82" fill="none" stroke="url(#mg)" strokeWidth="15" strokeLinejoin="round" strokeLinecap="round" />
  </svg>
);

interface NavItem {
  to: string;
  label: string;
  idx: string;
  roles?: string[];
}

const NAV: NavItem[] = [
  { to: "/", label: "Dashboard", idx: "01" },
  { to: "/projetos", label: "Projetos", idx: "02" },
  { to: "/transacoes", label: "Transações", idx: "03" },
  { to: "/planejamento", label: "Planejamento", idx: "04", roles: ["admin", "socio"] },
  { to: "/anotacoes", label: "Anotações", idx: "05" },
  { to: "/solicitacoes", label: "Solicitações", idx: "06" },
  { to: "/recorrencia", label: "Recorrência", idx: "07", roles: ["admin", "socio"] },
  { to: "/usuarios", label: "Usuários", idx: "08", roles: ["admin", "socio"] },
];

export function AppShell() {
  const { user, logout } = useAuth();
  const navigate = useNavigate();
  if (!user) return null;

  const items = NAV.filter((n) => !n.roles || n.roles.includes(user.role));

  return (
    <div className="shell">
      <aside className="sidebar">
        <div className="brand">
          {MARK}
          <div className="wm">
            MORFOS
            <small>FINANCE</small>
          </div>
        </div>

        <nav className="side-nav">
          {items.map((n) => (
            <NavLink key={n.to} to={n.to} end={n.to === "/"} className="side-link">
              <span className="side-idx mono">{n.idx}</span>
              <span>{n.label}</span>
            </NavLink>
          ))}
        </nav>

        <div className="side-foot">
          <div className="side-user">
            <div className="avatar">{user.nome.charAt(0).toUpperCase()}</div>
            <div className="side-user-body">
              <div className="side-user-name">{user.nome}</div>
              <div className="side-user-role mono">{ROLE_LABEL[user.role]}</div>
            </div>
          </div>
          <button
            className="btn btn-ghost btn-sm side-logout"
            onClick={() => {
              logout();
              navigate("/login");
            }}
          >
            Sair
          </button>
        </div>
      </aside>

      <main className="content">
        <Outlet />
      </main>
    </div>
  );
}
