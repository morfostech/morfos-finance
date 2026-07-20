CREATE EXTENSION IF NOT EXISTS citext;

-- generic updated_at trigger
CREATE OR REPLACE FUNCTION set_updated_at() RETURNS trigger AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TYPE user_role        AS ENUM ('admin','socio','colaborador');
CREATE TYPE project_status   AS ENUM ('ativo','pausado','concluido','cancelado');
CREATE TYPE installment_type AS ENUM ('entrada','finalizacao');
CREATE TYPE tx_type          AS ENUM ('ganho','despesa');
CREATE TYPE tx_origem        AS ENUM ('implementacao','recorrencia','avulso');
CREATE TYPE proposal_type    AS ENUM ('pdf','docx');
CREATE TYPE attachment_owner AS ENUM ('transaction','installment');

CREATE TABLE users (
  id                   BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  nome                 TEXT NOT NULL,
  email                CITEXT UNIQUE NOT NULL,
  senha_hash           TEXT NOT NULL,
  role                 user_role NOT NULL DEFAULT 'colaborador',
  must_change_password BOOLEAN NOT NULL DEFAULT TRUE,
  ativo                BOOLEAN NOT NULL DEFAULT TRUE,
  created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TRIGGER users_set_updated_at BEFORE UPDATE ON users
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE projects (
  id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  nome                TEXT NOT NULL,
  cliente             TEXT,
  valor_implementacao NUMERIC(14,2) CHECK (valor_implementacao >= 0),
  valor_mensal        NUMERIC(14,2) CHECK (valor_mensal >= 0),
  dia_vencimento      SMALLINT CHECK (dia_vencimento BETWEEN 1 AND 31),
  data_inicio         DATE,
  data_fim            DATE,
  status              project_status NOT NULL DEFAULT 'ativo',
  created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
  CHECK (valor_implementacao IS NOT NULL OR valor_mensal IS NOT NULL),
  CHECK (data_fim IS NULL OR data_inicio IS NULL OR data_fim >= data_inicio)
);
CREATE TRIGGER projects_set_updated_at BEFORE UPDATE ON projects
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE project_installments (
  id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  tipo       installment_type NOT NULL,
  valor      NUMERIC(14,2) NOT NULL CHECK (valor >= 0),
  pago_em    DATE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (project_id, tipo)
);
CREATE TRIGGER installments_set_updated_at BEFORE UPDATE ON project_installments
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE project_members (
  project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  user_id    BIGINT NOT NULL REFERENCES users(id)    ON DELETE CASCADE,
  PRIMARY KEY (project_id, user_id)
);

CREATE TABLE project_proposals (
  id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  project_id   BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  url          TEXT NOT NULL,
  arquivo_tipo proposal_type NOT NULL,
  descricao    TEXT,
  created_by   BIGINT REFERENCES users(id),
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE expense_categories (
  id   BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  nome TEXT UNIQUE NOT NULL
);

CREATE TABLE transactions (
  id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  tipo        tx_type NOT NULL,
  valor       NUMERIC(14,2) NOT NULL CHECK (valor > 0),
  data        DATE NOT NULL,
  project_id  BIGINT REFERENCES projects(id),
  user_id     BIGINT REFERENCES users(id),
  origem      tx_origem,
  category_id BIGINT REFERENCES expense_categories(id),
  descricao   TEXT,
  created_by  BIGINT NOT NULL REFERENCES users(id),
  deleted_at  TIMESTAMPTZ,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  CHECK (tipo <> 'despesa' OR origem      IS NULL),
  CHECK (tipo <> 'ganho'   OR category_id IS NULL)
);
CREATE TRIGGER transactions_set_updated_at BEFORE UPDATE ON transactions
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE INDEX idx_tx_active  ON transactions (data)       WHERE deleted_at IS NULL;
CREATE INDEX idx_tx_project ON transactions (project_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_tx_user    ON transactions (user_id)    WHERE deleted_at IS NULL;

CREATE TABLE attachments (
  id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  owner_type attachment_owner NOT NULL,
  owner_id   BIGINT NOT NULL,
  url        TEXT NOT NULL,
  descricao  TEXT,
  created_by BIGINT REFERENCES users(id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_attach_owner ON attachments (owner_type, owner_id);
