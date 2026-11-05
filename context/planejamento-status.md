# Response Guard — Planejamento e Status

## 1) Objetivo do projeto
Construir uma API em Go para executar sequências de requisições HTTP e detectar regressões comparando respostas em dois modos:
- **DUAL**: A vs B
- **SINGLE**: BEFORE vs AFTER

Base funcional definida em [spec.md](spec.md).

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

## 3) Status atual da implementação (2026-11-05)

### Entregue
- Fundação Go, configuração YAML e interface de repositório.
- Persistência SQL com SQLite/Postgres e migrações Goose para suites, steps, runs, snapshots e diffs.
- Constraints de domínio para `run.mode`, `run.status` e `snapshot.variant`.
- Endpoints de Infra, Suites, Steps e Runs; runner sequencial para DUAL e SINGLE.
- Snapshots com status HTTP, body, SHA256, latência e tamanho; diffs JSON/texto com ignore paths e deltas de latência/tamanho.
- Dockerfile, Compose com Postgres e migrações no startup, OpenAPI/Swagger e collection Postman unificada.
- `go test ./...` passa na branch `chore/infra`.

### Entregas de 2026-10-16
- **CORS aberto para testes** ([middleware.go](../internal/httpapi/middleware.go)): `Access-Control-Allow-Origin: *`, métodos `GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS`, ecoa qualquer header pedido em `Access-Control-Request-Headers`, `Expose-Headers: *`, `Max-Age: 86400`; preflight `OPTIONS` responde `204` em qualquer rota. Configuração temporária, sem credenciais; restringir antes de expor o serviço.
- **JSONs completos na comparação**: `GET /runs/{id}/diffs` passou a devolver, em cada diff, `before` e `after` com `variant`, `status_code`, `latency_ms`, `bytes`, `body_hash` e `body` inteiro (como JSON quando o body é JSON válido, senão string). Mapeamento: DUAL → A/B, SINGLE → BEFORE/AFTER, usando o snapshot mais recente por step+variant. Campos antigos (`diff_json`, `summary`, deltas) foram mantidos.
- `GET /runs/{id}/diffs` agora responde `404` com payload JSON padrão quando a run não existe (antes devolvia `[]`).
- OpenAPI atualizado (schema `ComparedResponse`, campos `before`/`after`, `404` em diffs). Testes novos: `TestDualRunDiffsReturnFullBodies` (upstreams fake A/B) e `TestListDiffsRunNotFound`.

### Entregas de 2026-11-05
- **Erros padronizados**: todas as respostas 404/405 da API usam o payload JSON padrão, inclusive rotas desconhecidas; 405 envia o header `Allow`. `GET /suites/{id}` e `GET /runs/{id}` não expõem mais `sql: no rows in result set` e respondem `suite not found`/`run not found`.
- **OpenAPI alinhado** aos status retornados pelos handlers (`415` e `500` por operação; regra geral de 404/405 na descrição da API).
- **Swagger UI corrigido**: `/docs/` entrava em loop de redirecionamento (o handler redirecionava para `/docs/index.html` e o `http.FileServer` redirecionava de volta). Agora `/docs/` serve a página.
- Testes novos: `TestSingleRunDiffsReturnBeforeAfter`, `TestMethodNotAllowedUsesJSONError`, `TestNotFoundUsesJSONError`, `TestDocsRedirectStillWorks`.

### Verificação de ambiente
- Em 2026-11-05 o stack Compose foi reconstruído com `docker compose up -d --build` e o fluxo passou em Postgres: Swagger UI, `/health`, `/ready`, `/suites`, suite → step → run DUAL → diffs, run SINGLE (start → continue → diffs com `before`/`after`), erros 404/405/415 e preflight CORS.
- O teste deixou no volume local as suites `smoke-2026-11-05` e `smoke-2026-11-05-single` (não há endpoint para apagar suites).
- A collection atual é [shadow_api_collection.json](../shadow_api_collection.json), com 14 requests em Infra, Suites, Steps e Runs. As collections específicas antigas permanecem como referência.

### Referências principais
- Contrato funcional: [spec.md](spec.md)
- Handoff e checklist executável: [todo.md](todo.md)
- Contrato HTTP: [openapi.yaml](../openapi.yaml)
- Guia de uso: [README.md](../README.md) e [QUICKSTART.md](../QUICKSTART.md)
- Implementação SQL: [internal/repository/sqlrepo/sqlrepo.go](../internal/repository/sqlrepo/sqlrepo.go)

---

## 4) Lacunas confirmadas

- Comparar `status_code` dos snapshots além do body. Atualmente ele é persistido, mas não influencia `passed`/`diff_json`.
- Executar `extractors_json` para disponibilizar valores extraídos nos steps seguintes. Atualmente o campo é validado e persistido, mas o runner não o consome.
- **Step ou suite inexistente**: `PUT /steps/{id}` responde `200` e `DELETE /steps/{id}` responde `204` mesmo sem step correspondente (o repositório não verifica `RowsAffected`); `GET /suites/{id}/steps` responde `[]` e `POST /suites/{id}/steps` não valida a suite antes do insert.
- **Constraints ausentes em bancos antigos**: os `CHECK` de `mode`/`status`/`variant` foram adicionados editando `001_init.sql` (commit `4690c2b`). Bancos que já tinham aplicado a migration não os recebem; o volume Postgres local está nessa situação (confirmado em 2026-11-05). Requer uma nova migration, que também pode incluir o status `FAILED`.

### Riscos identificados na análise de 2026-10-16

Robustez da execução:
- **Run presa em `RUNNING`**: se qualquer step falha (timeout, DNS, conexão recusada), o handler devolve 500 mas nunca atualiza o status. Não existe estado `FAILED` no domínio nem na constraint da tabela `run` ([handlers.go](../internal/httpapi/handlers.go), `handleRunDual`/`handleRunSingleStart`/`continueRun`).
- **Execução síncrona presa ao request**: o runner usa `r.Context()`; desconexão do cliente ou timeout de proxy cancela a run no meio, caindo no item anterior. Suites longas tendem a estourar timeouts de gateway.
- **Erros ignorados**: retorno de `UpdateRunStatus` é descartado (`_ =`).
- `latestSnapshot` no runner lista todos os snapshots da run a cada step (O(n²)); irrelevante hoje, cresce com suites grandes.

Qualidade do diff ([diff.go](../internal/diff/diff.go)):
- **Truncamento quebra a comparação JSON**: body acima de `max_body_bytes` é cortado; o JSON truncado não faz parse e a comparação cai para texto literal (um único change `$.body`). Além disso `bytes` grava o tamanho truncado, não o real, distorcendo `delta_bytes`. Não há flag indicando truncamento.
- **Ignore paths só para objetos**: `$.a.b` funciona; índices de array e curingas (`$.items[*].id`, `$.items[0].ts`) não são suportados.
- **Ordem do `diff_json` não determinística**: chaves de objeto são iteradas via map do Go; a mesma comparação pode gerar listas em ordem diferente. Ordenar as chaves resolve.
- Arrays são comparados por índice; inserção/remoção no início gera cascata de changes.

Testes:
- Pacotes `runner` e `sqlrepo` não têm testes unitários; `runner` só é exercitado indiretamente por `TestDualRunDiffsReturnFullBodies`.
- O schema de teste em `handlers_test.go` duplica as migrações manualmente e pode divergir delas.

Segurança:
- **SSRF**: a API faz requests para qualquer `base_a`/`base_b`/`base` enviado pelo usuário, sem autenticação, agora com CORS `*`. Não expor fora de rede interna/ambiente de teste.
- Bodies completos (podem conter dados sensíveis/tokens das APIs testadas) ficam persistidos e agora são devolvidos em `/diffs`.

Repositório/higiene:
- Fins de linha normalizados em commit separado: `.gitattributes` define `* text=auto eol=lf`, mantendo CRLF apenas em `*.bat`.
- `spec.md`, `todo.md` e `specs/` foram movidos para `context/`. `shadow_api_collection.json` está versionado; falta decidir a remoção do antigo `postman_collection.json`.

## 5) Evoluções pós-MVP

- Paralelismo controlado por step.
- Variáveis entre steps e extractors JSONPath.
- Thresholds configuráveis de métricas.
- Exportação de runs e UI de visualização.
- Armazenamento externo para corpos grandes.
- Logs estruturados por run/step.

## 6) Branch e workflow

- Trabalho atual na branch `chore/infra`, acompanhando `origin/chore/infra`.
- Não criar commits diretamente na `main`; publicar alterações em branches de trabalho e abrir PR.
