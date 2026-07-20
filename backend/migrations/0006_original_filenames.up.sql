ALTER TABLE attachments
  ADD COLUMN nome_arquivo TEXT;

ALTER TABLE project_proposals
  ADD COLUMN nome_arquivo TEXT;

-- Older rows keep NULL because the original multipart filename was not stored
-- anywhere and cannot be reconstructed from the randomized object key.
