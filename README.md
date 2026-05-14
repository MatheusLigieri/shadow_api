# Response Guard (MVP)

API para executar sequências de requests HTTP e comparar respostas para detectar regressões.

## Documentation

- **Swagger UI:** http://localhost:8080/docs
- **OpenAPI Spec:** [openapi.yaml](openapi.yaml)
- **Postman Collection:** [postman_collection.json](postman_collection.json)

### Import Postman Collection

1. Open Postman
2. Click **Import** → **Upload Files**
3. Select `postman_collection.json`
4. Collection variables:
   - `baseUrl`: http://localhost:8080
   - `suiteId`, `stepId`, `runId`: Updated from responses

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
docker-compose up --build
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
- `GET /runs/{id}/diffs` - List diffs

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

