# Response Guard — Planejamento e Status

## 1) Objetivo do projeto
Construir uma API em Go para executar sequências de requisições HTTP e detectar regressões comparando respostas em dois modos:
- **DUAL**: A vs B
- **SINGLE**: BEFORE vs AFTER

Base funcional definida em [spec.md](../spec.md).

---

## 2) Planejamento macro (MVP)

### Fase A — Fundação
- Estrutura base do projeto Go
- Configuração YAML
- Camada de repositório por interface
- Migrações com **goose**

### Fase B — Domínio e Persistência
- Modelos: suite, suite_step, run, snapshot, diff
- Implementação SQL (SQLite/Postgres)
- Índices e relações essenciais

### Fase C — API de Suites/Steps
- CRUD de suites
- CRUD de steps
- Ordenação por `position`

### Fase D — Runner e Execução
- Executor sequencial
- HTTP client com timeout
- Coleta de `status_code`, `latency_ms`, `bytes`, `body`
- Persistência de snapshots

### Fase E — Comparação e Diffs
- Diff JSON simples com ignore paths
- Diff textual
- Hash SHA256
- Persistência de diffs

### Fase F — Infra e DX
- Dockerfile + docker-compose
- OpenAPI/Swagger
- Coleção Postman

---

## 3) O que já foi feito

## Atualizações recentes
- Criado `todo.md` na raiz como guia de handoff do agent (feito + pendencias + checklist de execucao).
- Revalidação após novas rotas: `go test ./...` executado com sucesso.
- Ajustes P0 de execução concluídos:
  - middleware de `Content-Type` permite `POST`/`PUT` sem body
  - política de `max_body_bytes` aplicada com truncamento (sem erro)
- Endpoints de observabilidade adicionados:
  - `GET /health` (liveness)
  - `GET /ready` (readiness com checagem de banco)
- OpenAPI atualizado para incluir `/health` e `/ready`
- `go.mod` estabilizado removendo dependência inválida `github.com/jackc/pgx/v5/stdlib` (o pacote continua via `github.com/jackc/pgx/v5`)
- Dependências sincronizadas com `go mod tidy`
- Build/compilação validados com `go test ./...` (passando)
- Ajuste de validação de URL para exigir URI válida com `http/https` e `host`
- Pasta `documentation/` criada com coleções Postman organizadas por endpoint:
  - `suites-endpoints.json` - CRUD de suites com exemplos
  - `steps-endpoints.json` - CRUD de steps com exemplos
  - `runs-endpoints.json` - Execução dual/single e monitoramento com exemplos
- Infra Docker ajustada para execução mais completa:
  - `Dockerfile` atualizado para `golang:1.23-alpine`
  - inclusão de toolchain C (`gcc`/`musl-dev`) para build com CGO
  - inclusão do binário `goose` no container da API
  - `docker-compose` atualizado para executar migrações na inicialização da API
- Execução Docker validada com smoke tests:
  - `docker-compose up -d` com `api` e `postgres` em estado `Up`
  - `GET /suites` retornando `200`
  - `POST /suites` retornando objeto criado
  - `GET /docs/` retornando `301` (redirect esperado para Swagger UI)

## Estrutura e base
- Projeto Go inicializado
- Loader de configuração em YAML
- Conexão de banco por `database.driver` (sqlite/postgres)
- Interface de repositório definida

## Banco e migrações
- Migração inicial criada com goose: [migrations/001_init.sql](../migrations/001_init.sql)
- Tabelas criadas: `suite`, `suite_step`, `run`, `snapshot`, `diff`
- Índices principais criados

## Repositório
- Implementação SQL com placeholders para SQLite/Postgres: [internal/repository/sqlrepo/sqlrepo.go](../internal/repository/sqlrepo/sqlrepo.go)

## Endpoints de Suite/Step
- `POST /suites`
- `GET /suites`
- `GET /suites/{id}`
- `POST /suites/{id}/steps`
- `GET /suites/{id}/steps`
- `PUT /steps/{id}`
- `DELETE /steps/{id}`

## Runner e Runs
- `POST /runs/dual`
- `POST /runs/single/start`
- `POST /runs/{id}/continue`
- `GET /runs/{id}`
- `GET /runs/{id}/diffs`
- Execução sequencial com snapshots (`A`, `B`, `BEFORE`, `AFTER`)
- Diffs persistidos por step

## Validações iniciais
- URL
- método HTTP
- JSON em `headers_json` / `extractors_json`
- campos obrigatórios
- validação de `Content-Type` para POST/PUT

## Documentação e ferramentas
- OpenAPI: [openapi.yaml](../openapi.yaml)
- Swagger UI em `/docs`
- Coleção Postman: [postman_collection.json](../postman_collection.json)
- Guia rápido: [QUICKSTART.md](../QUICKSTART.md)

## Infra
- Dockerfile
- docker-compose (API + Postgres)
- configs local/docker

---

## 4) Pendências e ajustes necessários (curto prazo)

### P0 — Estabilidade de build
- ✅ Corrigido `go.mod` (dependência inválida removida)
- ✅ `go mod tidy` executado
- ✅ `go test ./...` validado

### P0 — Qualidade funcional
- ✅ Ajustado comportamento do middleware de `Content-Type` para endpoints sem corpo (ex.: `POST /runs/{id}/continue`)
- ✅ Definida política de limite de body (`max_body_bytes`) com truncamento sem erro

### P0 — Ambiente de execução local
- Monitorar recorrência de instabilidade intermitente do Docker Desktop/Engine no host (erro eventual de API/pipe), apesar de o fluxo principal estar funcionando.

### P1 — Consistência de contrato
- Garantir alinhamento 1:1 entre handlers e OpenAPI
- Padronizar respostas de erro (códigos e payload)

### P1 — Banco de dados
- Validar migração em SQLite e Postgres reais
- Adicionar constraints/checks úteis (enum-like para status/mode/variant)

---

## 5) Backlog do MVP (a concluir)

- Cálculo e persistência explícita de `delta_latency` e `delta_bytes` por comparação
- Melhorar diff JSON para cenários complexos (arrays e tipos mistos)
- Cobertura de testes de integração (HTTP + DB)
- Logs estruturados por run/step

---

## 6) Evoluções pós-MVP (planejado)

- Paralelismo controlado por step
- Variáveis entre steps
- Extractors com JSONPath real
- Thresholds configuráveis de métricas
- Exportação de runs
- UI de visualização
- Armazenamento externo para corpos grandes

---

## 7) Próximos passos sugeridos (execução)
1. Fechar ajustes de validação e erros HTTP.
2. Garantir execução completa de migrações em SQLite/Postgres.
3. Implementar deltas de métricas e testes de integração.
4. Congelar contrato OpenAPI + Postman para baseline do MVP.
