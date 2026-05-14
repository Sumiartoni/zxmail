"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { Button } from "@/components/shared/button";
import { PageHero } from "@/components/shared/page-hero";
import { SectionCard } from "@/components/shared/section-card";
import { StatusBadge } from "@/components/shared/status-badge";
import { useToast } from "@/components/providers/toast-provider";
import { DNSRecordCard } from "@/components/ui/dns-record-card";
import { EmptyState } from "@/components/ui/empty-state";
import { ErrorState } from "@/components/ui/error-state";
import { LoadingSkeleton } from "@/components/ui/loading-skeleton";
import { useAuth } from "@/components/providers/auth-provider";
import { formatDateTime } from "@/lib/utils";
import type { DomainRecord } from "@/types/zxmail";

export function DomainsClient() {
  const { api } = useAuth();
  const { pushToast } = useToast();
  const [domains, setDomains] = useState<DomainRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    let mounted = true;

    async function loadDomains() {
      try {
        const nextDomains = await api.listDomains();
        if (mounted) {
          setDomains(nextDomains);
        }
      } catch (nextError) {
        if (mounted) {
          setError(nextError instanceof Error ? nextError.message : "Failed to load domains");
        }
      } finally {
        if (mounted) {
          setLoading(false);
        }
      }
    }

    loadDomains();
    return () => {
      mounted = false;
    };
  }, [api]);

  async function verifyDomain(domainID: string) {
    setLoading(true);
    try {
      const result = await api.verifyDomain(domainID);
      const nextDomains = await api.listDomains();
      setDomains(nextDomains);
      setError("");
      pushToast({
        title: result.verified ? "Domain verified" : "Verification completed",
        description: result.verified
          ? "All required DNS records were found."
          : "Some required DNS records are still missing.",
        tone: result.verified ? "success" : "info",
      });
    } catch (nextError) {
      setError(nextError instanceof Error ? nextError.message : "Verification failed");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="space-y-6">
      <PageHero
        eyebrow="Domains"
        title="Add, verify, and activate sending domains"
        description="Each domain stays guided from record generation to DNS verification so teams can get to SMTP credentials without guessing the next step."
        actions={
          <Link href="/domains/new" className="inline-flex items-center justify-center rounded-[16px] bg-[var(--primary)] px-4 py-2.5 text-sm font-semibold text-white shadow-[0_14px_32px_rgba(23,105,255,0.24)] transition hover:bg-[#1458d6]">
            New domain wizard
          </Link>
        }
      />

      {error ? <ErrorState description={error} retryLabel="Retry" onRetry={() => void api.listDomains().then(setDomains)} /> : null}

      {loading ? (
        <div className="grid gap-4">
          <LoadingSkeleton className="h-64" />
          <LoadingSkeleton className="h-64" />
        </div>
      ) : null}

      {!loading && domains.length > 0 ? (
        <div className="grid gap-4">
          {domains.map((domain) => (
            <SectionCard
              key={domain.id}
              title={domain.name}
              description={`Created ${formatDateTime(domain.created_at)}${domain.verified_at ? ` · verified ${formatDateTime(domain.verified_at)}` : ""}`}
            >
              <div className="space-y-5">
                <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
                  <div className="flex flex-wrap items-center gap-3">
                    <StatusBadge value={domain.verified ? "verified" : "pending"} />
                    <span className="text-sm text-[var(--muted)]">
                      {domain.dns_checks.filter((check) => check.found).length}/{domain.dns_checks.length} required records found
                    </span>
                    <span className="text-sm text-[var(--muted)]">DKIM selector: {domain.dkim_selector}</span>
                  </div>
                  <div className="flex flex-wrap gap-3">
                    <Button disabled={loading} onClick={() => verifyDomain(domain.id)}>
                      {domain.verified ? "Re-check DNS" : "Verify domain"}
                    </Button>
                    <Link href="/credentials" className="inline-flex items-center justify-center rounded-[16px] border border-[var(--border)] bg-white px-4 py-2.5 text-sm font-semibold transition hover:bg-[#f8fbff]">
                      Create credential
                    </Link>
                  </div>
                </div>

                <div className="rounded-[24px] border border-[rgba(184,92,0,0.15)] bg-[rgba(184,92,0,0.06)] p-4 text-sm leading-7 text-[var(--foreground)]">
                  <p className="font-semibold">Cloudflare warning</p>
                  <p className="mt-2">{domain.warnings[0]}</p>
                </div>

                <div className="grid gap-4 xl:grid-cols-2">
                  {domain.dns_requirements.map((record) => (
                    <DNSRecordCard
                      key={`${domain.id}-${record.type}-${record.name}`}
                      record={record}
                      check={domain.dns_checks.find((check) => check.name === record.name)}
                      verified={domain.verified}
                    />
                  ))}
                </div>
              </div>
            </SectionCard>
          ))}
        </div>
      ) : null}

      {!loading && domains.length === 0 ? (
        <EmptyState
          title="No domains yet"
          description="Start the onboarding wizard to generate SPF, DKIM placeholder, DMARC, and the resolver-backed verification path."
          action={
            <Link href="/domains/new" className="inline-flex items-center justify-center rounded-[16px] bg-[var(--primary)] px-4 py-2.5 text-sm font-semibold text-white">
              Start domain wizard
            </Link>
          }
        />
      ) : null}
    </div>
  );
}
