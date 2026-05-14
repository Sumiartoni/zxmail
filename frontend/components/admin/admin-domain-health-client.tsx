"use client";

import { useEffect, useState } from "react";
import { useAuth } from "@/components/providers/auth-provider";
import { useToast } from "@/components/providers/toast-provider";
import { Button } from "@/components/shared/button";
import { DataTable } from "@/components/ui/data-table";
import { EmptyState } from "@/components/ui/empty-state";
import { ErrorState } from "@/components/ui/error-state";
import { LoadingSkeleton } from "@/components/ui/loading-skeleton";
import { PageHeader } from "@/components/ui/page-header";
import { formatShortDate } from "@/lib/utils";
import type { DomainHealthRecord } from "@/types/zxmail";

export function AdminDomainHealthClient() {
  const { api } = useAuth();
  const { pushToast } = useToast();
  const [domains, setDomains] = useState<DomainHealthRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  async function load() {
    setDomains(await api.listAdminDomainHealth());
  }

  useEffect(() => {
    let mounted = true;
    (async () => {
      try {
        const nextDomains = await api.listAdminDomainHealth();
        if (mounted) {
          setDomains(nextDomains);
          setError(null);
        }
      } catch (nextError) {
        if (mounted) {
          setError(nextError instanceof Error ? nextError.message : "Failed to load domain health");
        }
      } finally {
        if (mounted) {
          setLoading(false);
        }
      }
    })();
    return () => {
      mounted = false;
    };
  }, [api]);

  return (
    <div className="space-y-6">
      <PageHeader
        eyebrow="Domain health"
        title="DNS and traffic health snapshots"
        description="SPF, DKIM, DMARC, quota posture, and bounce or rejection indicators are summarized per domain. SMTP-related DNS still must remain DNS only in Cloudflare."
        actions={
          <Button
            onClick={async () => {
              await api.recheckAllDomains();
              await load();
              pushToast({ tone: "success", title: "Domain recheck requested" });
            }}
          >
            Recheck all domains
          </Button>
        }
      />
      {loading ? (
        <LoadingSkeleton className="h-80" />
      ) : error ? (
        <ErrorState description={error} retryLabel="Retry" onRetry={() => window.location.reload()} />
      ) : (
        <DataTable
          rows={domains}
          getRowKey={(domain) => domain.domain_id}
          emptyState={<EmptyState title="No health snapshots yet" description="Run domain rechecks or wait for scheduled jobs to populate health snapshots." />}
          columns={[
            {
              key: "domain",
              header: "Domain",
              render: (domain) => <p className="font-semibold">{domain.domain_name}</p>,
            },
            {
              key: "score",
              header: "Score",
              render: (domain) => <p className="text-sm font-semibold text-[var(--foreground)]">{domain.health_score}</p>,
            },
            {
              key: "dns",
              header: "DNS",
              render: (domain) => (
                <p className="text-sm text-[var(--muted)]">
                  SPF {domain.spf_found ? "yes" : "no"} · DKIM {domain.dkim_found ? "yes" : "no"} · DMARC {domain.dmarc_found ? "yes" : "no"}
                </p>
              ),
            },
            {
              key: "checked_at",
              header: "Checked",
              render: (domain) => <p className="text-sm text-[var(--muted)]">{formatShortDate(domain.checked_at)}</p>,
            },
          ]}
        />
      )}
    </div>
  );
}
