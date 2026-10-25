# portfolio-api

Go REST API for my personal site: CV, trips, career milestones and blog posts, backed by Postgres. The frontend lives in [`portfolio-web`](../portfolio-web).

## Architecture

```
cmd/
  server/        HTTP server entrypoint (runs migrations on boot)
  seedcv/        one-off CV import (JSON -> Postgres, single tx, count-verified)
internal/
  config/        env loading + validation
  entity/        business entities and transport-agnostic errors
  dto/           request/response shapes and entity <-> dto mapping
  repository/    Postgres (pgx) data access, plain SQL; interfaces + impls
  service/       business logic; depends only on repository interfaces
  controller/    HTTP handlers; depend only on service interfaces
  middleware/    security headers, CORS, rate limit, body limit, logging, recover
  response/      JSON envelope + error -> HTTP status mapping
  database/      pgx pool + embedded golang-migrate runner
  server/        dependency wiring, router, graceful shutdown
migrations/      golang-migrate SQL (embedded into the binary)
seed/cv.json     CV content, taken from the PDF CV
scripts/         export-cv-data.mjs: legacy src/data/*.ts -> JSON
```

Dependencies point inward: `controller -> service -> repository -> entity`.

## API (v1)

| Method | Path          | Auth   | Description                                                      |
|--------|---------------|--------|------------------------------------------------------------------|
| GET    | `/healthz`    | none   | Liveness/readiness (DB ping)                                     |
| GET    | `/api/v1/cv`  | none   | Experiences, projects, education and skills in one payload       |

Responses use the envelope `{ "response_code", "response_message", "data" }`.

## Local development

```bash
cp .env.example .env            # then set POSTGRES_PASSWORD / DATABASE_URL
docker compose up -d            # Postgres on 127.0.0.1:5433
go run ./cmd/seedcv             # import seed/cv.json
go run ./cmd/server             # http://localhost:8080
curl -s localhost:8080/api/v1/cv | jq
```

Reload the CV after editing `seed/cv.json`: `go run ./cmd/seedcv -replace`.

## Security defaults

- Exact-origin CORS allowlist (`CORS_ORIGIN`, `*` is rejected in production), no credentialed cookies.
- Security headers on every response (nosniff, frame DENY, CSP `default-src 'none'`, no-referrer, HSTS in production); `Cache-Control: no-store` unless an endpoint opts in.
- Per-IP rate limiting on public routes; forwarded IP headers are trusted only with `TRUST_PROXY=true`.
- Request body cap (`MAX_BODY_BYTES`), server read/write/idle timeouts, per-request timeout.
- Parameterized SQL only, DB `CHECK` constraints mirror service validation.
- Internal errors are logged server-side; clients get a generic message. Logs never include headers, query strings or bodies.
- Admin password is stored only as a bcrypt hash (`ADMIN_PASSWORD_BCRYPT_HASH`); config refuses a non-bcrypt value.
- Distroless, non-root container image; secrets come from the host (Fly secrets), never from git.
