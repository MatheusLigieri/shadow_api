# Response Guard — Spec simplificada

## 1. Objetivo

API em Golang para executar uma sequência de requests HTTP e comparar respostas para detectar regressões.

### Status da implementação (2026-11-05)

O MVP está implementado e `go test ./...` passa. A collection Postman unificada está em `../shadow_api_collection.json`. O smoke test Docker/Postgres e o fluxo mínimo passaram em 2026-11-05. Limitações conhecidas: status HTTP ainda não participa do diff; `extractors_json` é armazenado/validado, mas não executado.

### Modos de execução

**Dual**
- Executa requests em endpoint A e endpoint B
- Compara A vs B

**Single**
- Executa requests em endpoint
- Salva snapshot (before)
- Fica em estado WAITING_USER
- Usuário chama continue
- Executa novamente (after)
- Compara before vs after

---

## 2. Configuração

Arquivo único `config.yaml`.

```yaml
app:
  port: 8080
  log_level: "info"
  max_body_bytes: 1048576

database:
  driver: "sqlite" # sqlite | postgres

  sqlite:
    path: "./data.db"

  postgres:
    dsn: "postgres://user:pass@localhost/db?sslmode=disable"

runner:
  timeout_ms: 15000
  ignore_json_paths:
    - "$.timestamp"
    - "$.requestId"
```

Regras:
- `database.driver` define qual implementação será usada
- Camada de repositório deve ser abstraída por interface

---

## 3. Modelo de dados

### suite
- id
- name

### suite_step
- id
- suite_id
- position
- method
- path
- headers_json
- body_text
- extractors_json (opcional)

### run
- id
- suite_id
- mode (`DUAL` | `SINGLE`)
- status (`RUNNING | WAITING_USER | COMPLETED`)
- base_a
- base_b
- base_single
- created_at

### snapshot
- id
- run_id
- step_id
- variant (`A | B | BEFORE | AFTER`)
- status_code
- body
- body_hash
- latency_ms
- bytes
- created_at

### diff
- id
- run_id
- step_id
- passed
- diff_json
- summary
- delta_latency_ms (B - A ou AFTER - BEFORE)
- delta_bytes (B - A ou AFTER - BEFORE)

---

## 4. Endpoints

### Infra
- `GET /health` - liveness
- `GET /ready` - readiness com consulta ao banco

### Suite
- `POST /suites`
- `GET /suites`
- `GET /suites/{id}`

### Steps
- `POST /suites/{id}/steps`
- `GET /suites/{id}/steps`
- `PUT /steps/{id}`
- `DELETE /steps/{id}`

### Run

#### Dual
`POST /runs/dual`

```json
{
  "suite_id": "uuid",
  "base_a": "https://old",
  "base_b": "https://new"
}
```

#### Single start
`POST /runs/single/start`

```json
{
  "suite_id": "uuid",
  "base": "https://api"
}
```

#### Continue
`POST /runs/{id}/continue`

#### Consulta
- `GET /runs/{id}`
- `GET /runs/{id}/diffs`

---

## 5. Estratégia de comparação

### JSON
- Parse para mapa
- Remover paths ignorados
- Comparação profunda
- Saída em lista de mudanças

Exemplo:

```json
[
  {
    "path": "$.price",
    "before": 100,
    "after": 120
  }
]
```

### Texto
- Comparação direta de string

### Corpos não JSON e hash
- Corpos não JSON são comparados como texto literal.
- SHA256 é persistido em cada snapshot; o hash não é usado atualmente como uma estratégia de comparação binária separada.
- `status_code` é salvo no snapshot, mas ainda não participa do diff.

---

## 6. Métricas armazenadas

Por snapshot:
- latency_ms
- bytes
- status_code

Comparação:
- delta_latency_ms
- delta_bytes

Sem thresholds automáticos no MVP.

---

## 7. Estados de execução

- RUNNING
- WAITING_USER (apenas single)
- COMPLETED

---

## 8. Status do MVP

As fases de estrutura, persistência SQLite/Postgres, migrações, API de suites/steps, runner, modos DUAL/SINGLE, diffs, OpenAPI/Swagger, Docker e collection Postman estão implementadas. Testes automatizados passam com `go test ./...`.

Pendências para fechar o contrato funcional:
- Comparar `status_code` junto com o body.
- Implementar execução de extractors entre steps.
- Responder 404 em `PUT`/`DELETE /steps/{id}` para step inexistente e validar a suite em `/suites/{id}/steps`.
- Criar migration com as constraints de domínio para bancos que aplicaram a versão antiga de `001_init.sql`.

---

## 9. Possíveis evoluções futuras

- Paralelismo controlado
- Variáveis entre steps
- Extractors JSONPath (ainda não executados pelo runner)
- Thresholds de métricas
- Exportação de runs
- UI de visualização
- Armazenamento externo para bodies grandes
