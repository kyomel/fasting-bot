# Repository Guidelines

## Project Overview

Go WhatsApp bot for intermittent-fasting tracking (module `fasting-bot`).
Users register via `/daftar`, start fasts (`/puasa`, presets like `/if-168`,
`/omad`, `/water-48`), log breaks (`/buka`), and get automatic start/end
notifications plus stats, streaks, badges, leaderboard. A small REST API
(`POST /api/v1/users/register`, `GET /healthz`) shares the same usecase layer.
All user-facing bot text is Indonesian with emoji + `*bold*` WhatsApp markup.

## Architecture & Data Flow

Clean Architecture, dependency direction strictly:

```ts
delivery/whatsapp + delivery/http
  -> usecase.FastingUsecase (single facade, ~21 methods)
    -> repository interfaces (User, Schedule, Notification, Badge)
      -> domain (zero external deps)
```

- Composition root: `cmd/fasting-bot/main.go` wires everything by hand
  (constructor injection, no DI framework): `godotenv.Load` -> `config.Load()`
  -> `database.New()` -> `newRepositories()` -> `NewFastingUsecase(...)` ->
  WhatsApp client + `NewCommandHandler` -> `NewScheduler` -> `startAPIServer`.
- Inbound WhatsApp: group message -> `isAuthorized()` group-JID gate ->
  `processCommand()` switch on leading token -> usecase method -> repository SQL.
- Inbound HTTP: JSON body -> `delivery/http/server.go` handler -> same
  `FastingUsecase` (e.g. `RegisterUserAPI`).
- Outbound/proactive: cron `Scheduler` queries repositories for
  `NotificationTarget`s -> `Notifier` port (`internal/usecase/notifier.go`,
  implemented by `internal/infrastructure/whatsapp/notifier.go`) ->
  `whatsmeow` send.
- Database: PostgreSQL only (`DB_CONNECTION` required, `database.New()`
  fails fast when empty). Schema via embedded goose migrations
  (`migrations/postgres/00001..00009`, applied at startup). Shared streak/date
  helpers live in `persistence/schedule_shared.go`. SQLite remains only for
  the WhatsApp session store (`infrastructure/whatsapp/client.go` via
  whatsmeow `sqlstore`), never for application data.

## Key Directories

- `cmd/fasting-bot/` - sole entry point (`main.go`), composition root.
- `internal/domain/` - pure entities (`entities.go`), branded `ID` (`ids.go`),
  fasting catalogue (`fasting_types.go`), metabolic phases, motivation pools,
  badge rules. No external imports.
- `internal/repository/` - persistence contracts (`interfaces.go`) + sentinel
  errors (`errors.go`: `ErrNotFound`, `ErrConflict`).
- `internal/usecase/` - business logic, one file per concern: `user.go`,
  `register.go` (API registration), `password.go` (bcrypt),
  `schedule.go`, `record.go`, `stats.go`, `phases.go`, `motivation.go`,
  `badge.go`, `notifier.go` (outbound port).
- `internal/infrastructure/database/` - `postgres.go` (`DB` type, `New()` +
  `NewPostgres()`, pgx pool, embedded goose migrations in
  `migrations/postgres/`).
- `internal/infrastructure/persistence/` - PostgreSQL repo implementations
  (`*_postgres.go`); `schedule_shared.go` (streak/date helpers);
  `user_constraint.go` maps pg `23505` to `ErrConflict`.
- `internal/delivery/whatsapp/` - `command_handler.go` (command switch),
  `scheduler.go` (cron jobs + message builders).
- `internal/delivery/http/` - `server.go` (REST routes, status-code mapping).
- `migrations/postgres/` - root copy of goose migrations `00001..00009`
  (mirrors the embedded set; keep both in sync when adding one).
- `deploy/` - `fasting-bot.service` (systemd), `monitor.sh`
  (healthcheck/backup/restore/reset-session).

## Development Commands

```bash
make setup   # cp .env.example .env if missing + go mod download
make run     # setup + go run ./cmd/fasting-bot
make build   # go build -o bin/fasting-bot ./cmd/fasting-bot
make test    # go test ./...
make race    # go test -race ./...
make tidy    # go mod tidy
```

Notes: `build-linux` cross-compiles with
`CGO_ENABLED=1 GOOS=linux GOARCH=amd64` (CGO required for
`mattn/go-sqlite3`, used by the WhatsApp session store). No linter configured; CI (`.github/workflows/deploy.yml`,
push to `main`) builds the Linux binary and deploys via SCP/SSH + systemd
restart - it does **not** run tests, so run `make test` locally before push.

## Code Conventions & Common Patterns

- Formatting: `gofmt`-clean (no config, no linter). Verify with
  `gofmt -l internal/ cmd/` before yielding.
- Naming: `domain.ID` branded UUID string for all entity IDs (UUIDv7 via
  `domain.NewID()`; stored as native Postgres UUID).
- Error handling: sentinel errors, checked with `errors.Is` -
  `repository.ErrNotFound`, `repository.ErrConflict`,
  `usecase.ErrValidation`. Repos never leak `sql.ErrNoRows`; HTTP maps
  `ErrValidation` -> 400, `ErrConflict` -> 409. Wrap with `%w`, prefix
  Indonesian messages (`gagal ...: %w`).
- Vertical slice for new features: domain entity -> repo interface method ->
  Postgres repo impl -> usecase method -> delivery handler case.
  Update the interface fakes in tests when widening an interface.
- Repos use prepared statements held on the struct
  (`findByPhoneStmt`, `createStmt`, ...). Postgres `$1` placeholders with
  `RETURNING` clauses for generated IDs.
- Time: always `config.Location` (default `Asia/Jakarta`). Layouts in
  `fasting_usecase.go`: store `2006-01-02 15:04`, display `02-01-2006 15:04`,
  input `DD-MM-YYYY HH:MM`. `users.updated_at` is maintained by a Postgres
  trigger (`00009`) - never set it from Go.
- Passwords: bcrypt via `usecase.HashPassword` / `CheckPassword`
  (`golang.org/x/crypto/bcrypt`, `DefaultCost`). Store hash only; response
  DTOs (`RegisterResult`) exclude it.
- Phone contract: `+`-prefixed digits, leading `0` -> `62`
  (see `normalizePhone` in `command_handler.go`, mirrored by
  `normalizeRegisterPhone` in `register.go` - keep both in sync).
- Constraints: `/puasa` 1-168h, `/puasa-dry` max 48h; streak +1 on on-time
  `/buka`, reset after 24h idle.

## Important Files

| Path | Role |
|---|---|
| `cmd/fasting-bot/main.go` | Entry point + wiring (Postgres repos, usecase, WA, scheduler, API) |
| `internal/config/config.go` | Env config + defaults; `SecureFilePath`; timezone loading |
| `internal/usecase/fasting_usecase.go` | `FastingUsecase` interface + shared time/format helpers |
| `internal/delivery/whatsapp/command_handler.go` | Command switch; add `case` + handler + `/bantuan` line for new commands |
| `internal/delivery/whatsapp/scheduler.go` | Cron cadences (1-min start/end, 5-min motivation, 3-day cleanup, daily 15:00 group update, 4-h streak reset) |
| `internal/delivery/http/server.go` | REST routes + error->status mapping |
| `internal/repository/interfaces.go` | Persistence contracts (depend on domain only) |
| `migrations/postgres/00001_create_users.sql` | Canonical `users` schema |
| `.env.example` / `internal/config/config.go` | Env keys: `BOT_NUMBER`, `ADMIN_NUMBER`, `ALLOWED_GROUP_JID`, `GROUP_NAME`, `DB_CONNECTION` (required), `SESSION_PATH` (WA session SQLite file), `APP_TIMEZONE`, `API_ADDR` (`:8080`), `QR_CODE_PATH/HOST` |
| `README.md` | Command reference + setup/troubleshooting |
| `DEPLOY.md`, `deploy/` | CI/CD, systemd, backup/restore ops |

## Runtime/Tooling Preferences

- Runtime: Go (see `go 1.25.7` directive in `go.mod`; CI pins its own
  version in `.github/workflows/deploy.yml`). Package manager: native Go
  modules (`go.mod`/`go.sum`, no vendoring).
- CGO must stay enabled (go-sqlite3 for the WhatsApp session store). Keep the dependency set boring:
  stdlib + `net/http` ServeMux routing, `pgx`, `whatsmeow`, `goose`, `cron`.
- Config is env-driven (`internal/config` package vars, `.env` via godotenv
  locally, systemd `EnvironmentFile` in prod). Never hardcode paths/numbers.

## Testing & QA

- Framework: stdlib `testing` only, no assertion libs. Table-driven tests
  and hand-rolled fake repos (in-memory map + mutex, e.g. `registerFakeRepo`,
  `motivationUserRepo`) are the established pattern - follow it.
- Layout: `internal/domain/*_test.go` (pure logic), `internal/usecase/*_test.go`
  (fakes), `command_handler_test.go` (no DB), `schedule_logic_test.go`
  (pure streak rules in the `persistence` package, no DB).
- Postgres integration tests (`*_postgres_test.go`) run against
  a real DB via `database.MigratePostgres` + `TRUNCATE ... CASCADE` reset,
  gated on `TEST_DATABASE_URL` (skipped when unset):
  `TEST_DATABASE_URL=postgres://... go test ./internal/infrastructure/persistence/`.
- Coverage: no enforced threshold; add one focused test per non-trivial
  behavior (validation branches, duplicates, boundary values). Trivial
  one-liners need no test.
