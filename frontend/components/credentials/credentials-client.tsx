"use client";

import { useEffect, useState } from "react";
import { Button } from "@/components/shared/button";
import { PageHero } from "@/components/shared/page-hero";
import { SectionCard } from "@/components/shared/section-card";
import { StatusBadge } from "@/components/shared/status-badge";
import { useAuth } from "@/components/providers/auth-provider";
import { useToast } from "@/components/providers/toast-provider";
import { CredentialSecretModal } from "@/components/ui/credential-secret-modal";
import { EmptyState } from "@/components/ui/empty-state";
import { ErrorState } from "@/components/ui/error-state";
import { Modal } from "@/components/shared/modal";
import { QuotaUsageBar } from "@/components/ui/quota-usage-bar";
import { formatDateTime } from "@/lib/utils";
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
  const { pushToast } = useToast();
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
      pushToast({
        title: "Credential created",
        description: "The SMTP secret is ready to copy once before the modal closes.",
        tone: "success",
      });
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
        description="Create credentials only for verified domains, surface quota state clearly, and keep SMTP connection details easy to scan for developers."
        actions={
          <Button onClick={() => setModalOpen(true)}>Create credential</Button>
        }
      />

      {error ? <ErrorState description={error} /> : null}

      <SectionCard
        title="Credential inventory"
        description="Quota state is read from the backend response. Passwords are intentionally absent here after the one-time reveal flow."
      >
        {credentials.length > 0 ? (
          <div className="space-y-4">
            {credentials.map((entry) => (
              <div
                key={entry.credential.id}
                className="rounded-[26px] border border-[var(--border)] bg-white p-5 shadow-[var(--shadow-sm)]"
              >
                <div className="flex flex-col gap-5 lg:flex-row lg:items-start lg:justify-between">
                  <div className="max-w-2xl">
                    <div className="flex flex-wrap items-center gap-3">
                      <h3 className="text-xl font-semibold tracking-[-0.03em]">
                        {entry.credential.label || entry.credential.username}
                      </h3>
                      <StatusBadge value={entry.credential.status} />
                    </div>
                    <p className="mt-2 text-sm leading-7 text-[var(--muted)]">
                      {entry.credential.domain_name} · created {formatDateTime(entry.credential.created_at)}
                    </p>
                    <div className="mt-4 rounded-[22px] border border-[var(--border)] bg-[#fbfdff] p-4 text-sm leading-7 text-[var(--muted)]">
                      <p><span className="font-semibold text-[var(--foreground)]">Host:</span> {entry.smtp.host}</p>
                      <p><span className="font-semibold text-[var(--foreground)]">Ports:</span> {entry.smtp.starttls_port} STARTTLS / {entry.smtp.tls_port} TLS</p>
                      <p><span className="font-semibold text-[var(--foreground)]">Username:</span> {entry.smtp.username}</p>
                      <p className="mt-2 text-xs uppercase tracking-[0.16em]">{entry.smtp.password_note}</p>
                    </div>
                  </div>

                  <div className="grid w-full gap-3 lg:max-w-[420px]">
                    <QuotaUsageBar
                      label="Per-minute rate cap"
                      used={entry.credential.per_minute_used}
                      limit={entry.credential.per_minute_limit}
                    />
                    <QuotaUsageBar
                      label="Daily quota"
                      used={entry.credential.daily_used}
                      limit={entry.credential.daily_limit}
                    />
                    <QuotaUsageBar
                      label="Monthly quota"
                      used={entry.credential.monthly_used}
                      limit={entry.credential.monthly_limit}
                    />
                  </div>
                </div>
                <p className="mt-4 text-sm leading-7 text-[var(--muted)]">{entry.credential.enforcement_note}</p>
              </div>
            ))}
          </div>
        ) : !loading ? (
          <EmptyState
            title="No credentials yet"
            description="Create the first SMTP identity from a verified domain. The secret will be shown once immediately after creation."
          />
        ) : null}
      </SectionCard>

      <Modal
        open={modalOpen}
        title="Create SMTP credential"
        description="The password is revealed once immediately after creation and is never persisted in browser storage."
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

      <CredentialSecretModal
        credential={secretResult}
        open={Boolean(secretResult)}
        onClose={() => setSecretResult(null)}
      />
    </div>
  );
}
