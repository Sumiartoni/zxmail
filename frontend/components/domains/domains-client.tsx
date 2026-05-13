"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { Button } from "@/components/shared/button";
import { CopyButton } from "@/components/shared/copy-button";
import { PageHero } from "@/components/shared/page-hero";
import { SectionCard } from "@/components/shared/section-card";
import { StatusBadge } from "@/components/shared/status-badge";
import { useAuth } from "@/components/providers/auth-provider";
import { formatDateTime } from "@/lib/utils";
import type { DomainRecord } from "@/types/zxmail";

export function DomainsClient() {
  const { api } = useAuth();
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
      await api.verifyDomain(domainID);
      const nextDomains = await api.listDomains();
      setDomains(nextDomains);
      setError("");
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
        title="Onboard sending domains with DNS-ready record output"
        description="Add custom domains, hand off exact SPF, DKIM, and DMARC values, then run backend verification checks against public resolvers."
        actions={
          <Link
            href="/domains/new"
            className="inline-flex items-center justify-center rounded-full bg-[var(--accent)] px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-[#9f411e]"
          >
            New domain wizard
          </Link>
        }
      />

      {error ? (
        <div className="rounded-3xl border border-[#d8ad9f] bg-[#fff1ec] px-4 py-3 text-sm text-[#8d2d11]">
          {error}
        </div>
      ) : null}

      <div className="grid gap-4 xl:grid-cols-2">
        {domains.map((domain) => (
          <SectionCard
            key={domain.id}
            title={domain.name}
            description={`Created ${formatDateTime(domain.created_at)}${domain.verified_at ? `, verified ${formatDateTime(domain.verified_at)}` : ""}`}
          >
            <div className="space-y-4">
              <div className="flex flex-wrap items-center gap-3">
                <StatusBadge value={domain.verified ? "verified" : "pending"} />
                <span className="text-sm text-[var(--muted)]">
                  {domain.dns_checks.filter((check) => check.found).length}/{domain.dns_checks.length} required records found
                </span>
              </div>

              <div className="grid gap-3">
                {domain.dns_requirements.slice(0, 3).map((record) => (
                  <div
                    key={`${domain.id}-${record.name}`}
                    className="rounded-2xl border border-[var(--line)] bg-white/75 p-4"
                  >
                    <div className="flex items-start justify-between gap-3">
                      <div>
                        <p className="eyebrow">{record.type}</p>
                        <p className="mt-1 font-semibold">{record.name}</p>
                      </div>
                      <CopyButton value={record.value} />
                    </div>
                    <p className="mt-3 font-mono text-xs leading-6 text-[var(--muted)]">
                      {record.value}
                    </p>
                  </div>
                ))}
              </div>

              <div className="rounded-2xl border border-[#e0bb8e] bg-[#fff4df] p-4 text-sm leading-7 text-[#7b5a13]">
                {domain.warnings[0]}
              </div>

              <div className="flex flex-wrap gap-3">
                <Button disabled={loading} onClick={() => verifyDomain(domain.id)}>
                  {domain.verified ? "Re-check DNS" : "Verify domain"}
                </Button>
                <Link
                  href="/domains/new"
                  className="inline-flex items-center justify-center rounded-full border border-[var(--line)] bg-white/70 px-4 py-2.5 text-sm font-semibold transition hover:bg-white"
                >
                  Open wizard
                </Link>
              </div>
            </div>
          </SectionCard>
        ))}
      </div>

      {!loading && domains.length === 0 ? (
        <SectionCard
          title="No domains yet"
          description="Start the onboarding wizard to create the first verified sending identity."
        >
          <Link
            href="/domains/new"
            className="inline-flex items-center justify-center rounded-full bg-[var(--accent)] px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-[#9f411e]"
          >
            Start domain wizard
          </Link>
        </SectionCard>
      ) : null}
    </div>
  );
}
