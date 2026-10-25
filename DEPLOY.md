# Deploying

Production layout: **portfolio-api** on Fly.io (Go binary), Postgres on Fly Managed Postgres (or Neon), **portfolio-web** on Vercel, images on Cloudflare R2. The API runs migrations on boot, so deploying is just shipping the binary.

Pick the two public names first; several settings reference them:

- **API**: `https://maulanaiskak-api.fly.dev` (the Fly app name is global, already set in `fly.toml`)
- **Web**: `https://<web-project>.vercel.app` or your own domain

## 1. Cloudflare R2 (images)

1. Cloudflare dashboard → R2 → **Create bucket** (e.g. `portfolio-images`, location hint Asia-Pacific).
2. Bucket → Settings → **Public access**: connect a custom domain on Cloudflare (recommended, e.g. `images.yourdomain.com`), or enable the `r2.dev` URL (rate-limited; fine to start). This is `R2_PUBLIC_BASE_URL`.
3. R2 → **Manage API tokens** → create a token with **Object Read & Write** scoped to *this bucket only*. Copy the Access Key ID, Secret Access Key, and the S3 endpoint `https://<account-id>.r2.cloudflarestorage.com`. The secret is shown once.

## 2. Fly.io (API)

```bash
brew install flyctl
```

```bash
fly auth login
```

```bash
fly apps create maulanaiskak-api
```

### Database

Fly's self-run `fly postgres` clusters are deprecated; use one of:

- **Fly Managed Postgres** (paid; same private network, backups handled):

  ```bash
  fly mpg create --name maulanaiskak-api-db --region sin
  ```

  ```bash
  fly mpg attach <cluster-id> --app maulanaiskak-api   # sets the DATABASE_URL secret
  ```

- **Neon** (free tier; region Singapore): create a project, copy the pooled connection string (it includes `sslmode=require`), then
  `fly secrets set --stage --app maulanaiskak-api DATABASE_URL='postgres://...'`.

### Secrets

Generate the admin password hash locally (prompted, no echo):

```bash
go run ./cmd/hashpassword
```

Stage the secrets (encrypted by Fly; applied on the first deploy; never commit them). Save the `REVALIDATE_SECRET` value, because Vercel needs the same one:

```bash
openssl rand -hex 32
```

```bash
fly secrets set --stage --app maulanaiskak-api \
  JWT_SECRET="$(openssl rand -base64 48)" \
  ADMIN_EMAIL="you@example.com" \
  ADMIN_PASSWORD_BCRYPT_HASH='$2a$12$...paste...' \
  CORS_ORIGIN="https://<web-project>.vercel.app" \
  WEB_REVALIDATE_URL="https://<web-project>.vercel.app/api/revalidate" \
  REVALIDATE_SECRET="...the value above..." \
  R2_ENDPOINT="https://<account-id>.r2.cloudflarestorage.com" \
  R2_ACCESS_KEY_ID="..." \
  R2_SECRET_ACCESS_KEY="..." \
  R2_BUCKET="portfolio-images" \
  R2_PUBLIC_BASE_URL="https://pub-xxxx.r2.dev"
```

### Deploy and seed the CV

```bash
fly deploy
```

```bash
curl https://maulanaiskak-api.fly.dev/healthz
```

Seed the CV through the admin API (prompts for the admin password; replaces any stored CV):

```bash
scripts/seed-remote.sh https://maulanaiskak-api.fly.dev you@example.com
```

### CI/CD

`.github/workflows/deploy.yml` deploys on every push to `main` after tests pass. Create a deploy token:

```bash
fly tokens create deploy --app maulanaiskak-api
```

Store it as an Actions secret named `FLY_API_TOKEN` in a GitHub environment called `production` (repo → Settings → Environments → New environment → `production` → Add secret).

### Rotating credentials

- **Admin password**: generate a new hash, `fly secrets set ADMIN_PASSWORD_BCRYPT_HASH='...'`. The restart updates the row and invalidates every existing session token.
- **JWT secret**: `fly secrets set JWT_SECRET=...` logs everyone out.
- **R2 keys**: create a new token, set the secrets, then delete the old token in Cloudflare.

## 3. Vercel (web)

Import the `portfolio-web` repository at vercel.com/new (framework Next.js, defaults are fine) and set, for Production and Preview:

| Variable | Value |
|---|---|
| `NEXT_PUBLIC_API_BASE_URL` | `https://maulanaiskak-api.fly.dev` |
| `NEXT_PUBLIC_IMAGE_ORIGINS` | same as `R2_PUBLIC_BASE_URL` |
| `NEXT_PUBLIC_SITE_URL` | `https://<web-project>.vercel.app` |
| `REVALIDATE_SECRET` | same as the API (mark **Sensitive**) |
| `NEXT_PUBLIC_FORMSPREE_ID` | optional, enables the contact form |

If the production URL differs from what you put in `CORS_ORIGIN` / `WEB_REVALIDATE_URL`, update those on Fly (`fly secrets set ...` restarts the API). When you add a custom domain later, update both secrets and `NEXT_PUBLIC_SITE_URL`, then redeploy the web app.

## 4. Smoke test after cutover

- The web URL renders the CV from the API.
- `/admin` → sign in → create a draft post → it is **not** visible at `/blog`.
- Publish it → visible at `/blog` within seconds.
- Upload a photo to a trip → it is served from `R2_PUBLIC_BASE_URL`.
- Delete the test content.
