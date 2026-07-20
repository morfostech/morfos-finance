import { NotesPanel } from "../components/NotesPanel";

export function Notes() {
  return (
    <div>
      <header className="page-head">
        <span className="kicker">Anotações</span>
        <h1>Minhas anotações</h1>
        <p>
          Para anotar sobre um projeto ou transação específica, use o botão de notas dentro
          da própria tela. Solicitações de colaboradores passam pela aprovação de admin ou sócio.
        </p>
      </header>
      <NotesPanel ownerType="geral" title="Anotações gerais" />
    </div>
  );
}
