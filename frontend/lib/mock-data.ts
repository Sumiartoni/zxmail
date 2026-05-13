export const stackHighlights = [
  {
    title: "Backend lane",
    description: "Go + Gin API surface with health endpoints and Production v1 route map.",
    note: "Scaffold includes auth, orgs, domains, credentials, logs, suppressions, admin, and Postal webhook handlers.",
  },
  {
    title: "Persistence lane",
    description: "PostgreSQL migration baseline aligned to domain onboarding and SMTP operations.",
    note: "Users, memberships, domains, DNS records, credentials, logs, bounces, suppressions, quotas, and webhook events are all reserved.",
  },
  {
    title: "Redis lane",
    description: "Dedicated for queue jobs, quota windows, and rate limiting.",
    note: "Next implementation pass should wire DNS verification polling and minute or daily counters through Redis-backed workers.",
  },
  {
    title: "Infra lane",
    description: "Docker Compose and Caddy or Postal templates are prepared at repo root.",
    note: "Scope stays single-node Production v1. No Kubernetes, IP pool automation, or multi-node orchestration.",
  },
];

export const adminMetrics = [
  { label: "Verified domains", value: "12", note: "Placeholder metric card for onboarding health." },
  { label: "Live credentials", value: "34", note: "Reserved for active SMTP identities across organizations." },
  { label: "Bounce rate", value: "0.8%", note: "Will be derived from Postal webhook events." },
  { label: "Queue health", value: "Ready", note: "Redis-backed job processors and webhook workers." },
];

export const customerMetrics = [
  { label: "Domains", value: "02", note: "Pending and verified sending identities in one view." },
  { label: "Credentials", value: "04", note: "Scoped SMTP users with show-once secrets and quotas." },
  { label: "Daily quota", value: "61%", note: "Prepared for daily and monthly enforcement panels." },
  { label: "Suppressions", value: "07", note: "Recipient safety rail sourced from bounces and manual actions." },
];

export const operatorChecks = [
  "Health endpoints for liveness and readiness are already exposed under /health.",
  "Webhook signature validation is scaffolded before persistence is implemented.",
  "Initial migration reserves the data model for roles, quotas, and suppression handling.",
  "Compose structure keeps Postal, Postgres, Redis, API, frontend, and reverse proxy in one deployable topology.",
];

export const dnsRecordsPreview = [
  {
    type: "TXT",
    host: "zxmail.site",
    value: "v=spf1 a mx include:postal.zxmail.site ~all",
  },
  {
    type: "TXT",
    host: "postal._domainkey.zxmail.site",
    value: "k=rsa; p=PUBLIC_KEY_PLACEHOLDER",
  },
  {
    type: "TXT",
    host: "_dmarc.zxmail.site",
    value: "v=DMARC1; p=none; rua=mailto:dmarc@zxmail.site",
  },
];
