# zxMail Production v1

zxMail is a self-hosted transactional email control plane built around Postal. This repository is scaffolded specifically for Production v1, with the scope locked to auth, organizations, roles, domain onboarding, DNS verification, SMTP credentials, Postal integration, webhook ingestion, send logs, bounces, suppressions, quota enforcement, admin or customer dashboards, health endpoints, and Docker Compose deployment.

## Repository layout
- `backend/`: Go + Gin API scaffold, platform adapters, route placeholders, and SQL migrations.
- `frontend/`: Next.js + TypeScript + Tailwind dashboard scaffold for operator and customer flows.
- `infra/`: reverse proxy and Postal configuration templates for single-node Docker Compose deployments.
- `docs/`: architecture notes and the recommended implementation order.
- `docker-compose.yml`: Production v1 topology for Postgres, Redis, backend, frontend, and Caddy, with Postal documented as an external SMTP core deployment.
- `.env.example`: baseline environment variables for application and infrastructure wiring.

## Backend status
- Health endpoints are implemented at `/health`, `/health/live`, and `/health/ready`.
- Production v1 route groups exist for auth, organizations, domains, credentials, logs, bounces, suppressions, admin, and Postal webhooks.
- Initial migration creates Production v1 tables only: users, organizations, domains, SMTP credentials, send logs, bounces, suppressions, DNS checks, webhooks, and audit logs.
- Postal integration layer lives in `backend/internal/postal` and currently exposes a real reachability check plus explicit placeholder operations for capabilities that still need confirmed Postal API wiring.

## Frontend status
- Landing page explains the Production v1 scope boundary.
- Dashboard shell and Production v1 routes are implemented for:
  - `/login`
  - `/dashboard`
  - `/admin`
  - `/domains`
  - `/domains/new`
  - `/credentials`
  - `/logs`
  - `/suppressions`
  - `/admin/customers`
  - `/admin/logs`
  - `/admin/system`
- Frontend includes HttpOnly cookie-based auth handling, API wrapper, domain onboarding wizard, credential creation modal, logs filters, and responsive sidebar layout.

## Docker Compose topology
- Public entrypoint: `caddy`
- Internal-only services: `backend`, `frontend`, `postgres`, `redis`
- PostgreSQL and Redis are not published to the host or internet.
- Frontend is exposed through Caddy on `/`.
- Backend is exposed through Caddy on `/api/*`, `/health*`, and `/webhooks/*`.
- PostgreSQL data uses a persistent named volume: `postgres_data`.

Postal note:
- Production v1 still uses Postal as the SMTP core.
- This repository's main Compose stack does not run Postal directly.
- Deploy Postal separately on the same VPS or a dedicated mail host, then point `POSTAL_BASE_URL` and webhook settings back to zxMail.
- See `infra/postal/README.md` and `infra/postal/postal.example.yml`.

## Getting started
1. Copy `.env.example` to `.env` and replace every placeholder secret.
2. Set `FIRST_ADMIN_EMAIL` and `FIRST_ADMIN_PASSWORD` if you want the API to bootstrap the initial admin automatically on startup.
3. Set `FRONTEND_ORIGIN`, `COOKIE_DOMAIN`, `NEXT_PUBLIC_APP_NAME`, and `NEXT_PUBLIC_API_BASE_URL` to the public dashboard/browser origin, for example `https://dashboard.zxmail.site` and `.zxmail.site`.
4. Review `LOGIN_MAX_FAILURES`, `LOGIN_FAILURE_WINDOW_MINUTES`, and `LOGIN_LOCKOUT_MINUTES` for your login throttling policy before going live.
5. Set your encryption keys:
   - `ENCRYPTION_KEY_ID` identifies the legacy single-key path.
   - `ENCRYPTION_KEY` is the legacy/plain migration key and can remain populated while old encrypted rows still exist.
   - `ENCRYPTION_KEYS` contains comma-separated `key_id:base64keymaterial` entries for the keyring used by new writes.
   - `ACTIVE_ENCRYPTION_KEY_ID` selects which key encrypts new or rotated SMTP credentials.
6. Set `POSTAL_BASE_URL`, `POSTAL_API_KEY`, and `POSTAL_WEBHOOK_SECRET` to your Postal deployment values.
7. Start the stack with `docker compose up --build -d`.
8. Apply the SQL migrations using your preferred migration runner.
9. Configure Postal separately and point its webhook to `https://dashboard.zxmail.site/webhooks/postal/event`.

## Migration commands
Apply the baseline migration with `psql`:
```bash
psql "$DATABASE_URL" -f backend/migrations/000001_init.up.sql
```

Rollback the baseline migration:
```bash
psql "$DATABASE_URL" -f backend/migrations/000001_init.down.sql
```

If your database was created from an older baseline before key rotation support, add the new column without re-encrypting existing rows:
```sql
ALTER TABLE smtp_credentials
ADD COLUMN IF NOT EXISTS password_key_id TEXT;
```

On PowerShell, the same commands look like:
```powershell
psql $env:DATABASE_URL -f backend/migrations/000001_init.up.sql
psql $env:DATABASE_URL -f backend/migrations/000001_init.down.sql
```

## Postal integration
The backend reads these environment variables for Postal integration:
- `POSTAL_BASE_URL`
- `POSTAL_API_KEY`
- `POSTAL_WEBHOOK_SECRET`
- `SMTP_PUBLIC_HOST`

Current integration status:
- `healthCheck`: implemented as a real HTTP reachability probe to the configured Postal base URL.
- `createServerPlaceholder`: intentionally returns a manual-setup error until the exact Postal server creation contract is confirmed.
- `createCredentialPlaceholder`: intentionally returns an unsupported-operation error until credential provisioning is mapped to a verified Postal API flow.
- `getMessagePlaceholder`: intentionally returns an unsupported-operation error until message lookup is mapped to a verified Postal API flow.

This is deliberate. zxMail does not fake Postal success for operations that have not yet been wired to a confirmed Postal API endpoint.

## Deployment notes
- `dashboard.zxmail.site` may sit behind Cloudflare proxy if you want CDN or WAF behavior for the dashboard.
- `smtp.zxmail.site` must be `DNS only` in Cloudflare. Never proxy the SMTP hostname.
- PTR/rDNS at the VPS provider must point to `smtp.zxmail.site`.
- Postal must own SMTP ports `25`, `465`, and `587`.
- The dashboard/API stack can live behind Caddy on ports `80` and `443`, but SMTP traffic must terminate on Postal, not on Cloudflare proxy.
- `/health` is a liveness endpoint and should stay `200 OK` as long as the API process is serving HTTP.
- `/health/ready` is the readiness endpoint and may return `503` when PostgreSQL or Redis are not ready yet.
- If readiness fails, inspect backend logs for wrapped ping errors such as `ping postgres host=...` or `ping redis addr=...`.

## Quota and rate limiting
- PostgreSQL stores credential usage counters for `daily_used` and `monthly_used`, plus admin-managed limits for per-minute, daily, and monthly caps.
- Redis stores short-lived per-credential minute buckets so the dashboard can mark a credential as `limited` when the current minute cap has been reached.
- Postal webhook `accepted` events update PostgreSQL usage counters and the Redis advisory minute bucket.
- Admin controls are available at:
  - `PATCH /api/v1/admin/credentials/:id/quota`
  - `POST /api/v1/admin/credentials/:id/disable`
  - `POST /api/v1/admin/credentials/:id/enable`

Important limitation for Production v1:
- Customers send directly to Postal, not through a zxMail SMTP gateway.
- Because of that, quota and rate limiting are authoritative for dashboard state and post-send accounting, but pre-send enforcement remains limited until an SMTP gateway is added in front of Postal.

## Login protection
- Login throttling uses Redis and is keyed by the combination of normalized email and client IP address.
- Default policy:
  - `LOGIN_MAX_FAILURES=5`
  - `LOGIN_FAILURE_WINDOW_MINUTES=10`
  - `LOGIN_LOCKOUT_MINUTES=15`
- When the lockout is active, `POST /api/v1/auth/login` returns `429 Too Many Requests` and does not reveal whether the email exists.

## Encryption key rotation
- `smtp_credentials.password_key_id` stores which key encrypted each credential secret.
- New and rotated secrets always use `ACTIVE_ENCRYPTION_KEY_ID`.
- Decryption resolves the stored `password_key_id`; legacy rows without that value can still fall back to `ENCRYPTION_KEY_ID`.
- zxMail does not automatically re-encrypt all stored secrets during rotation.

Manual rotation flow:
1. Keep the old key available in config, either through `ENCRYPTION_KEY` plus `ENCRYPTION_KEY_ID` or by adding the old key to `ENCRYPTION_KEYS`.
2. Add the new base64-encoded key material to `ENCRYPTION_KEYS`.
3. Change `ACTIVE_ENCRYPTION_KEY_ID` to the new key id.
4. Restart the backend.
5. New credentials and rotated credentials will start using the new `password_key_id`.
6. Optional cleanup after validation:
   - backfill missing legacy ids without re-encrypting:
     ```sql
     UPDATE smtp_credentials
     SET password_key_id = 'legacy-v1'
     WHERE password_key_id IS NULL;
     ```
   - rotate individual credentials over time if you want all stored secrets to move to the new key.

## Recommended implementation order
1. Implement auth, password hashing, JWT issuance, and organization access checks.
2. Add repositories and services behind the scaffolded Go handlers.
3. Build the domain onboarding flow, DNS record generation, and Redis-backed verification jobs.
4. Implement SMTP credential issuance with Postal integration and encrypted secret storage.
5. Persist webhook events into send logs, bounces, suppressions, and credential quota usage.
6. Connect the Next.js dashboard to real backend APIs and enforce admin vs customer navigation.

## Explicitly excluded in this scaffold
- Billing and Stripe
- Kubernetes and multi-node orchestration
- IP pool automation and seed testing
- DMARC aggregate parsing
- Vault and SSO
- Advanced deliverability tooling
