# Backend Commands

## Run
```bash
go run ./cmd/api
```

Set `FIRST_ADMIN_EMAIL` and `FIRST_ADMIN_PASSWORD` before running if you want the first admin user to be created automatically.
Login throttling is controlled by:
- `LOGIN_MAX_FAILURES`
- `LOGIN_FAILURE_WINDOW_MINUTES`
- `LOGIN_LOCKOUT_MINUTES`
Credential secret encryption is controlled by:
- `ENCRYPTION_KEY_ID`
- `ENCRYPTION_KEY`
- `ENCRYPTION_KEYS`
- `ACTIVE_ENCRYPTION_KEY_ID`

## Build
```bash
go build ./cmd/api
```

## Test
```bash
go test ./...
```

## Migrations
Apply:
```bash
psql "$DATABASE_URL" -f migrations/000001_init.up.sql
```

Rollback:
```bash
psql "$DATABASE_URL" -f migrations/000001_init.down.sql
```

For existing databases that were created before key rotation support:
```sql
ALTER TABLE smtp_credentials
ADD COLUMN IF NOT EXISTS password_key_id TEXT;
```

## Postal wiring
Postal integration code lives under `internal/postal`.

Environment variables used by the client:
```bash
POSTAL_BASE_URL=http://postal:5000
POSTAL_API_KEY=replace-me
POSTAL_WEBHOOK_SECRET=replace-me
SMTP_PUBLIC_HOST=smtp.zxmail.site
```

Current behavior:
- `HealthCheck` performs a real HTTP reachability probe against the configured Postal base URL.
- `CreateServerPlaceholder`, `CreateCredentialPlaceholder`, and `GetMessagePlaceholder` do not fake success.
- Each placeholder returns an explicit error until the exact Postal API endpoint and auth contract are confirmed for this deployment.

## Quota controls
- `PATCH /api/v1/admin/credentials/:id/quota` updates per-minute, daily, and monthly limits, and can reset the stored usage counters.
- `POST /api/v1/admin/credentials/:id/disable` disables a credential from admin controls.
- `POST /api/v1/admin/credentials/:id/enable` re-enables a credential from admin controls.

Production v1 limitation:
- Customers still submit mail directly to Postal.
- Redis-backed per-minute rate limiting and PostgreSQL daily or monthly counters drive dashboard status and post-send accounting, but they cannot fully block mail before send until zxMail adds an SMTP gateway in front of Postal.

## Auth throttling
- `POST /api/v1/auth/login` uses Redis-backed throttling per email plus client IP.
- Default behavior is 5 failed attempts within 10 minutes, then a 15-minute temporary lockout.
- Locked requests return `429 Too Many Requests` with a `Retry-After` header and keep the same generic login error posture regarding account existence.

## Encryption key rotation
- SMTP credential secrets now persist the `password_key_id` used during encryption.
- New and rotated secrets always use `ACTIVE_ENCRYPTION_KEY_ID`.
- Existing rows are not automatically re-encrypted.
- Keep legacy keys configured until all old secrets have been rotated or otherwise migrated.

## Make targets
```bash
make tidy
make build
make test
make check
make run
```

The API expects PostgreSQL and Redis to be reachable through the environment variables defined in the root `.env.example`.

## Health semantics
- `GET /health` and `GET /health/live` are liveness endpoints. They return `200 OK` when the HTTP server is alive.
- `GET /health/ready` performs PostgreSQL and Redis checks and may return `503 Service Unavailable` until dependencies are reachable.
- Startup and readiness failures now include wrapped connection details such as PostgreSQL host/db/user or Redis addr/db to make deployment debugging less blind.
