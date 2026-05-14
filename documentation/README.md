# Postman Collections - Shadow API

This directory contains organized Postman collections for all Shadow API endpoints.

## Structure

- `suites-endpoints.json` - Suite CRUD operations
- `steps-endpoints.json` - Step CRUD operations  
- `runs-endpoints.json` - Run execution and monitoring

## How to Import

### Individual Collection
1. Open Postman
2. Click **Import**
3. Select the desired JSON file
4. Start using!

### All Collections
Import each file in sequence to have complete API coverage.

## Environment Variables

After importing, update these variables in your environment:

```
baseUrl = http://localhost:8080
suiteId = (populated from responses)
stepId = (populated from responses)
runId = (populated from responses)
```

## Quick Links

- [Suites API](suites-endpoints.json)
- [Steps API](steps-endpoints.json)
- [Runs API](runs-endpoints.json)
