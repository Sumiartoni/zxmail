# zxMail Architecture Notes

## Scope boundary
- Production v1 remains the stable base.
- Production Ready v2 extends that base incrementally with manual billing, usage metering, deliverability indicators, retention, and worker jobs.
- Single-node deployment still uses Docker Compose.
- Postal remains the SMTP core.
- Stripe, Kubernetes, multi-node orchestration, IP pool automation, seed testing, DMARC aggregate parsing, Vault, and SSO remain excluded.

## Components
- `frontend/`: Next.js app-router dashboard for operator and customer experiences.
- `backend/`: Go + Gin API, webhook receiver, health endpoints, and worker entry lanes.
- `postgres`: primary system of record for auth, orgs, domains, credentials, logs, bounces, and suppressions.
- `redis`: queue coordination, rate limiting, and quota counters.
- `postal`: SMTP core responsible for accepting and relaying mail.
- `caddy`: reverse proxy for dashboard and API entrypoints.

## Core flows
1. User authenticates through backend-issued JWT.
2. Organization member creates a domain and receives DNS records.
3. DNS verification worker polls records until domain status becomes verified.
4. Organization member creates SMTP credentials and sees the password exactly once.
5. Postal accepts SMTP traffic and posts lifecycle events to `/webhooks/postal/event`.
6. Backend persists send logs, bounce outcomes, suppression state, and quota counters.
7. Billing and payment approvals update subscription state, invoice state, and credential restriction state.
8. Worker jobs generate deliverability snapshots, alerts, and retention cleanup results.

## Data model direction
- Auth and authorization: `users`, `organizations`.
- Sending identity: `domains`, `dns_checks`, `smtp_credentials`.
- Delivery and safety: `send_logs`, `bounces`, `suppressions`, `webhooks`.
- Billing and usage: `plans`, `subscriptions`, `invoices`, `payments`, `usage_records`, `quota_events`.
- Governance and operations: `audit_logs`, `deliverability_snapshots`, `domain_health_checks`, `system_alerts`, `worker_job_runs`.
