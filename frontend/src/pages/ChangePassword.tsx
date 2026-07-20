import { useState, type FormEvent } from "react";
import { useNavigate } from "react-router-dom";
import { api } from "../lib/api";
import { useAuth } from "../lib/auth";
import { ErrorBanner } from "../components/ui";
import "./auth.css";

export function ChangePassword() {
  const { user, refresh, logout } = useAuth();
  const navigate = useNavigate();
  const forced = user?.must_change_password ?? false;

  const [atual, setAtual] = useState("");
  const [nova, setNova] = useState("");
  const [confirma, setConfirma] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function submit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    if (nova.length < 8) return setError("A nova senha deve ter ao menos 8 caracteres.");
    if (nova !== confirma) return setError("A confirmação não confere.");
    setBusy(true);
    try {
      await api.post("/auth/change-password", { senha_atual: atual, nova_senha: nova });
      await refresh();
      navigate("/", { replace: true });
    } catch (err) {
      setError(err instanceof Error ? err.message : "Falha ao trocar a senha");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="auth-form-wrap" style={{ minHeight: "100vh" }}>
      <form className="auth-form" onSubmit={submit}>
        <span className="kicker">Segurança</span>
        <h1>{forced ? "Defina sua senha" : "Trocar senha"}</h1>
        <p className="muted">
          {forced
            ? "Este é seu primeiro acesso. Crie uma nova senha para continuar."
            : "Confirme a senha atual e defina uma nova."}
        </p>

        {error && <ErrorBanner>{error}</ErrorBanner>}

        <div className="field">
          <label>Senha atual</label>
          <input type="password" autoComplete="current-password" value={atual} onChange={(e) => setAtual(e.target.value)} required />
        </div>
        <div className="field">
          <label>Nova senha</label>
          <input type="password" autoComplete="new-password" value={nova} onChange={(e) => setNova(e.target.value)} required />
        </div>
        <div className="field">
          <label>Confirmar nova senha</label>
          <input type="password" autoComplete="new-password" value={confirma} onChange={(e) => setConfirma(e.target.value)} required />
        </div>

        <button className="btn btn-primary" disabled={busy} style={{ marginTop: 6 }}>
          {busy ? "Salvando…" : "Salvar nova senha"}
        </button>
        <button
          type="button"
          className="btn btn-ghost btn-sm"
          onClick={() => {
            logout();
            navigate("/login");
          }}
        >
          Sair
        </button>
      </form>
    </div>
  );
}
