"use client";

import { useState } from "react";
import Link from "next/link";
import { Button } from "@/components/shared/button";
import { SectionCard } from "@/components/shared/section-card";
import { StatusBadge } from "@/components/shared/status-badge";
import { useToast } from "@/components/providers/toast-provider";
import { CodeBlock } from "@/components/ui/code-block";
import { DNSRecordCard } from "@/components/ui/dns-record-card";
import { ErrorState } from "@/components/ui/error-state";
import { Stepper } from "@/components/ui/stepper";
import { useAuth } from "@/components/providers/auth-provider";
import { formatDateTime } from "@/lib/utils";
import type { DomainRecord } from "@/types/zxmail";

export function DomainOnboardingWizard() {
  const { api } = useAuth();
  const { pushToast } = useToast();
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
      pushToast({
        title: "Domain created",
        description: "Copy the generated DNS records, publish them, then run verification.",
        tone: "success",
      });
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
      pushToast({
        title: result.verified ? "Verification passed" : "Verification needs more time",
        description: result.verified
          ? "The domain is now ready for SMTP credential creation."
          : "Check missing TXT records and run verification again after DNS propagates.",
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
      <SectionCard
        title="Domain onboarding wizard"
        description="A guided four-step flow: add the domain, publish DNS records, verify with public resolvers, then continue to SMTP credentials."
      >
        <Stepper
          current={step}
          steps={[
            { id: 1, label: "Add domain", detail: "Create the sending identity" },
            { id: 2, label: "Publish DNS", detail: "SPF, DKIM, DMARC" },
            { id: 3, label: "Verify DNS", detail: "Resolver-backed checks" },
          ]}
        />

        {message ? (
          <div className="mt-5 rounded-[22px] border border-[rgba(23,105,255,0.16)] bg-[rgba(23,105,255,0.06)] px-4 py-3 text-sm text-[var(--foreground)]">
            {message}
          </div>
        ) : null}
        {error ? <div className="mt-5"><ErrorState description={error} /></div> : null}
      </SectionCard>

      {step === 1 ? (
        <SectionCard
          title="Create sending domain"
          description="Transactional-only sending in Production v1. DNS changes stay manual by design, with no Cloudflare automation."
        >
          <div className="grid gap-6 lg:grid-cols-[1fr_0.92fr]">
            <div>
              <label className="grid gap-2">
                <span className="text-sm font-semibold text-[var(--foreground)]">Domain</span>
                <input
                  className="field"
                  placeholder="example.com"
                  value={domainName}
                  onChange={(event) => setDomainName(event.target.value)}
                />
              </label>
              <p className="mt-3 text-sm leading-7 text-[var(--muted)]">
                Use the root domain you want to send from. zxMail will generate SPF, DKIM placeholder, DMARC, and optional MX guidance.
              </p>
              <div className="mt-5">
                <Button disabled={loading || !domainName.trim()} onClick={handleCreateDomain}>
                  {loading ? "Creating..." : "Create domain"}
                </Button>
              </div>
            </div>
            <div className="rounded-[24px] border border-[var(--border)] bg-[#fbfdff] p-5">
              <p className="eyebrow">What happens next</p>
              <ul className="mt-4 space-y-3 text-sm leading-7 text-[var(--muted)]">
                <li>1. zxMail generates exact DNS requirements.</li>
                <li>2. You publish them at your DNS provider.</li>
                <li>3. Verification checks TXT records using public resolvers.</li>
                <li>4. Verified domains become eligible for SMTP credentials.</li>
              </ul>
            </div>
          </div>
        </SectionCard>
      ) : null}

      {domain && step >= 2 ? (
        <SectionCard
          title={`DNS checklist for ${domain.name}`}
          description="Copy these exact values into your registrar or DNS panel. SMTP records must stay DNS only, not proxied."
        >
          <div className="space-y-4">
            <div className="grid gap-4 xl:grid-cols-2">
              {domain.dns_requirements.map((record) => (
                <DNSRecordCard
                  key={`${record.type}-${record.name}`}
                  record={record}
                  check={domain.dns_checks.find((check) => check.name === record.name)}
                  verified={domain.verified}
                />
              ))}
            </div>

            <div className="rounded-[24px] border border-[rgba(184,92,0,0.16)] bg-[rgba(184,92,0,0.06)] p-5 text-sm leading-7 text-[var(--foreground)]">
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
              <Link href="/domains" className="inline-flex items-center justify-center rounded-[16px] border border-[var(--border)] bg-white px-4 py-2.5 text-sm font-semibold transition hover:bg-[#f8fbff]">
                Back to domains
              </Link>
            </div>
          </div>
        </SectionCard>
      ) : null}

      {domain && step === 3 ? (
        <SectionCard
          title="Verification result"
          description="The backend checks required TXT records through public resolvers and stores each result in dns_checks."
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
                className="rounded-[22px] border border-[var(--border)] bg-white p-4"
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
            {domain.verified ? (
              <div className="rounded-[22px] border border-[rgba(8,127,91,0.16)] bg-[rgba(8,127,91,0.06)] p-4 text-sm leading-7 text-[var(--foreground)]">
                <p className="font-semibold">Ready to create SMTP credentials</p>
                <p className="mt-2">
                  This domain is verified and can now be used on the credentials page to issue SMTP access.
                </p>
                <div className="mt-4">
                  <Link href="/credentials" className="inline-flex items-center justify-center rounded-[16px] bg-[var(--primary)] px-4 py-2.5 text-sm font-semibold text-white">
                    Continue to credentials
                  </Link>
                </div>
              </div>
            ) : (
              <CodeBlock label="Verification note">
                Wait for TXT records to propagate, then run verification again. SPF, DKIM placeholder, and DMARC all need to be discoverable before zxMail marks the domain as verified.
              </CodeBlock>
            )}
          </div>
        </SectionCard>
      ) : null}
    </div>
  );
}
