"use client";

import { useEffect, useState } from "react";
import { Button } from "@/components/shared/button";
import { CopyButton } from "@/components/shared/copy-button";
import { Modal } from "@/components/shared/modal";
import { PageHero } from "@/components/shared/page-hero";
import { SectionCard } from "@/components/shared/section-card";
import { StatusBadge } from "@/components/shared/status-badge";
import { useAuth } from "@/components/providers/auth-provider";
import { formatDateTime, formatNumber } from "@/lib/utils";
import type {
  CredentialResponse,
  CredentialSecretResponse,
  DomainRecord,
} from "@/types/zxmail";

type CreateFormState = {
  domain_id: string;
  label: string;
  per_minute_limit: string;
  daily_limit: string;
  monthly_limit: string;
};

const initialForm: CreateFormState = {
  domain_id: "",
  label: "",
  per_minute_limit: "",
  daily_limit: "",
  monthly_limit: "",
};

export function CredentialsClient() {
  const { api } = useAuth();
  const [domains, setDomains] = useState<DomainRecord[]>([]);
  const [credentials, setCredentials] = useState<CredentialResponse[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [modalOpen, setModalOpen] = useState(false);
  const [form, setForm] = useState<CreateFormState>(initialForm);
  const [secretResult, setSecretResult] = useState<CredentialSecretResponse | null>(null);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    let mounted = true;

    async function loadData() {
      try {
        const [nextDomains, nextCredentials] = await Promise.all([
          api.listDomains(),
          api.listCredentials(),
        ]);
        if (!mounted) {
          return;
        }
        setDomains(nextDomains.filter((domain) => domain.verified));
        setCredentials(nextCredentials);
      } catch (nextError) {
        if (mounted) {
          setError(
            nextError instanceof Error ? nextError.message : "Failed to load credentials",
          );
        }
      } finally {
        if (mounted) {
          setLoading(false);
        }
      }
    }

    loadData();
    return () => {
      mounted = false;
    };
  }, [api]);

  async function refreshCredentials() {
    const nextCredentials = await api.listCredentials();
    setCredentials(nextCredentials);
  }

  async function submitCredential() {
    setSubmitting(true);
    setError("");
    try {
      const response = await api.createCredential({
        domain_id: form.domain_id,
        label: form.label,
        per_minute_limit: form.per_minute_limit ? Number(form.per_minute_limit) : null,
        daily_limit: form.daily_limit ? Number(form.daily_limit) : null,
        monthly_limit: form.monthly_limit ? Number(form.monthly_limit) : null,
      });
      await refreshCredentials();
      setSecretResult(response);
      setModalOpen(false);
      setForm(initialForm);
    } catch (nextError) {
      setError(nextError instanceof Error ? nextError.message : "Failed to create credential");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="space-y-6">
      <PageHero
        eyebrow="SMTP credentials"
        title="Issue scoped SMTP identities with show-once secrets"
        description="Create credentials only for verified domains, surface quota state, and expose Postal-compatible SMTP connection info without ever re-displaying an old password."
        actions={
          <Button onClick={() => setModalOpen(true)}>Create credential</Button>
        }
      />

      {error ? (
        <div className="rounded-3xl border border-[#d8ad9f] bg-[#fff1ec] px-4 py-3 text-sm text-[#8d2d11]">
          {error}
        </div>
      ) : null}

      <SectionCard
        title="Credential inventory"
        description="Quota state is read from the backend response. A credential can move between enabled, limited, and disabled."
      >
        <div className="space-y-4">
          {credentials.map((entry) => (
            <div
              key={entry.credential.id}
              className="rounded-3xl border border-[var(--line)] bg-white/75 p-5"
            >
              <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
                <div>
                  <div className="flex flex-wrap items-center gap-3">
                    <h3 className="text-xl font-semibold">
                      {entry.credential.label || entry.credential.username}
                    </h3>
                    <StatusBadge value={entry.credential.status} />
                  </div>
                  <p className="mt-2 text-sm leading-7 text-[var(--muted)]">
                    {entry.credential.domain_name} · created{" "}
                    {formatDateTime(entry.credential.created_at)}
                  </p>
                  <div className="mt-4 rounded-2xl bg-[#f8f3ea] p-4 text-sm leading-7 text-[var(--muted)]">
                    <p>
                      <span className="font-semibold text-[var(--ink)]">Host:</span>{" "}
                      {entry.smtp.host}
                    </p>
                    <p>
                      <span className="font-semibold text-[var(--ink)]">Ports:</span>{" "}
                      {entry.smtp.starttls_port} STARTTLS / {entry.smtp.tls_port} TLS
                    </p>
                    <p>
                      <span className="font-semibold text-[var(--ink)]">Username:</span>{" "}
                      {entry.smtp.username}
                    </p>
                    <p className="mt-2 text-xs uppercase tracking-[0.16em]">
                      {entry.smtp.password_note}
                    </p>
                  </div>
                </div>

                <div className="grid gap-3 sm:grid-cols-3 lg:w-[440px]">
                  <QuotaCell
                    label="Per minute"
                    used={entry.credential.per_minute_used}
                    limit={entry.credential.per_minute_limit}
                  />
                  <QuotaCell
                    label="Daily"
                    used={entry.credential.daily_used}
                    limit={entry.credential.daily_limit}
                  />
                  <QuotaCell
                    label="Monthly"
                    used={entry.credential.monthly_used}
                    limit={entry.credential.monthly_limit}
                  />
                </div>
              </div>
              <p className="mt-4 text-sm leading-7 text-[var(--muted)]">
                {entry.credential.enforcement_note}
              </p>
            </div>
          ))}

          {!loading && credentials.length === 0 ? (
            <div className="rounded-2xl border border-dashed border-[var(--line)] px-4 py-8 text-center text-sm text-[var(--muted)]">
              No credentials yet. Create the first SMTP identity from a verified domain.
            </div>
          ) : null}
        </div>
      </SectionCard>

      <Modal
        open={modalOpen}
        title="Create SMTP credential"
        description="The password will be revealed once immediately after creation. It is never persisted in browser storage."
        onClose={() => setModalOpen(false)}
      >
        <div className="grid gap-4 md:grid-cols-2">
          <select
            className="field md:col-span-2"
            value={form.domain_id}
            onChange={(event) => setForm((current) => ({ ...current, domain_id: event.target.value }))}
          >
            <option value="">Choose a verified domain</option>
            {domains.map((domain) => (
              <option key={domain.id} value={domain.id}>
                {domain.name}
              </option>
            ))}
          </select>
          <input
            className="field md:col-span-2"
            placeholder="Credential label"
            value={form.label}
            onChange={(event) => setForm((current) => ({ ...current, label: event.target.value }))}
          />
          <input
            className="field"
            type="number"
            placeholder="Per-minute cap"
            value={form.per_minute_limit}
            onChange={(event) =>
              setForm((current) => ({ ...current, per_minute_limit: event.target.value }))
            }
          />
          <input
            className="field"
            type="number"
            placeholder="Daily quota"
            value={form.daily_limit}
            onChange={(event) =>
              setForm((current) => ({ ...current, daily_limit: event.target.value }))
            }
          />
          <input
            className="field"
            type="number"
            placeholder="Monthly quota"
            value={form.monthly_limit}
            onChange={(event) =>
              setForm((current) => ({ ...current, monthly_limit: event.target.value }))
            }
          />
        </div>

        <div className="mt-6 flex flex-wrap gap-3">
          <Button disabled={submitting || !form.domain_id} onClick={submitCredential}>
            {submitting ? "Creating..." : "Create credential"}
          </Button>
          <Button variant="ghost" onClick={() => setModalOpen(false)}>
            Cancel
          </Button>
        </div>
      </Modal>

      <Modal
        open={Boolean(secretResult)}
        title="SMTP password revealed once"
        description="Copy this secret now. Once you close this modal, the frontend clears it from memory and the backend will not return it again."
        onClose={() => setSecretResult(null)}
      >
        {secretResult ? (
          <div className="space-y-5">
            <div className="grid gap-3 md:grid-cols-2">
              <div className="rounded-2xl border border-[var(--line)] bg-white/70 p-4">
                <p className="eyebrow">Username</p>
                <p className="mt-2 break-all font-mono text-sm">
                  {secretResult.smtp.username}
                </p>
                <div className="mt-3">
                  <CopyButton value={secretResult.smtp.username} />
                </div>
              </div>
              <div className="rounded-2xl border border-[var(--line)] bg-white/70 p-4">
                <p className="eyebrow">Password</p>
                <p className="mt-2 break-all font-mono text-sm">{secretResult.secret}</p>
                <div className="mt-3">
                  <CopyButton value={secretResult.secret} />
                </div>
              </div>
            </div>
            <div className="rounded-2xl border border-[#e0bb8e] bg-[#fff4df] p-4 text-sm leading-7 text-[#7b5a13]">
              Save this secret in your application immediately. zxMail will not show it again after this modal closes.
            </div>
          </div>
        ) : null}
      </Modal>
    </div>
  );
}

function QuotaCell({
  label,
  used,
  limit,
}: {
  label: string;
  used: number;
  limit?: number | null;
}) {
  return (
    <div className="rounded-2xl border border-[var(--line)] bg-white/75 p-4">
      <p className="eyebrow">{label}</p>
      <p className="mt-2 text-2xl font-semibold tracking-[-0.04em]">
        {formatNumber(used)}
      </p>
      <p className="mt-1 text-sm text-[var(--muted)]">
        / {limit !== null && limit !== undefined ? formatNumber(limit) : "unlimited"}
      </p>
    </div>
  );
}
