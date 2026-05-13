CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('admin', 'customer')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_login TIMESTAMPTZ
);

CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    owner_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE domains (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name TEXT NOT NULL UNIQUE,
    verified BOOLEAN NOT NULL DEFAULT FALSE,
    dkim_selector TEXT NOT NULL,
    dkim_public TEXT NOT NULL,
    spf_record TEXT NOT NULL,
    dmarc_record TEXT NOT NULL,
    verified_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE smtp_credentials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    domain_id UUID REFERENCES domains(id) ON DELETE SET NULL,
    username TEXT NOT NULL UNIQUE,
    password_enc TEXT NOT NULL,
    password_key_id TEXT,
    label TEXT,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used_at TIMESTAMPTZ,
    quota_per_minute_limit INTEGER,
    quota_daily_limit INTEGER,
    quota_daily_used INTEGER NOT NULL DEFAULT 0,
    quota_monthly_limit INTEGER,
    quota_monthly_used INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE send_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    domain_id UUID REFERENCES domains(id) ON DELETE SET NULL,
    credential_id UUID REFERENCES smtp_credentials(id) ON DELETE SET NULL,
    postal_message_id TEXT,
    message_id_header TEXT,
    from_addr TEXT NOT NULL,
    to_addr TEXT NOT NULL,
    subject TEXT,
    status TEXT NOT NULL CHECK (status IN ('accepted', 'delivered', 'bounced', 'deferred', 'rejected')),
    raw_event JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE bounces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    domain_id UUID REFERENCES domains(id) ON DELETE SET NULL,
    credential_id UUID REFERENCES smtp_credentials(id) ON DELETE SET NULL,
    recipient TEXT NOT NULL,
    reason TEXT,
    postal_message_id TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    disabled BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE suppressions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    recipient TEXT NOT NULL,
    source TEXT NOT NULL CHECK (source IN ('bounce', 'manual', 'complaint')),
    reason TEXT,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    released_at TIMESTAMPTZ
);

CREATE TABLE dns_checks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    domain_id UUID NOT NULL REFERENCES domains(id) ON DELETE CASCADE,
    record_type TEXT NOT NULL,
    name TEXT NOT NULL,
    expected_value TEXT NOT NULL,
    found_value TEXT,
    found BOOLEAN NOT NULL DEFAULT FALSE,
    checked_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE webhooks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source TEXT NOT NULL,
    secret TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL,
    action TEXT NOT NULL,
    target_type TEXT NOT NULL,
    target_id UUID,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_role ON users (role);
CREATE INDEX idx_organizations_owner_user_id ON organizations (owner_user_id);
CREATE INDEX idx_domains_organization_id ON domains (organization_id);
CREATE INDEX idx_domains_verified ON domains (verified);
CREATE INDEX idx_domains_name ON domains (name);
CREATE INDEX idx_smtp_credentials_username ON smtp_credentials (username);
CREATE INDEX idx_smtp_credentials_organization_id ON smtp_credentials (organization_id);
CREATE INDEX idx_smtp_credentials_domain_id ON smtp_credentials (domain_id);
CREATE INDEX idx_smtp_credentials_enabled ON smtp_credentials (enabled);
CREATE INDEX idx_smtp_credentials_quota_per_minute_limit ON smtp_credentials (quota_per_minute_limit);
CREATE INDEX idx_send_logs_domain_id ON send_logs (domain_id);
CREATE INDEX idx_send_logs_credential_id ON send_logs (credential_id);
CREATE INDEX idx_send_logs_postal_message_id ON send_logs (postal_message_id);
CREATE INDEX idx_send_logs_message_id_header ON send_logs (message_id_header);
CREATE INDEX idx_send_logs_to_addr ON send_logs (to_addr);
CREATE INDEX idx_send_logs_status ON send_logs (status);
CREATE INDEX idx_send_logs_created_at ON send_logs (created_at DESC);
CREATE INDEX idx_send_logs_status_created_at ON send_logs (status, created_at DESC);
CREATE UNIQUE INDEX idx_send_logs_postal_message_status_unique ON send_logs (postal_message_id, status) WHERE postal_message_id IS NOT NULL;
CREATE INDEX idx_bounces_recipient ON bounces (recipient);
CREATE INDEX idx_bounces_domain_id ON bounces (domain_id);
CREATE INDEX idx_bounces_credential_id ON bounces (credential_id);
CREATE INDEX idx_bounces_disabled ON bounces (disabled);
CREATE INDEX idx_suppressions_organization_recipient ON suppressions (organization_id, recipient);
CREATE INDEX idx_suppressions_recipient ON suppressions (recipient);
CREATE INDEX idx_suppressions_active ON suppressions (active);
CREATE INDEX idx_dns_checks_domain_id ON dns_checks (domain_id);
CREATE INDEX idx_dns_checks_record_type_name ON dns_checks (record_type, name);
CREATE INDEX idx_dns_checks_found ON dns_checks (found);
CREATE INDEX idx_webhooks_source ON webhooks (source);
CREATE INDEX idx_audit_logs_actor_user_id ON audit_logs (actor_user_id);
CREATE INDEX idx_audit_logs_organization_id ON audit_logs (organization_id);
CREATE INDEX idx_audit_logs_created_at ON audit_logs (created_at DESC);
