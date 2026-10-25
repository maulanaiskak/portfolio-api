# portfolio-api

Go REST API for my personal site: CV, trips, career milestones and blog posts, backed by Postgres. The frontend lives in [`portfolio-web`](../portfolio-web).

## Architecture

```
cmd/
  server/        HTTP server entrypoint (runs migrations on boot)
  seedcv/        one-off CV import (JSON -> Postgres, single tx, count-verified)
  hashpassword/  bcrypt hash for ADMIN_PASSWORD_BCRYPT_HASH (no-echo prompt)
internal/
  config/        env loading + validation
  entity/        business entities and transport-agnostic errors
  dto/           request/response shapes and entity <-> dto mapping
  repository/    Postgres (pgx) data access, plain SQL; interfaces + impls
  service/       business logic; depends only on repository interfaces
  controller/    HTTP handlers; depend only on service interfaces
  middleware/    auth (JWT), security headers, CORS, rate limit, body limit, request id, logging, recover
  security/      bcrypt + JWT (HS256) helpers
  response/      JSON envelope + error -> HTTP status mapping
  database/      pgx pool + embedded golang-migrate runner
  server/        dependency wiring, router, graceful shutdown
migrations/      golang-migrate SQL (embedded into the binary)
seed/cv.json     CV content, taken from the PDF CV
scripts/         export-cv-data.mjs: legacy src/data/*.ts -> JSON
```

Dependencies point inward: `controller -> service -> repository -> entity`.

## API (v1)

All JSON responses use the envelope `{ "response_code", "response_message", "data", "total_data"? }`.
`response_code` is `00` on success, `E0x` for business errors, `H0x` for transport errors.

**Public** (rate-limited per IP, `Cache-Control: public, max-age=60`)

| Method | Path | Description |
|---|---|---|
| GET | `/healthz` | Liveness/readiness (DB ping) |
| GET | `/api/v1/cv` | Experiences, projects, education, skills |
| GET | `/api/v1/trips?page=&limit=` | Published trips (list view, no body) |
| GET | `/api/v1/trips/{slug}` | Published trip with markdown body and photos |
| GET | `/api/v1/milestones?page=&limit=` | Published career milestones |
| GET | `/api/v1/milestones/{slug}` | Published milestone |
| GET | `/api/v1/posts?page=&limit=&tag=` | Published blog posts, optional tag filter |
| GET | `/api/v1/posts/tags` | Tags with counts |
| GET | `/api/v1/posts/{slug}` | Published post |
| POST | `/api/v1/auth/login` | `{email, password}` → `{token, expiresAt}` (5/min per IP) |

**Admin** (`Authorization: Bearer <token>`, `Cache-Control: no-store`)

| Method | Path | Description |
|---|---|---|
| GET | `/api/v1/admin/me` | Current admin |
| GET, PUT | `/api/v1/admin/cv` | Read / replace the whole CV (one transaction) |
| GET, POST | `/api/v1/admin/{trips,milestones,posts}` | List (any status, `?status=`) / create |
| GET, PUT, DELETE | `/api/v1/admin/{trips,milestones,posts}/{id}` | Read / update / delete |
| POST | `/api/v1/admin/trips/{id}/photos` | Add photo `{url, caption, sortOrder}` |
| PUT, DELETE | `/api/v1/admin/trips/{id}/photos/{photoId}` | Update caption/order, delete |
| POST | `/api/v1/admin/uploads` | `multipart/form-data` field `file` (JPEG/PNG/WebP) → `{url, width, height}` |

Slugs are generated from the title when omitted. Publishing sets `publishedAt` on first publish; drafts are never visible on public routes.

## Local development

```bash
cp .env.example .env            # set POSTGRES_PASSWORD, DATABASE_URL, JWT_SECRET
go run ./cmd/hashpassword       # paste into ADMIN_PASSWORD_BCRYPT_HASH='...' with ADMIN_EMAIL
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
- Admin password is stored only as a bcrypt hash (`ADMIN_PASSWORD_BCRYPT_HASH`); config refuses a non-bcrypt value. Single admin: changing `ADMIN_EMAIL` removes the old account.
- JWT: HS256 only (alg pinned), issuer/audience/expiry checked, ≥32-byte secret enforced. Tokens embed a password fingerprint, so rotating the password revokes every session.
- Login: same error and same bcrypt cost for unknown email vs wrong password (no account enumeration), 5 attempts/min per IP.
- Uploads: format detected from bytes (not filename/Content-Type), JPEG/PNG/WebP only, ≤40MP, re-encoded server-side, which strips EXIF/GPS metadata (orientation applied first) and caps size at 2560px. Keys are random UUIDs.
- Strict JSON decoding: unknown fields, trailing data and oversized bodies are rejected.
- Distroless, non-root container image; secrets come from the host (Fly secrets), never from git.
