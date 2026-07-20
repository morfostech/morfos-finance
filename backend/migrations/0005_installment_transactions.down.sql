DELETE FROM transactions
WHERE installment_id IS NOT NULL
  AND descricao IN (
    'Parcela de implementação (entrada)',
    'Parcela de implementação (finalização)'
  );

DROP INDEX IF EXISTS transactions_installment_id_unique;
ALTER TABLE transactions DROP COLUMN IF EXISTS installment_id;
