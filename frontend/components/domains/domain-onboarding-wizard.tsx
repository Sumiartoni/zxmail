"use client";

import { useState } from "react";
import Link from "next/link";
import { Button } from "@/components/shared/button";
import { CopyButton } from "@/components/shared/copy-button";
import { SectionCard } from "@/components/shared/section-card";
import { StatusBadge } from "@/components/shared/status-badge";
import { useAuth } from "@/components/providers/auth-provider";
import { formatDateTime } from "@/lib/utils";
import type { DomainRecord } from "@/types/zxmail";

export function DomainOnboardingWizard() {
  const { api } = useAuth();
  const [step, setStep] = useState(1);
  const [domainName, setDomainName] = useState("");
  const [domain, setDomain] = useState<DomainRecord | null>(null);
  const [loading, setLoading] = useState(false);
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");

  async function handleCreateDomain() {
    setLoading(true);
    setError("");
    setMessage("");
    try {
      const response = await api.createDomain(domainName);
      setDomain(response.domain);
      setStep(2);
      setMessage("Domain created. Add the records below before verifying.");
    } catch (nextError) {
      setError(nextError instanceof Error ? nextError.message : "Failed to create domain");
    } finally {
      setLoading(false);
    }
  }

  async function handleVerifyDomain() {
    if (!domain) {
      return;
    }

    setLoading(true);
    setError("");
    setMessage("");
    try {
      const result = await api.verifyDomain(domain.id);
      setDomain((current) =>
        current
          ? {
              ...current,
              verified: result.verified,
              verified_at: result.verified_at ?? null,
              dns_checks: result.dns_checks,
            }
          : current,
      );
      setStep(3);
      setMessage(
        result.verified
          ? "All required DNS records were found."
          : "Verification ran, but some records are still missing.",
      );
    } catch (nextError) {
      setError(nextError instanceof Error ? nextError.message : "Verification failed");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="space-y-6">
      <SectionCard
        title="Domain onboarding wizard"
        description="Create a sending domain, copy registrar-ready DNS records, then verify propagation using the backend DNS checks."
      >
        <div className="grid gap-3 md:grid-cols-3">
          {[
            { value: 1, label: "Domain name" },
            { value: 2, label: "DNS records" },
            { value: 3, label: "Verify" },
          ].map((item) => (
            <div
              key={item.value}
              className={`rounded-3xl border px-4 py-4 ${
                step >= item.value
                  ? "border-[var(--accent)] bg-[#fff2e8]"
                  : "border-[var(--line)] bg-white/65"
              }`}
            >
              <p className="eyebrow">Step {item.value}</p>
              <p className="mt-2 text-lg font-semibold">{item.label}</p>
            </div>
          ))}
        </div>

        {message ? (
          <div className="mt-5 rounded-3xl border border-[#d4c29f] bg-[#fff7df] px-4 py-3 text-sm text-[#6b5320]">
            {message}
          </div>
        ) : null}
        {error ? (
          <div className="mt-5 rounded-3xl border border-[#d8ad9f] bg-[#fff1ec] px-4 py-3 text-sm text-[#8d2d11]">
            {error}
          </div>
        ) : null}
      </SectionCard>

      {step === 1 ? (
        <SectionCard
          title="Create sending domain"
          description="Transactional-only sending in Production v1. Customer DNS changes stay manual and Cloudflare automation is intentionally excluded."
        >
          <div className="grid gap-4 lg:grid-cols-[1fr_auto]">
            <input
              className="field"
              placeholder="example.com"
              value={domainName}
              onChange={(event) => setDomainName(event.target.value)}
            />
            <Button disabled={loading || !domainName.trim()} onClick={handleCreateDomain}>
              {loading ? "Creating..." : "Create domain"}
            </Button>
          </div>
          <p className="mt-4 text-sm leading-7 text-[var(--muted)]">
            You will receive SPF, DKIM placeholder, DMARC, and optional MX guidance.
            The dashboard also warns that SMTP host records must remain DNS only.
          </p>
        </SectionCard>
      ) : null}

      {domain && step >= 2 ? (
        <SectionCard
          title={`DNS checklist for ${domain.name}`}
          description="Copy these exact values into your registrar or Cloudflare DNS panel. SMTP records must stay DNS only, not proxied."
        >
          <div className="space-y-4">
            {domain.dns_requirements.map((record) => (
              <div
                key={`${record.type}-${record.name}`}
                className="rounded-3xl border border-[var(--line)] bg-white/75 p-5"
              >
                <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
                  <div className="space-y-2">
                    <div className="flex flex-wrap items-center gap-3">
                      <span className="eyebrow">{record.type}</span>
                      <StatusBadge value={domain.verified ? "verified" : "pending"} />
                    </div>
                    <h3 className="text-lg font-semibold">{record.name}</h3>
                    <p className="text-sm leading-7 text-[var(--muted)]">{record.note}</p>
                  </div>
                  <div className="flex gap-2">
                    <CopyButton value={record.name} label="Copy name" />
                    <CopyButton value={record.value} label="Copy value" />
                  </div>
                </div>
                <div className="mt-4 rounded-2xl bg-[#f8f3ea] p-4 font-mono text-xs leading-6 text-[var(--muted)]">
                  {record.value}
                </div>
              </div>
            ))}

            <div className="rounded-3xl border border-[#e0bb8e] bg-[#fff4df] p-5 text-sm leading-7 text-[#7b5a13]">
              <p className="font-semibold">Cloudflare warning</p>
              <p className="mt-2">
                SMTP records and public hostname records must stay DNS only. Do not
                orange-cloud the SMTP host, or deliverability and TLS behavior will break.
              </p>
            </div>

            <div className="flex flex-wrap gap-3">
              <Button disabled={loading} onClick={handleVerifyDomain}>
                {loading ? "Verifying..." : "Verify now"}
              </Button>
              <Link
                href="/domains"
                className="inline-flex items-center justify-center rounded-full border border-[var(--line)] bg-white/70 px-4 py-2.5 text-sm font-semibold transition hover:bg-white"
              >
                Back to domains
              </Link>
            </div>
          </div>
        </SectionCard>
      ) : null}

      {domain && step === 3 ? (
        <SectionCard
          title="Verification result"
          description="The backend checks TXT records via public resolvers and stores the result in dns_checks."
        >
          <div className="space-y-3">
            <div className="flex flex-wrap items-center gap-3">
              <StatusBadge value={domain.verified ? "verified" : "pending"} />
              <span className="text-sm text-[var(--muted)]">
                Last checked {formatDateTime(domain.dns_checks[0]?.checked_at)}
              </span>
            </div>
            {domain.dns_checks.map((check) => (
              <div
                key={check.id}
                className="rounded-2xl border border-[var(--line)] bg-white/70 p-4"
              >
                <div className="flex items-center justify-between gap-3">
                  <div>
                    <p className="font-semibold">{check.name}</p>
                    <p className="mt-1 text-xs uppercase tracking-[0.16em] text-[var(--muted)]">
                      {check.record_type}
                    </p>
                  </div>
                  <StatusBadge value={check.found ? "verified" : "pending"} />
                </div>
                <p className="mt-3 text-sm leading-7 text-[var(--muted)]">
                  Expected: <span className="font-mono text-xs">{check.expected_value}</span>
                </p>
                <p className="mt-1 text-sm leading-7 text-[var(--muted)]">
                  Found:{" "}
                  <span className="font-mono text-xs">
                    {check.found_value || "Not found yet"}
                  </span>
                </p>
              </div>
            ))}
          </div>
        </SectionCard>
      ) : null}
    </div>
  );
}
