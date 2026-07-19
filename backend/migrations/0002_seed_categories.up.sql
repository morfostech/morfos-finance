INSERT INTO expense_categories (nome) VALUES
  ('Software'),
  ('Salários'),
  ('Aluguel'),
  ('Impostos'),
  ('Marketing'),
  ('Infraestrutura'),
  ('Outros')
ON CONFLICT (nome) DO NOTHING;
