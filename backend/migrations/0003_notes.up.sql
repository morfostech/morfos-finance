CREATE TYPE note_owner AS ENUM ('project', 'transaction', 'installment', 'geral');

-- User-scoped annotations. Collaborator mutations are applied through the
-- reviewed change-request workflow. owner_id is NULL for general notes.
CREATE TABLE notes (
  id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  owner_type note_owner NOT NULL,
  owner_id   BIGINT,
  texto      TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CHECK (owner_type = 'geral' OR owner_id IS NOT NULL)
);
CREATE TRIGGER notes_set_updated_at BEFORE UPDATE ON notes
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE INDEX idx_notes_user_owner ON notes (user_id, owner_type, owner_id);
