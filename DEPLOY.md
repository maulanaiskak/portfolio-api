# Deploying

Production layout: **portfolio-api** on Fly.io (Go binary + Fly Postgres), **portfolio-web** on Vercel, images on Cloudflare R2. The API runs migrations on boot, so deploying is just shipping the binary.

## 1. Cloudflare R2 (images)

1. Cloudflare dashboard → R2 → **Create bucket** (e.g. `portfolio-images`).
2. Bucket → Settings → **Public access**: connect a custom domain (recommended, e.g. `images.yourdomain.com`) or enable the `r2.dev` URL. This is `R2_PUBLIC_BASE_URL`.
3. R2 → **Manage API tokens** → create a token with **Object Read & Write** scoped to *this bucket only*. Note the Access Key ID, Secret Access Key, and the S3 endpoint `https://<account-id>.r2.cloudflarestorage.com`.

## 2. Fly.io (API + Postgres)

```bash
brew install flyctl
fly auth login
```

```bash
fly apps create <your-api-name>          # then set `app` in fly.toml to match
```

```bash
fly postgres create --name <your-api-name>-db --region sin
```

```bash
fly postgres attach <your-api-name>-db --app <your-api-name>   # sets DATABASE_URL
```

Generate the admin password hash locally (prompted, no echo):

```bash
go run ./cmd/hashpassword
```

Set secrets (values are encrypted by Fly; never commit them):

```bash
fly secrets set --app <your-api-name> \
  JWT_SECRET="$(openssl rand -base64 48)" \
  ADMIN_EMAIL="you@example.com" \
  ADMIN_PASSWORD_BCRYPT_HASH='$2a$12$...paste...' \
  CORS_ORIGIN="https://yourdomain.com,https://www.yourdomain.com" \
  R2_ENDPOINT="https://<account-id>.r2.cloudflarestorage.com" \
  R2_ACCESS_KEY_ID="..." \
  R2_SECRET_ACCESS_KEY="..." \
  R2_BUCKET="portfolio-images" \
  R2_PUBLIC_BASE_URL="https://images.yourdomain.com" \
  WEB_REVALIDATE_URL="https://yourdomain.com/api/revalidate" \
  REVALIDATE_SECRET="$(openssl rand -hex 32)"
```

Keep a copy of `REVALIDATE_SECRET`; the web app needs the same value.

Deploy and seed the CV once:

```bash
fly deploy
```

```bash
fly proxy 15432:5432 --app <your-api-name>-db
```

In another terminal, with `DATABASE_URL` pointing at `localhost:15432` (from `fly postgres connect` credentials):

```bash
JWT_SECRET=unused-but-required-32-bytes-minimum DATABASE_URL='postgres://...@localhost:15432/...' go run ./cmd/seedcv
```

Check it:

```bash
curl https://<your-api-name>.fly.dev/api/v1/cv
```

### CI/CD

`.github/workflows/deploy.yml` deploys on every push to `main` after tests pass. Create a deploy token and add it as a GitHub Actions secret named `FLY_API_TOKEN` (in a `production` environment):

```bash
fly tokens create deploy --app <your-api-name>
```

### Rotating credentials

- **Admin password**: generate a new hash, `fly secrets set ADMIN_PASSWORD_BCRYPT_HASH='...'`. The restart updates the row and invalidates every existing session token.
- **JWT secret**: `fly secrets set JWT_SECRET=...` logs everyone out.
- **R2 keys**: create a new token, set the secrets, then delete the old token in Cloudflare.

## 3. Vercel (web)

See `portfolio-web/README.md`. Required env vars there: `NEXT_PUBLIC_API_BASE_URL=https://<your-api-name>.fly.dev`, `NEXT_PUBLIC_IMAGE_ORIGINS=https://images.yourdomain.com`, `NEXT_PUBLIC_SITE_URL`, and `REVALIDATE_SECRET` (same as the API).

## 4. Smoke test after cutover

- `https://yourdomain.com` renders the CV from the API.
- `/admin` → sign in → create a draft post → it is **not** visible at `/blog`.
- Publish it → visible at `/blog` within seconds.
- Upload a photo to a trip → it is served from `R2_PUBLIC_BASE_URL`.
- Delete the test content.
