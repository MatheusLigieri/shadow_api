# Postman Collections - Shadow API

## Unified Collection

Import [`../shadow_api_collection.json`](../shadow_api_collection.json) for the current collection. It groups all 14 requests into four folders: Infra, Suites, Steps, and Runs. The collection variables are `baseUrl`, `suiteId`, `stepId`, and `runId`.

The older files in this directory are endpoint-specific reference collections and are retained for compatibility:

- `suites-endpoints.json`
- `steps-endpoints.json`
- `runs-endpoints.json`

## Current Status

The API tests pass with `go test ./...`. A Docker/Postgres smoke flow last passed on 2026-11-05. Start the stack with `docker compose up -d --build` before exercising the requests.
