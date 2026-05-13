"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { MetricCard } from "@/components/dashboard/metric-card";
import { PageHero } from "@/components/shared/page-hero";
import { SectionCard } from "@/components/shared/section-card";
import { StatusBadge } from "@/components/shared/status-badge";
import { useAuth } from "@/components/providers/auth-provider";
import { formatDateTime } from "@/lib/utils";
import type { CredentialResponse, DomainRecord, SendLog, SuppressionRecord } from "@/types/zxmail";

export function DashboardOverviewClient() {
  const { api } = useAuth();
  const [domains, setDomains] = useState<DomainRecord[]>([]);
  const [credentials, setCredentials] = useState<CredentialResponse[]>([]);
  const [logs, setLogs] = useState<SendLog[]>([]);
  const [suppressions, setSuppressions] = useState<SuppressionRecord[]>([]);

  useEffect(() => {
    let mounted = true;

    async function loadOverview() {
      const [nextDomains, nextCredentials, logsResult, nextSuppressions] =
        await Promise.all([
          api.listDomains(),
          api.listCredentials(),
          api.listLogs({ limit: 5, offset: 0 }),
          api.listSuppressions(),
        ]);
      if (!mounted) {
        return;
      }
      setDomains(nextDomains);
      setCredentials(nextCredentials);
      setLogs(logsResult.logs);
      setSuppressions(nextSuppressions);
    }

    loadOverview().catch(() => {});
    return () => {
      mounted = false;
    };
  }, [api]);

  const verifiedDomains = domains.filter((domain) => domain.verified).length;
  const limitedCredentials = credentials.filter(
    (entry) => entry.credential.status === "limited",
  ).length;

  return (
    <div className="space-y-6">
      <PageHero
        eyebrow="Dashboard"
        title="Customer workspace for onboarding, credentials, and observability"
        description="Production v1 keeps the focus on verified domains, Postal-ready credentials, searchable logs, and suppression hygiene without Phase 2 billing or multi-node operations."
        actions={
          <>
            <Link
              href="/domains/new"
              className="inline-flex items-center justify-center rounded-full bg-[var(--accent)] px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-[#9f411e]"
            >
              Start domain wizard
            </Link>
            <Link
              href="/credentials"
              className="inline-flex items-center justify-center rounded-full border border-[var(--line)] bg-white/70 px-4 py-2.5 text-sm font-semibold transition hover:bg-white"
            >
              Manage credentials
            </Link>
          </>
        }
      />

      <section className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <MetricCard
          label="Verified domains"
          value={String(verifiedDomains).padStart(2, "0")}
          note="Domains that have passed required TXT verification checks."
        />
        <MetricCard
          label="Credentials"
          value={String(credentials.length).padStart(2, "0")}
          note="Scoped SMTP identities available for Production v1 transactional traffic."
        />
        <MetricCard
          label="Limited credentials"
          value={String(limitedCredentials).padStart(2, "0")}
          note="Credentials that reached at least one minute, daily, or monthly threshold."
        />
        <MetricCard
          label="Suppressions"
          value={String(suppressions.length).padStart(2, "0")}
          note="Recipients blocked because of bounce handling or manual safety actions."
        />
      </section>

      <div className="grid gap-4 xl:grid-cols-[1.1fr_0.9fr]">
        <SectionCard
          title="Recent domain state"
          description="Quick view of verification progress and the Cloudflare DNS-only rule."
        >
          <div className="space-y-3">
            {domains.slice(0, 3).map((domain) => (
              <div
                key={domain.id}
                className="rounded-2xl border border-[var(--line)] bg-white/70 p-4"
              >
                <div className="flex items-center justify-between gap-3">
                  <div>
                    <p className="font-semibold">{domain.name}</p>
                    <p className="mt-1 text-sm text-[var(--muted)]">
                      {domain.dns_checks.filter((check) => check.found).length}/
                      {domain.dns_checks.length} required records found
                    </p>
                  </div>
                  <StatusBadge value={domain.verified ? "verified" : "pending"} />
                </div>
                <p className="mt-3 text-sm leading-7 text-[var(--muted)]">
                  {domain.warnings[0]}
                </p>
              </div>
            ))}
          </div>
        </SectionCard>

        <SectionCard
          title="Recent email events"
          description="Webhook-backed lifecycle events surfaced with searchable status badges."
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
                      {log.to_addr} · {formatDateTime(log.created_at)}
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
