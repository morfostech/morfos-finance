CREATE TYPE vp_transaction_type AS ENUM ('venda', 'compra');
CREATE TYPE vp_transaction_status AS ENUM ('negociando', 'concluida', 'recusada', 'cancelada');
CREATE TYPE vp_offer_status AS ENUM ('aberta', 'liberada', 'pendente', 'bloqueada', 'encerrada');

CREATE TABLE vp_settings (
  id           SMALLINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
  credit_limit NUMERIC(14,2) NOT NULL DEFAULT 0 CHECK (credit_limit >= 0),
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
INSERT INTO vp_settings (id, credit_limit) VALUES (1, 0);
CREATE TRIGGER vp_settings_set_updated_at BEFORE UPDATE ON vp_settings
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE vp_transactions (
  id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  tipo          vp_transaction_type NOT NULL,
  status        vp_transaction_status NOT NULL DEFAULT 'concluida',
  valor         NUMERIC(14,2) NOT NULL CHECK (valor > 0),
  data          DATE NOT NULL,
  permutante    TEXT NOT NULL,
  oferta        TEXT NOT NULL,
  project_id    BIGINT REFERENCES projects(id),
  voucher_code  TEXT,
  observacoes   TEXT,
  created_by    BIGINT NOT NULL REFERENCES users(id),
  deleted_at    TIMESTAMPTZ,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TRIGGER vp_transactions_set_updated_at BEFORE UPDATE ON vp_transactions
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE INDEX idx_vp_transactions_active ON vp_transactions (data DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_vp_transactions_project ON vp_transactions (project_id) WHERE deleted_at IS NULL;

CREATE TABLE vp_offers (
  id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  titulo       TEXT NOT NULL,
  descricao    TEXT,
  valor        NUMERIC(14,2) CHECK (valor > 0),
  negociavel   BOOLEAN NOT NULL DEFAULT FALSE,
  status       vp_offer_status NOT NULL DEFAULT 'aberta',
  external_url TEXT,
  created_by   BIGINT NOT NULL REFERENCES users(id),
  deleted_at   TIMESTAMPTZ,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  CHECK (negociavel OR valor IS NOT NULL)
);
CREATE TRIGGER vp_offers_set_updated_at BEFORE UPDATE ON vp_offers
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE INDEX idx_vp_offers_active ON vp_offers (status) WHERE deleted_at IS NULL;
