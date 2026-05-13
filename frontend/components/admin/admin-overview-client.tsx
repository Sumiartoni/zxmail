"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { MetricCard } from "@/components/dashboard/metric-card";
import { PageHero } from "@/components/shared/page-hero";
import { SectionCard } from "@/components/shared/section-card";
import { StatusBadge } from "@/components/shared/status-badge";
import { useAuth } from "@/components/providers/auth-provider";
import { formatDateTime } from "@/lib/utils";
import type { CredentialResponse, DomainRecord, OrganizationRecord, SendLog } from "@/types/zxmail";

export function AdminOverviewClient() {
  const { api } = useAuth();
  const [organizations, setOrganizations] = useState<OrganizationRecord[]>([]);
  const [domains, setDomains] = useState<DomainRecord[]>([]);
  const [credentials, setCredentials] = useState<CredentialResponse[]>([]);
  const [logs, setLogs] = useState<SendLog[]>([]);

  useEffect(() => {
    let mounted = true;

    async function loadAdminOverview() {
      const [nextOrganizations, nextDomains, nextCredentials, logsResult] =
        await Promise.all([
          api.listOrganizations(),
          api.listDomains(),
          api.listCredentials(),
          api.listLogs({ limit: 5, offset: 0 }),
        ]);
      if (!mounted) {
        return;
      }
      setOrganizations(nextOrganizations);
      setDomains(nextDomains);
      setCredentials(nextCredentials);
      setLogs(logsResult.logs);
    }

    loadAdminOverview().catch(() => {});
    return () => {
      mounted = false;
    };
  }, [api]);

  const deliveredCount = logs.filter((log) => log.status === "delivered").length;
  const bounceCount = logs.filter((log) => log.status === "bounced").length;

  return (
    <div className="space-y-6">
      <PageHero
        eyebrow="Admin overview"
        title="Operator view across organizations, credentials, and delivery events"
        description="Production v1 keeps the global operator surface intentionally narrow: organization management, logs, system health, quotas, and deliverability basics."
        actions={
          <>
            <Link
              href="/admin/customers"
              className="inline-flex items-center justify-center rounded-full bg-[var(--accent)] px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-[#9f411e]"
            >
              Manage customers
            </Link>
            <Link
              href="/admin/system"
              className="inline-flex items-center justify-center rounded-full border border-[var(--line)] bg-white/70 px-4 py-2.5 text-sm font-semibold transition hover:bg-white"
            >
              System health
            </Link>
          </>
        }
      />

      <section className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <MetricCard
          label="Organizations"
          value={String(organizations.length).padStart(2, "0")}
          note="Customer organizations currently provisioned inside the control plane."
        />
        <MetricCard
          label="Verified domains"
          value={String(domains.filter((domain) => domain.verified).length).padStart(2, "0")}
          note="Sending domains that passed required TXT checks."
        />
        <MetricCard
          label="Active credentials"
          value={String(credentials.filter((item) => item.credential.enabled).length).padStart(2, "0")}
          note="Enabled SMTP identities across all organizations."
        />
        <MetricCard
          label="Bounce events"
          value={String(bounceCount).padStart(2, "0")}
          note={`${deliveredCount} delivered events in the same current sample window.`}
        />
      </section>

      <div className="grid gap-4 xl:grid-cols-[0.95fr_1.05fr]">
        <SectionCard
          title="Newest customer organizations"
          description="Admin-created customers own their organization and can only access their own scoped data."
        >
          <div className="space-y-3">
            {organizations.slice(0, 4).map((organization) => (
              <div
                key={organization.id}
                className="rounded-2xl border border-[var(--line)] bg-white/70 p-4"
              >
                <div className="flex items-center justify-between gap-3">
                  <div>
                    <p className="font-semibold">{organization.name}</p>
                    <p className="mt-1 text-sm text-[var(--muted)]">
                      {organization.owner_email} · {formatDateTime(organization.created_at)}
                    </p>
                  </div>
                  <StatusBadge value="active" />
                </div>
              </div>
            ))}
          </div>
        </SectionCard>

        <SectionCard
          title="Latest message activity"
          description="Use the admin logs page for full filtering by domain, credential, message ID, recipient, and date range."
        >
          <div className="space-y-3">
            {logs.map((log) => (
              <div
                key={log.id}
                className="rounded-2xl border border-[var(--line)] bg-white/70 p-4"
              >
                <div className="flex items-center justify-between gap-3">
                  <div>
                    <p className="font-semibold">{log.subject || "No subject"}</p>
                    <p className="mt-1 text-sm text-[var(--muted)]">
                      {log.domain_name} · {log.to_addr}
                    </p>
                  </div>
                  <StatusBadge value={log.status} />
                </div>
              </div>
            ))}
          </div>
        </SectionCard>
      </div>
    </div>
  );
}
