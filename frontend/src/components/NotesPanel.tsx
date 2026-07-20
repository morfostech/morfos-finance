import { useState, type FormEvent } from "react";
import { api } from "../lib/api";
import { useAsync } from "../lib/hooks";
import { date } from "../lib/format";
import type { Note, NoteOwner } from "../lib/types";
import { useAuth } from "../lib/auth";
import { Empty, ErrorBanner, SectionHead, Spinner } from "./ui";
import "./notes.css";

/**
 * Private per-user notes attached to a project, transaction, installment, or
 * standing alone ("geral"). Every viewer only ever sees their own notes — the
 * backend scopes reads to the caller. Collaborator mutations become requests;
 * admin and partner mutations are applied directly.
 */
export function NotesPanel({
  ownerType,
  ownerId,
  idx,
  title = "Minhas anotações",
  bare = false,
}: {
  ownerType: NoteOwner;
  ownerId?: number;
  idx?: string;
  title?: string;
  /** Skip the card/panel wrapper — use when already inside a Modal. */
  bare?: boolean;
}) {
  const { user } = useAuth();
  const requiresApproval = user?.role === "colaborador";
  const query = ownerId != null ? `?owner_type=${ownerType}&owner_id=${ownerId}` : `?owner_type=${ownerType}`;
  const { data, loading, error, reload } = useAsync<Note[]>(() => api.get(`/notes${query}`), [query]);
  const [texto, setTexto] = useState("");
  const [busy, setBusy] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [editText, setEditText] = useState("");

  async function submit(e: FormEvent) {
    e.preventDefault();
    if (!texto.trim()) return;
    setFormError(null);
    setNotice(null);
    setBusy(true);
    try {
      if (requiresApproval) {
        await api.post("/change-requests", {
          action: "note_create",
          payload: { owner_type: ownerType, owner_id: ownerId ?? null, texto },
        });
        setNotice("Solicitação enviada para aprovação.");
      } else {
        await api.post("/notes", { owner_type: ownerType, owner_id: ownerId ?? null, texto });
        reload();
      }
      setTexto("");
    } catch (err) {
      setFormError(err instanceof Error ? err.message : "Falha ao salvar");
    } finally {
      setBusy(false);
    }
  }

  async function saveEdit(id: number) {
    if (!editText.trim()) return;
    setFormError(null);
    setNotice(null);
    try {
      if (requiresApproval) {
        await api.post("/change-requests", { action: "note_update", payload: { note_id: id, texto: editText } });
        setNotice("Alteração enviada para aprovação.");
      } else {
        await api.put(`/notes/${id}`, { texto: editText });
        reload();
      }
      setEditingId(null);
    } catch (err) {
      setFormError(err instanceof Error ? err.message : "Falha ao enviar alteração");
    }
  }

  async function remove(id: number) {
    setFormError(null);
    setNotice(null);
    try {
      if (requiresApproval) {
        await api.post("/change-requests", { action: "note_delete", payload: { note_id: id } });
        setNotice("Exclusão enviada para aprovação.");
      } else {
        await api.del(`/notes/${id}`);
        reload();
      }
    } catch (err) {
      setFormError(err instanceof Error ? err.message : "Falha ao enviar exclusão");
    }
  }

  return (
    <div className={bare ? "notes-panel" : "card panel notes-panel"}>
      {title && <SectionHead idx={idx} title={title} />}

      <form onSubmit={submit} className="notes-form">
        <textarea
          placeholder="Escreva uma anotação…"
          value={texto}
          onChange={(e) => setTexto(e.target.value)}
          rows={2}
        />
        <button className="btn btn-primary btn-sm" disabled={busy || !texto.trim()}>
          {busy ? "Enviando…" : requiresApproval ? "Solicitar anotação" : "Anotar"}
        </button>
      </form>
      {formError && <ErrorBanner>{formError}</ErrorBanner>}
      {notice && <div className="notes-notice">{notice}</div>}

      {loading ? (
        <Spinner />
      ) : error ? (
        <ErrorBanner>{error}</ErrorBanner>
      ) : !data || data.length === 0 ? (
        <Empty>Nenhuma anotação ainda.</Empty>
      ) : (
        <div className="notes-list">
          {data.map((n) => (
            <div key={n.id} className="note-item">
              {editingId === n.id ? (
                <div className="notes-form">
                  <textarea value={editText} onChange={(e) => setEditText(e.target.value)} rows={2} />
                  <div style={{ display: "flex", gap: 8 }}>
                    <button className="btn btn-ghost btn-sm" onClick={() => setEditingId(null)}>Cancelar</button>
                    <button className="btn btn-primary btn-sm" onClick={() => saveEdit(n.id)}>
                      {requiresApproval ? "Solicitar" : "Salvar"}
                    </button>
                  </div>
                </div>
              ) : (
                <>
                  <p className="note-text">{n.texto}</p>
                  <div className="note-meta">
                    <span className="mono muted">{date(n.updated_at)}</span>
                    <div className="note-actions">
                      <button
                        className="btn btn-ghost btn-sm"
                        onClick={() => {
                          setEditingId(n.id);
                          setEditText(n.texto);
                        }}
                      >
                        {requiresApproval ? "Solicitar edição" : "Editar"}
                      </button>
                      <button className="btn btn-danger btn-sm" onClick={() => remove(n.id)}>
                        {requiresApproval ? "Solicitar exclusão" : "Excluir"}
                      </button>
                    </div>
                  </div>
                </>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
