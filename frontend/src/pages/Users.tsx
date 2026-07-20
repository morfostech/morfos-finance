import { useState, type FormEvent } from "react";
import { api } from "../lib/api";
import { useAsync } from "../lib/hooks";
import { ROLE_LABEL } from "../lib/format";
import type { Role, User } from "../lib/types";
import { Empty, ErrorBanner, SectionHead, Select, Spinner } from "../components/ui";
import { Modal } from "../components/Modal";
import "./pages.css";

export function Users() {
  const { data, loading, error, reload } = useAsync<User[]>(() => api.get("/users"), []);
  const [creating, setCreating] = useState(false);
  const [resetting, setResetting] = useState<User | null>(null);

  return (
    <div>
      <header className="page-head">
        <span className="kicker">05 / Usuários</span>
        <h1>Usuários</h1>
        <p>Cadastre pessoas e defina o cargo. A senha inicial exige troca no primeiro acesso.</p>
      </header>

      <SectionHead
        title="Equipe"
        action={<button className="btn btn-primary btn-sm" onClick={() => setCreating(true)}>+ Novo usuário</button>}
      />

      {loading ? (
        <Spinner />
      ) : error ? (
        <ErrorBanner>{error}</ErrorBanner>
      ) : !data || data.length === 0 ? (
        <Empty>Nenhum usuário.</Empty>
      ) : (
        <div className="card table-wrap">
          <table>
            <thead>
              <tr>
                <th>Nome</th>
                <th>E-mail</th>
                <th>Cargo</th>
                <th>Status</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {data.map((u) => (
                <tr key={u.id}>
                  <td style={{ fontWeight: 600 }}>{u.nome}</td>
                  <td className="muted">{u.email}</td>
                  <td><span className="pill pill-neutral">{ROLE_LABEL[u.role]}</span></td>
                  <td>
                    {u.must_change_password ? (
                      <span className="pill pill-pending">1º acesso</span>
                    ) : (
                      <span className="pill pill-ok">ativo</span>
                    )}
                  </td>
                  <td style={{ textAlign: "right" }}>
                    <button className="btn btn-ghost btn-sm" onClick={() => setResetting(u)}>Resetar senha</button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {creating && (
        <NewUserModal onClose={() => setCreating(false)} onCreated={() => { setCreating(false); reload(); }} />
      )}
      {resetting && (
        <ResetModal user={resetting} onClose={() => setResetting(null)} onDone={() => { setResetting(null); reload(); }} />
      )}
    </div>
  );
}

function NewUserModal({ onClose, onCreated }: { onClose: () => void; onCreated: () => void }) {
  const [nome, setNome] = useState("");
  const [email, setEmail] = useState("");
  const [senha, setSenha] = useState("");
  const [role, setRole] = useState<Role>("colaborador");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function submit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    if (senha.length < 8) return setError("Senha inicial deve ter ao menos 8 caracteres.");
    setBusy(true);
    try {
      await api.post("/users", { nome, email, senha_inicial: senha, role });
      onCreated();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Falha ao criar");
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal title="Novo usuário" onClose={onClose}>
      <form onSubmit={submit} style={{ display: "flex", flexDirection: "column", gap: 16 }}>
        {error && <ErrorBanner>{error}</ErrorBanner>}
        <div className="field">
          <label>Nome *</label>
          <input value={nome} onChange={(e) => setNome(e.target.value)} required />
        </div>
        <div className="field">
          <label>E-mail *</label>
          <input type="email" value={email} onChange={(e) => setEmail(e.target.value)} required />
        </div>
        <div className="form-row">
          <div className="field">
            <label>Senha inicial *</label>
            <input type="text" value={senha} onChange={(e) => setSenha(e.target.value)} required />
          </div>
          <div className="field">
            <label>Cargo *</label>
            <Select
              ariaLabel="Cargo do usuário"
              value={role}
              onChange={(value) => setRole(value as Role)}
              options={[
                { value: "colaborador", label: "Colaborador" },
                { value: "socio", label: "Sócio" },
                { value: "admin", label: "Admin" },
              ]}
            />
          </div>
        </div>
        <div className="modal-actions">
          <button type="button" className="btn btn-ghost btn-sm" onClick={onClose}>Cancelar</button>
          <button className="btn btn-primary btn-sm" disabled={busy}>{busy ? "Criando…" : "Criar"}</button>
        </div>
      </form>
    </Modal>
  );
}

function ResetModal({ user, onClose, onDone }: { user: User; onClose: () => void; onDone: () => void }) {
  const [senha, setSenha] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function submit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    if (senha.length < 8) return setError("A nova senha deve ter ao menos 8 caracteres.");
    setBusy(true);
    try {
      await api.post(`/users/${user.id}/reset-password`, { nova_senha: senha });
      onDone();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Falha ao resetar");
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal title={`Resetar senha · ${user.nome}`} onClose={onClose}>
      <form onSubmit={submit} style={{ display: "flex", flexDirection: "column", gap: 16 }}>
        {error && <ErrorBanner>{error}</ErrorBanner>}
        <p className="muted" style={{ fontSize: 13 }}>
          O usuário será obrigado a trocar a senha no próximo acesso.
        </p>
        <div className="field">
          <label>Nova senha inicial *</label>
          <input type="text" value={senha} onChange={(e) => setSenha(e.target.value)} required />
        </div>
        <div className="modal-actions">
          <button type="button" className="btn btn-ghost btn-sm" onClick={onClose}>Cancelar</button>
          <button className="btn btn-primary btn-sm" disabled={busy}>{busy ? "Salvando…" : "Resetar"}</button>
        </div>
      </form>
    </Modal>
  );
}
