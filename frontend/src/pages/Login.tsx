import { useState, type FormEvent } from "react";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../lib/auth";
import { ErrorBanner } from "../components/ui";
import "./auth.css";

export function Login() {
  const { login } = useAuth();
  const navigate = useNavigate();
  const [email, setEmail] = useState("");
  const [senha, setSenha] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function submit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setBusy(true);
    try {
      const user = await login(email, senha);
      navigate(user.must_change_password ? "/trocar-senha" : "/", { replace: true });
    } catch (err) {
      setError(err instanceof Error ? err.message : "Falha no login");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="auth-page">
      <div className="auth-aside">
        <div className="auth-brand">
          <svg viewBox="0 0 100 100" className="auth-mk" aria-hidden>
            <defs>
              <linearGradient id="ag" x1="8" y1="16" x2="92" y2="86" gradientUnits="userSpaceOnUse">
                <stop offset="0" stopColor="#16B0A0" />
                <stop offset="0.42" stopColor="#0C7C75" />
                <stop offset="1" stopColor="#D08B57" />
              </linearGradient>
            </defs>
            <path d="M15 82 V27 L50 48 L85 27 V82" fill="none" stroke="url(#ag)" strokeWidth="15" strokeLinejoin="round" strokeLinecap="round" />
          </svg>
          <div className="wm">
            MORFOS <small>FINANCE</small>
          </div>
        </div>
        <p className="auth-tag">
          Controle financeiro interno — <span className="grad-text">projetos, recorrência e caixa</span> num só lugar.
        </p>
        <div className="auth-hud mono">
          <i /> SISTEMA INTERNO · MORFOS TECH
        </div>
      </div>

      <div className="auth-form-wrap">
        <form className="auth-form" onSubmit={submit}>
          <span className="kicker">01 / Acesso</span>
          <h1>Entrar</h1>
          <p className="muted">Use o e-mail e a senha da sua conta.</p>

          {error && <ErrorBanner>{error}</ErrorBanner>}

          <div className="field">
            <label htmlFor="email">E-mail</label>
            <input id="email" type="email" autoComplete="username" value={email} onChange={(e) => setEmail(e.target.value)} required />
          </div>
          <div className="field">
            <label htmlFor="senha">Senha</label>
            <input id="senha" type="password" autoComplete="current-password" value={senha} onChange={(e) => setSenha(e.target.value)} required />
          </div>

          <button className="btn btn-primary" disabled={busy} style={{ marginTop: 6 }}>
            {busy ? "Entrando…" : "Entrar"}
          </button>
        </form>
      </div>
    </div>
  );
}
