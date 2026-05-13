import type {
  AppUser,
  CredentialResponse,
  CredentialSecretResponse,
  DNSCheck,
  DNSRequirement,
  DomainRecord,
  LogsFilterState,
  OrganizationRecord,
  SendLog,
  SuppressionRecord,
  SystemHealth,
} from "@/types/zxmail";

const ORG_ID = "org_demo";
const ADMIN_ID = "user_admin";
const CUSTOMER_ID = "user_customer";

const cloudflareWarning =
  "Cloudflare SMTP records must stay DNS only. Do not proxy the SMTP hostname.";

let demoDomains: DomainRecord[] = [
  buildDomain({
    id: "dom_1",
    name: "acme-mail.com",
    verified: true,
    createdAt: "2026-05-05T10:12:00Z",
    verifiedAt: "2026-05-05T11:48:00Z",
  }),
  buildDomain({
    id: "dom_2",
    name: "alerts.acme-mail.com",
    verified: false,
    createdAt: "2026-05-06T08:31:00Z",
  }),
];

let demoCredentials: CredentialResponse[] = [
  {
    credential: {
      id: "cred_1",
      organization_id: ORG_ID,
      domain_id: "dom_1",
      domain_name: "acme-mail.com",
      username: "apikey_9r3s1j0x",
      label: "Primary app",
      enabled: true,
      status: "enabled",
      created_at: "2026-05-05T12:10:00Z",
      last_used_at: "2026-05-07T03:11:00Z",
      per_minute_limit: 90,
      per_minute_used: 22,
      daily_limit: 5000,
      daily_used: 1850,
      monthly_limit: 120000,
      monthly_used: 28440,
      limited: false,
      exceeded: [],
      enforcement_note:
        "Production v1 customers send directly to Postal. Pre-send quota enforcement is limited until zxMail adds an SMTP gateway in front of Postal.",
    },
    smtp: {
      host: "smtp.zxmail.site",
      starttls_port: "587",
      tls_port: "465",
      username: "apikey_9r3s1j0x",
      password_note:
        "Password is shown only once when the credential is created or rotated.",
    },
  },
  {
    credential: {
      id: "cred_2",
      organization_id: ORG_ID,
      domain_id: "dom_1",
      domain_name: "acme-mail.com",
      username: "apikey_0g4v2m2w",
      label: "Billing notifier",
      enabled: true,
      status: "limited",
      created_at: "2026-05-04T08:20:00Z",
      last_used_at: "2026-05-07T02:42:00Z",
      per_minute_limit: 20,
      per_minute_used: 20,
      daily_limit: 1000,
      daily_used: 1000,
      monthly_limit: 20000,
      monthly_used: 10420,
      limited: true,
      exceeded: ["per_minute", "daily"],
      enforcement_note:
        "Production v1 customers send directly to Postal. Pre-send quota enforcement is limited until zxMail adds an SMTP gateway in front of Postal.",
    },
    smtp: {
      host: "smtp.zxmail.site",
      starttls_port: "587",
      tls_port: "465",
      username: "apikey_0g4v2m2w",
      password_note:
        "Password is shown only once when the credential is created or rotated.",
    },
  },
];

const demoLogs: SendLog[] = [
  buildLog({
    id: "log_1",
    domainId: "dom_1",
    domainName: "acme-mail.com",
    credentialId: "cred_1",
    credentialName: "Primary app",
    status: "accepted",
    to: "maya@acme.test",
    subject: "Verify your email",
    createdAt: "2026-05-07T03:11:21Z",
  }),
  buildLog({
    id: "log_2",
    domainId: "dom_1",
    domainName: "acme-mail.com",
    credentialId: "cred_1",
    credentialName: "Primary app",
    status: "delivered",
    to: "maya@acme.test",
    subject: "Verify your email",
    createdAt: "2026-05-07T03:11:28Z",
  }),
  buildLog({
    id: "log_3",
    domainId: "dom_1",
    domainName: "acme-mail.com",
    credentialId: "cred_2",
    credentialName: "Billing notifier",
    status: "bounced",
    to: "finance@broken-mail.test",
    subject: "Invoice available",
    createdAt: "2026-05-07T02:41:20Z",
    reason: "550 user unknown",
  }),
  buildLog({
    id: "log_4",
    domainId: "dom_2",
    domainName: "alerts.acme-mail.com",
    credentialId: "cred_2",
    credentialName: "Billing notifier",
    status: "deferred",
    to: "ops@delayed-mail.test",
    subject: "Background job completed",
    createdAt: "2026-05-06T22:13:17Z",
    reason: "421 temporary rate limit",
  }),
  buildLog({
    id: "log_5",
    domainId: "dom_1",
    domainName: "acme-mail.com",
    credentialId: "cred_1",
    credentialName: "Primary app",
    status: "rejected",
    to: "blocked@policy-mail.test",
    subject: "Password reset",
    createdAt: "2026-05-06T17:33:40Z",
    reason: "policy reject",
  }),
];

const demoSuppressions: SuppressionRecord[] = [
  {
    id: "sup_1",
    organization_id: ORG_ID,
    recipient: "finance@broken-mail.test",
    source: "bounce",
    reason: "550 user unknown",
    active: true,
    created_at: "2026-05-07T02:41:20Z",
    released_at: null,
  },
  {
    id: "sup_2",
    organization_id: ORG_ID,
    recipient: "qa@legacy-mail.test",
    source: "manual",
    reason: "customer requested stop",
    active: true,
    created_at: "2026-05-05T07:20:00Z",
    released_at: null,
  },
];

let demoOrganizations: OrganizationRecord[] = [
  {
    id: ORG_ID,
    name: "Acme Systems",
    owner_user_id: CUSTOMER_ID,
    owner_email: "owner@acme-mail.com",
    created_at: "2026-05-01T05:00:00Z",
  },
  {
    id: "org_2",
    name: "Northline Labs",
    owner_user_id: "user_2",
    owner_email: "ops@northline.test",
    created_at: "2026-05-03T08:15:00Z",
  },
];

export const demoSystemHealth: SystemHealth = {
  api: "healthy",
  postal: "manual-check",
  redis: "healthy",
  postgres: "healthy",
  notes: [
    "Postal credential provisioning still requires confirmed endpoint contracts before live server creation is automated.",
    "SMTP hostname must remain DNS only in Cloudflare to preserve direct SMTP delivery.",
    "Quota state is authoritative in the dashboard, but Production v1 pre-send enforcement is limited while customers connect directly to Postal.",
  ],
};

export function createPreviewSession(email: string) {
  const role = email.toLowerCase().includes("admin") ? "admin" : "customer";
  const user: AppUser = {
    id: role === "admin" ? ADMIN_ID : CUSTOMER_ID,
    email,
    role,
    organization_id: role === "customer" ? ORG_ID : undefined,
  };

  return {
    user,
  };
}

export async function listDomains() {
  return clone(demoDomains);
}

export async function createDomain(name: string) {
  const domain = buildDomain({
    id: crypto.randomUUID(),
    name,
    verified: false,
    createdAt: new Date().toISOString(),
  });
  demoDomains = [domain, ...demoDomains];
  return clone({
    domain,
    dns_requirements: domain.dns_requirements,
    dns_checks: domain.dns_checks,
    warnings: domain.warnings,
  });
}

export async function verifyDomain(domainID: string) {
  demoDomains = demoDomains.map((domain) => {
    if (domain.id !== domainID) {
      return domain;
    }

    const verifiedDomain: DomainRecord = {
      ...domain,
      verified: true,
      verified_at: new Date().toISOString(),
      dns_checks: domain.dns_checks.map((check) => ({
        ...check,
        found: true,
        found_value: check.expected_value,
        checked_at: new Date().toISOString(),
      })),
    };

    return verifiedDomain;
  });

  return clone(
    demoDomains.find((domain) => domain.id === domainID) ?? demoDomains[0],
  );
}

export async function listCredentials() {
  return clone(demoCredentials);
}

export async function createCredential(input: {
  domain_id: string;
  label: string;
  per_minute_limit?: number | null;
  daily_limit?: number | null;
  monthly_limit?: number | null;
}) {
  const domain = demoDomains.find((entry) => entry.id === input.domain_id);
  if (!domain) {
    throw new Error("domain not found");
  }

  const username = `apikey_${Math.random().toString(36).slice(2, 10)}`;
  const secret = `secret_${Math.random().toString(36).slice(2, 14)}`;
  const response: CredentialSecretResponse = {
    credential: {
      id: crypto.randomUUID(),
      organization_id: ORG_ID,
      domain_id: domain.id,
      domain_name: domain.name,
      username,
      label: input.label,
      enabled: true,
      status: "enabled",
      created_at: new Date().toISOString(),
      last_used_at: null,
      per_minute_limit: input.per_minute_limit ?? null,
      per_minute_used: 0,
      daily_limit: input.daily_limit ?? null,
      daily_used: 0,
      monthly_limit: input.monthly_limit ?? null,
      monthly_used: 0,
      limited: false,
      exceeded: [],
      enforcement_note:
        "Production v1 customers send directly to Postal. Pre-send quota enforcement is limited until zxMail adds an SMTP gateway in front of Postal.",
    },
    smtp: {
      host: "smtp.zxmail.site",
      starttls_port: "587",
      tls_port: "465",
      username,
      password_note:
        "Password is shown only once when the credential is created or rotated.",
    },
    secret,
  };

  demoCredentials = [response, ...demoCredentials];
  return clone(response);
}

export async function listLogs(filters?: Partial<LogsFilterState>) {
  let results = [...demoLogs];

  if (filters?.domain_id) {
    results = results.filter((item) => item.domain_id === filters.domain_id);
  }
  if (filters?.credential_id) {
    results = results.filter(
      (item) => item.credential_id === filters.credential_id,
    );
  }
  if (filters?.message_id) {
    results = results.filter((item) =>
      `${item.message_id_header ?? ""} ${item.postal_message_id ?? ""}`
        .toLowerCase()
        .includes(filters.message_id!.toLowerCase()),
    );
  }
  if (filters?.recipient) {
    results = results.filter((item) =>
      item.to_addr.toLowerCase().includes(filters.recipient!.toLowerCase()),
    );
  }
  if (filters?.status) {
    results = results.filter((item) => item.status === filters.status);
  }
  if (filters?.from) {
    results = results.filter((item) =>
      item.from_addr.toLowerCase().includes(filters.from!.toLowerCase()),
    );
  }
  if (filters?.to) {
    results = results.filter((item) =>
      item.to_addr.toLowerCase().includes(filters.to!.toLowerCase()),
    );
  }
  if (filters?.date_from) {
    results = results.filter(
      (item) => item.created_at >= `${filters.date_from}T00:00:00Z`,
    );
  }
  if (filters?.date_to) {
    results = results.filter(
      (item) => item.created_at <= `${filters.date_to}T23:59:59Z`,
    );
  }

  const limit = filters?.limit ?? 25;
  const offset = filters?.offset ?? 0;

  return clone({
    logs: results.slice(offset, offset + limit),
    total: results.length,
  });
}

export async function listSuppressions() {
  return clone(demoSuppressions);
}

export async function listOrganizations() {
  return clone(demoOrganizations);
}

export async function createOrganization(input: {
  name: string;
  owner_email: string;
  owner_password: string;
}) {
  const organization: OrganizationRecord = {
    id: crypto.randomUUID(),
    name: input.name,
    owner_email: input.owner_email,
    owner_user_id: crypto.randomUUID(),
    created_at: new Date().toISOString(),
  };

  demoOrganizations = [organization, ...demoOrganizations];
  return clone(organization);
}

function buildDomain(input: {
  id: string;
  name: string;
  verified: boolean;
  createdAt: string;
  verifiedAt?: string;
}) {
  const dkimSelector = "zxmail";
  const dnsRequirements = buildDNSRequirements(input.name, dkimSelector);
  const dnsChecks = dnsRequirements
    .filter((record) => record.required)
    .map<DNSCheck>((record) => ({
      id: crypto.randomUUID(),
      record_type: record.type,
      name: record.name,
      expected_value: record.value,
      found: input.verified,
      found_value: input.verified ? record.value : null,
      checked_at: input.verified ? input.verifiedAt ?? input.createdAt : input.createdAt,
    }));

  return {
    id: input.id,
    organization_id: ORG_ID,
    name: input.name,
    verified: input.verified,
    verified_at: input.verified ? input.verifiedAt ?? input.createdAt : null,
    created_at: input.createdAt,
    dkim_selector: dkimSelector,
    dkim_public:
      "v=DKIM1; k=rsa; p=MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAv1PreviewPostalKey",
    spf_record: `v=spf1 include:smtp.zxmail.site ~all`,
    dmarc_record: `v=DMARC1; p=none; rua=mailto:dmarc@${input.name}`,
    warnings: [cloudflareWarning],
    dns_requirements: dnsRequirements,
    dns_checks: dnsChecks,
  } satisfies DomainRecord;
}

function buildDNSRequirements(domain: string, selector: string): DNSRequirement[] {
  return [
    {
      type: "TXT",
      name: domain,
      value: "v=spf1 include:smtp.zxmail.site ~all",
      note: "Required SPF record for Postal-based delivery.",
      required: true,
    },
    {
      type: "TXT",
      name: `${selector}._domainkey.${domain}`,
      value:
        "v=DKIM1; k=rsa; p=MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAv1PreviewPostalKey",
      note: "DKIM placeholder until Postal-managed key material is wired live.",
      required: true,
    },
    {
      type: "TXT",
      name: `_dmarc.${domain}`,
      value: `v=DMARC1; p=none; rua=mailto:dmarc@${domain}`,
      note: "Recommended DMARC baseline for launch.",
      required: true,
    },
    {
      type: "MX",
      name: domain,
      value: `10 route.${domain}`,
      note: "Optional bounce-routing note only. Add if you want explicit mail routing guidance.",
      required: false,
    },
  ];
}

function buildLog(input: {
  id: string;
  domainId: string;
  domainName: string;
  credentialId: string;
  credentialName: string;
  status: SendLog["status"];
  to: string;
  subject: string;
  createdAt: string;
  reason?: string;
}) {
  return {
    id: input.id,
    domain_id: input.domainId,
    domain_name: input.domainName,
    credential_id: input.credentialId,
    credential_name: input.credentialName,
    postal_message_id: `postal_${input.id}`,
    message_id_header: `<${input.id}@${input.domainName}>`,
    from_addr: `noreply@${input.domainName}`,
    to_addr: input.to,
    subject: input.subject,
    status: input.status,
    reason: input.reason,
    created_at: input.createdAt,
    raw_event: {
      event: input.status,
      reason: input.reason,
      message: {
        id: `postal_${input.id}`,
        message_id: `<${input.id}@${input.domainName}>`,
        credential: input.credentialName,
        domain: input.domainName,
      },
    },
  } satisfies SendLog;
}

function clone<T>(value: T): T {
  return JSON.parse(JSON.stringify(value)) as T;
}
