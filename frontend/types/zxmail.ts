export type UserRole = "admin" | "customer";

export type LogStatus =
  | "accepted"
  | "delivered"
  | "bounced"
  | "deferred"
  | "rejected";

export type CredentialStatus = "enabled" | "limited" | "disabled";

export type DomainStatus = "verified" | "pending";

export type AppUser = {
  id: string;
  email: string;
  role: UserRole;
  organization_id?: string;
};

export type AuthSession = {
  user: AppUser;
};

export type DNSRequirement = {
  type: "TXT" | "CNAME" | "MX";
  name: string;
  value: string;
  note: string;
  required: boolean;
};

export type DNSCheck = {
  id: string;
  record_type: string;
  name: string;
  expected_value: string;
  found_value?: string | null;
  found: boolean;
  checked_at: string;
};

export type DomainRecord = {
  id: string;
  organization_id: string;
  name: string;
  verified: boolean;
  verified_at?: string | null;
  created_at: string;
  dkim_selector: string;
  dkim_public: string;
  spf_record: string;
  dmarc_record: string;
  warnings: string[];
  dns_requirements: DNSRequirement[];
  dns_checks: DNSCheck[];
};

export type SMTPConnectionInfo = {
  host: string;
  starttls_port: string;
  tls_port: string;
  username: string;
  password_note: string;
};

export type CredentialRecord = {
  id: string;
  organization_id: string;
  domain_id: string;
  domain_name: string;
  username: string;
  label?: string;
  enabled: boolean;
  status: CredentialStatus;
  created_at: string;
  last_used_at?: string | null;
  per_minute_limit?: number | null;
  per_minute_used: number;
  daily_limit?: number | null;
  daily_used: number;
  monthly_limit?: number | null;
  monthly_used: number;
  limited: boolean;
  exceeded: string[];
  enforcement_note: string;
};

export type CredentialResponse = {
  credential: CredentialRecord;
  smtp: SMTPConnectionInfo;
};

export type CredentialSecretResponse = CredentialResponse & {
  secret: string;
};

export type SendLog = {
  id: string;
  domain_id?: string | null;
  domain_name?: string | null;
  credential_id?: string | null;
  credential_name?: string | null;
  postal_message_id?: string | null;
  message_id_header?: string | null;
  from_addr: string;
  to_addr: string;
  subject?: string | null;
  status: LogStatus;
  reason?: string | null;
  created_at: string;
  raw_event: Record<string, unknown>;
};

export type SuppressionRecord = {
  id: string;
  organization_id: string;
  recipient: string;
  source: "bounce" | "manual" | "complaint";
  reason?: string | null;
  active: boolean;
  created_at: string;
  released_at?: string | null;
};

export type OrganizationRecord = {
  id: string;
  name: string;
  owner_user_id: string;
  owner_email: string;
  created_at: string;
};

export type LogsFilterState = {
  domain_id: string;
  credential_id: string;
  message_id: string;
  recipient: string;
  status: string;
  from: string;
  to: string;
  date_from: string;
  date_to: string;
  limit: number;
  offset: number;
};

export type SystemHealth = {
  api: "healthy" | "degraded";
  postal: "ready" | "manual-check";
  redis: "healthy" | "degraded";
  postgres: "healthy" | "degraded";
  notes: string[];
};

export type PaymentProviderCode = "manual_bank_transfer" | "manual_qris";

export type PlanRecord = {
  id: string;
  code: string;
  name: string;
  description: string;
  active: boolean;
  currency: string;
  price_monthly: number;
  daily_quota?: number | null;
  monthly_quota?: number | null;
  per_minute_quota?: number | null;
  credential_quota?: number | null;
  trial_days: number;
  overage_price_per_email: number;
  payment_methods: PaymentProviderCode[];
  created_at: string;
  updated_at: string;
};

export type SubscriptionRecord = {
  id: string;
  organization_id: string;
  plan_id: string;
  status: "trialing" | "active" | "past_due" | "expired" | "suspended" | "canceled";
  starts_at: string;
  current_period_start: string;
  current_period_end: string;
  trial_ends_at?: string | null;
  expired_at?: string | null;
  suspended_at?: string | null;
  quota_daily_override?: number | null;
  quota_monthly_override?: number | null;
  quota_per_minute_override?: number | null;
  notes?: string;
  created_at: string;
  updated_at: string;
};

export type SubscriptionView = {
  subscription: SubscriptionRecord;
  plan: PlanRecord;
  payment_status: string;
};

export type InvoiceRecord = {
  id: string;
  organization_id: string;
  subscription_id?: string | null;
  invoice_number: string;
  status: "draft" | "issued" | "paid" | "failed" | "void";
  currency: string;
  amount: number;
  due_at?: string | null;
  period_start: string;
  period_end: string;
  issued_at: string;
  paid_at?: string | null;
  failed_at?: string | null;
};

export type PaymentRecord = {
  id: string;
  organization_id: string;
  invoice_id?: string | null;
  provider_code: PaymentProviderCode;
  status: "pending" | "approved" | "rejected" | "failed";
  amount: number;
  reference?: string | null;
  submitted_at: string;
  approved_at?: string | null;
  rejected_at?: string | null;
  notes?: string | null;
};

export type UsageOverview = {
  organization_id: string;
  accepted_today: number;
  accepted_month: number;
  delivered_month: number;
  bounced_month: number;
  deferred_month: number;
  rejected_month: number;
  effective_daily_quota?: number | null;
  effective_monthly_quota?: number | null;
  effective_per_minute_quota?: number | null;
  overage_count: number;
  status: string;
  last_updated_at: string;
};

export type DeliverabilityOverview = {
  accepted_count: number;
  delivered_count: number;
  bounced_count: number;
  deferred_count: number;
  rejected_count: number;
  bounce_rate: number;
  deferred_rate: number;
  rejected_rate: number;
  delivered_rate: number;
  open_alerts: number;
  average_health_score: number;
};

export type DomainHealthRecord = {
  domain_id: string;
  domain_name: string;
  spf_found: boolean;
  dkim_found: boolean;
  dmarc_found: boolean;
  mx_note_found: boolean;
  rdns_status: string;
  bounce_rate: number;
  deferred_rate: number;
  rejected_rate: number;
  quota_limited: boolean;
  health_score: number;
  checked_at: string;
  last_verified_at?: string | null;
};

export type AlertRecord = {
  id: string;
  severity: "info" | "warning" | "critical";
  status: "open" | "resolved";
  alert_type: string;
  title: string;
  message: string;
  created_at: string;
  resolved_at?: string | null;
};

export type AdminOverview = {
  total_email_sent: number;
  delivered: number;
  bounced: number;
  rejected: number;
  active_customers: number;
  active_domains: number;
  open_alerts: number;
  past_due_payments: number;
};

export type AdminOrganizationDetail = {
  id: string;
  name: string;
  suspended: boolean;
  suspended_reason?: string;
  created_at: string;
  retention_days: number;
  current_subscription_status: string;
  payment_status: string;
  verified_domains: number;
  enabled_credentials: number;
  latest_send_activity_at?: string | null;
  risk_score: number;
  bounce_rate: number;
};

export type RiskRecord = {
  organization_id: string;
  name: string;
  risk_score: number;
  bounce_rate: number;
  suspended: boolean;
  payment_status: string;
};

export type RetentionPolicy = {
  organization_id: string;
  name: string;
  retention_days: number;
};

export type CleanupResult = {
  dry_run: boolean;
  matched_logs: number;
  deleted_logs: number;
  organizations: number;
};

export type AuditLogRecord = {
  id: string;
  actor_user_id?: string | null;
  actor_email?: string | null;
  organization_id?: string | null;
  action: string;
  target_type: string;
  target_id?: string | null;
  metadata: Record<string, unknown>;
  created_at: string;
};

export type AdminSystemHealth = {
  postgres: string;
  redis: string;
  postal: string;
  worker: string;
  queue: string;
  notes: Record<string, string>;
};

export type QueueHealth = {
  mode: string;
  pending: number;
  in_progress: number;
  note: string;
};
