# Morfos Finance

Sistema interno de controle financeiro da Morfos Tech. Backend em Go (Chi + PostgreSQL), frontend em React + TypeScript, identidade visual alinhada ao site da Morfos.

> Módulos: **auth ✅ · projetos ✅ · transações ✅ · planejamento ✅ · orçamentos ✅ · recorrência ✅ · Via Permuta ✅ · anexos ✅ · dashboards ✅ · frontend/tema ✅**.

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

# 5. Frontend (em outro terminal)
cd ../frontend
npm install
npm run dev                   # http://localhost:5173 (proxy /api -> :8080)
```

A API aplica as migrations pendentes automaticamente ao subir. `go run ./cmd/seed` é
idempotente — se o admin já existe, não faz nada. O front (Vite) faz proxy de
`/api` e `/uploads` para o backend, então basta abrir `http://localhost:5173`.

### Variáveis de ambiente

Ver [`backend/.env.example`](backend/.env.example). Essenciais: `DATABASE_URL`, `JWT_SECRET`.
Para produção, troque `JWT_SECRET` por um valor longo e aleatório e defina
`SEED_ADMIN_EMAIL` / `SEED_ADMIN_SENHA` antes de rodar o seed.

Para usar **Supabase Storage**, crie primeiro o bucket, habilite o protocolo S3
em `Storage > Configuration > S3` e gere as credenciais S3 de servidor. Configure
`S3_ENDPOINT`, `S3_BUCKET`, `S3_ACCESS_KEY_ID`, `S3_SECRET_ACCESS_KEY` e
`S3_REGION` com os valores mostrados nessa tela. As chaves comuns `anon` e
`service_role` não são usadas pelo cliente S3 atual. Como os anexos são salvos
com URL direta, o bucket deve ser público e `S3_PUBLIC_BASE_URL` deve ser a URL
pública do bucket. Para bucket privado, será necessário adicionar geração de URL
assinada no backend.

### Deploy no Railway

O `Dockerfile` gera o frontend e o backend em uma única imagem. Em produção, o
servidor Go entrega a SPA e a API no mesmo domínio. O `railway.json` configura o
health check em `/health`.

Crie um PostgreSQL no mesmo projeto e configure `DATABASE_URL` como referência a
`${{Postgres.DATABASE_URL}}`. Também são necessárias `JWT_SECRET`, `JWT_TTL` e as
variáveis `S3_*` descritas acima. O backend só ativa o Supabase Storage quando
endpoint, bucket, access key e secret key estiverem presentes; antes disso, usa
o armazenamento local temporário do container.

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
| GET    | `/api/users`                      | Admin / Sócio | Lista usuários                              |
| POST   | `/api/users`                      | Admin / Sócio | Cria usuário com senha inicial              |
| POST   | `/api/users/{id}/reset-password`  | Admin / Sócio | Reseta senha (força troca no próximo login) |

**Papéis:** `admin` e `socio` têm **o mesmo acesso** — veem e editam tudo
(projetos, transações, anexos, usuários). A diferença é só de contexto: no
dashboard, o sócio (e o admin) pode alternar entre a visão da empresa e uma
**visão individual** (seus próprios ganhos/despesas/projetos), como um
colaborador teria. `colaborador` só vê a própria área e qualquer alteração é
enviada como solicitação para aprovação de admin/sócio. Usuários novos
nascem com `must_change_password = true`.

## API — módulo Projetos

| Método | Rota                                          | Auth        | Descrição                                                    |
|--------|-----------------------------------------------|-------------|-------------------------------------------------------------|
| GET    | `/api/projects`                               | Autenticado | Lista projetos (colaborador vê só os alocados)              |
| GET    | `/api/projects/{id}`                          | Autenticado | Projeto + parcelas + membros (colaborador só se alocado)   |
| POST   | `/api/projects`                               | Admin / Sócio | Cria projeto; gera parcelas 50/50 se houver implementação   |
| PUT    | `/api/projects/{id}`                          | Admin / Sócio | Atualiza campos; reconcilia parcelas de implementação       |
| PUT    | `/api/projects/{id}/members`                  | Admin / Sócio | Define a lista de colaboradores alocados                    |
| PATCH  | `/api/projects/{id}/installments/{iid}`       | Admin / Sócio | Marca parcela paga (`pago_em`) ou pendente (`null`)         |

**Fontes de receita:** um projeto tem `valor_implementacao` e/ou `valor_mensal`
(ao menos um). A implementação vira **duas parcelas** — `entrada` (50%, arredondada
para baixo) e `finalizacao` (o restante) — que sempre somam o valor total.

**Regra de reconciliação de parcelas** (no `PUT`): alterar/remover o valor de
implementação **com uma parcela já paga** retorna `409`; sem parcela paga, as
parcelas são regeradas. Mensalidade sozinha não gera parcelas.

Marcar uma parcela como paga cria ou restaura automaticamente uma transação de
ganho de implementação vinculada à parcela. Voltar a parcela para pendente faz
soft delete dessa transação. Transações gerenciadas por parcelas não podem ser
editadas ou excluídas diretamente.

## API — módulo Transações & Categorias

| Método | Rota                        | Auth        | Descrição                                              |
|--------|-----------------------------|-------------|--------------------------------------------------------|
| GET    | `/api/transactions`         | Autenticado | Lista com filtros (colaborador vê só as próprias)      |
| GET    | `/api/transactions/{id}`    | Autenticado | Uma transação (colaborador só as próprias)             |
| POST   | `/api/transactions`         | Admin / Sócio | Cria ganho/despesa (carimba `created_by`)              |
| PUT    | `/api/transactions/{id}`    | Admin / Sócio | Edita transação                                        |
| DELETE | `/api/transactions/{id}`    | Admin / Sócio | Soft delete (`deleted_at`; a linha permanece)          |
| GET    | `/api/categories`           | Autenticado | Lista categorias de despesa                            |
| POST   | `/api/categories`           | Admin / Sócio | Cria categoria                                         |
| DELETE | `/api/categories/{id}`      | Admin / Sócio | Remove categoria (`409` se em uso por transações)      |

**Regras de transação:** `valor` positivo (centavos) e `data` obrigatórios.
Ganhos manuais aceitam `origem` (`recorrencia`/`avulso`) e nunca categoria;
implementação é gerada exclusivamente pelo pagamento de parcelas, enquanto
recorrência exige `project_id`;
`despesa` aceita `category_id` e nunca origem. `user_id` é opcional; `project_id`
continua opcional para ganhos avulsos e despesas.

**Filtros do `GET /api/transactions`** (query string): `from`, `to` (`YYYY-MM-DD`),
`tipo`, `origem`, `project_id`, `user_id`, `category_id`. Para colaborador, o
`user_id` é sempre forçado ao próprio, ignorando o parâmetro.

## API — Planejamento e orçamentos

Planejamentos representam contas futuras e ficam separados das transações
realizadas. Uma baixa cria a transação correspondente de forma atômica. A
criação pode repetir o lançamento mensalmente por até 24 meses; vencimentos no
fim do mês são ajustados para o último dia válido.

| Método | Rota                              | Descrição |
|--------|-----------------------------------|-----------|
| GET    | `/api/planning`                   | Lista por período e situação (`aberto`/`realizado`) |
| POST   | `/api/planning`                   | Cria uma ou várias provisões mensais |
| PUT    | `/api/planning/{id}`              | Edita uma provisão em aberto |
| DELETE | `/api/planning/{id}`              | Exclui uma provisão em aberto |
| POST   | `/api/planning/{id}/complete`     | Dá baixa e cria a transação realizada |
| GET    | `/api/planning/cash-flow`         | Projeta entradas, saídas e saldo por data |
| GET    | `/api/budgets`                    | Compara orçamento e despesa realizada no mês |
| PUT    | `/api/budgets`                    | Cria ou atualiza limite de uma categoria |
| DELETE | `/api/budgets/{id}`               | Remove um limite mensal |

O frontend também exporta a lista filtrada de transações em CSV compatível com
planilhas brasileiras (UTF-8, separador `;` e valores em reais).

## API — Via Permuta

VP é tratado como um livro auxiliar independente: usa centésimos inteiros para
manter precisão, mas nunca é somado ao caixa, ganhos ou despesas em reais. O
saldo VP considera somente vendas e compras concluídas; negociações abertas não
alteram a posição. O disponível é `saldo VP + limite aprovado`.

| Método | Rota                                      | Descrição |
|--------|-------------------------------------------|-----------|
| GET    | `/api/via-permuta/summary`                | Posição, vendas/compras do período, tickets e indicadores |
| GET/PUT| `/api/via-permuta/settings`               | Consulta ou ajusta o limite de crédito VP |
| GET/POST | `/api/via-permuta/transactions`        | Lista ou cria vendas/compras VP |
| PUT/DELETE | `/api/via-permuta/transactions/{id}` | Edita ou faz soft delete de uma movimentação |
| GET/POST | `/api/via-permuta/offers`              | Lista ou cria ofertas do catálogo |
| PUT/DELETE | `/api/via-permuta/offers/{id}`       | Edita ou faz soft delete de uma oferta |

Movimentações aceitam status `negociando`, `concluida`, `recusada` ou
`cancelada`, vínculo opcional com projeto, código de voucher e observações.
Ofertas aceitam valor fixo ou negociável e guardam o link externo do anúncio na
Via Permuta. Todas as rotas deste módulo são restritas a admin/sócio.

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

## API — módulo Anexos

Uploads são `multipart/form-data` com o campo **`file`** e `descricao` opcional.
O nome original sanitizado é persistido em `nome_arquivo` e retornado pela API;
a chave interna do objeto permanece única para evitar colisões entre uploads.

| Método | Rota                                                    | Auth        | Descrição                                  |
|--------|---------------------------------------------------------|-------------|--------------------------------------------|
| POST   | `/api/transactions/{id}/attachments`                    | Admin / Sócio | Comprovante de transação (PDF/PNG/JPG/JPEG)|
| GET    | `/api/transactions/{id}/attachments`                    | Autenticado | Lista comprovantes da transação            |
| POST   | `/api/projects/{id}/installments/{iid}/attachments`     | Admin / Sócio | Comprovante de parcela                      |
| DELETE | `/api/attachments/{id}`                                 | Admin / Sócio | Remove comprovante (DB + objeto)           |
| POST   | `/api/projects/{id}/proposals`                          | Admin / Sócio | Proposta comercial (PDF/DOCX)              |
| GET    | `/api/projects/{id}/proposals`                          | Autenticado | Lista propostas do projeto                 |
| DELETE | `/api/proposals/{id}`                                   | Admin / Sócio | Remove proposta                             |

**Storage** atrás de uma interface (`internal/storage`): sem variáveis `S3_*`, os
arquivos vão para **disco local** (`UPLOAD_DIR`, servidos em `/uploads`); com
`S3_ENDPOINT` + `S3_BUCKET`, usa **S3-compatible** (Cloudflare R2 por padrão, via
`minio-go`) — a mesma lógica troca de backend só pelo `.env`.

**Validação:** comprovantes aceitam PDF/PNG/JPG/JPEG; propostas PDF/DOCX; tamanho
máximo `MAX_UPLOAD_MB` (default 10). O dono (transação/parcela/projeto) precisa
existir, senão `404`.

## API — módulo Dashboards

| Método | Rota                         | Auth          | Descrição                                        |
|--------|------------------------------|---------------|--------------------------------------------------|
| GET    | `/api/dashboard/company`     | Admin / Sócio | Visão financeira completa da empresa             |
| GET    | `/api/dashboard/me`          | Autenticado   | Visão pessoal (colaborador: próprios números)    |

**Período** via `from`/`to` (`YYYY-MM-DD`); sem parâmetros usa o **mês atual**.
Isso cobre os filtros diário/semanal/mensal/anual/personalizado — o cliente
escolhe o intervalo.

**`company`** traz: `saldo_em_caixa` (acumulado de todos os tempos, entrou −
saiu), `ganhos`/`despesas`/`resultado` do período, `ganhos_por_origem`,
`despesas_por_categoria`, `implementacao` (total × recebido × a_receber das
parcelas), `parcelas_pendentes`, `recorrencia_mes` (resumo do mês de `to`),
`recorrencia_futura` (previsão dos 12 meses seguintes), e os
recortes `por_projeto` e `por_colaborador`.

**`me`** traz `ganhos`/`despesas`/`saldo` do colaborador no período e seus
`projetos` alocados.

## API — solicitações e anotações

Anotações são sempre listadas no escopo do próprio usuário. Admin e sócio podem
criar, editar e excluir diretamente. Colaboradores enviam uma solicitação, e a
mutação só é aplicada quando um admin ou sócio a aprova.

| Método | Rota                                  | Auth          | Descrição                         |
|--------|---------------------------------------|---------------|-----------------------------------|
| GET    | `/api/notes`                          | Autenticado   | Lista as próprias anotações       |
| POST   | `/api/notes`                          | Admin / Sócio | Cria anotação diretamente         |
| PUT    | `/api/notes/{id}`                     | Admin / Sócio | Edita a própria anotação          |
| DELETE | `/api/notes/{id}`                     | Admin / Sócio | Exclui a própria anotação         |
| GET    | `/api/change-requests`                | Autenticado   | Próprias solicitações ou fila completa |
| POST   | `/api/change-requests`                | Colaborador   | Solicita criação/edição/exclusão  |
| POST   | `/api/change-requests/{id}/approve`   | Admin / Sócio | Aprova e aplica a alteração       |
| POST   | `/api/change-requests/{id}/reject`    | Admin / Sócio | Rejeita com justificativa         |

## Frontend (React + TypeScript)

SPA em Vite + React 18 + TS, **CSS Modules/vanilla com os tokens da Morfos**
(`src/styles/tokens.css`, extraídos do site) — tema escuro, acentos teal/copper,
tipografia Space Grotesk / Manrope / Space Mono, kickers em maiúsculas e seções
numeradas, sem lib de UI.

- **Login** por e-mail/senha, com **troca obrigatória no 1º acesso**.
- **Dashboard** — visão de empresa (admin/sócio) com saldo em caixa, a receber ×
  recebido, ganhos por origem, despesas por categoria, recorrência do mês e
  recortes por projeto/colaborador; visão pessoal (colaborador) com seus números
  e projetos.
- **Projetos** — lista, criação, e detalhe com parcelas (marcar paga/pendente),
  colaboradores e upload de propostas.
- **Transações** — lista com filtros (tipo/período), criação e soft delete.
- **Recorrência** — resumo mensal + linha do tempo do ano (admin/sócio).
- **Usuários** — cadastro e reset de senha (admin).
- **Anotações** — notas por usuário, projeto ou transação.
- **Solicitações** — aprovação/rejeição das alterações pedidas por colaboradores.

O `AuthContext` guarda o JWT em `localStorage`; rotas são protegidas por
autenticação e por papel. Valores monetários são formatados de centavos para BRL
no cliente.

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
