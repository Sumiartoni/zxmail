ALTER TABLE organizations
    ADD COLUMN IF NOT EXISTS suspended BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS suspended_reason TEXT,
    ADD COLUMN IF NOT EXISTS suspended_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS retention_days INTEGER NOT NULL DEFAULT 90,
    ADD COLUMN IF NOT EXISTS quota_daily_override INTEGER,
    ADD COLUMN IF NOT EXISTS quota_monthly_override INTEGER,
    ADD COLUMN IF NOT EXISTS quota_per_minute_override INTEGER,
    ADD COLUMN IF NOT EXISTS settings JSONB NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

ALTER TABLE domains
    ADD COLUMN IF NOT EXISTS last_rechecked_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS current_health_score INTEGER NOT NULL DEFAULT 0;

ALTER TABLE smtp_credentials
    ADD COLUMN IF NOT EXISTS manually_limited BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS manual_limit_reason TEXT,
    ADD COLUMN IF NOT EXISTS manual_limit_updated_at TIMESTAMPTZ;

ALTER TABLE send_logs
    ADD COLUMN IF NOT EXISTS organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL;

ALTER TABLE bounces
    ADD COLUMN IF NOT EXISTS organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL;

UPDATE send_logs l
SET organization_id = d.organization_id
FROM domains d
WHERE l.domain_id = d.id
  AND l.organization_id IS NULL;

UPDATE bounces b
SET organization_id = d.organization_id
FROM domains d
WHERE b.domain_id = d.id
  AND b.organization_id IS NULL;

CREATE TABLE IF NOT EXISTS plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    active BOOLEAN NOT NULL DEFAULT TRUE,
    currency TEXT NOT NULL DEFAULT 'IDR',
    price_monthly BIGINT NOT NULL DEFAULT 0,
    daily_quota INTEGER,
    monthly_quota INTEGER,
    per_minute_quota INTEGER,
    credential_quota INTEGER,
    trial_days INTEGER NOT NULL DEFAULT 0,
    overage_price_per_email BIGINT NOT NULL DEFAULT 0,
    payment_methods JSONB NOT NULL DEFAULT '[]'::jsonb,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    plan_id UUID NOT NULL REFERENCES plans(id) ON DELETE RESTRICT,
    status TEXT NOT NULL CHECK (status IN ('trialing', 'active', 'past_due', 'expired', 'suspended', 'canceled')),
    starts_at TIMESTAMPTZ NOT NULL,
    current_period_start TIMESTAMPTZ NOT NULL,
    current_period_end TIMESTAMPTZ NOT NULL,
    trial_ends_at TIMESTAMPTZ,
    expired_at TIMESTAMPTZ,
    suspended_at TIMESTAMPTZ,
    quota_daily_override INTEGER,
    quota_monthly_override INTEGER,
    quota_per_minute_override INTEGER,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS invoices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    subscription_id UUID REFERENCES subscriptions(id) ON DELETE SET NULL,
    invoice_number TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL CHECK (status IN ('draft', 'issued', 'paid', 'failed', 'void')),
    currency TEXT NOT NULL DEFAULT 'IDR',
    amount BIGINT NOT NULL,
    due_at TIMESTAMPTZ,
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    issued_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    paid_at TIMESTAMPTZ,
    failed_at TIMESTAMPTZ,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    invoice_id UUID REFERENCES invoices(id) ON DELETE SET NULL,
    provider_code TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('pending', 'approved', 'rejected', 'failed')),
    amount BIGINT NOT NULL,
    reference TEXT,
    submitted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    approved_at TIMESTAMPTZ,
    rejected_at TIMESTAMPTZ,
    notes TEXT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS usage_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    credential_id UUID REFERENCES smtp_credentials(id) ON DELETE SET NULL,
    domain_id UUID REFERENCES domains(id) ON DELETE SET NULL,
    send_log_id UUID REFERENCES send_logs(id) ON DELETE CASCADE,
    metric TEXT NOT NULL CHECK (metric IN ('accepted', 'delivered', 'bounced', 'deferred', 'rejected', 'overage')),
    quantity INTEGER NOT NULL DEFAULT 1,
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    period_day DATE NOT NULL,
    period_month DATE NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE TABLE IF NOT EXISTS quota_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    credential_id UUID REFERENCES smtp_credentials(id) ON DELETE SET NULL,
    event_type TEXT NOT NULL,
    reason TEXT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS deliverability_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    domain_id UUID REFERENCES domains(id) ON DELETE SET NULL,
    credential_id UUID REFERENCES smtp_credentials(id) ON DELETE SET NULL,
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    accepted_count INTEGER NOT NULL DEFAULT 0,
    delivered_count INTEGER NOT NULL DEFAULT 0,
    bounced_count INTEGER NOT NULL DEFAULT 0,
    deferred_count INTEGER NOT NULL DEFAULT 0,
    rejected_count INTEGER NOT NULL DEFAULT 0,
    bounce_rate NUMERIC(7,4) NOT NULL DEFAULT 0,
    rejection_rate NUMERIC(7,4) NOT NULL DEFAULT 0,
    deferred_rate NUMERIC(7,4) NOT NULL DEFAULT 0,
    delivered_rate NUMERIC(7,4) NOT NULL DEFAULT 0,
    score INTEGER NOT NULL DEFAULT 0,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS domain_health_checks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    domain_id UUID NOT NULL REFERENCES domains(id) ON DELETE CASCADE,
    spf_found BOOLEAN NOT NULL DEFAULT FALSE,
    dkim_found BOOLEAN NOT NULL DEFAULT FALSE,
    dmarc_found BOOLEAN NOT NULL DEFAULT FALSE,
    mx_note_found BOOLEAN NOT NULL DEFAULT FALSE,
    rdns_status TEXT NOT NULL DEFAULT 'manual',
    bounce_rate NUMERIC(7,4) NOT NULL DEFAULT 0,
    rejection_rate NUMERIC(7,4) NOT NULL DEFAULT 0,
    deferred_rate NUMERIC(7,4) NOT NULL DEFAULT 0,
    quota_limited BOOLEAN NOT NULL DEFAULT FALSE,
    score INTEGER NOT NULL DEFAULT 0,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    checked_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS system_alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    domain_id UUID REFERENCES domains(id) ON DELETE SET NULL,
    credential_id UUID REFERENCES smtp_credentials(id) ON DELETE SET NULL,
    severity TEXT NOT NULL CHECK (severity IN ('info', 'warning', 'critical')),
    status TEXT NOT NULL CHECK (status IN ('open', 'resolved')),
    alert_type TEXT NOT NULL,
    title TEXT NOT NULL,
    message TEXT NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS worker_job_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_name TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('running', 'succeeded', 'failed')),
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_usage_records_send_log_metric_unique
    ON usage_records (send_log_id, metric)
    WHERE send_log_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_subscriptions_active_org
    ON subscriptions (organization_id)
    WHERE status IN ('trialing', 'active', 'past_due', 'suspended');

CREATE INDEX IF NOT EXISTS idx_send_logs_organization_id ON send_logs (organization_id);
CREATE INDEX IF NOT EXISTS idx_bounces_organization_id ON bounces (organization_id);
CREATE INDEX IF NOT EXISTS idx_organizations_suspended ON organizations (suspended);
CREATE INDEX IF NOT EXISTS idx_organizations_retention_days ON organizations (retention_days);
CREATE INDEX IF NOT EXISTS idx_smtp_credentials_manually_limited ON smtp_credentials (manually_limited);
CREATE INDEX IF NOT EXISTS idx_plans_active ON plans (active);
CREATE INDEX IF NOT EXISTS idx_subscriptions_organization_id ON subscriptions (organization_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_status ON subscriptions (status);
CREATE INDEX IF NOT EXISTS idx_invoices_organization_id ON invoices (organization_id);
CREATE INDEX IF NOT EXISTS idx_invoices_status ON invoices (status);
CREATE INDEX IF NOT EXISTS idx_payments_organization_id ON payments (organization_id);
CREATE INDEX IF NOT EXISTS idx_payments_status ON payments (status);
CREATE INDEX IF NOT EXISTS idx_payments_provider_code ON payments (provider_code);
CREATE INDEX IF NOT EXISTS idx_usage_records_organization_id ON usage_records (organization_id);
CREATE INDEX IF NOT EXISTS idx_usage_records_credential_id ON usage_records (credential_id);
CREATE INDEX IF NOT EXISTS idx_usage_records_domain_id ON usage_records (domain_id);
CREATE INDEX IF NOT EXISTS idx_usage_records_metric ON usage_records (metric);
CREATE INDEX IF NOT EXISTS idx_usage_records_period_day ON usage_records (period_day DESC);
CREATE INDEX IF NOT EXISTS idx_usage_records_period_month ON usage_records (period_month DESC);
CREATE INDEX IF NOT EXISTS idx_quota_events_organization_id ON quota_events (organization_id);
CREATE INDEX IF NOT EXISTS idx_quota_events_credential_id ON quota_events (credential_id);
CREATE INDEX IF NOT EXISTS idx_quota_events_created_at ON quota_events (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_deliverability_snapshots_org_window ON deliverability_snapshots (organization_id, period_end DESC);
CREATE INDEX IF NOT EXISTS idx_deliverability_snapshots_domain_window ON deliverability_snapshots (domain_id, period_end DESC);
CREATE INDEX IF NOT EXISTS idx_domain_health_checks_domain_checked_at ON domain_health_checks (domain_id, checked_at DESC);
CREATE INDEX IF NOT EXISTS idx_system_alerts_organization_id ON system_alerts (organization_id);
CREATE INDEX IF NOT EXISTS idx_system_alerts_status ON system_alerts (status);
CREATE INDEX IF NOT EXISTS idx_system_alerts_created_at ON system_alerts (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_worker_job_runs_job_name_started_at ON worker_job_runs (job_name, started_at DESC);
