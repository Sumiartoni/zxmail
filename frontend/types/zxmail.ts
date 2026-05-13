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
