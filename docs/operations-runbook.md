# Operations Runbook

Common tasks:
- approve payment: `/admin/payments`
- mark invoice paid or failed: `/admin/invoices`
- suspend customer: `/admin/organizations/[id]`
- run retention cleanup: `/admin/retention`
- recheck domain health: `/admin/domain-health`

Backups:
- database backup path should be configured outside the app, for example via `BACKUP_PATH`
- example backup command: `docker compose exec -T postgres pg_dump -U "$POSTGRES_USER" -d "$POSTGRES_DB" > backups/zxmail-$(date +%F).sql`
- example restore command: `cat backups/zxmail-YYYY-MM-DD.sql | docker compose exec -T postgres psql -U "$POSTGRES_USER" -d "$POSTGRES_DB"`
- do not delete invoices, payments, or audit logs during cleanup

Worker:
- starts from `go run ./cmd/worker`
- exposes `/health`
- runs subscription expiry checks, resets, snapshots, alerts, and retention cleanup on schedule
