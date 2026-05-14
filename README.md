# zxMail Production Ready v2

zxMail is a self-hosted transactional email control plane built around Postal. This repository now extends the stable Production v1 foundation into Production Ready v2 with gateway-agnostic billing, subscription management, usage metering, deliverability indicators, retention controls, scheduled workers, and expanded admin/customer dashboards.

## Repository layout
- `backend/`: Go + Gin API scaffold, platform adapters, route placeholders, and SQL migrations.
- `frontend/`: Next.js + TypeScript + Tailwind dashboard scaffold for operator and customer flows.
- `infra/`: reverse proxy and Postal configuration templates for single-node Docker Compose deployments.
- `docs/`: architecture notes and the recommended implementation order.
- `docker-compose.yml`: Production Ready v2 topology for Postgres, Redis, backend, worker, frontend, and Caddy, with Postal documented as an external SMTP core deployment.
- `.env.example`: baseline environment variables for application and infrastructure wiring.

## Backend status
- Health endpoints are implemented at `/health`, `/health/live`, and `/health/ready`.
- `/ready` is available as a readiness alias for operational tooling.
- Production v1 route groups remain intact for auth, organizations, domains, credentials, logs, bounces, suppressions, admin, and Postal webhooks.
- Production Ready v2 route groups are available under `/api/v2` for plans, subscriptions, invoices, payments, usage, deliverability, alerts, retention, and advanced admin control.
- Initial migration creates Production v1 tables only: users, organizations, domains, SMTP credentials, send logs, bounces, suppressions, DNS checks, webhooks, and audit logs.
- Production Ready v2 migration adds plans, subscriptions, invoices, payments, usage records, quota events, deliverability snapshots, domain health checks, system alerts, and worker job runs.
- Postal integration layer lives in `backend/internal/postal` and currently exposes a real reachability check plus explicit placeholder operations for capabilities that still need confirmed Postal API wiring.

## Frontend status
- Landing page explains the current zxMail positioning without exposing unfinished Phase 3 features.
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
- Production Ready v2 routes now also exist for:
  - `/billing`
  - `/usage`
  - `/deliverability`
  - `/alerts`
  - `/settings`
  - `/admin/overview`
  - `/admin/organizations`
  - `/admin/organizations/[id]`
  - `/admin/billing`
  - `/admin/payments`
  - `/admin/invoices`
  - `/admin/usage`
  - `/admin/deliverability`
  - `/admin/domain-health`
  - `/admin/alerts`
  - `/admin/retention`
  - `/admin/audit-logs`
- Frontend includes HttpOnly cookie-based auth handling, API wrapper, guided domain onboarding wizard, credential creation modal, logs filters, message detail drawer, and responsive sidebar layout.

## Frontend UI
- Design direction is a zxMail-specific blend of business dashboard clarity and developer-first email tooling, without copying Brevo, Mailjet, or Resend branding.
- Shared frontend primitives now live around:
  - `frontend/components/layout/` for `AppShell`
  - `frontend/components/providers/` for auth and toast wiring
  - `frontend/components/shared/` for stable wrappers like button, modal, page hero, section card, status badge, and copy button
  - `frontend/components/ui/` for reusable dashboard primitives such as `PageHeader`, `MetricCard`, `DataTable`, `EmptyState`, `ErrorState`, `LoadingSkeleton`, `Drawer`, `Stepper`, `DNSRecordCard`, `CredentialSecretModal`, `MessageTimeline`, `QuotaUsageBar`, and `HealthStatusCard`
- The current redesign covers:
  - login
  - customer dashboard
  - domains list
  - domain onboarding wizard
  - credentials
  - logs
  - suppressions
  - admin overview
  - admin customers
  - admin logs
  - admin system
- Preview mode remains available only when `NEXT_PUBLIC_API_BASE_URL` is empty outside production. It is intended for UI inspection only, not fake success in production.
- `npm run typecheck` now runs `next typegen` first so a clean checkout can typecheck without requiring a prior build.

Frontend API notes:
- The redesign is wired to existing backend endpoints for login, me, domains, credentials, logs, suppressions, organizations, health, billing, usage, deliverability, alerts, retention, and admin system surfaces.
- Plan creation and richer quota override editing remain API-ready but are not fully exposed as browser forms yet.
- No Stripe, seed testing, warm-up, IP pool, or Phase 3 UI is included.

## Docker Compose topology
- Public entrypoint: `caddy`
- Internal-only services: `backend`, `worker`, `frontend`, `postgres`, `redis`
- PostgreSQL and Redis are not published to the host or internet.
- Frontend is exposed through Caddy on `/`.
- Backend is exposed through Caddy on `/api/*`, `/health*`, and `/webhooks/*`.
- PostgreSQL data uses a persistent named volume: `postgres_data`.
- Redis uses a persistent named volume: `redis_data`.

Postal note:
- Production Ready v2 still uses Postal as the SMTP core.
- This repository's main Compose stack does not run Postal directly.
- Deploy Postal separately on the same VPS or a dedicated mail host, then point `POSTAL_BASE_URL` and webhook settings back to zxMail.
- See `infra/postal/README.md` and `infra/postal/postal.example.yml`.

## Getting started
1. Copy `.env.example` to `.env` and replace every placeholder secret.
2. Set `FIRST_ADMIN_EMAIL` and `FIRST_ADMIN_PASSWORD` if you want the API to bootstrap the initial admin automatically on startup.
3. Set `DASHBOARD_HOST`, `API_HOST`, `FRONTEND_ORIGIN`, `COOKIE_DOMAIN`, `NEXT_PUBLIC_APP_NAME`, and `NEXT_PUBLIC_API_BASE_URL` to your public browser and API origins, for example `app.zxmail.site`, `api.zxmail.site`, `https://app.zxmail.site`, and `.zxmail.site`.
4. Review `LOGIN_MAX_FAILURES`, `LOGIN_FAILURE_WINDOW_MINUTES`, and `LOGIN_LOCKOUT_MINUTES` for your login throttling policy before going live.
5. Set your encryption keys:
   - `ENCRYPTION_KEY_ID` identifies the legacy single-key path.
   - `ENCRYPTION_KEY` is the legacy/plain migration key and can remain populated while old encrypted rows still exist.
   - `ENCRYPTION_KEYS` contains comma-separated `key_id:base64keymaterial` entries for the keyring used by new writes.
   - `ACTIVE_ENCRYPTION_KEY_ID` selects which key encrypts new or rotated SMTP credentials.
6. Set `POSTAL_BASE_URL`, `POSTAL_API_KEY`, and `POSTAL_WEBHOOK_SECRET` to your Postal deployment values.
7. Apply the SQL migrations using your preferred migration runner.
8. Start the stack with `docker compose up --build -d`.
9. Configure Postal separately and point its webhook to `https://app.zxmail.site/webhooks/postal/event`.

## Migration commands
Apply the baseline migration with `psql`:
```bash
psql "$DATABASE_URL" -f backend/migrations/000001_init.up.sql
psql "$DATABASE_URL" -f backend/migrations/000002_production_v2.up.sql
```

Rollback the baseline migration:
```bash
psql "$DATABASE_URL" -f backend/migrations/000001_init.down.sql
psql "$DATABASE_URL" -f backend/migrations/000002_production_v2.down.sql
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
psql $env:DATABASE_URL -f backend/migrations/000002_production_v2.up.sql
psql $env:DATABASE_URL -f backend/migrations/000002_production_v2.down.sql
```

## Backup and restore
Database backup example:
```bash
docker compose exec -T postgres pg_dump -U "$POSTGRES_USER" -d "$POSTGRES_DB" > backups/zxmail-$(date +%F).sql
```

Database restore example:
```bash
cat backups/zxmail-2026-05-14.sql | docker compose exec -T postgres psql -U "$POSTGRES_USER" -d "$POSTGRES_DB"
```

Notes:
- keep backups outside the application containers on durable storage
- do not delete invoices, payments, or audit logs as part of routine cleanup
- test restore on a staging database before relying on backups for production recovery

## Production Ready v2
- Billing and subscriptions:
  - plans and subscriptions are gateway-agnostic
  - supported providers today: `manual_bank_transfer`, `manual_qris`
  - customer plan changes still require admin approval
- Usage and quota:
  - PostgreSQL is the source of truth
  - Redis keeps fast advisory counters
  - accepted webhook events drive usage records
- Deliverability:
  - bounce, deferred, rejected, and delivered rates are shown as health indicators
  - no seed testing, inbox placement claim, or FBL ingestion is implemented
- Worker:
  - daily and monthly reset
  - subscription expiry check
  - deliverability snapshots and alerts
  - retention cleanup

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
- `app.zxmail.site` may sit behind Cloudflare proxy if you want CDN or WAF behavior for the dashboard.
- `api.zxmail.site` may also sit behind Cloudflare proxy if you want the API on a dedicated hostname through the same Caddy origin.
- `smtp.zxmail.site` must be `DNS only` in Cloudflare. Never proxy the SMTP hostname.
- PTR/rDNS at the VPS provider must point to `smtp.zxmail.site`.
- Postal must own SMTP ports `25`, `465`, and `587`.
- The dashboard/API stack can live behind Caddy on ports `80` and `443`, but SMTP traffic must terminate on Postal, not on Cloudflare proxy.
- `/health` is a liveness endpoint and should stay `200 OK` as long as the API process is serving HTTP.
- `/health/ready` and `/ready` are readiness endpoints and may return `503` when PostgreSQL or Redis are not ready yet.
- If readiness fails, inspect backend logs for wrapped ping errors such as `ping postgres host=...` or `ping redis addr=...`.
- Worker health is available on the worker service itself at `/health`.
- Keep `smtp.zxmail.site` as `DNS only` in Cloudflare and ensure PTR/rDNS maps back to it.
- Keep `CORS_ALLOW_ORIGINS`, `COOKIE_DOMAIN`, `POSTAL_WEBHOOK_SECRET`, and encryption keys synchronized across deploys.

## Frontend UI v2
- Customer pages added:
  - billing
  - usage
  - deliverability
  - alerts
  - settings
- Admin pages added:
  - overview
  - organizations and organization detail
  - billing
  - payments
  - invoices
  - usage
  - deliverability
  - domain health
  - alerts
  - retention
  - audit logs
- Reusable UI added:
  - `PlanCard`
  - `InvoiceTable`
  - `PaymentStatusBadge`
  - `SubscriptionStatusCard`
  - `UsageChart`
  - `QuotaProgress`
  - `DeliverabilityScoreCard`
  - `DomainHealthChecklist`
  - `AlertCenter`
  - `AdminActionPanel`
  - `RiskBadge`
  - `RetentionPolicyForm`
  - `AuditLogTable`

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

## Documentation map
- [docs/production-v2.md](docs/production-v2.md)
- [docs/billing-manual.md](docs/billing-manual.md)
- [docs/usage-quota.md](docs/usage-quota.md)
- [docs/deliverability-health.md](docs/deliverability-health.md)
- [docs/deployment.md](docs/deployment.md)
- [docs/operations-runbook.md](docs/operations-runbook.md)
- [docs/security.md](docs/security.md)

## Explicitly excluded in this scaffold
- Stripe
- Kubernetes and multi-node orchestration
- IP pool automation and seed testing
- DMARC aggregate parsing and FBL ingestion
- Vault and SSO
- Inbox placement claims or advanced deliverability workbench
