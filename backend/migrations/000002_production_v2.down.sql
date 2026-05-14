DROP TABLE IF EXISTS worker_job_runs;
DROP TABLE IF EXISTS system_alerts;
DROP TABLE IF EXISTS domain_health_checks;
DROP TABLE IF EXISTS deliverability_snapshots;
DROP TABLE IF EXISTS quota_events;
DROP TABLE IF EXISTS usage_records;
DROP TABLE IF EXISTS payments;
DROP TABLE IF EXISTS invoices;
DROP TABLE IF EXISTS subscriptions;
DROP TABLE IF EXISTS plans;

DROP INDEX IF EXISTS idx_send_logs_organization_id;
DROP INDEX IF EXISTS idx_bounces_organization_id;
DROP INDEX IF EXISTS idx_organizations_suspended;
DROP INDEX IF EXISTS idx_organizations_retention_days;
DROP INDEX IF EXISTS idx_smtp_credentials_manually_limited;

ALTER TABLE bounces
    DROP COLUMN IF EXISTS organization_id;

ALTER TABLE send_logs
    DROP COLUMN IF EXISTS organization_id;

ALTER TABLE smtp_credentials
    DROP COLUMN IF EXISTS manually_limited,
    DROP COLUMN IF EXISTS manual_limit_reason,
    DROP COLUMN IF EXISTS manual_limit_updated_at;

ALTER TABLE domains
    DROP COLUMN IF EXISTS last_rechecked_at,
    DROP COLUMN IF EXISTS current_health_score;

ALTER TABLE organizations
    DROP COLUMN IF EXISTS suspended,
    DROP COLUMN IF EXISTS suspended_reason,
    DROP COLUMN IF EXISTS suspended_at,
    DROP COLUMN IF EXISTS retention_days,
    DROP COLUMN IF EXISTS quota_daily_override,
    DROP COLUMN IF EXISTS quota_monthly_override,
    DROP COLUMN IF EXISTS quota_per_minute_override,
    DROP COLUMN IF EXISTS settings,
    DROP COLUMN IF EXISTS updated_at;
