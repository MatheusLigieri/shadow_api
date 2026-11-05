# Quick Start Guide - Shadow API

## 1. Start the API

**Local (SQLite):**
```bash
./setup.sh
go run ./cmd/api
```

**Docker (Postgres):**
```bash
docker compose up --build
```

API will be available at: http://localhost:8080

## 2. Access Documentation

Open in browser: http://localhost:8080/docs

## 3. Create Your First Test

### Step 1: Create a Suite
```bash
curl -X POST http://localhost:8080/suites \
  -H "Content-Type: application/json" \
  -d '{"name":"User API Test"}'
```

Response:
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "User API Test"
}
```

Save the `id` for next steps.

### Step 2: Add Steps to Suite
```bash
curl -X POST http://localhost:8080/suites/550e8400-e29b-41d4-a716-446655440000/steps \
  -H "Content-Type: application/json" \
  -d '{
    "position": 1,
    "method": "GET",
    "path": "/api/users",
    "headers_json": "{\"Authorization\":\"Bearer token123\"}",
    "body_text": ""
  }'
```

### Step 3: Run Dual Mode (A vs B)
```bash
curl -X POST http://localhost:8080/runs/dual \
  -H "Content-Type: application/json" \
  -d '{
    "suite_id": "550e8400-e29b-41d4-a716-446655440000",
    "base_a": "https://api-old.example.com",
    "base_b": "https://api-new.example.com"
  }'
```

### Step 4: Check Results
```bash
curl http://localhost:8080/runs/{run_id}/diffs
```

Response shows differences found:
```json
[
  {
    "id": "...",
    "run_id": "...",
    "step_id": "...",
    "passed": false,
    "diff_json": "[{\"path\":\"$.price\",\"before\":100,\"after\":120}]",
    "summary": "1 change(s)",
    "delta_latency_ms": -4,
    "delta_bytes": 12
  }
]
```

## 4. Single Mode (Before/After)

### Start (Capture Before)
```bash
curl -X POST http://localhost:8080/runs/single/start \
  -H "Content-Type: application/json" \
  -d '{
    "suite_id": "550e8400-e29b-41d4-a716-446655440000",
    "base": "https://api.example.com"
  }'
```

Run status will be `WAITING_USER`. Deploy your changes, then:

### Continue (Capture After)
```bash
curl -X POST http://localhost:8080/runs/{run_id}/continue \
  -H "Content-Type: application/json"
```

### Check Diffs
```bash
curl http://localhost:8080/runs/{run_id}/diffs
```

## 5. Import Postman Collection

1. Open Postman
2. **Import** → Select `shadow_api_collection.json`
3. Set `baseUrl`; the collection uses `suiteId`, `stepId`, and `runId` variables
4. Start testing!

## Common Use Cases

### Ignore Dynamic Fields
Edit `config.yaml`:
```yaml
runner:
  ignore_json_paths:
    - "$.timestamp"
    - "$.requestId"
    - "$.metadata.generatedAt"
```

### Increase Timeout
```yaml
runner:
  timeout_ms: 30000  # 30 seconds
```

### Limit Response Size
```yaml
app:
  max_body_bytes: 2097152  # 2MB
```

## Troubleshooting

**Migration errors:**
```bash
goose -dir ./migrations sqlite ./data/shadow.db status
goose -dir ./migrations sqlite ./data/shadow.db up
```

**Docker cleanup:**
```bash
docker compose down -v
docker compose up --build
```

**Check logs:**
```bash
docker compose logs -f api
```

## Current Implementation Notes

- `go test ./...` passes.
- Docker/Postgres, Swagger UI, and the suite → step → DUAL/SINGLE run → diffs flow last passed a smoke test on 2026-11-05. Start the stack with `docker compose up -d --build`.
- Each item from `GET /runs/{run_id}/diffs` also has `before` and `after` objects with the variant, status code, latency, bytes, hash, and full body of the compared responses.
- HTTP status codes are saved in snapshots but are not compared yet. `extractors_json` is stored but extractors are not executed.
