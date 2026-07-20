import { Navigate, Route, Routes } from "react-router-dom";
import { useAuth } from "./lib/auth";
import { AppShell } from "./components/AppShell";
import { Spinner } from "./components/ui";
import { Login } from "./pages/Login";
import { ChangePassword } from "./pages/ChangePassword";
import { Dashboard } from "./pages/Dashboard";
import { Projects } from "./pages/Projects";
import { ProjectDetail } from "./pages/ProjectDetail";
import { Transactions } from "./pages/Transactions";
import { Recurrence } from "./pages/Recurrence";
import { Users } from "./pages/Users";
import { Notes } from "./pages/Notes";
import { ChangeRequests } from "./pages/ChangeRequests";
import type { Role } from "./lib/types";

export function App() {
  const { user, loading } = useAuth();

  if (loading) {
    return (
      <div style={{ minHeight: "100vh", display: "grid", placeItems: "center" }}>
        <Spinner label="Iniciando…" />
      </div>
    );
  }

  return (
    <Routes>
      <Route path="/login" element={user ? <Navigate to="/" replace /> : <Login />} />

      <Route element={<Protected />}>
        <Route path="/trocar-senha" element={<ChangePassword />} />
      </Route>

      <Route element={<Protected requireStablePassword />}>
        <Route element={<AppShell />}>
          <Route path="/" element={<Dashboard />} />
          <Route path="/projetos" element={<Projects />} />
          <Route path="/projetos/:id" element={<ProjectDetail />} />
          <Route path="/transacoes" element={<Transactions />} />
          <Route path="/anotacoes" element={<Notes />} />
          <Route path="/solicitacoes" element={<ChangeRequests />} />
          <Route element={<RoleGate roles={["admin", "socio"]} />}>
            <Route path="/recorrencia" element={<Recurrence />} />
          </Route>
          <Route element={<RoleGate roles={["admin", "socio"]} />}>
            <Route path="/usuarios" element={<Users />} />
          </Route>
        </Route>
      </Route>

      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}

import { Outlet } from "react-router-dom";

function Protected({ requireStablePassword }: { requireStablePassword?: boolean }) {
  const { user } = useAuth();
  if (!user) return <Navigate to="/login" replace />;
  if (requireStablePassword && user.must_change_password) {
    return <Navigate to="/trocar-senha" replace />;
  }
  return <Outlet />;
}

function RoleGate({ roles }: { roles: Role[] }) {
  const { user } = useAuth();
  if (!user || !roles.includes(user.role)) return <Navigate to="/" replace />;
  return <Outlet />;
}
