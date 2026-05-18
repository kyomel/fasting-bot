# fasting-bot

Starter Go project for a fasting bot. The current focus is a readable foundation that can run: Fiber serves health/API entry points, SQLite is opened and migrated, and a context-aware bot runner starts beside the API.

## Structure

```text
cmd/fasting-bot/       application entry point
internal/app/          dependency wiring and graceful shutdown
internal/bot/          long-running bot loop placeholder
internal/config/       environment loading and typed config
internal/database/     SQLite connection and starter migration
internal/http/         Fiber app, middleware, and routes
```

This follows common Go maintainability patterns: keep `main` small, put private application code under `internal`, isolate configuration/database setup, and make long-running workers stop through `context.Context`.

## Run locally

```sh
cp .env.example .env
go run ./cmd/fasting-bot
```

Then check:

```sh
curl http://localhost:3000/healthz
curl http://localhost:3000/readyz
curl http://localhost:3000/api/v1/
```

## Notes for the next plan

- Add fasting domain models and service methods under `internal/domain` and `internal/service`.
- Add SQLite repositories under `internal/repository/sqlite`.
- Keep Fiber handlers thin in `internal/http`; they should parse requests, call services, and return responses.
- Keep bot integrations in `internal/bot`; they should share services with the API instead of duplicating fasting logic.
