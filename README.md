# Morfos Finance

Sistema interno de controle financeiro da Morfos Tech. Backend em Go (Chi + PostgreSQL), frontend em React + TypeScript, identidade visual alinhada ao site da Morfos.

> Em construção por módulos: **auth ✅ → projetos ✅ → transações ✅ → recorrência ✅ → anexos → dashboards → tema/UI**.

> **Valores monetários** trafegam na API em **centavos** (inteiro), nunca float. Ex.: `500000` = R$ 5.000,00.

## Stack

- **Backend:** Go 1.25, Chi router, PostgreSQL (pgx), JWT (HS256), senhas com argon2id.
- **Frontend:** React 18 + TypeScript + Vite, CSS Modules com os tokens da Morfos _(próximos módulos)_.
- **Storage de anexos:** S3-compatible via `.env` (padrão Cloudflare R2) _(módulo de anexos)_.
- **Arquitetura:** camadas `handlers → services → repositories`, migrations versionadas embutidas no binário, segredos via `.env`.

## Rodar localmente

Pré-requisitos: Go 1.25+ e Docker.

```bash
# 1. Subir o Postgres
docker compose up -d

# 2. Configurar o backend
cd backend
cp .env.example .env          # ajuste JWT_SECRET e as credenciais do admin

# 3. Criar o admin inicial (roda as migrations + seed de categorias antes)
go run ./cmd/seed

# 4. Subir a API
go run ./cmd/api              # http://localhost:8080
```

A API aplica as migrations pendentes automaticamente ao subir. `go run ./cmd/seed` é
idempotente — se o admin já existe, não faz nada.

### Variáveis de ambiente

Ver [`backend/.env.example`](backend/.env.example). Essenciais: `DATABASE_URL`, `JWT_SECRET`.
Para produção, troque `JWT_SECRET` por um valor longo e aleatório e defina
`SEED_ADMIN_EMAIL` / `SEED_ADMIN_SENHA` antes de rodar o seed.

## Testes

```bash
cd backend
go test ./...
```

Cobrem hashing/verificação de senha, emissão/parse de JWT, regras de login
(senha errada, usuário inativo, e-mail case-insensitive), troca de senha e o
gating de permissões por cargo (admin/sócio/colaborador).

## API — módulo Auth

| Método | Rota                              | Auth        | Descrição                                   |
|--------|-----------------------------------|-------------|---------------------------------------------|
| GET    | `/health`                         | —           | Healthcheck                                 |
| POST   | `/api/auth/login`                 | —           | Login por e-mail/senha, retorna JWT         |
| GET    | `/api/auth/me`                    | Autenticado | Dados do usuário atual                      |
| POST   | `/api/auth/change-password`       | Autenticado | Troca a própria senha (cobre 1º login)      |
| GET    | `/api/users`                      | Admin       | Lista usuários                              |
| POST   | `/api/users`                      | Admin       | Cria usuário com senha inicial              |
| POST   | `/api/users/{id}/reset-password`  | Admin       | Reseta senha (força troca no próximo login) |

**Papéis:** `admin` (vê/edita tudo, gerencia usuários), `socio` (visão financeira
completa, somente leitura), `colaborador` (apenas a própria área). Usuários novos
nascem com `must_change_password = true`.

## API — módulo Projetos

| Método | Rota                                          | Auth        | Descrição                                                    |
|--------|-----------------------------------------------|-------------|-------------------------------------------------------------|
| GET    | `/api/projects`                               | Autenticado | Lista projetos (colaborador vê só os alocados)              |
| GET    | `/api/projects/{id}`                          | Autenticado | Projeto + parcelas + membros (colaborador só se alocado)   |
| POST   | `/api/projects`                               | Admin       | Cria projeto; gera parcelas 50/50 se houver implementação   |
| PUT    | `/api/projects/{id}`                          | Admin       | Atualiza campos; reconcilia parcelas de implementação       |
| PUT    | `/api/projects/{id}/members`                  | Admin       | Define a lista de colaboradores alocados                    |
| PATCH  | `/api/projects/{id}/installments/{iid}`       | Admin       | Marca parcela paga (`pago_em`) ou pendente (`null`)         |

**Fontes de receita:** um projeto tem `valor_implementacao` e/ou `valor_mensal`
(ao menos um). A implementação vira **duas parcelas** — `entrada` (50%, arredondada
para baixo) e `finalizacao` (o restante) — que sempre somam o valor total.

**Regra de reconciliação de parcelas** (no `PUT`): alterar/remover o valor de
implementação **com uma parcela já paga** retorna `409`; sem parcela paga, as
parcelas são regeradas. Mensalidade sozinha não gera parcelas.

## API — módulo Transações & Categorias

| Método | Rota                        | Auth        | Descrição                                              |
|--------|-----------------------------|-------------|--------------------------------------------------------|
| GET    | `/api/transactions`         | Autenticado | Lista com filtros (colaborador vê só as próprias)      |
| GET    | `/api/transactions/{id}`    | Autenticado | Uma transação (colaborador só as próprias)             |
| POST   | `/api/transactions`         | Admin       | Cria ganho/despesa (carimba `created_by`)              |
| PUT    | `/api/transactions/{id}`    | Admin       | Edita transação                                        |
| DELETE | `/api/transactions/{id}`    | Admin       | Soft delete (`deleted_at`; a linha permanece)          |
| GET    | `/api/categories`           | Autenticado | Lista categorias de despesa                            |
| POST   | `/api/categories`           | Admin       | Cria categoria                                         |
| DELETE | `/api/categories/{id}`      | Admin       | Remove categoria (`409` se em uso por transações)      |

**Regras de transação:** `valor` positivo (centavos) e `data` obrigatórios.
`ganho` aceita `origem` (`implementacao`/`recorrencia`/`avulso`) e nunca categoria;
`despesa` aceita `category_id` e nunca origem. `project_id`/`user_id` opcionais.

**Filtros do `GET /api/transactions`** (query string): `from`, `to` (`YYYY-MM-DD`),
`tipo`, `origem`, `project_id`, `user_id`, `category_id`. Para colaborador, o
`user_id` é sempre forçado ao próprio, ignorando o parâmetro.

## API — módulo Recorrência

| Método | Rota                          | Auth          | Descrição                                             |
|--------|-------------------------------|---------------|-------------------------------------------------------|
| GET    | `/api/recurrence`             | Admin / Sócio | Resumo do mês: previsto × recebido × pendente         |
| GET    | `/api/recurrence/timeline`    | Admin / Sócio | 12 resumos mensais do ano (linha do tempo)            |

**Sem tabela de faturas.** A recorrência é calculada de `valor_mensal` + período
do projeto (`data_inicio`/`data_fim`, ambos opcionais = em aberto), cruzando com
as transações `ganho` de `origem=recorrencia` no mês:

- **previsto** = `valor_mensal` se o projeto está ativo no mês (0 se inativo);
- **recebido** = soma dos ganhos de recorrência do projeto no mês;
- **pendente** = `previsto − recebido`, nunca negativo.

Um projeto entra no resultado do mês se estiver **ativo** naquele mês **ou** se
tiver recebido recorrência nele. Parâmetros: `ano`, `mes` (default = mês atual),
`project_id` (opcional). `timeline` aceita `ano` e `project_id`.

## Estrutura

```
morfos-finance/
├── docker-compose.yml            # Postgres local
├── assets/branding/              # logo e material de identidade da Morfos
├── backend/
│   ├── cmd/api/                  # servidor HTTP
│   ├── cmd/seed/                 # provisiona o admin inicial
│   ├── internal/
│   │   ├── config/ database/ migrate/
│   │   ├── domain/               # entidades + erros de domínio
│   │   ├── auth/                 # argon2id + JWT
│   │   ├── repository/           # acesso a Postgres (pgx)
│   │   ├── service/              # regras de negócio
│   │   └── http/                 # router, middlewares, handlers, respostas
│   └── migrations/               # *.up.sql / *.down.sql (embutidas no binário)
└── frontend/                     # (próximos módulos)
```
