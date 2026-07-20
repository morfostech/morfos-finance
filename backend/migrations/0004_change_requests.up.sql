CREATE TYPE change_request_action AS ENUM ('note_create', 'note_update', 'note_delete');
CREATE TYPE change_request_status AS ENUM ('pending', 'processing', 'approved', 'rejected');

CREATE TABLE change_requests (
  id             BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  requester_id   BIGINT NOT NULL REFERENCES users(id),
  action         change_request_action NOT NULL,
  payload        JSONB NOT NULL,
  status         change_request_status NOT NULL DEFAULT 'pending',
  reviewer_id    BIGINT REFERENCES users(id),
  review_comment TEXT,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  reviewed_at    TIMESTAMPTZ
);

CREATE INDEX idx_change_requests_queue ON change_requests (status, created_at DESC);
CREATE INDEX idx_change_requests_requester ON change_requests (requester_id, created_at DESC);
