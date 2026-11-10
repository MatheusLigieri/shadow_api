# TODO - Shadow API

## Objetivo
Guia rapido de handoff para o agent com o status real do projeto (feito e pendente), sincronizado com `planejamento-status.md`.

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
- [x] Completar alinhamento handlers/OpenAPI e payload padrão para todas as respostas 404/405
- [x] Validar migracoes em SQLite e Postgres reais
- [x] Adicionar constraints de dominio (`mode`/`status`/`variant`)

### P2
- [x] Persistir `delta_latency` e `delta_bytes` por comparacao
- [x] Melhorar diff JSON para cenarios complexos (arrays/tipos mistos)
- [x] Criar testes de integracao HTTP + DB
- [x] Adicionar endpoint de healthcheck/readiness (`/health`, `/ready`)

## Checklist handoff
- [x] Rodar `go test ./...`
- [x] Smoke test Docker/Postgres e fluxo minimo executados anteriormente
- [x] Reiniciar Docker Compose e revalidar `/docs`, `/health`, `/ready`, `/suites` e suite -> step -> dual run -> diffs (2026-11-05)
- [x] Criar collection Postman unica em `shadow_api_collection.json` (14 requests em 4 pastas)
- [x] Atualizar todos os documentos de status e referencias da collection
- [x] Atualizar `context/planejamento-status.md` apos cada entrega

## Lacunas funcionais confirmadas
- [ ] Comparar `status_code` dos snapshots junto com o body
- [ ] Executar `extractors_json` para compartilhar valores entre steps
- [x] Padronizar payload JSON para todos os 404/405 e completar respostas no OpenAPI
- [ ] Responder 404 em `PUT`/`DELETE /steps/{id}` para step inexistente e validar a suite em `/suites/{id}/steps`
- [ ] Criar migration com os `CHECK` de `mode`/`status`/`variant` para bancos que aplicaram a versao antiga de `001_init.sql` (inclui o volume Postgres local)

## Feito em 2026-11-05
- [x] 404/405 com payload JSON padrao em todas as rotas, header `Allow` no 405 e mensagens `suite not found`/`run not found`
- [x] OpenAPI com `415`/`500` por operacao e regra geral de 404/405
- [x] Corrigido loop de redirecionamento em `/docs/` (Swagger UI nao carregava)
- [x] Testes do fluxo SINGLE em `/diffs`, dos erros 404/405 e de `/docs`
- [x] Smoke test Docker/Postgres revalidado

## Feito em 2026-10-16
- [x] CORS totalmente aberto para testes (qualquer origem, metodos incluindo PATCH/HEAD, ecoa headers do preflight)
- [x] `GET /runs/{id}/diffs` devolve `before`/`after` com o JSON inteiro de cada resposta comparada + status_code, latencia, bytes e hash
- [x] `GET /runs/{id}/diffs` responde 404 JSON para run inexistente
- [x] OpenAPI e README atualizados; testes `TestDualRunDiffsReturnFullBodies` e `TestListDiffsRunNotFound`

## Riscos/pendencias levantados na analise (detalhes em `planejamento-status.md`, secao 4)
### Alta prioridade
- [ ] Adicionar status `FAILED` (modelo, migration/constraint, handlers) e marcar a run quando um step falha
- [ ] Desacoplar execucao do request (contexto proprio / execucao assincrona) para nao cancelar run quando o cliente desconecta
- [ ] Tratar erro de `UpdateRunStatus` em vez de ignorar
- [ ] Incluir `status_code` no diff (ja listado acima)

### Media prioridade
- [ ] Sinalizar truncamento por `max_body_bytes` e gravar tamanho real em `bytes`
- [ ] Suportar arrays/curingas em `ignore_json_paths` (`$.items[*].id`)
- [ ] Ordenar chaves no diff JSON para saida deterministica
- [ ] Testes unitarios para `runner` e `sqlrepo`; gerar schema de teste a partir das migrations

### Seguranca / higiene
- [ ] Restringir CORS e avaliar allowlist de hosts (SSRF) antes de expor fora do ambiente de teste
- [ ] Avaliar mascaramento de dados sensiveis nos bodies devolvidos em `/diffs`
- [x] Adicionar `.gitattributes` (`eol=lf`) e normalizar CRLF em commit separado
- [x] Versionar `context/` e `shadow_api_collection.json`
- [ ] Decidir remocao do `postman_collection.json` antigo

## Bugs corrigidos (branch chore/infra)
- `diff_json` retornava `null` em vez de `[]` quando nao havia diferencas (slice nil no marshal)
- `POST /runs/dual` e `POST /runs/single/start` retornavam status desatualizado na resposta (`RUNNING` em vez de `COMPLETED`/`WAITING_USER`)

## Workflow de branches
- Nunca commitar diretamente na `main`
- Usar branches por funcionalidade/tipo: `feat/`, `fix/`, `chore/`
- PRs para merge na `main`

## Arquivos importantes
- `cmd/api/main.go`
- `internal/httpapi/handlers.go`
- `internal/httpapi/middleware.go`
- `internal/runner/runner.go`
- `internal/diff/diff.go`
- `internal/repository/sqlrepo/sqlrepo.go`
- `migrations/001_init.sql`
- `openapi.yaml`
- `context/spec.md`
- `context/planejamento-status.md`
- `context/todo.md`
- `shadow_api_collection.json`
