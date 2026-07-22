CREATE TABLE planned_entries (
  id                    BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  tipo                  tx_type NOT NULL,
  valor                 NUMERIC(14,2) NOT NULL CHECK (valor > 0),
  due_date               DATE NOT NULL,
  project_id             BIGINT REFERENCES projects(id),
  user_id                BIGINT REFERENCES users(id),
  origem                 tx_origem,
  category_id            BIGINT REFERENCES expense_categories(id),
  descricao              TEXT NOT NULL,
  actual_transaction_id  BIGINT UNIQUE REFERENCES transactions(id),
  created_by             BIGINT NOT NULL REFERENCES users(id),
  deleted_at             TIMESTAMPTZ,
  created_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
  CHECK (tipo <> 'despesa' OR origem IS NULL),
  CHECK (tipo <> 'ganho' OR category_id IS NULL)
);
CREATE TRIGGER planned_entries_set_updated_at BEFORE UPDATE ON planned_entries
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE INDEX idx_planned_open_due ON planned_entries (due_date)
  WHERE deleted_at IS NULL AND actual_transaction_id IS NULL;

CREATE TABLE expense_budgets (
  id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  category_id BIGINT NOT NULL REFERENCES expense_categories(id),
  ano         INTEGER NOT NULL CHECK (ano BETWEEN 2000 AND 2200),
  mes         SMALLINT NOT NULL CHECK (mes BETWEEN 1 AND 12),
  valor       NUMERIC(14,2) NOT NULL CHECK (valor > 0),
  created_by  BIGINT NOT NULL REFERENCES users(id),
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (category_id, ano, mes)
);
CREATE TRIGGER expense_budgets_set_updated_at BEFORE UPDATE ON expense_budgets
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();
