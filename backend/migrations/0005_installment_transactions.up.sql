ALTER TABLE transactions
  ADD COLUMN installment_id BIGINT REFERENCES project_installments(id) ON DELETE CASCADE;

CREATE UNIQUE INDEX transactions_installment_id_unique
  ON transactions (installment_id)
  WHERE installment_id IS NOT NULL;

-- Reuse an equivalent existing implementation gain when one can be matched
-- one-to-one with a paid installment.
WITH paid AS (
  SELECT id, project_id, valor, pago_em,
         ROW_NUMBER() OVER (PARTITION BY project_id, valor, pago_em ORDER BY id) AS position
  FROM project_installments
  WHERE pago_em IS NOT NULL
), gains AS (
  SELECT id, project_id, valor, data,
         ROW_NUMBER() OVER (PARTITION BY project_id, valor, data ORDER BY id) AS position
  FROM transactions
  WHERE deleted_at IS NULL
    AND tipo = 'ganho'
    AND origem = 'implementacao'
    AND installment_id IS NULL
)
UPDATE transactions t
SET installment_id = paid.id
FROM paid
JOIN gains
  ON gains.project_id = paid.project_id
 AND gains.valor = paid.valor
 AND gains.data = paid.pago_em
 AND gains.position = paid.position
WHERE t.id = gains.id;

-- Backfill any paid installment that still has no financial transaction.
WITH actor AS (
  SELECT id
  FROM users
  ORDER BY CASE role WHEN 'admin' THEN 0 WHEN 'socio' THEN 1 ELSE 2 END, id
  LIMIT 1
)
INSERT INTO transactions (
  tipo, valor, data, project_id, origem, descricao, created_by, installment_id
)
SELECT
  'ganho', i.valor, i.pago_em, i.project_id, 'implementacao',
  'Parcela de implementação (' ||
    CASE i.tipo WHEN 'entrada' THEN 'entrada' ELSE 'finalização' END || ')',
  actor.id, i.id
FROM project_installments i
CROSS JOIN actor
WHERE i.pago_em IS NOT NULL
  AND NOT EXISTS (
    SELECT 1 FROM transactions t WHERE t.installment_id = i.id
  );
