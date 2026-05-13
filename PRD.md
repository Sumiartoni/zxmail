# PRD

## 1. Overview
zxMail — layanan SMTP mandiri berbasis domain custom (zxmail.site) yang menjalankan Postal pada VPS dengan dedicated IP untuk menyajikan email transactional (OTP, reset password, invoice, notifikasi sistem) bagi produk SaaS. Tujuan Phase 1: luncurkan layanan SMTP self-hosted yang dapat memberikan kredensial SMTP per-customer, memastikan deliverability dasar (SPF / DKIM / DMARC PASS) dan observability dasar (logs + bounce handling). Tanpa ketergantungan pada ESP pihak ketiga.

Tujuan utama Phase 1:
- Deploy Postal di VPS dengan dedicated IP, Docker + Nginx/Caddy + Cloudflare.
- Self-service domain onboarding (DNS hints + verification).
- Generate SMTP credentials per customer; provide SMTP config.
- Capture send logs, bounces, basic quota enforcement.
- Dashboard minimal untuk admin & customer (create domain, credentials, logs).

Target pengguna: tim engineering/founder SaaS, agensi white‑label, organisasi dengan kebutuhan kontrol data.

## 2. Requirements

Functional
- Host Postal SMTP server on VPS with dedicated IP(s).
- Provide per-customer SMTP credentials (username/password or API key).
- Provide DNS record guidance (SPF, DKIM, DMARC, MX, rDNS) and verify propagation.
- Capture send events, deliveries, bounces, and basic metrics.
- Expose REST admin/customer API + web dashboard for credential and domain management.
- Enforce per-credential sending quotas / rate limits (configurable).
- Secure storage of credentials and logs (encrypted at rest where applicable).

Non-functional
- Availability: target 99.9% for SMTP & API.
- Latency: API responses <200ms typical.
- Scalability: support thousands of daily transactional messages per VPS initially; horizontal scale via more VPS + IPs later.
- Security: TLS for SMTP (STARTTLS / 465), HTTPS for APIs, RBAC for admin console, secret encryption.
- Compliance: allow self-hosting for data control; logging retention policy configurable.

Phase 1 Scope (in-scope)
- Postal deployment, DNS guidance and verification, SMTP credentials, logs, bounce handling, UI for onboarding and credentials, PostgreSQL persistence, basic quota enforcement.

Out of Scope (Phase 1)
- Billing and payments, multi-tenant billing plans, advanced deliverability tooling (seed lists, inbox placement analytics), automated IP warming orchestration (future roadmap).

## 3. Core Features

Priority MVP features (Phase 1)
1. Postal deployment template (Docker compose / Helm-like) with dedicated IP support.
2. Domain onboarding wizard: add domain → show required DNS records (SPF, DKIM selector, DMARC, MX) → verify propagation.
3. Generate SMTP credentials (apikey_xxx) per customer; display SMTP config:
   - Host: smtp.zxmail.site
   - Port: 587 (STARTTLS) / 465 (TLS)
   - Username: apikey_xxx
   - Password: secret_xxx
4. Send logs and events: accepted, delivered, bounced, rejected — searchable by message-id, recipient, domain.
5. Bounce handling pipeline: Postal webhooks → zxMail event processor → mark addresses as bounced / disabled.
6. Quota & rate limiting: per-credential daily/monthly limits + per-minute rate cap.
7. Admin dashboard: manage domains, credentials, view logs, view IP reputation basics.
8. Security: HTTPS, credential encryption, RBAC for admin.

Secondary (Phase 1+)
- rDNS guidance and operator checklist.
- Export logs and basic metrics CSV.
- Health checks and alerting hooks (webhook or email for admin).

## 4. User Flow

1. Provisioning zxMail (operator)
   - Provision VPS with dedicated IP (Hetzner/OVH).
   - Deploy Postal using Docker Compose + Nginx/Caddy reverse proxy, set Cloudflare DNS for dashboard & SMTP hostname.
   - Configure rDNS with provider to match sending hostname.

2. Customer onboarding
   - Customer registers / admin approves.
   - Customer adds sending domain in dashboard.
   - System shows required DNS records: SPF, DKIM (selector + public key), MX (optional), DMARC suggestion.
   - Customer adds DNS records at registrar/Cloudflare.
   - System polls DNS and marks domain verified when records propagate.

3. Credential issuance
   - Customer requests SMTP credentials for domain or app.
   - System generates apikey_xxx + secret, stores securely, shows SMTP connection string and sample SMTP config/code snippet.
   - Optional: customer can rotate/revoke credentials.

4. Sending email
   - Customer app connects to smtp.zxmail.site:587 using provided credentials.
   - Postal accepts messages and attempts delivery.
   - Postal emits event webhooks to zxMail (accepted, delivered, bounced, deferred).
   - zxMail stores event, updates logs, applies bounce handling, enforces quota.

5. Observability & handling
   - Customer checks logs, filters by message-id/recipient.
   - On bounce event, email address can be auto-disabled (configurable).
   - Admin monitors IP reputation signals and logs for manual action.

## 5. Architecture & Integrations

High-level components
- Postal (SMTP server) in Docker on VPS — mail delivery & inbound handling.
- Reverse proxy: Nginx or Caddy for HTTPS termination and routing (dashboard/API).
- Cloudflare: DNS + optional proxy for dashboard; SMTP hostname should be DNS-only (no CF proxy) to ensure correct MX/rDNS and deliverability.
- Backend service (Go — Gin/Fiber): REST API, worker pool for processing Postal webhooks, quota enforcement, DNS verification, credential management.
- PostgreSQL: persistent store.
- Workers: background processing for DNS polling, logs aggregation, webhook processing.
- Storage: local disk or object store for logs/attachments (Phase 1: local disk).
- Monitoring: Prometheus + Grafana (optional), alerting.

API DOCS: No external APIs required.
- zxMail does not require third-party ESP APIs. All integrations are internal (Postal, DNS via registrar UI by customer). Below are internal HTTP endpoints (REST) your zxMail backend exposes and the Postal webhook endpoints it consumes.

Internal REST API Endpoints (Backend)
- Authentication: token-based (JWT) for dashboard/API users.

1) POST /api/v1/auth/login
- Auth: none
- Request:
  - email: string
  - password: string
- Response:
  - token: jwt
  - user: { id, email, role }
- Notes: standard login for dashboard access.

2) POST /api/v1/domains
- Auth: Bearer token
- Request:
  - name: string (e.g., example.com)
  - purpose: enum [transactional] (optional)
- Response:
  - domain_id: uuid
  - dns_requirements: [{ type, name, value, note }]
- Notes: creates domain record and returns DNS record list to add (SPF, DKIM public key + selector, DMARC example, MX if needed).

3) GET /api/v1/domains/{domain_id}
- Auth: Bearer token
- Response:
  - id, name, verified: bool, dns_requirements, verification_status: [{record, found: bool, value_match: bool}]

4) POST /api/v1/domains/{domain_id}/verify
- Auth: Bearer token
- Request: none
- Response:
  - status: pending|verified|failed
- Notes: triggers DNS polling job to check records and DKIM selector/pkey.

5) POST /api/v1/credentials
- Auth: Bearer token
- Request:
  - domain_id: uuid
  - label: string (optional)
  - max_daily: integer (optional)
- Response:
  - credential_id: uuid
  - username: string (apikey_xxx)
  - password: string (plaintext only on creation)
  - smtp: { host: smtp.zxmail.site, port: 587, tls: starttls }
- Notes: stores hashed/ encrypted secret; show password once.

6) GET /api/v1/credentials/{credential_id}
- Auth: Bearer token
- Response:
  - id, username, domain_id, label, created_at, enabled, quota { daily_used, daily_limit }

7) POST /api/v1/credentials/{credential_id}/revoke
- Auth: Bearer token
- Request: none
- Response:
  - success: true

8) GET /api/v1/logs
- Auth: Bearer token
- Query params: domain_id, credential_id, message_id, recipient, status, from, to, limit, offset
- Response:
  - logs: [{ id, message_id, from, to, subject, status, timestamp, raw_event }]
- Notes: paginated.

9) GET /api/v1/metrics/domain/{domain_id}
- Auth: Bearer token
- Response:
  - stats: { delivered, bounced, deferred, rejected, accepted } over configurable window.

Postal Webhook Endpoint(s)
- Postal will POST events to zxMail to inform of message lifecycle.

A) POST /webhooks/postal/event
- Auth: webhook secret header X-Webhook-Signature (HMAC)
- Request payload (example):
  - event: string (accepted|delivered|bounced|deferred|rejected)
  - message: {
      id: string (Postal message id),
      message_id: string (original Message-ID),
      from: string,
      to: string,
      subject: string,
      size: int,
      credential: string (username used),
      domain: string
    }
  - reason: string (for bounces/rejects)
  - timestamp: ISO8601
- Response: 200 OK
- Processing:
  - Persist event to logs table.
  - Apply bounce handling: on bounce -> mark recipient as bounced and optionally disable credential or notify customer.
  - Update quotas.

Outbound SMTP connection info (for customers)
- Host: smtp.zxmail.site (A record to VPS dedicated IP, DO NOT proxy via Cloudflare)
- Ports:
  - 587 (STARTTLS) — recommended
  - 465 (TLS) — optional
- Auth:
  - Username: apikey_{credential_id}
  - Password: secret (shown once)
- Sample SMTP flow: AUTH PLAIN/LOGIN → MAIL FROM → RCPT TO → DATA

Notes about DNS & Cloudflare
- Dashboard should warn: SMTP host must be DNS-only (no Cloudflare proxy orange-cloud), DKIM public keys and SPF must be added at registrar/Cloudflare. rDNS must be configured at VPS provider to match sending hostname.

Security integrations
- TLS (Let's Encrypt via Caddy or certbot with Nginx)
- Secrets encrypted in PostgreSQL (application-level encryption for passwords).
- Webhook signatures for Postal → zxMail.

No external third-party ESP APIs are required or used in Phase 1.

## 6. Database Schema

Postgres (concise core tables)

Table: users
- id: uuid (PK)
- email: text (unique)
- password_hash: text
- role: enum('admin','customer')
- created_at: timestamptz
- last_login: timestamptz

Table: organizations
- id: uuid (PK)
- name: text
- owner_user_id: uuid -> users.id
- created_at: timestamptz

Table: domains
- id: uuid (PK)
- organization_id: uuid -> organizations.id
- name: text (example.com)
- verified: boolean (default false)
- dkim_selector: text
- dkim_public: text
- spf_record: text
- dmarc_record: text
- created_at: timestamptz
- verified_at: timestamptz

Table: smtp_credentials
- id: uuid (PK)
- organization_id: uuid
- domain_id: uuid (nullable)
- username: text (apikey_xxx, unique)
- password_enc: text (encrypted)
- label: text
- enabled: boolean
- created_at: timestamptz
- last_used_at: timestamptz
- quota_daily_limit: integer (nullable)
- quota_daily_used: integer (default 0)

Table: send_logs
- id: uuid (PK)
- domain_id: uuid
- credential_id: uuid
- postal_message_id: text
- message_id_header: text
- from_addr: text
- to_addr: text
- subject: text
- status: enum('accepted','delivered','bounced','deferred','rejected')
- raw_event: jsonb
- created_at: timestamptz

Table: bounces
- id: uuid (PK)
- recipient: text
- domain_id: uuid
- credential_id: uuid
- reason: text
- postal_message_id: text
- created_at: timestamptz
- disabled: boolean (auto-disable flag)

Table: ip_addresses
- id: uuid (PK)
- ip: inet
- assigned_to: text (e.g., VPS id)
- rDNS: text
- reputation_notes: text
- created_at: timestamptz

Table: dns_checks
- id: uuid
- domain_id: uuid
- record_type: text
- name: text
- expected_value: text
- found_value: text
- found: boolean
- checked_at: timestamptz

Table: webhooks
- id: uuid
- source: text (postal)
- secret: text (HMAC secret)
- created_at: timestamptz

Indexes: logs on message_id, to_addr, status, timestamp; credentials on username.

Retention & Archival
- send_logs retention default 90 days; optional archival to compressed storage/export.

## 7. Constraints

Operational constraints
- rDNS control: operator must set rDNS at VPS provider to match sending hostname — cannot be automated via DNS alone.
- Deliverability is not guaranteed; requires correct DNS, IP warming, sender reputation practices.
- Cloudflare: SMTP hostname must be DNS-only (no proxy). Mistakes cause delivery failures.
- IP warming: new dedicated IPs require gradual volume ramp-up to build reputation—must be operationally managed.
- DNS propagation delays: verification may take minutes to >24 hours across resolvers.
- Scale limits: single VPS + Postal suitable for small–medium transactional volume. For high volume, need horizontal scaling and multiple IPs.
- Legal / compliance: operator is responsible for compliance with spam laws (CAN-SPAM, GDPR, etc.) and content policies for hosted customers.
- No third-party deliverability mitigation: Phase 1 lacks features like seed list testing, ISP feedback loops automation (future).
- Security: secrets must be handled securely; show password only once at creation; rotate/revoke functionality required.
- Backup & DR: DB backups required; Postal mail queues must be included in backup strategy where needed.
- Monitoring & alerting: operator must configure host-level monitoring (disk, queue sizes, bounced rates).

Roadmap notes (beyond Phase 1)
- Add billing, multi-plan quotas, automated IP warming, advanced deliverability tooling (inbox placement), feedback loop integrations, dedicated customer dashboards per organization, multi-VPS orchestration.



---

Business Requirements Document (BRD) — Phase 2 (zxMail)
1. Purpose
- Extend zxMail Phase 1 into a multi-node, scalable transactional SMTP platform with advanced deliverability tooling, billing/subscription management, automated IP warm-up orchestration, ISP feedback loop processing, and stronger operational automation while retaining self-hosting, privacy and control goals from Phase 1.

2. Goals / Success Criteria
- Multi-VPS / multi-IP orchestration: manage multiple Postal instances (nodes) with an IP pool, automatic assignment & rotation.
- Automated IP warm-up engine: schedule staged ramp-up, automatically adjust per-IP and per-credential rate caps.
- Billing & subscription: tiered plans (free/dev, standard, pro) with quota enforcement, usage metering, and invoicing integration.
- Deliverability tooling: seed-list inbox placement testing, automated DMARC aggregate parsing, complaint (FBL) ingestion & processing, DKIM rotation helper, reputation signals dashboard.
- Enhanced observability: centralized logs (Loki/Elastic), metrics, tracing (OpenTelemetry), alerting, and SLO-based alerts.
- Operational automation: IaC for provisioning (Terraform), rDNS automation where provider API supports it (Hetzner/OVH), health-driven autoscaling of worker pools.
- Security & compliance: per-tenant KMS-backed secret management (Vault), MFA for admin, SSO (SAML/OIDC) optional, per-tenant data retention policies and export controls.
- Availability and performance: maintain 99.9% SMTP/API availability, scale to tens of millions monthly by horizontal expansion, API latencies <200ms under normal load.

3. Scope (in-scope)
- Multi-node Postal orchestration, IP pools, warm-up automation, billing & subscriptions, seed tests, FBL ingestion pipelines, DMARC aggregate parsing, DKIM rotation tooling, suppression lists, tenant-level retention policy UI, CSV export, advanced quotas (per-plan + per-credential), Redis/KV rate-limits, RabbitMQ/Kafka event bus, MinIO object store for exports.

4. Out-of-Scope (Phase 2)
- Full mailbox-level inbox placement analytics tied to ISPs beyond seed-list results, full managed deliverability service (human-driven outreach), hosted shared IP reputation guarantee, full GAAP accounting (accounting is simple invoices/payments).

5. Functional Requirements (high level)
- IP Pool Management: create/manage IP pools, tag IPs (warmup, cold, dedicated), assign IPs to Postal nodes.
- Warm-up Manager: schedule warm-up plan per IP/pool using historical throughput and adjustable schedule; auto-adjust per-minute and per-day limits; provide warm-up status and recommendations.
- Billing & Plans: create plans with quotas, rate caps, overages, trial periods; subscribe orgs; generate invoices, process payments (Stripe integration), record payments.
- FBL & Complaint Processing: ingest ARF/Complaint reports from partnered ISPs, normalize, mark recipients as complained and push to suppression list; offer complaint reason classification.
- Seed-list Testing: manage seed-lists, schedule sends via customer credentials, collect delivery/inbox results, provide pass/fail and simple placement metrics.
- DMARC/Aggregate & Forensic Parser: ingest aggregate DMARC reports (email/webhook), parse, surface sources of failure and ISP-level signals.
- DKIM Rotation Helper: generate new selectors, stage rotation (rollout + verification), and automate DNS hints if API token provided.
- Multi-VPS orchestration: node registration, health checks, auto-deploy Postal via templates (Helm/Docker Compose/k3s), and node draining for maintenance.
- Billing UI & Usage API: usage endpoints for plan metering, overage billing calculation, invoice retrieval, payment webhooks.
- Enhanced Admin Dashboard: IP reputation timeline, warm-up progress, complaint rates, plan utilization, and alerting hooks.
- Retention & Exports: configurable retention per-organization for send_logs and bounces with admin overrides; CSV/JSON exports stored to object store.
- Security: SSO/SAML or OIDC, MFA, RBAC expanded (super-admin, operator, billing-admin, organization-admin); Vault-backed secrets; webhook HMAC/SIG verification.

6. Non-functional Requirements
- Availability 99.9% for SMTP & API.
- API latency <200ms typical; SLOs for 95/99th percentile documented.
- Scale: support multi-million monthly sends via sharded nodes and IP pools.
- Security: TLS everywhere, secrets encryption with Vault/KMS, audit logging, per-tenant key scoping.
- Observability: full metrics + logs + traces, alerting to PagerDuty/Slack.

Detailed Tech Stack — Phase 2
1. Core services
- Backend API & workers: Go (1.21+) using Fiber or Gin for API; gRPC for internal node communication and warm-up orchestration. Rationale: high throughput, low latency, static binary deployment.
- Web UI: React + TypeScript + Vite; component library (Mantine or Chakra).
- Postal: continued use as MTA; run Postal inside containers on each node. Use proven Postal setup from Phase 1 and extend orchestration.
- Container orchestration: Kubernetes (k8s) for central deployments, or lightweight k3s for small operator installs; Helm charts for Postal + backend. For smaller operators, provide Terraform + Docker Compose templates.
- Event bus: RabbitMQ for task events + Kafka optional for large scale. Use RabbitMQ for transactional reliability and easier operational management.
- Rate-limiting and caching: Redis (6/7) for counters, distributed locks, token-bucket rate limiting.
- Persistent storage:
  - PostgreSQL (13+) for core relational data.
  - MinIO (S3-compatible) for exports, large logs, seed-test assets.
  - Local disk for Postal queues (backed by snapshots/backup policy).
- Secrets & KMS:
  - HashiCorp Vault for secret encryption & per-tenant keys; integrate with application via transit KMS.
  - Optional cloud KMS (AWS/GCP/Azure) mapping for hosted setups.
- Monitoring & Observability:
  - Prometheus for metrics; Grafana dashboards.
  - Loki for logs or ElasticSearch + Kibana for large setups.
  - OpenTelemetry + Jaeger for tracing.
  - Alertmanager + Opsgenie/PagerDuty/Slack hooks.
- CI/CD: GitHub Actions + Terraform Cloud; image registry (GitHub Container Registry).
- IaC: Terraform with cloud/Hetzner/OVH providers, Ansible for node bootstrap (when k8s not used).
- Backup: pgBackRest for Postgres, MinIO lifecycle policies, periodic Postal queue snapshot.
- Payment: Stripe for billing, webhooks for payment events.
- Sentry for error tracking.
- SMTP hostname & DNS: Cloudflare optional via API token integration to assist customers who allow DNS automation.

Operational integrations & automation
- DNS automation: Cloudflare API integration (optional, customer must grant limited token) to auto-insert DKIM/SPF/DMARC for verified domains.
- rDNS automation: use Hetzner/OVH APIs to set PTR records if operator has provider credentials.
- Node registration: nodes register to control plane via mTLS, receive config (IP assignments, rate limits).
- Autoscaling: k8s HPA for backend workers, custom controller for Postal node scaling via Terraform + provider APIs.

Security & Compliance
- All secrets encrypted in Vault; DB column-level encryption for password_enc with Vault/GCP KMS.
- Webhooks verified with HMAC; JWTs signed with rotating keys.
- Admin actions audited in audit_log table; SSO + MFA for higher privilege roles.

API Documentation — Phase 2 (Internal REST + webhook endpoints)
- Auth: Bearer JWT for API; mTLS for node-to-control-plane; header X-Request-ID for tracing.

Common error responses:
- 401 Unauthorized, 403 Forbidden, 422 Unprocessable Entity, 429 Too Many Requests, 500 Internal Server Error.
- All success responses use JSON with top-level "data" object unless listed otherwise.

1) POST /api/v2/auth/login
- Auth: none
- Request JSON:
  {
    "email": "user@example.com",
    "password": "string",
    "mfa_code": "string (optional)"
  }
- Response 200:
  {
    "token": "jwt_token",
    "user": { "id":"uuid", "email":"user@example.com", "role":"organization_admin", "org_id":"uuid" },
    "expires_at": "2026-XX-XXT..Z"
  }

2) POST /api/v2/auth/sso/authorize
- Start SSO; returns redirect URL
- Request:
  {
    "provider": "saml|oidc",
    "redirect_uri": "https://app.zxmail.site/oauth/callback"
  }
- Response:
  {
    "redirect": "https://idp.example/sso?..."
  }

3) POST /api/v2/domains
- Create domain; optionally attempt DNS automation if cloud token provided.
- Auth: Bearer
- Request:
  {
    "organization_id": "uuid",
    "name": "example.com",
    "purpose": "transactional",
    "auto_dns": { "provider": "cloudflare", "token_id": "vault-ref" } // optional: Vault ref to token
  }
- Response 201:
  {
    "domain_id":"uuid",
    "dns_requirements":[
      { "type":"TXT","name":"@","value":"v=spf1 mx -all","note":"Add SPF" },
      { "type":"TXT","name":"_dmarc","value":"v=DMARC1; p=none; rua=mailto:dmarc@zxmail.site","note":"DMARC aggregate" },
      { "type":"TXT","name":"zxmail._domainkey","value":"v=DKIM1; k=rsa; p=MIIBIjANB...","note":"DKIM selector zxmail" }
    ],
    "auto_dns_status": "pending|applied|failed"
  }

4) POST /api/v2/domains/{domain_id}/verify
- Trigger DNS verification job
- Auth: Bearer
- Request: {}
- Response:
  {
    "status":"pending",
    "job_id":"uuid"
  }

5) GET /api/v2/domains/{domain_id}
- Returns domain details incl DMARC/aggregate summary
- Response:
  {
    "id":"uuid",
    "name":"example.com",
    "verified":true,
    "dkim_selector":"zxmail",
    "dkim_public":"MIIB...",
    "spf_record":"v=spf1 mx -all",
    "dmarc_record":"v=DMARC1; p=none;",
    "dmarc_summary": { "failures": 12, "sources": [{"ip":"1.2.3.4","count":10}] },
    "dns_requirements":[...],
    "created_at":"iso",
    "verified_at":"iso"
  }

6) POST /api/v2/credentials
- Create SMTP credential with optional plan limits and node affinity
- Auth: Bearer
- Request:
  {
    "organization_id":"uuid",
    "domain_id":"uuid",
    "label":"service-worker",
    "quota": { "daily": 10000, "monthly": 300000 },
    "plan_id":"uuid (optional)",
    "assign_node_id":"uuid (optional)"
  }
- Response 201:
  {
    "credential_id":"uuid",
    "username":"apikey_xxx",
    "password":"plain_secret_shown_once",
    "smtp": {"host":"smtp.zxmail.site","ports":[587,465],"tls":"starttls"},
    "quota_applied": {"daily":10000,"monthly":300000}
  }
- Notes: password only returned once; stored encrypted with Vault.

7) POST /api/v2/credentials/{credential_id}/rotate
- Rotate password/keys
- Request:
  {
    "rotate_password": true,
    "revoke_old": true
  }
- Response:
  {
    "credential_id":"uuid",
    "username":"apikey_xxx",
    "password":"new_plain_secret"
  }

8) POST /api/v2/credentials/{credential_id}/revoke
- Request: {}
- Response: { "success": true }

9) GET /api/v2/credentials/{credential_id}
- Response:
  {
    "id":"uuid",
    "username":"apikey_xxx",
    "domain_id":"uuid",
    "label":"service-worker",
    "enabled":true,
    "quota":{"daily_limit":10000,"daily_used":123},
    "last_used_at":"iso"
  }

10) GET /api/v2/logs
- Query params: domain_id, credential_id, message_id, recipient, status, from, to, limit, offset, include_raw
- Response:
  {
    "logs": [
      {
        "id":"uuid",
        "message_id_header":"<abc@domain.com>",
        "postal_message_id":"postal-123",
        "from_addr":"noreply@example.com",
        "to_addr":"user@example.com",
        "subject":"string",
        "status":"bounced",
        "timestamp":"iso",
        "raw_event": { ... } // included when include_raw=true
      }
    ],
    "paging": {"limit":50,"offset":0,"total":12345}
  }

11) POST /api/v2/logs/export
- Kick off CSV/JSON export to MinIO; returns job id
- Request:
  {
    "domain_id":"uuid",
    "from":"iso",
    "to":"iso",
    "format":"csv|json",
    "fields":["message_id_header","from_addr","to_addr","status","timestamp"]
  }
- Response:
  {
    "job_id":"uuid",
    "status":"queued"
  }

12) POST /api/v2/ip-pools
- Create IP pool and tag IPs
- Request:
  {
    "name":"warmup-pool-1",
    "description":"New cold IP pool",
    "ips":[ "198.51.100.10", "198.51.100.11" ],
    "tags":["warmup"]
  }
- Response:
  {
    "pool_id":"uuid",
    "ips":[{"id":"uuid","ip":"198.51.100.10","rDNS":"pending"}]
  }

13) POST /api/v2/warmup-jobs
- Schedule an automated warm-up plan
- Request:
  {
    "pool_id":"uuid",
    "start_date":"iso",
    "duration_days":14,
    "initial_daily_volume":100,
    "ramp_strategy":"linear|aggressive|custom",
    "custom_schedule":[ {"day":1,"daily_limit":100},{"day":2,"daily_limit":300} ],
    "notify_webhook":"https://ops.example/hook" // optional
  }
- Response:
  {
    "warmup_job_id":"uuid",
    "status":"scheduled",
    "next_run":"iso"
  }

14) GET /api/v2/warmup-jobs/{warmup_job_id}
- Response includes progress, current per-IP limits, recommended adjustments, and events.

15) POST /api/v2/billing/plans
- Create plan (admin only)
- Request:
  {
    "name":"Pro",
    "monthly_price_cents":5000,
    "quota": { "monthly": 300000 },
    "rate_limits": { "per_minute": 200, "per_second": 3 },
    "overage_rate_cents_per_1000": 50
  }
- Response: plan object with id.

16) POST /api/v2/subscriptions
- Subscribe organization to a plan (calls Stripe)
- Request:
  {
    "organization_id":"uuid",
    "plan_id":"uuid",
    "payment_method_id":"pm_abc",
    "trial_days":14
  }
- Response:
  {
    "subscription_id":"sub_xxx",
    "status":"active|trialing|past_due",
    "current_period_end":"iso"
  }

17) GET /api/v2/invoices/{org_id}
- Returns invoice list; payments, links to download PDF.

18) POST /api/v2/webhooks/config
- Register a new outbound webhook for events (e.g., deliverability alerts)
- Request:
  {
    "organization_id":"uuid",
    "url":"https://customer.example/webhook",
    "events":["bounce","complaint","warmup.progress"],
    "secret_vault_ref":"vault:secret/ref"
  }
- Response:
  { "webhook_id":"uuid", "verified":false }

19) POST /api/v2/webhooks/verify
- Challenge-response verify webhook
- Request:
  { "webhook_id":"uuid", "challenge":"random_string" }
- Response: { "success": true }

20) POST /webhooks/postal/event (Postal → zxMail)
- Auth: X-Webhook-Signature: HMAC-SHA256(secret, body)
- Payload (exact):
  {
    "event":"accepted|delivered|bounced|deferred|rejected",
    "message":{
      "id":"postal_message_id",
      "message_id":"<original-message-id@example.com>",
      "from":"sender@domain.com",
      "to":"recipient@domain.com",
      "subject":"string",
      "size": 12345,
      "credential":"apikey_xxx",
      "domain":"example.com",
      "node_id":"node-uuid"
    },
    "reason":"string (for bounce/reject)",
    "smtp_response":"string (optional)",
    "timestamp":"2026-05-06T15:04:05Z"
  }
- Response: 200 OK with JSON { "status":"ok" }
- Processing: persist, increment quotas, route to suppression/complaint pipeline for bounces/complaints, emit events on RabbitMQ.

21) POST /webhooks/fbl (ISP → zxMail) — Complaint ingestion (ARF)
- Auth: header X-FBL-Signature: HMAC or TLS client cert (negotiated per ISP)
- Payload:
  {
    "arf_version":"1.0",
    "raw_arf":"base64 of ARF email" ,
    "parsed":[
      { "recipient":"user@example.com","source_ip":"1.2.3.4","original_message_id":"<...>","feedback_type":"abuse","arrival_date":"iso" }
    ],
    "provider":"gmail|yahoo|outlook|other",
    "timestamp":"iso"
  }
- Response: 200 { "status":"processed" }
- Processing: insert into complaints table, mark recipient as complained, increment complaint rates on domain & IP, notify customer via configured webhooks/email.

22) POST /api/v2/seed-tests
- Schedule a seed-list test for a credential
- Request:
  {
    "credential_id":"uuid",
    "seed_list_id":"uuid",
    "subject":"Seed test subject",
    "message_template":"string or template id",
    "send_at":"iso (optional immediate)"
  }
- Response:
  {
    "seed_test_id":"uuid",
    "status":"scheduled"
  }
- Results available via GET /api/v2/seed-tests/{id}/results

23) POST /api/v2/nodes/register
- For Postal/node registration to control plane (mTLS preferred)
- Request:
  {
    "node_id":"uuid",
    "ip":"198.51.100.10",
    "capabilities":["smtp","webhook-forwarder"],
    "postal_version":"1.0.0",
    "auth_token":"node_jwt_or_mtls_cert_fingerprint"
  }
- Response:
  { "node_id":"uuid", "status":"registered", "config_url":"https://controlplane/api/v2/nodes/{id}/config" }

24) GET /api/v2/metrics/domain/{domain_id}
- Extended metrics: delivered, bounced, deferred, rejected, accepted, complaint_rate, reputation_score (0-100), warmup_status
- Response:
  {
    "stats": { "delivered":1000,"bounced":10,"accepted":1010,"complaints":2 },
    "reputation": { "score": 72, "notes":[{"date":"iso","note":"Spike in bounces"}] }
  }

25) POST /api/v2/suppression/disable-recipient
- Request:
  {
    "recipient":"user@example.com",
    "domain_id":"uuid",
    "reason":"hard_bounce|complaint|manual",
    "source":"postal|fbl|manual"
  }
- Response:
  { "suppression_id":"uuid", "disabled":true }

Webhook Security & Signatures
- All inbound webhooks (Postal, FBL) must include HMAC header X-Webhook-Signature computed with secret stored in Vault. Support also mutual-TLS for ISP FBLs.
- Outbound webhook calls include signature header X-ZXMAIL-SIGNATURE and use retry/backoff with dead-letter queue.

Eventing & Worker Model
- Inbound webhooks persist to raw_events, then produce to RabbitMQ queue for processing by dedicated workers (bounce handler, complaint handler, warmup manager, billing meter).
- Workers are idempotent; jobs have unique idempotency keys: postal_message_id or webhook_event_id.

Entity Relationship Diagram (ERD) — Mermaid
(erDiagram syntax)

erDiagram
  USERS ||--o{ ORGANIZATIONS : owns
  ORGANIZATIONS ||--o{ USERS : members
  ORGANIZATIONS ||--o{ DOMAINS : manages
  ORGANIZATIONS ||--o{ SMTP_CREDENTIALS : issues
  DOMAINS ||--o{ SMTP_CREDENTIALS : used_by
  DOMAINS ||--o{ SEND_LOGS : produces
  SMTP_CREDENTIALS ||--o{ SEND_LOGS : used_in
  SEND_LOGS ||--o{ BOUNCES : may_produce
  SEND_LOGS ||--o{ COMPLAINTS : may_produce
  DOMAINS ||--o{ DNS_CHECKS : has
  ORGANIZATIONS ||--o{ SUBSCRIPTIONS : has
  SUBSCRIPTIONS ||--o{ INVOICES : bills
  ORGANIZATIONS ||--o{ INVOICES : receives
  IP_POOLS ||--o{ IP_ADDRESSES : contains
  IP_ADDRESSES ||--o{ NODES : assigned_to
  NODES ||--o{ SMTP_CREDENTIALS : hosts
  WARMUP_JOBS ||--o{ IP_POOLS : targets
  SEED_TESTS ||--o{ SMTP_CREDENTIALS : uses
  WEBHOOKS ||--o{ ORGANIZATIONS : belongs_to
  AUDIT_LOG ||--o{ USERS : actor
  SUPPRESSIONS ||--o{ ORGANIZATIONS : belongs_to

Database Schema (Phase 2) — core tables (PostgreSQL)
- Common: use UUID PKs (gen_random_uuid()) and timestamptz defaults.

Table: users
- id: uuid PK
- organization_id: uuid FK -> organizations.id (nullable for admins)
- email: text UNIQUE NOT NULL
- password_hash: text (nullable for SSO-only)
- role: enum('super_admin','operator','billing_admin','organization_admin','user')
- mfa_enabled: boolean default false
- created_at: timestamptz default now()
- last_login: timestamptz
- sso_provider: text nullable
Indexes: email unique, org_id.

Table: organizations
- id: uuid PK
- name: text NOT NULL
- owner_user_id: uuid FK -> users.id
- billing_customer_id: text (stripe customer id)
- retention_days: integer default 90
- created_at: timestamptz

Table: domains
- id: uuid PK
- organization_id: uuid FK -> organizations.id
- name: text UNIQUE per org
- verified: boolean default false
- dkim_selector: text
- dkim_public: text
- spf_record: text
- dmarc_record: text
- dmarc_summary: jsonb (cached aggregate)
- auto_dns_provider: text nullable
- auto_dns_status: enum('none','pending','applied','failed')
- created_at: timestamptz
- verified_at: timestamptz
Indexes: (organization_id, name), name.

Table: smtp_credentials
- id: uuid PK
- organization_id: uuid FK
- domain_id: uuid FK nullable
- username: text UNIQUE NOT NULL (apikey_xxx)
- password_enc: text NOT NULL (Vault-wrapped)
- label: text
- enabled: boolean default true
- created_at: timestamptz
- last_used_at: timestamptz
- quota_daily_limit: bigint nullable
- quota_daily_used: bigint default 0
- quota_monthly_limit: bigint nullable
- quota_monthly_used: bigint default 0
- plan_id: uuid FK -> billing_plans.id nullable
- assigned_node_id: uuid FK -> nodes.id nullable
Indexes: username unique, organization_id.

Table: send_logs (partitioning recommended by created_at month)
- id: uuid PK
- domain_id: uuid FK
- credential_id: uuid FK
- postal_message_id: text indexed
- message_id_header: text indexed
- from_addr: text
- to_addr: text indexed
- subject: text
- status: enum('accepted','delivered','bounced','deferred','rejected') indexed
- raw_event: jsonb
- size: integer
- node_id: uuid
- created_at: timestamptz default now()
Indexes: (to_addr), (message_id_header), (status, created_at), partition by range(created_at).

Table: bounces
- id: uuid PK
- recipient: text indexed
- domain_id: uuid FK
- credential_id: uuid FK
- reason: text
- postal_message_id: text
- bounce_type: enum('hard','soft','policy','spam')
- created_at: timestamptz
- disabled: boolean default false
Indexes: recipient.

Table: complaints
- id: uuid PK
- recipient: text
- domain_id: uuid FK
- credential_id: uuid FK
- provider: text
- reason: text
- raw_report: jsonb
- created_at: timestamptz
- processed: boolean default false

Table: ip_addresses
- id: uuid PK
- ip: inet UNIQUE
- pool_id: uuid FK -> ip_pools.id nullable
- assigned_node_id: uuid FK -> nodes.id nullable
- rDNS: text
- status: enum('available','assigned','in_warmup','cold','blacklisted')
- reputation_notes: text
- created_at: timestamptz

Table: ip_pools
- id: uuid PK
- name: text
- description: text
- tags: text[] default '{}'
- created_at: timestamptz

Table: nodes
- id: uuid PK
- name: text
- ip: inet
- provider: text
- postal_version: text
- last_heartbeat: timestamptz
- config_hash: text
- status: enum('online','offline','draining','provisioning')
- created_at: timestamptz
Indexes: ip unique.

Table: warmup_jobs
- id: uuid PK
- pool_id: uuid FK
- organization_id: uuid FK nullable (if warmup per customer)
- start_date: date
- duration_days: integer
- initial_daily_volume: bigint
- strategy: enum('linear','aggressive','custom')
- custom_schedule: jsonb nullable
- status: enum('scheduled','in_progress','paused','completed','failed')
- progress: jsonb (per ip day progress)
- created_at: timestamptz
- updated_at: timestamptz

Table: billing_plans
- id: uuid PK
- name: text
- monthly_price_cents: integer
- quota_monthly: bigint
- rate_limits: jsonb (per_minute/per_second)
- overage_rate_cents_per_1000: integer
- created_at: timestamptz

Table: subscriptions
- id: uuid PK
- organization_id: uuid FK
- plan_id: uuid FK
- stripe_subscription_id: text
- status: enum('active','trialing','past_due','canceled')
- current_period_end: timestamptz
- created_at: timestamptz

Table: invoices
- id: uuid PK
- organization_id: uuid FK
- stripe_invoice_id: text
- amount_cents: bigint
- status: enum('pending','paid','failed')
- pdf_url: text
- created_at: timestamptz

Table: dns_checks
- id: uuid PK
- domain_id: uuid FK
- record_type: text
- name: text
- expected_value: text
- found_value: text
- found: boolean
- checked_at: timestamptz

Table: webhooks
- id: uuid PK
- organization_id: uuid FK
- url: text
- events: text[] // e.g., ['bounce','complaint']
- secret_vault_ref: text (Vault)
- last_verified_at: timestamptz
- created_at: timestamptz

Table: seed_tests
- id: uuid PK
- organization_id: uuid FK
- credential_id: uuid FK
- seed_list: jsonb
- subject: text
- message_template: text
- scheduled_at: timestamptz
- status: enum('scheduled','running','completed','failed')
- results: jsonb
- created_at: timestamptz

Table: suppressions
- id: uuid PK
- organization_id: uuid FK
- recipient: text indexed
- reason: enum('bounce','complaint','manual','unsubscribed')
- source: text
- created_at: timestamptz
- expires_at: timestamptz nullable
- metadata: jsonb

Table: audit_log
- id: uuid PK
- actor_user_id: uuid FK
- action: text
- target_type: text
- target_id: uuid
- details: jsonb
- created_at: timestamptz

Table: raw_events
- id: uuid PK
- source: text
- raw_payload: jsonb
- processed: boolean
- created_at: timestamptz

Indexes & partitioning recommendations
- Partition send_logs by monthly range on created_at.
- Use partial indexes on send_logs(status='bounced') for fast bounce queries.
- GIN indexes for jsonb fields (dmarc_summary, results).
- Unique constraints: smtp_credentials.username.

Retention & Archival
- Default retention per org: send_logs 90 days; override per organization (up to 365 days).
- Archival job: export to MinIO and flag archived; purge physical storage after retention expiry.

Operational flows mapping to PRD Phase 1 (tie-in)
- Phase 1 features remain core: Postal deployment, domain onboarding, DKIM/SPF/DMARC guidance & verification, SMTP credential issuance, send logs, bounce handling, quotas.
- Phase 2 extends Phase 1 by:
  - Multi-node orchestration so the Postal deployment template becomes centrally managed via nodes and Helm charts.
  - IP pool and automated warm-up to address deliverability over new dedicated IPs (operator still controls rDNS; attempted automation when provider API available).
  - Billing & subscription systems enabling plan-based quota enforcement and overage billing.
  - Deliverability tooling added: seed-list tests, DMARC aggregates parsing, complaint ingestion and classification (FBL) to assist operators in improving reputation.
  - Automation of DNS additions where customers permit via Cloudflare API tokens stored in Vault.
  - Enhanced observability (prometheus/grafana/loki) and alerting that builds upon Phase 1 basic logs.
  - Stronger security: Vault for secrets, SSO/MFA, audit logs—extending Phase 1 credential encryption & RBAC.

Operational constraints & notes (Phase 2)
- rDNS still requires operator/provider credentials; automation is conditional on provider APIs and operator consent.
- IP warming must be conservative; warm-up recommendations given but final send pacing enforced by operator policy and per-credential caps.
- Billing introduces legal/tax complexity—Phase 2 includes Stripe integration only; operators remain responsible for invoicing compliance.
- Privacy: per-tenant retention and export must be honored; data deletion requests supported.
- Scaling: architecture supports horizontal scaling of nodes & workers; database will require read replicas and partitioning for high volume (send_logs).

Migration & Backward compatibility
- Data model changes are additive; migrations must run with zero-downtime where possible (use rollout patterns for column adds, new tables).
- API v1 remains supported for Phase 1 customers; new capabilities exposed via /api/v2; provide shim endpoints.

Operational runbook highlights
- Node onboarding: nodes register, heartbeat monitored; when decommissioning, mark draining and wait for queues drained.
- Warm-up failure: when IP reputation drops or complaints spike, pause warm-up and notify admin.
- Backup/DR: nightly DB backups, weekly full Postal queue snapshot, quick restore playbooks.

Deliverables for Phase 2
- Helm charts + Terraform modules for multi-node Postal deployments.
- API & web UI extensions (billing, warm-up, IP pools, seed-tests).
- RabbitMQ workers for FBL & warm-up orchestration.
- Vault integration for secrets; Stripe billing integration.
- Documentation: operator runbooks (rDNS, IP warm-up playbook), customer docs for DNS automation opt-in, and API docs.

This Phase 2 specification extends Phase 1's self-hosted, privacy-first SMTP service to a scalable, operationally automated platform enabling managed IP pools, warm-up, billing, deliverability tooling, and advanced observability while preserving the core principles of zxMail.

---

UI/UX Structure
- Purpose: Phase 3 combines operational/operator surfaces (multi-node, IP pools, warm-up, billing) with customer-facing flows (domain onboarding, credential issuance, observability). Below are the main screens, their components, API touchpoints, states, and acceptance criteria for engineers.

Main screens (priority order)
1) Auth / SSO / Login
   - Goals: Login (password + optional MFA) and SSO entry point; show role-based landing.
   - Components: email/password form, SSO button, MFA modal, error banner.
   - APIs: POST /api/v2/auth/login, POST /api/v2/auth/sso/authorize
   - States: idle, submitting (spinner), mfa_needed, success, error.
   - Acceptance: JWT returned, redirect to role-appropriate dashboard; show inline validation.

2) Operator / Admin System Overview (Global Dashboard)
   - Goals: single-pane health & signals: clustered SMTP health, IP reputation, warm-up progress, queued jobs, alerts.
   - Components: top metrics (TPS, accepted/delivered/bounced 24h), IP pool cards, warm-up jobs list, nodes list with statuses, recent alerts feed.
   - APIs: GET /api/v2/metrics/global, GET /api/v2/ip-pools, GET /api/v2/warmup-jobs?status=active, GET /api/v2/nodes
   - States: real-time (WS or polling), degraded/no-data, empty state.
   - Acceptance: metrics load <300ms for summary; real-time updates via WebSocket/Server-Sent-Events for critical alerts.

3) Organization Dashboard (Customer landing)
   - Goals: show org-level metrics, domains, credentials, quota usage, quick actions (create domain, create credential).
   - Components: metric tiles, domain list (verified state), credentials list, recent send_logs snippet.
   - APIs: GET /api/v2/metrics/domain/{domain_id}, GET /api/v2/domains, GET /api/v2/credentials
   - States: no-domains onboarding CTA, partial verification checklist.
   - Acceptance: create-domain opens onboarding wizard; clicking credential shows modal with SMTP details.

4) Domain Onboarding Wizard (critical)
   - Goals: guide domain DNS setup (SPF/DKIM/DMARC/MX) with verification polling and optional auto-DNS (Cloudflare) flow.
   - Components: stepper (Domain name → DNS records → Verification & rDNS notes → Done), copy-to-clipboard snippets, DNS check list with live pass/fail, auto-apply toggle, progress polling indicator.
   - APIs: POST /api/v2/domains, POST /api/v2/domains/{domain_id}/verify, GET /api/v2/domains/{domain_id}
   - States: waiting for DNS, verified, failed, auto_apply_pending, retry limit.
   - Acceptance: show per-record found/value_match; provide detailed remediation hints; retry/force-check button.

5) Credentials Management (critical)
   - Goals: create/rotate/revoke SMTP credentials; show quota, last_used_at, assigned node.
   - Components: create modal (label, domain select, quota fields), show-once password view, rotate/revoke actions, enable toggle, sample SMTP code snippets.
   - APIs: POST /api/v2/credentials, GET /api/v2/credentials/{id}, POST /api/v2/credentials/{id}/rotate, POST /api/v2/credentials/{id}/revoke
   - States: created (show password once), active, disabled, rotated (old invalid).
   - Acceptance: password shown once and then masked; creation returns encrypted store; UI shows current quota usage.

6) Message Logs & Message Detail (critical)
   - Goals: searchable, filterable log explorer; message-level timeline of Postal events and raw event JSON; actions (disable recipient, create suppression).
   - Components: search bar (message_id, recipient, credential, domain), filters (status, date range), results table with status badges, selectable row shows right-hand detail pane with event timeline and raw JSON, action buttons.
   - APIs: GET /api/v2/logs (with include_raw option), POST /api/v2/suppression/disable-recipient, GET /api/v2/logs/{id}
   - States: paginated, streaming tail-view (live), empty, partial results.
   - Acceptance: filters combined with debounced API (300ms); timeline shows accepted→delivered→bounced events with timestamps; raw JSON toggle.

7) IP Pools & Warm-up Manager
   - Goals: manage IP pools, schedule automated warm-up jobs, track per-IP daily limits and progress, pause/resume, recommendations.
   - Components: IP pool list, pool detail with per-IP cards (status, rDNS), warm-up job builder (start_date, duration, strategy, custom_schedule table), job progress timeline, alerts.
   - APIs: POST /api/v2/ip-pools, POST /api/v2/warmup-jobs, GET /api/v2/warmup-jobs/{id}, PATCH warmup control endpoints
   - States: scheduled, in_progress, paused, completed, failed.
   - Acceptance: job schedule saved, background worker picks it up; UI shows next_run and per-IP limits.

8) Nodes & Postal Instances
   - Goals: register/manage nodes, view heartbeat, drain nodes, view assigned IPs and Postal version.
   - Components: nodes table, node detail page, register node modal (mTLS fingerprint or node token), heartbeat chart, drain button with confirmation.
   - APIs: POST /api/v2/nodes/register, GET /api/v2/nodes
   - States: online/offline/draining/provisioning.
   - Acceptance: draining flag triggers node-side graceful shutdown; show last_heartbeat.

9) Deliverability Toolkit (seed tests, DMARC, FBL)
   - Goals: schedule seed-list tests and show results, DMARC aggregate summary, ingest FBLs and show complaints.
   - Components: seed test scheduler, results table with ISP placement; DMARC summary card, complaint list with provider and original message link.
   - APIs: POST /api/v2/seed-tests, GET /api/v2/seed-tests/{id}/results, POST /webhooks/fbl
   - States: scheduled/running/completed; parsing errors shown.

10) Billing & Subscriptions
    - Goals: plan selection, subscriptions, invoices, usage graphs, overage estimates.
    - Components: plan cards, subscription status, invoice list, payment methods.
    - APIs: POST /api/v2/subscriptions, GET /api/v2/invoices/{org_id}
    - States: trialing/active/past_due/canceled.

11) Webhooks & Notifications
    - Goals: configure outbound webhook endpoints for org events (bounce/complaint/warmup.progress).
    - Components: webhook list, verify ping, secret rotation, delivery logs.
    - APIs: POST /api/v2/webhooks/config, POST /api/v2/webhooks/verify
    - States: verified/unverified/failed_delivery.

12) Suppressions & Recipient Management
    - Goals: view and manage suppressed recipients, bulk import/export, expiry.
    - Components: suppression table, bulk upload, per-recipient actions.
    - APIs: POST /api/v2/suppression/disable-recipient, GET suppressions.

13) Settings: RBAC, SSO, Vault integrations, retention policies, audit log
    - Goals: admin controls for org and system behavior.
    - Components: user invite flows, role picker, SSO connector, Vault token management, data retention slider.
    - APIs: admin endpoints, audit_log reads.

Design rules for engineers
- UI must be responsive: sidebar collapses to icons; primary workflows optimized for 1024px+.
- Accessibility: all interactive elements must have keyboard focus, aria-labels, color contrast >=4.5:1 for text.
- Loading: use skeletons for tables/graphs; optimistic updates for enable/disable toggles with rollback on 4xx/5xx.
- Error handling: show contextual inline errors (field-level), toasts for success/failure; map HTTP 401→logout, 429→rate-limit toast with retry-after.
- Real-time: use WebSockets/SSE for critical flows (alerts, live logs tail, warm-up progress). Fallback to polling (10s) if WS fails.
- Pagination & large datasets: server-side pagination cursor-based for logs; client-side debounced searches.
- Security: sensitive secrets (password shown once) must not be persisted in app state; use ephemeral modal with copy button and zero retention.
- Audit trail: every mutating action triggers a write to audit_log and UI shows recent audit entries near action.

ASCII Wireframes (critical screens)
- Wireframes are annotated with component names and API endpoints to help programmers implement.

1) Operator / Admin System Overview (Global Dashboard)
```markdown
+-------------------------------------------------------------------------------------------+
| ZXMAIL (logo)                       | Search (msg id / recipient) | User ▾ | Org ▾ | Help ▾ |
+-------------------------------------------------------------------------------------------+
| Sidebar:                       | Top metrics (24h)                                          |
| - Dashboard (active)           | +-------------------------------------------------------+ |
| - IP Pools                     | | Accepted  12.3k | Delivered 11.8k | Bounced  220 |   | |
| - Warm-up Jobs                 | | Rejections 12 | TPS 8.2 | Complaint rate 0.02%       | |
| - Domains                      | +-------------------------------------------------------+ |
| - Credentials                  |                                                             |
| - Logs                         | Graph: Delivery rate (24h)                                 |
| - Nodes                        |  [graph line chart]                                        |
| - Seed Tests                   |                                                             |
| - Billing                      |                                                             |
| - Settings                     | Warm-up jobs (active):                                       |
| - Audit Log                    | +-------------------------------------------------------+       |
|                                | | Job Name     | Pool        | Progress | Next run  | Ctrl |  |
|                                | | warmup-1     | warmup-pool | 8/14 days| 2026-05-08| >    |  |
|                                | | orgA-warmup  | pool-2      | 3/14    | 2026-05-07| >    |  |
|                                | +-------------------------------------------------------+       |
|                                | IP Pools (top 3 by volume):                                   |
|                                | +--------------------------------------+   Alerts (feed)      |
|                                | | Pool: warmup-pool                    | -------------------  |
|                                | | IPs: 198.51.100.10(in_warmup)        |  [!] High bounce rate |
|                                | | Reputation: 72 / 100                 |  [!] Node offline     |
|                                | +--------------------------------------+ -------------------  |
+-------------------------------------------------------------------------------------------+
Notes/implementation:
- Metrics from GET /api/v2/metrics/global
- Warm-up jobs from GET /api/v2/warmup-jobs?status=active (subscribe to SSE / WS)
- IP pool quick actions: open pool detail -> GET /api/v2/ip-pools/{id}
```

2) Domain Onboarding Wizard (multi-step)
```markdown
+-------------------------------------------------------------------------------------------+
| < Back               Domain Onboarding -> Step 2/3: Add DNS records                        |
+-------------------------------------------------------------------------------------------+
| Progress: [Domain name] -> [DNS records] -> [Verify & rDNS] -> [Done]                     |
+-------------------------------------------------------------------------------------------+
| Left column (info)                        | Right column (DNS checklist)                 |
| - Domain: example.com                     | Domain status: NOT VERIFIED [Retry Check]    |
| - Owner: org@example.com                  | Last checked: 2026-05-06 10:23 UTC           |
| - Auto-DNS: OFF (toggle)                  |                                                 |
|                                           | 1) TXT @     SPF                            |
|                                           |    expected: v=spf1 mx include:smtp.zxmail.site -all |
|                                           |    status: ❌ not found                       |
|                                           |    [copy] [how to add in Cloudflare]         |
|                                           |                                               |
|                                           | 2) TXT zxmail._domainkey                     |
|                                           |    expected: v=DKIM1; p=MIIBIjAN...          |
|                                           |    status: ✅ found (value match)             |
|                                           |    [copy selector]                            |
|                                           |                                               |
|                                           | 3) TXT _dmarc                               |
|                                           |    expected: v=DMARC1; p=none; rua=mailto:...|
|                                           |    status: ❌ not found                       |
|                                           |                                               |
|                                           | [Verify now] [Start auto-check poll]         |
|                                           |                                               |
|                                           | rDNS note: set PTR on VPS provider to: send.example.com |
|                                           | [Open operator runbook]                      |
+-------------------------------------------------------------------------------------------+
Footer: [Cancel] [Save as Draft] [Next: Verify & rDNS]
Notes/implementation:
- Create domain: POST /api/v2/domains {name,...} returns dns_requirements
- Verify trigger: POST /api/v2/domains/{id}/verify -> job_id; poll GET /api/v2/domains/{id} for dns_requirements verification_status
- Auto-DNS: optional flow to call Cloudflare API via server when token present (POST includes auto_dns).
- Show copy-to-clipboard for record values; mask long DKIM keys with expand.
```

3) Message Logs & Message Detail (Explorer + Timeline)
```markdown
+-------------------------------------------------------------------------------------------+
| ZXMAIL > Logs                                                                             |
+-------------------------------------------------------------------------------------------+
| Filters: [domain ▾] [credential ▾] [status ▾] [recipient / message_id search ______] [date range] [Apply] |
+-------------------------------------------------------------------------------------------+
| Results (table)                                                                            |
| +----+----------------------+----------------------+------------+---------+-------------+--------+ |
| | #  | Message-ID           | From                 | To         | Status  | Time        | Actions| |
| +----+----------------------+----------------------+------------+---------+-------------+--------+ |
| | 1  | <abc@...>            | noreply@ex.com       | u@ex.com   | BOUNCED | 2026-05-06  | View   | |
| | 2  | <def@...>            | ops@ex.com           | v@ex.com   | DELIVER | 2026-05-06  | View   | |
| | 3  | <ghi@...>            | app@ex.com           | w@ex.com   | ACCEPT  | 2026-05-06  | View   | |
| +----+----------------------+----------------------+------------+---------+-------------+--------+ |
| Pagination: << 1 2 3 ... 50 >>                                                                 |
+-------------------------------------------------------------------------------------------+
| Detail pane (right)                                                                        |
| ---------------------------------------------------------------------------               |
| Message: <abc@...>   status: BOUNCED    postal_id: postal-12345                         |
| From: noreply@ex.com  To: u@ex.com  Subject: "Reset"  Size: 1234                         |
| ---------------------------------------------------------------------------               |
| Event timeline:                                                                         |
| [2026-05-06T12:01:02] ACCEPTED   (credential: apikey_xxx)                                 |
| [2026-05-06T12:01:08] DEFERRED    (smtp_response: 421 rate limit)                          |
| [2026-05-06T12:05:10] BOUNCED     (reason: 550 mailbox not found)  [Create suppression]    |
|                                                                                         |
| Raw event: [toggle]                                                                     |
| { "event":"bounced", "message": {...}, "reason":"550 mailbox not found", ... }           |
|                                                                                         |
| Actions: [Disable recipient] [Create suppression] [Send to webhook]                      |
|                                                                                         |
| Related logs: [Show all for recipient] [Show all for credential]                         |
+-------------------------------------------------------------------------------------------+
Notes/implementation:
- Logs: GET /api/v2/logs?domain_id=&credential_id=&recipient=&message_id=&status=&from=&to=&limit=
- Tail/live: subscribe to WS channel for selected domain to append new events to top.
- Detail: GET /api/v2/logs/{id} or include_raw=true on list; actions call POST /api/v2/suppression/disable-recipient
- Rate-limit UI: debounce search (300ms); table uses server-side cursor paging.
```

Implementation notes / Handoff checklist for developers
- Map each CTA to API endpoints listed in notes under each wireframe.
- Use feature flags for Phase 3 complex features (warm-up, billing, seed-tests) to enable progressive rollout.
- Provide a small design tokens file and component library (React + TypeScript) with components: MetricCard, DataTable (server-paged), Stepper, CodeSnippet (copy), Toasts, Modal (secret-show-once), Timeline, Badge (status colors).
- Live updates: implement WS topic naming convention: /ws/org/{org_id}/alerts, /ws/domain/{domain_id}/logs, /ws/warmup/{job_id}/progress.
- Security: never store plaintext password in browser; show it in memory only until modal closes. Ensure endpoints returning plaintext are called only once and client clears value immediately.

End.