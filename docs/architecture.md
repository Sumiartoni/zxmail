# zxMail Production v1 Architecture

## Scope boundary
- Production v1 only.
- Single-node deployment using Docker Compose.
- Postal remains the SMTP core.
- No billing, Stripe, Kubernetes, multi-node, IP pool automation, seed testing, DMARC aggregate parsing, Vault, SSO, or advanced deliverability tooling.

## Components
- `frontend/`: Next.js app-router dashboard for operator and customer experiences.
- `backend/`: Go + Gin API, webhook receiver, health endpoints, and worker entry lanes.
- `postgres`: primary system of record for auth, orgs, domains, credentials, logs, bounces, and suppressions.
- `redis`: queue coordination, rate limiting, and quota counters.
- `postal`: SMTP core responsible for accepting and relaying mail.
- `caddy`: reverse proxy for dashboard and API entrypoints.

## Production v1 flows
1. User authenticates through backend-issued JWT.
2. Organization member creates a domain and receives DNS records.
3. DNS verification worker polls records until domain status becomes verified.
4. Organization member creates SMTP credentials and sees the password exactly once.
5. Postal accepts SMTP traffic and posts lifecycle events to `/webhooks/postal/event`.
6. Backend persists send logs, bounce outcomes, suppression state, and quota counters.

## Data model direction
- Auth and authorization: `users`, `organizations`, `organization_members`.
- Sending identity: `domains`, `domain_dns_records`, `smtp_credentials`.
- Delivery and safety: `send_logs`, `bounces`, `suppressions`, `webhook_events`.
- Governance and operations: `quota_counters`, `audit_logs`, `webhook_endpoints`.
