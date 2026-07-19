DROP TABLE IF EXISTS attachments;
DROP TABLE IF EXISTS transactions;
DROP TABLE IF EXISTS expense_categories;
DROP TABLE IF EXISTS project_proposals;
DROP TABLE IF EXISTS project_members;
DROP TABLE IF EXISTS project_installments;
DROP TABLE IF EXISTS projects;
DROP TABLE IF EXISTS users;

DROP TYPE IF EXISTS attachment_owner;
DROP TYPE IF EXISTS proposal_type;
DROP TYPE IF EXISTS tx_origem;
DROP TYPE IF EXISTS tx_type;
DROP TYPE IF EXISTS installment_type;
DROP TYPE IF EXISTS project_status;
DROP TYPE IF EXISTS user_role;

DROP FUNCTION IF EXISTS set_updated_at;
