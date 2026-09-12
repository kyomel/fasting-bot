# Implement authentication and authorization API

## Decision
- Reuse existing bcrypt registration path and enforce the requested minimum of 8 password characters for registration and login input.
- Use opaque 256-bit random bearer access tokens in `Authorization: Bearer`, with only SHA-256 token hashes persisted. Do not add JWT or a new dependency: this API has one service, no external identity provider, and server-side revocation is simpler and safer.
- Store sessions in PostgreSQL with created/expiry/revocation timestamps; default TTL 24 hours.
- Add a `role` column with `user` default and `admin` allow-list constraint. Enforce authenticated-user access to self-scoped endpoints and admin-only access to explicit admin endpoints. Deny by default through middleware.
- Keep login errors generic (`invalid username or password`) and perform dummy bcrypt verification when the username is missing, reducing enumeration/timing differences.

## Call graph
```text
POST /api/v1/auth/login
  -> HTTP decoder and boundary validation
    -> usecase.Login
      -> UserRepository.FindByUsername
      -> CheckPassword / dummy CheckPassword
      -> crypto/rand access token
      -> AuthSessionRepository.Create(hash(token))
      -> public user projection

protected endpoint
  -> bearerToken middleware
    -> usecase.Authenticate(hash(token))
      -> AuthSessionRepository.FindActiveByTokenHash
      -> UserRepository.FindByID
    -> handler-level role/object policy
      -> existing usecase operation

POST /api/v1/auth/logout
  -> bearerToken middleware
    -> usecase.Logout
      -> AuthSessionRepository.RevokeByTokenHash
```

## Files
- Add `internal/domain/auth.go` for role constants/session record.
- Extend `domain.User` and `RegisterResult` with role.
- Extend repository contracts with auth-session operations.
- Add Postgres auth-session repository and migration `00010` in both canonical embedded and mirror directories.
- Extend usecase facade with Login/Authenticate/Logout and session repository injection; update constructor callsites and test fakes.
- Add HTTP login/logout/me endpoints and bearer middleware. Keep registration public, `/healthz` public. Add one protected self-scoped endpoint (`GET /api/v1/users/me`) and one admin-only endpoint (`GET /api/v1/admin/users`) only if existing repository/usecase data makes it safe without introducing unrelated scope; otherwise authorization is enforced through auth middleware and role helper ready for future protected routes.
- Add config for session TTL only if needed; use a constant for the requested simple deployment unless runtime configuration is already established.
- Add focused tests for short login password, successful login, generic failure, token revocation, expired/revoked token denial, and role denial.
- Update `.env.example`/README API docs with endpoints and HTTPS/proxy requirement. Never store secrets.
- Write research note under `docs/plans/2026-09-13-api-auth-research.md` with Exa-backed official sources.

## Verification
- Run `gofmt` on changed Go files.
- Run focused usecase/HTTP tests, then `go test ./...` and `go build ./...` if available.
- If PostgreSQL integration environment is absent, report it and rely on unit tests plus migration inspection; do not invent a DB result.
