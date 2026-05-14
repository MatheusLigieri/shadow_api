# Response Guard — Spec simplificada

## 1. Objetivo

API em Golang para executar uma sequência de requests HTTP e comparar respostas para detectar regressões.

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

---

## 4. Endpoints

### Suite
- `POST /suites`
- `GET /suites`
- `GET /suites/{id}`

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

### Binário
- Comparação por hash SHA256

---

## 6. Métricas armazenadas

Por snapshot:
- latency_ms
- bytes
- status_code

Comparação:
- delta_latency
- delta_bytes

Sem thresholds automáticos no MVP.

---

## 7. Estados de execução

- RUNNING
- WAITING_USER (apenas single)
- COMPLETED

---

## 8. TODO-list (MVP)

### Estrutura base
- Criar estrutura do projeto Go
- Loader de configuração YAML
- Interface de repositório
- Implementação SQLite
- Implementação Postgres
- Migrações

### Suite
- CRUD básico de suites
- CRUD de steps
- Ordenação por position

### Runner
- Executor sequencial de steps
- Cliente HTTP com timeout
- Captura de latency e bytes
- Persistência de snapshots

### Modos
- Execução Dual
- Execução Single (before)
- Endpoint continue (after)

### Diff
- Comparador JSON simples
- Comparador texto
- Comparador binário
- Persistência de diffs

### API
- Endpoints de run
- Consulta de status
- Listagem de diffs

### Infra
- Dockerfile
- docker-compose (api + postgres opcional)

---

## 9. Possíveis evoluções futuras

- Paralelismo controlado
- Variáveis entre steps
- Extractors JSONPath
- Thresholds de métricas
- Exportação de runs
- UI de visualização
- Armazenamento externo para bodies grandes
