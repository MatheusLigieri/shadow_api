# TODO - Shadow API

## Objetivo
Guia rapido de handoff para o agent com o status real do projeto (feito e pendente), sincronizado com `specs/planejamento-status.md`.

## Feito
- [x] Estrutura base em Go com configuracao YAML
- [x] Repositorio SQL com suporte SQLite e Postgres
- [x] Migracoes com goose (schema inicial)
- [x] Endpoints de suites e steps (CRUD)
- [x] Endpoints de runs (dual, single/start, continue, status, diffs)
- [x] Runner sequencial com snapshots (A, B, BEFORE, AFTER)
- [x] Diff basico de resposta (JSON/texto/hash)
- [x] Swagger/OpenAPI e colecoes Postman
- [x] Docker e docker-compose com smoke test validado
- [x] go.mod estabilizado e build/testes validado

## Em aberto

### P0
- [x] Ajustar validacao de Content-Type para endpoint POST sem body (ex.: `/runs/{id}/continue`)
- [x] Definir politica final de `max_body_bytes` (truncar sem erro)

### P1
- [x] Garantir alinhamento 1:1 entre handlers e OpenAPI
- [x] Padronizar payload/codigo de erro em todos os endpoints
- [x] Validar migracoes em SQLite e Postgres reais
- [x] Adicionar constraints de dominio (`mode`/`status`/`variant`)

### P2
- [x] Persistir `delta_latency` e `delta_bytes` por comparacao
- [x] Melhorar diff JSON para cenarios complexos (arrays/tipos mistos)
- [x] Criar testes de integracao HTTP + DB
- [x] Adicionar endpoint de healthcheck/readiness (`/health`, `/ready`)

## Checklist handoff
- [x] Rodar `go test ./...`
- [ ] Rodar `docker-compose up -d` e validar `/docs`, `/health`, `/ready` e `/suites`
- [ ] Executar fluxo minimo: criar suite -> criar step -> run dual -> consultar diffs
- [ ] Atualizar `specs/planejamento-status.md` apos cada entrega

## Arquivos importantes
- `cmd/api/main.go`
- `internal/httpapi/handlers.go`
- `internal/httpapi/middleware.go`
- `internal/runner/runner.go`
- `internal/diff/diff.go`
- `internal/repository/sqlrepo/sqlrepo.go`
- `migrations/001_init.sql`
- `openapi.yaml`
- `specs/planejamento-status.md`
