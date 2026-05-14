import {
  createCredential,
  createDomain,
  createOrganization,
  createPreviewSession,
  demoSystemHealth,
  listCredentials,
  listDomains,
  listLogs,
  listOrganizations,
  listSuppressions,
  verifyDomain,
} from "@/lib/demo-data";
import type {
  AdminOrganizationDetail,
  AdminOverview,
  AdminSystemHealth,
  AlertRecord,
  AuditLogRecord,
  AppUser,
  AuthSession,
  CleanupResult,
  CredentialResponse,
  CredentialSecretResponse,
  DeliverabilityOverview,
  DomainRecord,
  DomainHealthRecord,
  InvoiceRecord,
  LogsFilterState,
  OrganizationRecord,
  PaymentRecord,
  PlanRecord,
  QueueHealth,
  RetentionPolicy,
  RiskRecord,
  SendLog,
  SubscriptionView,
  SuppressionRecord,
  SystemHealth,
  UsageOverview,
} from "@/types/zxmail";

const baseUrl = process.env.NEXT_PUBLIC_API_BASE_URL?.replace(/\/$/, "") ?? "";
const nodeEnv = process.env.NODE_ENV ?? "development";

export const previewMode = nodeEnv !== "production" && baseUrl === "";

export type CreateCredentialPayload = {
  domain_id: string;
  label: string;
  per_minute_limit?: number | null;
  daily_limit?: number | null;
  monthly_limit?: number | null;
};

export type CreateOrganizationPayload = {
  name: string;
  owner_email: string;
  owner_password: string;
};

export type PlanPayload = {
  code: string;
  name: string;
  description: string;
  active?: boolean;
  currency: string;
  price_monthly: number;
  daily_quota?: number | null;
  monthly_quota?: number | null;
  per_minute_quota?: number | null;
  credential_quota?: number | null;
  trial_days: number;
  overage_price_per_email: number;
  payment_methods: string[];
};

export type SubscriptionAssignmentPayload = {
  plan_id: string;
  payment_provider: string;
  notes?: string;
  start_trial?: boolean;
};

export class ZxMailApiClient {
  async login(email: string, password: string): Promise<AuthSession> {
    if (previewMode) {
      if (!email.trim() || !password.trim()) {
        throw new Error("Email and password are required.");
      }
      return createPreviewSession(email.trim().toLowerCase());
    }

    return this.request<AuthSession>("/api/v1/auth/login", {
      method: "POST",
      body: JSON.stringify({ email, password }),
    });
  }

  async logout(): Promise<void> {
    if (previewMode) {
      return;
    }

    await this.request<{ success: boolean }>("/api/v1/auth/logout", {
      method: "POST",
    });
  }

  async me(): Promise<AppUser> {
    if (previewMode) {
      throw new Error("preview mode does not persist browser sessions");
    }

    const response = await this.request<{ user: AppUser }>("/api/v1/me");
    return response.user;
  }

  async listDomains(): Promise<DomainRecord[]> {
    if (previewMode) {
      return listDomains();
    }

    const response = await this.request<{ domains: DomainRecord[] }>("/api/v1/domains");
    return response.domains;
  }

  async createDomain(name: string) {
    if (previewMode) {
      return createDomain(name);
    }

    return this.request<{
      domain: DomainRecord;
      dns_requirements: DomainRecord["dns_requirements"];
      dns_checks: DomainRecord["dns_checks"];
      warnings: string[];
    }>("/api/v1/domains", {
      method: "POST",
      body: JSON.stringify({ name }),
    });
  }

  async verifyDomain(domainID: string) {
    if (previewMode) {
      const domain = await verifyDomain(domainID);
      return {
        status: domain.verified ? "verified" : "pending",
        verified: domain.verified,
        verified_at: domain.verified_at ?? null,
        required_records_total: domain.dns_checks.length,
        required_records_found: domain.dns_checks.filter((check) => check.found).length,
        dns_checks: domain.dns_checks,
        warnings: domain.warnings,
      };
    }

    return this.request<{
      status: string;
      verified: boolean;
      verified_at?: string | null;
      required_records_total: number;
      required_records_found: number;
      dns_checks: DomainRecord["dns_checks"];
      warnings: string[];
    }>(`/api/v1/domains/${domainID}/verify`, {
      method: "POST",
    });
  }

  async listCredentials(): Promise<CredentialResponse[]> {
    if (previewMode) {
      return listCredentials();
    }

    const response = await this.request<{ credentials: CredentialResponse[] }>(
      "/api/v1/credentials",
    );
    return response.credentials;
  }

  async createCredential(payload: CreateCredentialPayload): Promise<CredentialSecretResponse> {
    if (previewMode) {
      return createCredential(payload);
    }

    return this.request<CredentialSecretResponse>("/api/v1/credentials", {
      method: "POST",
      body: JSON.stringify(payload),
    });
  }

  async listLogs(filters: Partial<LogsFilterState>) {
    if (previewMode) {
      return listLogs(filters);
    }

    const query = new URLSearchParams();
    for (const [key, value] of Object.entries(filters)) {
      if (value === "" || value === undefined || value === null) {
        continue;
      }
      query.set(key, String(value));
    }

    const response = await this.request<{ logs: SendLog[]; total?: number }>(
      `/api/v1/logs${query.size > 0 ? `?${query.toString()}` : ""}`,
    );
    return {
      logs: response.logs,
      total: response.total ?? response.logs.length,
    };
  }

  async listSuppressions(): Promise<SuppressionRecord[]> {
    if (previewMode) {
      return listSuppressions();
    }

    const response = await this.request<{ suppressions: SuppressionRecord[] }>(
      "/api/v1/suppressions",
    );
    return response.suppressions;
  }

  async listOrganizations(): Promise<OrganizationRecord[]> {
    if (previewMode) {
      return listOrganizations();
    }

    const response = await this.request<{ organizations: OrganizationRecord[] }>(
      "/api/v1/admin/organizations",
    );
    return response.organizations;
  }

  async getOrganization(): Promise<OrganizationRecord> {
    const response = await this.request<{ organization: OrganizationRecord }>("/api/v2/organization");
    return response.organization;
  }

  async createOrganization(payload: CreateOrganizationPayload): Promise<OrganizationRecord> {
    if (previewMode) {
      return createOrganization(payload);
    }

    const response = await this.request<{ organization: OrganizationRecord }>(
      "/api/v1/admin/organizations",
      {
        method: "POST",
        body: JSON.stringify(payload),
      },
    );
    return response.organization;
  }

  async health(): Promise<SystemHealth> {
    if (previewMode) {
      return demoSystemHealth;
    }

    const liveResponse = await this.request<{ status: string }>("/health");
    const readyResponse = await this.fetchJSON<{
      status: string;
      checks?: {
        postgres?: { ready?: boolean };
        redis?: { ready?: boolean };
      };
    }>("/health/ready");

    return {
      api: liveResponse.status === "ok" ? "healthy" : "degraded",
      postal: "manual-check",
      redis: readyResponse.checks?.redis?.ready ? "healthy" : "degraded",
      postgres: readyResponse.checks?.postgres?.ready ? "healthy" : "degraded",
      notes: [
        "Postal live API provisioning still needs confirmed server and credential contracts.",
        "Liveness is served by /health. Dependency readiness is served by /health/ready.",
      ],
    };
  }

  async listPlans(): Promise<PlanRecord[]> {
    if (previewMode) {
      return [];
    }
    const response = await this.request<{ plans: PlanRecord[] }>("/api/v2/plans");
    return response.plans;
  }

  async createPlan(payload: PlanPayload): Promise<PlanRecord> {
    const response = await this.request<{ plan: PlanRecord }>("/api/v2/admin/plans", {
      method: "POST",
      body: JSON.stringify(payload),
    });
    return response.plan;
  }

  async updatePlan(planID: string, payload: PlanPayload): Promise<PlanRecord> {
    const response = await this.request<{ plan: PlanRecord }>(`/api/v2/admin/plans/${planID}`, {
      method: "PATCH",
      body: JSON.stringify(payload),
    });
    return response.plan;
  }

  async assignSubscription(organizationID: string, payload: SubscriptionAssignmentPayload): Promise<SubscriptionView> {
    return this.request<SubscriptionView>(`/api/v2/admin/organizations/${organizationID}/subscription`, {
      method: "POST",
      body: JSON.stringify(payload),
    });
  }

  async getSubscription(): Promise<SubscriptionView | null> {
    if (previewMode) {
      return null;
    }
    try {
      return await this.request<SubscriptionView>("/api/v2/subscription");
    } catch (error) {
      if (error instanceof Error && error.message === "subscription not found") {
        return null;
      }
      throw error;
    }
  }

  async listInvoices(): Promise<InvoiceRecord[]> {
    if (previewMode) {
      return [];
    }
    const response = await this.request<{ invoices: InvoiceRecord[] }>("/api/v2/invoices");
    return response.invoices;
  }

  async listAdminInvoices(): Promise<InvoiceRecord[]> {
    if (previewMode) {
      return [];
    }
    const response = await this.request<{ invoices: InvoiceRecord[] }>("/api/v2/admin/invoices");
    return response.invoices;
  }

  async markInvoicePaid(invoiceID: string): Promise<InvoiceRecord> {
    const response = await this.request<{ invoice: InvoiceRecord }>(`/api/v2/admin/invoices/${invoiceID}/mark-paid`, {
      method: "POST",
    });
    return response.invoice;
  }

  async markInvoiceFailed(invoiceID: string): Promise<InvoiceRecord> {
    const response = await this.request<{ invoice: InvoiceRecord }>(`/api/v2/admin/invoices/${invoiceID}/mark-failed`, {
      method: "POST",
    });
    return response.invoice;
  }

  async listPayments(): Promise<PaymentRecord[]> {
    if (previewMode) {
      return [];
    }
    const response = await this.request<{ payments: PaymentRecord[] }>("/api/v2/admin/payments");
    return response.payments;
  }

  async approvePayment(paymentID: string): Promise<PaymentRecord> {
    const response = await this.request<{ payment: PaymentRecord }>(`/api/v2/admin/payments/${paymentID}/approve`, {
      method: "POST",
    });
    return response.payment;
  }

  async rejectPayment(paymentID: string): Promise<PaymentRecord> {
    const response = await this.request<{ payment: PaymentRecord }>(`/api/v2/admin/payments/${paymentID}/reject`, {
      method: "POST",
    });
    return response.payment;
  }

  async getUsage(): Promise<UsageOverview> {
    return (await this.request<{ usage: UsageOverview }>("/api/v2/usage")).usage;
  }

  async getOrganizationUsage(organizationID: string): Promise<UsageOverview> {
    return (await this.request<{ usage: UsageOverview }>(`/api/v2/admin/organizations/${organizationID}/usage`)).usage;
  }

  async updateOrganizationQuota(organizationID: string, payload: {
    daily_quota?: number | null;
    monthly_quota?: number | null;
    per_minute_quota?: number | null;
    reason: string;
  }): Promise<UsageOverview> {
    return (await this.request<{ usage: UsageOverview }>(`/api/v2/admin/organizations/${organizationID}/quota`, {
      method: "PATCH",
      body: JSON.stringify(payload),
    })).usage;
  }

  async resetOrganizationUsage(organizationID: string): Promise<void> {
    await this.request<{ success: boolean }>(`/api/v2/admin/organizations/${organizationID}/reset-usage`, {
      method: "POST",
    });
  }

  async limitCredential(credentialID: string, reason: string): Promise<void> {
    await this.request<{ success: boolean }>(`/api/v2/admin/credentials/${credentialID}/limit`, {
      method: "POST",
      body: JSON.stringify({ reason }),
    });
  }

  async unlimitCredential(credentialID: string, reason: string): Promise<void> {
    await this.request<{ success: boolean }>(`/api/v2/admin/credentials/${credentialID}/unlimit`, {
      method: "POST",
      body: JSON.stringify({ reason }),
    });
  }

  async getDeliverabilityOverview(): Promise<DeliverabilityOverview> {
    return (await this.request<{ overview: DeliverabilityOverview }>("/api/v2/deliverability/overview")).overview;
  }

  async getDomainHealth(domainID: string): Promise<DomainHealthRecord> {
    return (await this.request<{ domain_health: DomainHealthRecord }>(`/api/v2/domains/${domainID}/health`)).domain_health;
  }

  async recheckDomain(domainID: string): Promise<DomainHealthRecord> {
    return (await this.request<{ domain_health: DomainHealthRecord }>(`/api/v2/domains/${domainID}/recheck`, {
      method: "POST",
    })).domain_health;
  }

  async listAlerts(): Promise<AlertRecord[]> {
    if (previewMode) {
      return [];
    }
    const response = await this.request<{ alerts: AlertRecord[] }>("/api/v2/alerts");
    return response.alerts;
  }

  async getAdminOverview(): Promise<AdminOverview> {
    return (await this.request<{ overview: AdminOverview }>("/api/v2/admin/overview")).overview;
  }

  async getAdminOrganizationDetail(organizationID: string): Promise<AdminOrganizationDetail> {
    return (await this.request<{ organization: AdminOrganizationDetail }>(`/api/v2/admin/organizations/${organizationID}/detail`)).organization;
  }

  async suspendOrganization(organizationID: string, reason: string): Promise<void> {
    await this.request<{ success: boolean }>(`/api/v2/admin/organizations/${organizationID}/suspend`, {
      method: "POST",
      body: JSON.stringify({ reason }),
    });
  }

  async unsuspendOrganization(organizationID: string): Promise<void> {
    await this.request<{ success: boolean }>(`/api/v2/admin/organizations/${organizationID}/unsuspend`, {
      method: "POST",
    });
  }

  async disableOrganizationCredentials(organizationID: string): Promise<void> {
    await this.request<{ success: boolean }>(`/api/v2/admin/organizations/${organizationID}/disable-credentials`, {
      method: "POST",
    });
  }

  async forceRotateCredential(credentialID: string): Promise<CredentialSecretResponse> {
    return this.request<CredentialSecretResponse>(`/api/v2/admin/credentials/${credentialID}/force-rotate`, {
      method: "POST",
    });
  }

  async listRiskOrganizations(): Promise<RiskRecord[]> {
    if (previewMode) {
      return [];
    }
    const response = await this.request<{ organizations: RiskRecord[] }>("/api/v2/admin/risk/organizations");
    return response.organizations;
  }

  async getAdminDeliverabilityOverview(): Promise<DeliverabilityOverview> {
    return (await this.request<{ overview: DeliverabilityOverview }>("/api/v2/admin/deliverability")).overview;
  }

  async listAdminDomainHealth(): Promise<DomainHealthRecord[]> {
    if (previewMode) {
      return [];
    }
    const response = await this.request<{ domains: DomainHealthRecord[] }>("/api/v2/admin/domains/health");
    return response.domains;
  }

  async recheckAllDomains(): Promise<void> {
    await this.request<{ success: boolean }>("/api/v2/admin/domains/recheck-all", { method: "POST" });
  }

  async resolveAlert(alertID: string): Promise<void> {
    await this.request<{ success: boolean }>(`/api/v2/admin/alerts/${alertID}/resolve`, { method: "POST" });
  }

  async listRetentionPolicies(): Promise<RetentionPolicy[]> {
    if (previewMode) {
      return [];
    }
    const response = await this.request<{ policies: RetentionPolicy[] }>("/api/v2/admin/retention");
    return response.policies;
  }

  async updateRetentionPolicy(organizationID: string, retentionDays: number): Promise<void> {
    await this.request<{ success: boolean }>(`/api/v2/admin/organizations/${organizationID}/retention`, {
      method: "PATCH",
      body: JSON.stringify({ retention_days: retentionDays }),
    });
  }

  async runRetentionCleanup(dryRun = true): Promise<CleanupResult> {
    return (await this.request<{ cleanup: CleanupResult }>(`/api/v2/admin/retention/run-cleanup?dry_run=${dryRun}`, {
      method: "POST",
    })).cleanup;
  }

  async listAuditLogs(limit = 100): Promise<AuditLogRecord[]> {
    if (previewMode) {
      return [];
    }
    const response = await this.request<{ audit_logs: AuditLogRecord[] }>(`/api/v2/admin/audit-logs?limit=${limit}`);
    return response.audit_logs;
  }

  async getAdminSystemHealth(): Promise<AdminSystemHealth> {
    return (await this.request<{ health: AdminSystemHealth }>("/api/v2/admin/system/health")).health;
  }

  async getQueueHealth(): Promise<QueueHealth> {
    return (await this.request<{ queues: QueueHealth }>("/api/v2/admin/system/queues")).queues;
  }

  async getPostalHealth(): Promise<{ reachable: boolean; note: string }> {
    return (await this.request<{ postal: { reachable: boolean; note: string } }>("/api/v2/admin/system/postal-health")).postal;
  }

  private async request<T>(path: string, init?: RequestInit): Promise<T> {
    const response = await this.fetch(path, init);
    if (!response.ok) {
      let message = "Request failed";
      try {
        const payload = (await response.json()) as { error?: string };
        if (payload.error) {
          message = payload.error;
        }
      } catch {}
      throw new Error(message);
    }

    return response.json() as Promise<T>;
  }

  private async fetchJSON<T>(path: string, init?: RequestInit): Promise<T> {
    const response = await this.fetch(path, init);
    return response.json() as Promise<T>;
  }

  private fetch(path: string, init?: RequestInit): Promise<Response> {
    const headers = new Headers(init?.headers);
    headers.set("Content-Type", "application/json");

    return fetch(`${baseUrl}${path}`, {
      ...init,
      headers,
      cache: "no-store",
      credentials: "include",
    });
  }
}
