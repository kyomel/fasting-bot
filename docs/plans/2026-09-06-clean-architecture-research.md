# Go Clean Architecture — Primary-Source Research

Research date: 2026-09-06. Purpose: establish a grounded reference for judging and
guiding `fasting-bot`'s architecture against canonical Go clean-architecture practice.
All external claims cite their source URLs; internal mentions cite repo files.

## 1. Source list

| Source | URL | Authority |
|---|---|---|
| golang-standards/project-layout | https://github.com/golang-standards/project-layout | De-facto community standard Go repo layout; explicitly *not* an official Go standard, and not a clean-architecture spec |
| Robert C. Martin, "The Clean Architecture" | https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html | The canonical origin of the pattern (with Hexagonal/Onion antecedents) |
| bxcodec/go-clean-arch | https://github.com/bxcodec/go-clean-arch | The most-cited Go clean-arch reference implementation; v1 (2017) → v4 (2024) shows how the community's mapping of the pattern evolved |
| manakuro/golang-clean-architecture | https://github.com/manakuro/golang-clean-architecture | Widely-used small-to-medium Go example (Echo + MySQL); concrete `registry` = hand-wired composition root |
| Kittipat.po, "Structuring a Go Project with Clean Architecture" (DEV, 2025) | https://dev.to/kittipat1413/structuring-a-go-project-with-clean-architecture-a-practical-example-3b3f | Recent real-world walkthrough; `cmd/ + internal/{api,domain,infra,server,usecase}` layout with manual DI |
| bmf-san, "Clean Architecture in Go: A Practical Implementation Guide" (DEV) | https://dev.to/bmf_san/clean-architecture-in-go-a-practical-implementation-guide-1jko | Smallest practical mapping (8 dirs / 22 files); direct layer→directory table |
| OneUptime, "How to Implement Clean Architecture in Go" (2026) | https://oneuptime.com/blog/post/2026-01-07-go-clean-architecture/view | Ports & Adapters framing: `domain / usecase / port / adapter`; entities as behavior-bearing structs |
| Fiber docs, "Clean Architecture" recipe | https://docs.gofiber.io/recipes/clean-architecture/ | Official framework example of the pattern; also a useful *negative* example (framework-tagged entities) |
| 0xKiire, "The Complete Guide to Clean Architecture" (2026) | https://0xkiire.com/clean-architecture-golang-guide/ | Consolidates principles (DIP, Stable Abstractions, Screaming Architecture) with Go code |
| CyberAgent Developers Blog, "Practical Guide To Apply Clean Architecture In Go" (2025) | https://developers.cyberagent.co.jp/blog/archives/59647/ | How a large monorepo applies the layers "practically, not strictly"; introduces `go-arch-lint` to enforce boundaries |

Exa queries used: `golang clean architecture project structure best practice`,
`golang-standards project-layout internal cmd pkg`,
`go clean architecture repository usecase dependency rule maintainability`.

## 2. Per-source takeaways

### 2.1 golang-standards/project-layout (https://github.com/golang-standards/project-layout)

- Explicitly scoped down: *"This is `NOT an official standard defined by the core Go dev team`"*,
  and *"This is a basic layout ... it doesn't try to cover the project structure you'd have
  with something like Clean Architecture."* → layout conventions and clean architecture are
  orthogonal; the layout doc must not be mistaken for an architecture spec.
- Proportioned guidance for small projects: *"If you are trying to learn Go or if you are
  building a PoC or a simple project for yourself this project layout is an overkill. Start
  with something really simple instead (a single `main.go` file and `go.mod` is more than
  enough)."* — over-layout is itself a cited pitfall.
- `/cmd`: *"Main applications for this project."* — *"It's common to have a small `main`
  function that imports and invokes the code from the `/internal` and `/pkg` directories and
  nothing else."* → thin entry points; substance never lives in `cmd`.
- `/internal`: *"Private application and library code ... this layout pattern is enforced by
  the Go compiler itself."* — not limited to top level; can be one `internal` per tree level,
  optionally split as `internal/app` and `internal/pkg`. This is the only compiler-enforced
  privacy mechanism in Go.
- `/pkg`: *"Library code that's ok to use by external applications ... think twice before you
  put something here"*; also *"some in the Go community don't recommend it"*.
- `/src`: *"You really don't want your Go code or Go projects to look like Java"* — avoid.

### 2.2 The Clean Architecture — Robert C. Martin (https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)

- Goals (the pattern's definition of done): systems that are *independent of frameworks*,
  *testable*, *independent of UI*, *independent of database*, *independent of any external agency*.
- The Dependency Rule, verbatim: *"source code dependencies can only point inwards. Nothing
  in an inner circle can know anything at all about something in an outer circle."*
- Four schematic circles (explicitly not mandatory — *"There's no rule that says you must
  always have just these four"*):
  - **Entities** — *"Enterprise wide business rules"*; least likely to change with externals.
  - **Use Cases** — *"application specific business rules"*; orchestrate entities; isolated
    from database/UI/frameworks.
  - **Interface Adapters** — convert data between use-case form and external form; MVC
    controllers/presenters, and *"all the SQL should be restricted to this layer"*.
  - **Frameworks & Drivers** — *"The Web is a detail. The database is a detail."*
- Crossing boundaries via DIP: the use case depends on an *interface declared in its own
  circle* (an "output port") that an outer-layer adapter implements — source-dependency
  opposes flow of control.
- Data crossing boundaries: *"isolated, simple, data structures are passed across the
  boundaries. We don't want to cheat and pass Entities or Database rows."* — never pass a DB
  `RowStructure` inward; *"data ... is always in the form that is most convenient for the
  inner circle."* (the core authority for DTO/entity separation).

### 2.3 bxcodec/go-clean-arch (https://github.com/bxcodec/go-clean-arch)

- Quotes Uncle Bob's five rules; maps the pattern to four Go layers: **Models, Repository,
  Usecase, Delivery**.
- Version history is the community's evolution (changelog in README):
  - v3 (2020): *"Introducing Domain package"* — domain models pulled into their own package.
  - v4 (2024, master): *"Declare Interfaces to the consuming side"*, *"Introduce `internal`
    package"*, *"Introduce `Service-focused` package."* — interfaces now live on the side that
    *consumes* them, not in a shared crate.
- Master tree (GitHub API `git/trees/master?recursive=1`): `domain/` (article.go, author.go,
  errors.go), `article/` (service.go + tests), `internal/repository/mysql/` (driver impls +
  tests), `internal/rest/` (+ `middleware/`, `mocks/`), `internal/workers/`, `app/main.go`.
- Author's note: the *arrangement* of directories varies across v1–v4; *"the foundational
  concept will remain consistent"* — don't treat any single tree as dogma.

### 2.4 manakuro/golang-clean-architecture (https://github.com/manakuro/golang-clean-architecture)

- Tree (GitHub API): `cmd/app/main.go`, `cmd/seed/main.go`, `db/migrations/`,
  `docker/`, and `pkg/` containing the layers:
  - `pkg/domain/model/` — domain structs (user.go).
  - `pkg/usecase/usecase/` — business logic (user.go).
  - `pkg/usecase/repository/` — **repository interfaces declared at the consuming side** (db.go, user.go; tiny files ~126–200 B).
  - `pkg/adapter/controller/` and `pkg/adapter/repository/` — adapters implementing those interfaces.
  - `pkg/infrastructure/datastore/` + `pkg/infrastructure/router/` — driver/framework glue.
  - `pkg/registry/` — `registry.go` + `user.go`: a hand-written **composition root** (no DI framework).
  - `pkg/config/` — config.go + config.yml.
- Takeaway: for a small service the whole CA structure can be ~12 packages; interfaces live
  next to the consumer (usecase), adapters implement them, and one `registry` wires everything.

### 2.5 Kittipat.po, DEV practical example (https://dev.to/kittipat1413/structuring-a-go-project-with-clean-architecture-a-practical-example-3b3f)

- Layout: `cmd/` (CLI entry points), `db/migrations/`, `docs/`,
  `internal/api/http/{handler,middleware,route}`, `internal/config`,
  `internal/domain/{cache,entity,errs,repository}` (repository *interfaces* live in domain),
  `internal/infra/{db,redis}` (implementations + mocks), `internal/server/dependency.go`
  (manual DI), `internal/usecase/{healthcheck,...}`, `internal/util/httpresponse`, `pkg/`.
- Mirrors the current community consensus: interfaces in domain, impls in infra, one
  `dependency.go` composition root, handlers thin.

### 2.6 bmf-san, DEV practical guide (https://dev.to/bmf_san/clean-architecture-in-go-a-practical-implementation-guide-1jko)

- Explicit layer→directory table for a minimal CMS: Frameworks & Drivers → `infrastructure/`
  (env.go, logger.go, router.go, sqlhandler.go), Use Cases → `usecases/` (interactors +
  `*_repository.go` interfaces), Interface Adapters → `interfaces/` (controllers,
  repository impls), Entities → `domain/`; `database/migrations/`, `main.go`.
- Demonstrates the minimal footprint of CA in Go (8 dirs / 22 files); rationale given is
  framework-independence for a long-lived solo project. Note the legacy choice of a shared
  `interfaces/` package — the current consensus (bxcodec v4, manakuro, Go idiom "accept
  interfaces, return structs") puts interfaces on the consuming side instead.

### 2.7 OneUptime guide, Ports & Adapters (https://oneuptime.com/blog/post/2026-01-07-go-clean-architecture/view)

- Frames CA as Hexagonal/Ports-and-Adapters: layers Domain → Use Case → Interface (Ports)
  → Infrastructure (Adapters).
- Structure: `cmd/api/main.go`; `internal/domain/{entity,repository}`; `internal/usecase/`;
  `internal/adapter/{repository/{postgres,memory}, handler/{http,grpc}}`;
  `internal/port/{input,output}`; `pkg/errors`.
- Entities are behavior-bearing: `NewUser` factory with validation, `Validate()`,
  `Deactivate()`, `CanPerformAction()` and domain error sentinels — the antidote to
  anemic/DTO-only entities.

### 2.8 Fiber official recipe (https://docs.gofiber.io/recipes/clean-architecture/)

- Official framework example: `api/` (handlers, routes, presenters), `pkg/` (core logic +
  entities + `book.Service` interface), `cmd/` (entry point).
- Useful *negative* example: `entities.Book` carries `bson:"_id"` / `primitive.ObjectID`
  tags — the entity is coupled to MongoDB, i.e. the Dependency Rule is violated at the innermost
  layer. Good checklist probe: do domain types import driver/framework packages?

### 2.9 0xKiire complete guide (https://0xkiire.com/clean-architecture-golang-guide/)

- Restates the Dependency Rule; adds supporting principles:
  - **DIP**: `UserService` depends on `UserRepository` interface, not `*sql.DB`; SQL in
    business logic shown as the canonical anti-example.
  - **Stable Abstractions Principle**: interfaces live in the stable (domain) layer;
    volatile implementations (e.g. `StripePaymentGateway`) in infrastructure.
  - **Screaming Architecture**: name packages after the domain (`users/`, `orders/`) not the
    framework (`controllers/`, `models/`, `views/`, `routes/`).
- Dedicated pitfalls section: tight coupling to concrete persistence, framework-centric
  naming, and missing boundary enforcement.

### 2.10 CyberAgent / go-arch-lint (https://developers.cyberagent.co.jp/blog/archives/59647/)

- Large-monorepo practice: adopt CA *"in a simple and practical way, while still applying
  useful ideas"*; layers domain → usecase → interfaces → infrastructure.
- Repository layer defined as *"a boundary between your business logic and data sources"* —
  Go interfaces abstract data access so core logic stays independent of DB/API details.
- The dependency rule in practice: *"usecase (inner layer) can define a repository interface
  (outer layer) without knowing how it's implemented"; the PostgreSQL adapter lives in the
  outermost layer.*
- Introduces **go-arch-lint** to enforce layer boundaries automatically — boundaries that
  aren't enforced erode (cited maintainability driver).

## 3. Distilled findings

### 3.1 Canonical layer layout for a small-to-medium Go service

Synthesized across 2.1–2.10 (bxcodec v4 tree, manakuro tree, kittipat1413, oneuptime;
delivery naming follows fasting-bot AGENTS.md "Architecture & Data Flow"):

```
cmd/<app>/main.go            # thin entry point + composition root (manual wiring)
internal/
  domain/                    # entities, value types, sentinel errors — zero external imports
  repository/  (or usecase/) # persistence CONTRACTS — interfaces on the consuming side
  usecase/                   # application rules; orchestrates domain + repository ports
  delivery/http, whatsapp    # adapters: handlers, status-code mapping, DTOs
  infrastructure/{persistence,database,whatsapp}  # driver/framework implementations
  config/                    # env-driven config package
db/migrations/               # schema (outside Go)
pkg/                         # OPTIONAL, only genuinely shareable libs (rarely needed)
```

### 3.2 Dependency rule wording (authoritative)

> "Source code dependencies can only point inwards. Nothing in an inner circle can know
> anything at all about something in an outer circle." — *The Clean Architecture* (2.2)

In Go this reduces to: `domain` imports nothing external; `usecase` imports `domain` +
interfaces it declares; `delivery`/`infrastructure` import `usecase` and implement those
interfaces. Dependencies point from delivery/infrastructure → usecase → domain.

### 3.3 Where interfaces live

Current consensus (v4 of the canonical example, 2.3; manakuro, 2.4; Go idiom *"accept
interfaces, return structs"*): **on the consuming side** — `usecase` declares the
repository interface it needs; `delivery` declares the service interface it needs; adapters
in `infrastructure`/`persistence` implement them. A single shared `interfaces/` package
(2.6) is the older pattern and tends to aggregate god-interfaces. fasting-bot already does
this: `internal/repository/interfaces.go` is consumed by `usecase` (AGENTS.md "Key
Directories").

### 3.4 DTO vs entity handling

- Entities are the inner, stable business objects; they may carry behavior/validation
  (2.7, OneUptime `User.Validate()`).
- Across each boundary pass *"simple data structures ... in the form that is most convenient
  for the inner circle"*; **never** pass DB rows/ORM models inward, **never** leak framework
  types into entities (2.2, 2.8 negative example).
- DTOs belong in the delivery layer (HTTP request/response structs); fasting-bot follows
  this — `RegisterResult` excludes the password hash, repos never leak `sql.ErrNoRows`
  (AGENTS.md "Code Conventions").

### 3.5 Config / DI conventions

- Config is env-driven and lives in a `config` package injected into components
  (manakuro `pkg/config`, fasting-bot `internal/config/config.go`); never hardcode paths or
  numbers (AGENTS.md).
- Composition root wires everything **by hand** (constructor injection) — no DI framework.
  One file/package per app: `cmd/app/main.go` (2.3, 2.4), `internal/server/dependency.go`
  (2.5), `registry` package (2.4). fasting-bot already does this in
  `cmd/fasting-bot/main.go` (AGENTS.md "Architecture & Data Flow").

### 3.6 Common maintainability pitfalls cited by the sources

1. SQL / driver calls inside business logic (2.2, 2.9).
2. Passing DB row/ORM structs across an inner boundary (2.2).
3. Framework-typed entities (bson/primitive tags in domain) (2.8).
4. `/src` Java-style layout and fat `cmd` (2.1).
5. Applying the full layout to a PoC — layout/CA as over-engineering (2.1, 2.10).
6. A shared `interfaces/` package / god-interfaces instead of consumer-owned interfaces (2.3 vs 2.6).
7. Anemic entities reduced to plain DTOs (2.7).
8. Architecture erosion — no enforced or documented layer boundaries (2.10).
9. Framework-centric package naming (`controllers/`, `models/`, `routes/`) (2.9 "Screaming Architecture").

## 4. Checklist — judging a Go clean-architecture codebase (10 items)

1. **Dependency rule holds.** `domain` has zero external imports; `usecase` never imports
   `delivery` or `infrastructure`; every import points inward (2.2).
2. **Interfaces are consumer-owned.** Repository/service interfaces are declared where they
   are consumed (`usecase`/`domain`), implemented by adapters — no shared `interfaces/` god
   package (2.3, 2.4).
3. **Delivery layers have no business logic.** Handlers parse/validate/format and delegate to
   usecases; error→status mapping lives in delivery (2.5, 2.10; AGENTS.md HTTP mapping).
4. **Data crosses boundaries as plain structs.** No ORM rows, `*sql.*` types, or framework
   types leak inward; domain types carry no driver tags; DTOs live in delivery (2.2, 2.8).
5. **Domain is behavior-bearing.** Entities/rules + sentinel errors live in `domain`, not an
   anemic DTO folder (2.7, 2.9).
6. **Composition root is explicit and manual.** One entry point wires dependencies by
   constructor injection; no global state, no service locator, no DI framework (2.4, 2.5).
7. **Config is injected, env-driven.** `config` package loaded at startup and passed down;
   paths/numbers never hardcoded (2.4; AGENTS.md).
8. **Testability is actually realized.** Usecase tests run against fakes/mocks with no DB or
   server; domain tests are pure (2.2, 2.3 mockery, 2.4; fasting-bot `*_test.go` fakes per
   AGENTS.md).
9. **Layout serves the size.** `cmd` is thin, `/internal` used (compiler-enforced privacy),
   no `/src`, no speculative `/pkg`; structure is proportional to project size, not copied
   wholesale (2.1).
10. **Boundaries are documented or enforced.** A dependency-direction diagram exists (or a
    tool like `go-arch-lint` runs); package names scream the domain, not the framework (2.9, 2.10).

## 5. Note on fasting-bot

Per AGENTS.md ("Architecture & Data Flow" / "Key Directories"), fasting-bot already matches
most of the checklist: layering `delivery → usecase → repository → domain`, consumer-owned
`repository/interfaces.go`, a thin `cmd/fasting-bot/main.go` composition root, env-driven
`internal/config`, and pure `internal/domain` with no external imports. External references
above therefore mostly serve as *validation* of the existing shape and as ammunition for
keeping boundaries honest (e.g. resisting driver types in `domain`, resisting DI frameworks).