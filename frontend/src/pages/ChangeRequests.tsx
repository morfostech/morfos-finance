import { useState } from "react";
import { useAuth } from "../lib/auth";
import { api } from "../lib/api";
import { date } from "../lib/format";
import { useAsync } from "../lib/hooks";
import type { ChangeRequest, ChangeRequestAction, ChangeRequestStatus } from "../lib/types";
import { Empty, ErrorBanner, Spinner } from "../components/ui";
import "./change-requests.css";

const ACTION_LABEL: Record<ChangeRequestAction, string> = {
  note_create: "Criar anotação",
  note_update: "Editar anotação",
  note_delete: "Excluir anotação",
};

const STATUS_LABEL: Record<ChangeRequestStatus, string> = {
  pending: "Pendente",
  approved: "Aprovada",
  rejected: "Rejeitada",
};

const STATUS_CLASS: Record<ChangeRequestStatus, string> = {
  pending: "pill-pending",
  approved: "pill-ok",
  rejected: "pill-danger",
};

export function ChangeRequests() {
  const { user } = useAuth();
  const canReview = user?.role === "admin" || user?.role === "socio";
  const { data, loading, error, reload } = useAsync<ChangeRequest[]>(() => api.get("/change-requests"), []);
  const [busyID, setBusyID] = useState<number | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);
  const [comments, setComments] = useState<Record<number, string>>({});

  async function review(id: number, approved: boolean) {
    const comment = comments[id] ?? "";
    if (!approved && !comment.trim()) return;
    setBusyID(id);
    setActionError(null);
    try {
      await api.post(`/change-requests/${id}/${approved ? "approve" : "reject"}`, { comment });
      reload();
    } catch (err) {
      setActionError(err instanceof Error ? err.message : "Falha ao revisar solicitação");
    } finally {
      setBusyID(null);
    }
  }

  return (
    <div>
      <header className="page-head">
        <span className="kicker">Controle de alterações</span>
        <h1>{canReview ? "Solicitações" : "Minhas solicitações"}</h1>
        <p>{canReview ? "Revise mudanças enviadas pelos colaboradores." : "Acompanhe as mudanças enviadas para admin e sócios."}</p>
      </header>

      {actionError && <ErrorBanner>{actionError}</ErrorBanner>}
      {loading ? <Spinner /> : error ? <ErrorBanner>{error}</ErrorBanner> : !data?.length ? (
        <Empty>Nenhuma solicitação encontrada.</Empty>
      ) : (
        <div className="request-list">
          {data.map((request) => (
            <article className="card request-item" key={request.id}>
              <div className="request-head">
                <div>
                  <span className="kicker">{ACTION_LABEL[request.action]}</span>
                  <h2>{canReview ? request.requester_name : ACTION_LABEL[request.action]}</h2>
                </div>
                <span className={`pill ${STATUS_CLASS[request.status]}`}>{STATUS_LABEL[request.status]}</span>
              </div>

              {request.payload.texto && <p className="request-text">{request.payload.texto}</p>}
              <div className="request-meta mono">
                <span>#{request.id}</span>
                <span>{date(request.created_at)}</span>
                {request.reviewer_name && <span>Revisada por {request.reviewer_name}</span>}
              </div>
              {request.review_comment && <p className="request-comment">{request.review_comment}</p>}

              {canReview && request.status === "pending" && (
                <div className="request-review">
                  <label htmlFor={`comment-${request.id}`}>Comentário da revisão</label>
                  <textarea
                    id={`comment-${request.id}`}
                    rows={2}
                    placeholder="Obrigatório para rejeitar"
                    value={comments[request.id] ?? ""}
                    onChange={(event) => setComments((current) => ({ ...current, [request.id]: event.target.value }))}
                  />
                  <div className="request-actions">
                    <button className="btn btn-danger btn-sm" disabled={busyID === request.id || !(comments[request.id] ?? "").trim()} onClick={() => review(request.id, false)}>Rejeitar</button>
                    <button className="btn btn-primary btn-sm" disabled={busyID === request.id} onClick={() => review(request.id, true)}>Aprovar</button>
                  </div>
                </div>
              )}
            </article>
          ))}
        </div>
      )}
    </div>
  );
}
