# Production Ready v2

Production Ready v2 extends zxMail beyond the original Production v1 control plane without replacing the working foundation.

Included:
- gateway-agnostic plans and subscriptions
- manual bank transfer and manual QRIS payment tracking
- invoices and payment approval flow
- usage metering with PostgreSQL source of truth and Redis advisory counters
- deliverability snapshots and alert generation
- retention policy and cleanup controls
- worker service for scheduled jobs
- expanded customer and admin dashboards

Still intentionally excluded:
- Stripe
- Kubernetes or multi-node orchestration
- IP pool automation
- automated warm-up
- seed testing
- FBL ingestion
- SSO enterprise
