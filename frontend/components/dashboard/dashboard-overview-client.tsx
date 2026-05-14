"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { MetricCard } from "@/components/dashboard/metric-card";
import { PageHero } from "@/components/shared/page-hero";
import { SectionCard } from "@/components/shared/section-card";
import { StatusBadge } from "@/components/shared/status-badge";
import { EmptyState } from "@/components/ui/empty-state";
import { LoadingSkeleton } from "@/components/ui/loading-skeleton";
import { useAuth } from "@/components/providers/auth-provider";
import { formatDateTime } from "@/lib/utils";
import type { CredentialResponse, DomainRecord, SendLog, SuppressionRecord } from "@/types/zxmail";

export function DashboardOverviewClient() {
  const { api } = useAuth();
  const [domains, setDomains] = useState<DomainRecord[]>([]);
  const [credentials, setCredentials] = useState<CredentialResponse[]>([]);
  const [logs, setLogs] = useState<SendLog[]>([]);
  const [suppressions, setSuppressions] = useState<SuppressionRecord[]>([]);
  const [loading, setLoading] = useState(true);

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
      setLoading(false);
    }

    loadOverview().catch(() => {
      if (mounted) {
        setLoading(false);
      }
    });
    return () => {
      mounted = false;
    };
  }, [api]);

  const verifiedDomains = domains.filter((domain) => domain.verified).length;
  const limitedCredentials = credentials.filter(
    (entry) => entry.credential.status === "limited",
  ).length;
  const deliveredCount = logs.filter((log) => log.status === "delivered").length;
  const bounceCount = logs.filter((log) => log.status === "bounced").length;

  return (
    <div className="space-y-6">
      <PageHero
        eyebrow="Dashboard"
        title="Customer workspace for domains, SMTP access, and email observability"
        description="Production v1 keeps the focus on verified domains, Postal-ready credentials, searchable logs, suppression hygiene, and quota posture without drifting into billing or multi-node operations."
        actions={
          <>
            <Link href="/domains/new" className="inline-flex items-center justify-center rounded-[16px] bg-[var(--primary)] px-4 py-2.5 text-sm font-semibold text-white shadow-[0_14px_32px_rgba(23,105,255,0.24)] transition hover:bg-[#1458d6]">
              Start domain wizard
            </Link>
            <Link href="/credentials" className="inline-flex items-center justify-center rounded-[16px] border border-[var(--border)] bg-white px-4 py-2.5 text-sm font-semibold transition hover:bg-[#f8fbff]">
              Manage credentials
            </Link>
          </>
        }
      />

      <section className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <MetricCard
          label="Verified domains"
          value={String(verifiedDomains).padStart(2, "0")}
          note="Domains ready to issue SMTP credentials."
        />
        <MetricCard
          label="Credentials"
          value={String(credentials.length).padStart(2, "0")}
          note="Scoped SMTP identities available for transactional traffic."
        />
        <MetricCard
          label="Delivered events"
          value={String(deliveredCount).padStart(2, "0")}
          note="Latest delivered activity from webhook-backed event processing."
        />
        <MetricCard
          label="Risk posture"
          value={`${limitedCredentials}/${bounceCount}`}
          note="Limited credentials versus bounce events in the latest sample window."
        />
      </section>

      <div className="grid gap-4 xl:grid-cols-[1.1fr_0.9fr]">
        <SectionCard
          title="Readiness snapshot"
          description="The quickest path to sending is verified domains first, then SMTP credentials."
        >
          {loading ? (
            <div className="grid gap-3">
              <LoadingSkeleton className="h-24" />
              <LoadingSkeleton className="h-24" />
              <LoadingSkeleton className="h-24" />
            </div>
          ) : domains.length > 0 ? (
            <div className="space-y-3">
              {domains.slice(0, 3).map((domain) => (
                <div
                  key={domain.id}
                  className="rounded-[22px] border border-[var(--border)] bg-white p-4"
                >
                  <div className="flex items-center justify-between gap-3">
                    <div>
                      <p className="font-semibold">{domain.name}</p>
                      <p className="mt-1 text-sm text-[var(--muted)]">
                        {domain.dns_checks.filter((check) => check.found).length}/{domain.dns_checks.length} required DNS records found
                      </p>
                    </div>
                    <StatusBadge value={domain.verified ? "verified" : "pending"} />
                  </div>
                  <p className="mt-3 text-sm leading-7 text-[var(--muted)]">{domain.warnings[0]}</p>
                </div>
              ))}
            </div>
          ) : (
            <EmptyState
              title="No domains yet"
              description="Create your first sending domain to begin the DNS verification flow."
              action={
                <Link href="/domains/new" className="inline-flex items-center justify-center rounded-[16px] bg-[var(--primary)] px-4 py-2.5 text-sm font-semibold text-white">
                  Add domain
                </Link>
              }
            />
          )}
        </SectionCard>

        <SectionCard
          title="Recent message activity"
          description="Webhook-backed lifecycle events are ready for quick triage from the dashboard."
        >
          {loading ? (
            <div className="grid gap-3">
              <LoadingSkeleton className="h-24" />
              <LoadingSkeleton className="h-24" />
              <LoadingSkeleton className="h-24" />
            </div>
          ) : logs.length > 0 ? (
            <div className="space-y-3">
              {logs.map((log) => (
                <div key={log.id} className="rounded-[22px] border border-[var(--border)] bg-white p-4">
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
          ) : (
            <EmptyState
              title="No events yet"
              description="Accepted, delivered, bounced, deferred, and rejected events will appear here once Postal webhooks start flowing."
            />
          )}
        </SectionCard>
      </div>

      <div className="grid gap-4 xl:grid-cols-[0.95fr_1.05fr]">
        <SectionCard
          title="Credential quota posture"
          description="Track which SMTP identities are close to rate or volume limits."
        >
          {credentials.length > 0 ? (
            <div className="space-y-3">
              {credentials.slice(0, 4).map((entry) => (
                <div key={entry.credential.id} className="rounded-[22px] border border-[var(--border)] bg-white p-4">
                  <div className="flex items-center justify-between gap-3">
                    <div>
                      <p className="font-semibold">{entry.credential.label || entry.credential.username}</p>
                      <p className="mt-1 text-sm text-[var(--muted)]">{entry.credential.domain_name}</p>
                    </div>
                    <StatusBadge value={entry.credential.status} />
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <EmptyState
              title="No credentials yet"
              description="Verified domains can be turned into SMTP credentials here once your DNS is ready."
            />
          )}
        </SectionCard>

        <SectionCard
          title="Suppressions and hygiene"
          description="Keep future sends safe by watching suppressions created from bounce handling."
        >
          {suppressions.length > 0 ? (
            <div className="space-y-3">
              {suppressions.slice(0, 4).map((entry) => (
                <div key={entry.id} className="rounded-[22px] border border-[var(--border)] bg-white p-4">
                  <div className="flex items-center justify-between gap-3">
                    <div>
                      <p className="font-semibold">{entry.recipient}</p>
                      <p className="mt-1 text-sm text-[var(--muted)]">
                        {entry.reason || "No reason provided"} · {formatDateTime(entry.created_at)}
                      </p>
                    </div>
                    <StatusBadge value={entry.active ? "active" : "released"} />
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <EmptyState
              title="No suppressions"
              description="Hard bounces and manual blocks will appear here when they need attention."
            />
          )}
        </SectionCard>
      </div>
    </div>
  );
}
