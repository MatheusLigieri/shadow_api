# Response Guard (MVP)

API em Go para executar sequências de requests HTTP e comparar respostas para detectar regressões.

## Implementation Status

- Core MVP implemented: suites, steps, sequential DUAL and SINGLE runs, snapshots, body diffs, and persisted latency/size deltas.
- `go test ./...` passes on the current `chore/infra` branch.
- Docker/Postgres smoke flow (Swagger UI, health/readiness, DUAL and SINGLE runs, diffs) last passed on 2026-11-05. Start the stack with `docker compose up -d --build`.
- Known gaps: response HTTP status codes are stored but not included in diff comparisons; `extractors_json` is validated and persisted but not executed; `PUT`/`DELETE /steps/{id}` report success for a step that does not exist; Postgres databases that applied `001_init.sql` before its `CHECK` constraints were added do not have them. See [the status doc](context/planejamento-status.md).

## Documentation

- **Swagger UI:** http://localhost:8080/docs/
- **OpenAPI Spec:** [openapi.yaml](openapi.yaml)
- **Unified Postman Collection:** [shadow_api_collection.json](shadow_api_collection.json)
- **Project context:** [spec](context/spec.md), [current status](context/planejamento-status.md), [handoff checklist](context/todo.md)

### Import Postman Collection

Import `shadow_api_collection.json` into Postman. It contains Infra, Suites, Steps, and Runs folders. Set `baseUrl` to `http://localhost:8080`; `suiteId`, `stepId`, and `runId` are collection variables used by the requests.

## Quick Start

### Local Development (SQLite)

**Linux/macOS:**
```bash
chmod +x setup.sh
./setup.sh
go run ./cmd/api
```

**Windows:**
```cmd
setup.bat
go run ./cmd/api
```

### Docker Setup (with Postgres)

```bash
docker compose up -d --build
```

The API will be available at `http://localhost:8080`

Postgres will be running at `localhost:5432` with:
- Database: `shadow_api`
- User: `postgres`
- Password: `postgres`

## Configuration

- **Local:** `config.yaml` (uses SQLite at `./data/shadow.db`)
- **Docker:** Uses `config.docker.yaml` (uses Postgres)
- **Example:** `config.example.yaml`

## Migrations

### SQLite
```bash
goose -dir ./migrations sqlite ./data/shadow.db up
goose -dir ./migrations sqlite ./data/shadow.db down
```

### Postgres
```bash
goose -dir ./migrations postgres "postgres://user:pass@localhost/db?sslmode=disable" up
goose -dir ./migrations postgres "postgres://user:pass@localhost/db?sslmode=disable" down
```

## API Endpoints

### CORS
- CORS is fully open for testing: `Access-Control-Allow-Origin: *`, methods `GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS`, any header requested in `Access-Control-Request-Headers` is echoed back, and preflight `OPTIONS` returns `204`. Restrict this before exposing the service.

### Errors
- Error responses are JSON: `{"error": "...", "timestamp": "..."}`. Unknown routes return `404`; an unsupported method on a known route returns `405` with an `Allow` header; a request body whose `Content-Type` is not `application/json` returns `415`.

### Infra
- `GET /health` - Liveness check
- `GET /ready` - Readiness check (database)

### Suites
- `POST /suites` - Create suite
- `GET /suites` - List suites
- `GET /suites/{id}` - Get suite by ID

### Steps
- `POST /suites/{id}/steps` - Create step
- `GET /suites/{id}/steps` - List steps
- `PUT /steps/{id}` - Update step
- `DELETE /steps/{id}` - Delete step

### Runs
- `POST /runs/dual` - Start dual run (A vs B)
- `POST /runs/single/start` - Start single run (capture before)
- `POST /runs/{id}/continue` - Continue single run (capture after)
- `GET /runs/{id}` - Get run status
- `GET /runs/{id}/diffs` - List diffs, each with the full compared responses in `before` / `after` (variant, status code, latency, bytes, hash and the complete body as JSON)

## Validations

- **URL validation**: All URLs must be properly formatted (http/https)
- **HTTP Method validation**: Only GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS are allowed
- **JSON validation**: headers_json and extractors_json fields must contain valid JSON
- **Required fields**: suite_id, base, method, path are validated
- **Body size limit**: Responses are truncated to max_body_bytes (default 1MB)
- **Content-Type**: Requests must have `Content-Type: application/json` for POST/PUT operations

## Testing

```bash
go test ./...
```
