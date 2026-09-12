# fasting-bot × Clean Architecture — Codebase Evidence Map

Research date: 2026-09-06. Companion to `docs/plans/2026-09-06-clean-architecture-research.md`
(external primary sources). **This document maps the actual code** with `file:line` evidence.
No files were changed.

## Layers table

| Path | Role | Clean-arch layer |
|---|---|---|
| `cmd/fasting-bot/main.go` | Sole entry point + composition root; hand-wires godotenv→config→DB→repos→usecase→WA→scheduler→API | Frameworks & Drivers / composition root |
| `internal/config/config.go` | Env-driven config (vars, `SecureFilePath`, timezone) | Config (see V3 — globals) |
| `internal/domain/*` | Pure entities, branded `ID`, fasting catalogue, metabolic phases, motivation pools, badge rules | Entities |
| `internal/repository/{errors,interfaces}.go` | Persistence contracts + sentinel errors | Repository (consuming-side interfaces) |
| `internal/usecase/*` | Business logic: registration, schedule, record, stats, phases, motivation, badges | Use Cases |
| `internal/delivery/http/server.go` | REST handlers, error→status mapping, DTOs | Interface Adapters |
| `internal/delivery/whatsapp/command_handler.go` | WA command switch, parsing, send/receive | Interface Adapters |
| `internal/delivery/whatsapp/scheduler.go` | Cron jobs + message builders | Interface Adapters (bypasses usecase — V8) |
| `internal/infrastructure/database/{sqlite,postgres}.go` | DB connections + schema management | Frameworks & Drivers |
| `internal/infrastructure/persistence/*` | Repo implementations per driver + shared pure logic | Frameworks & Drivers (implements contracts) |
| `internal/infrastructure/whatsapp/{client,notifier}.go` | WhatsApp client adapter + `usecase.Notifier` impl | Frameworks & Drivers |

## Dependency-direction check

Verified positively:

- `domain` imports **no** internal package anywhere (`ids.go` uses only stdlib + `google/uuid` — see V1).
- `usecase` imports `delivery`/`infrastructure` in **no** file; outer imports are `internal/config`, `internal/domain`, `internal/repository`.
- `repository` imports only `internal/domain`.
- Delivery and infrastructure import inward only.
- Composition root is the only place wiring concrete impls — manual constructor injection, no DI framework (matches external consensus: bxcodec v4, manakuro registry).

So the **Dependency Rule holds** at the package-import level. The findings below concern what crosses the boundaries and which layer owns what.

## Findings (with evidence)

### V1. Innermost layer couples to a SQL driver type
`internal/domain/ids.go` imports `database/sql/driver` and `github.com/google/uuid`; `domain.ID` implements `sql.Scanner` and `driver.Valuer`. The innermost circle knows the DB driver interface. This is exactly the anti-example the sources call out (Fiber recipe's `entities.Book` carrying `bson`/`primitive.ObjectID` tags — external-ref §2.8).

### V2. Presentation strings live inside business logic (largest structural deviation)
`FastingUsecase` facade (~20 methods) all return `(string, error)` — WhatsApp/HTTP formatting (emoji, `*bold*`, Indonesian copy) composed in the usecase layer:
- `record.go` — completion/cancellation message with badge unlocks, refeed tips.
- `stats.go` — leaderboard + stats message.
- `schedule.go` — schedule-saved confirmation + teaser.
- `user.go` — registration welcome text.

Per external-ref §2.2/§2.5, presenters belong in delivery; usecase should return data. Net effect: business logic is coupled to the WhatsApp channel.

### V3. Config is global mutable state, loaded twice, un-injected
`internal/config/config.go` declares package-level `var`s; `init()` calls `Load()`, **and** `cmd/fasting-bot/main.go` calls `config.Load()` again. Nothing is injected; `usecase` reads `config.Location` directly; delivery reads `config.AllowedGroupJID`/`config.GroupName`. Violates checklist items #6/#7 (composition root injects; config injected). Also makes the package hard to unit-test and introduces init-ordering hazards.

### V4. Ignored `db.Prepare` errors → nil-`*sql.Stmt` panic risk
Every repo constructor discards prepare errors with `_ =`:
- `user_repository_postgres.go` (6 statements), `schedule_repository_postgres.go`, `notification_repository_postgres.go`, and all three SQLite repos.

If a statement fails to compile at prepare time the field stays nil and the first call panics instead of returning a wrapped error.

### V5. Duplicated migration directories (split-brain)
Identical goose sets in **both** `internal/infrastructure/database/migrations/postgres/` and `migrations/postgres/` (00001..00009). Only the embedded copy is used (`postgres.go` `//go:embed`); the root mirror has nothing enforcing sync.

### V6. Two unrelated schema-maintenance mechanisms
Postgres gets versioned goose `.sql` migrations; SQLite gets a hand-rolled idempotent `migrate()` in Go (`sqlite.go`, `CREATE TABLE IF NOT EXISTS` + `addColumnIfMissing` via `PRAGMA table_info`). Same schema described two ways → drift risk.

### V7. `DATABASE_PATH` is dead config — SQLite fallback is effectively broken
`internal/config/config.go` declares `DatabasePath` but `Load()` never reads `DATABASE_PATH` from env; `.env.example` omits it too. `sqlite.go` calls `config.SecureFilePath(config.DatabasePath)` with a value that is always `""` → the error is discarded and `sql.Open` gets `file:` + empty path. README/DEPLOY.md/AGENTS.md all document `DATABASE_PATH` as the SQLite path; the code never loads it.

### V8. Scheduler bypasses the usecase layer
`scheduler.go` takes `repository.ScheduleRepository` + `repository.NotificationRepository` + `usecase.Notifier` directly — delivery reaches into repositories, skipping usecase (breaks the documented `delivery → usecase → repository`). It also re-implements proactive-motivation/streak business logic already present in `usecase/motivation.go`, duplicating cross-layer domain behavior.

### V9. Dead / speculative `queries.go` dialect abstraction
`internal/infrastructure/persistence/queries.go` defines a `dialect` enum + 9 helpers (`placeholders`, `boolLiteral`, `insertOrIgnore`, `intervalHours`, `intervalHoursParam`, `dateOf`, `minuteString`, `leftRight`, `charLength`) with **zero call sites**. Every SQLite/Postgres repo pair hand-rolled its dialect SQL inline. The abstraction built to DRY the dual-DB logic was never wired in; the duplication remains.

### V10. `domain.NewID()` appears unused
`internal/domain/ids.go` defines `NewID()` (UUIDv7). No persistence path calls it — Postgres gets IDs from `RETURNING id`, SQLite from `LastInsertId()` integer-as-text. Speculative/dead code.

### V11. Duplicated phone-normalization logic
`delivery/whatsapp/command_handler.go` `normalizePhone` and `usecase/register.go` `normalizeRegisterPhone` implement the same `0`→`62`/`+` logic. AGENTS.md explicitly warns "keep both in sync" — a cross-layer duplicate that can drift.

## Dual-DB strategy assessment

**Good:**
- Paired repos per driver returning the same contract (`repository/interfaces.go`) — the standard Go dual-impl approach.
- Shared **pure** logic in `schedule_shared.go` (streak/date rules) consumed by both drivers.
- Driver-agnostic sentinel errors: `ErrNotFound`/`ErrConflict` + `mapUserConstraintError` (`user_constraint.go`) maps pgconn `23505` and sqlite `ErrConstraint` to `ErrConflict`.
- Selection is a clean `newRepositories()` branch in the composition root on `DBConnection`.

**Weak:**
- The DRY abstraction shipped to unify dialects (`queries.go`) is unused; ~15 query methods stay hand-duplicated per driver.
- SQLite schema maintained in Go, Postgres in `.sql` — two sources of truth.
- `domain.ID` bridges integer-as-text (SQLite) vs UUID (Postgres) via a polymorphic `Scan` — latent correctness hazard.
- SQLite fallback can't use `DATABASE_PATH` (V7), so the fallback path is unverified end-to-end.

## Testability notes

**Strong:** usecase tests with hand-rolled fakes (`registerFakeRepo`, `motivation*Repo`); `command_handler_test.go` DB-free; SQLite repo tests on `:memory:`; Postgres integration tests gated on `TEST_DATABASE_URL` (skip when unset) using exported `MigratePostgres` + `TRUNCATE ... CASCADE`; pure domain tests.

**Gaps:**
- Zero tests for the cron `Scheduler` (the largest delivery component).
- Usecase tests assert presentation strings (emoji/substring) — they'd break on copy changes, not behavior (tied to V2).
- `config` package untestable (globals, V3); `queries.go` dead code untested (V9 — and shouldn't exist).
- `FastingUsecase` is a single ~20-method interface (god-interface) — hard to isolate one concern.

## Strengths

1. Dependency direction correct — `delivery → usecase → repository → domain`, no reverse imports.
2. Consumer-owned interfaces — `internal/repository/interfaces.go` + `usecase.Notifier` (declared in usecase, implemented by `infrastructure/whatsapp/notifier.go` with `var _ usecase.Notifier = (*Notifier)(nil)`).
3. Driver-agnostic error contracts — sentinels + `mapUserConstraintError`; repos never leak `sql.ErrNoRows`.
4. DTO discipline at the HTTP boundary — `registerRequest`/`errorResponse`, `DisallowUnknownFields`, `ErrValidation`→400 / `ErrConflict`→409, `RegisterResult` never carries `PasswordHash`.
5. Behavior-bearing domain — `EvaluateBadges`, `PhaseForElapsedHours`/`NextPhase`, `SmartNotificationPlanFor` are real rules, not strings.
6. Thin hand-wired composition root — no DI framework, no service locator.
7. SQL-injection hygiene — prepared statements + placeholders throughout both drivers.
8. Path hardening — `SecureFilePath` rejects `file:`, `?`, `#`, NUL, CR/LF; creates dirs `0700`.
9. Graceful timezone fallback — `loadLocation` → `time.Local` on bad `APP_TIMEZONE`.
10. Shared pure streak/date logic keeps semantics identical across dialects.