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
  AppUser,
  AuthSession,
  CredentialResponse,
  CredentialSecretResponse,
  DomainRecord,
  LogsFilterState,
  OrganizationRecord,
  SendLog,
  SuppressionRecord,
  SystemHealth,
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
