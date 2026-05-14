"use client";

import { useEffect, useState } from "react";
import { DeliverabilityScoreCard } from "@/components/deliverability/deliverability-score-card";
import { DomainHealthChecklist } from "@/components/deliverability/domain-health-checklist";
import { useAuth } from "@/components/providers/auth-provider";
import { Button } from "@/components/shared/button";
import { SectionCard } from "@/components/shared/section-card";
import { DataTable } from "@/components/ui/data-table";
import { EmptyState } from "@/components/ui/empty-state";
import { ErrorState } from "@/components/ui/error-state";
import { LoadingSkeleton } from "@/components/ui/loading-skeleton";
import { PageHeader } from "@/components/ui/page-header";
import type { DeliverabilityOverview, DomainHealthRecord, DomainRecord } from "@/types/zxmail";

export function DeliverabilityClient() {
  const { api } = useAuth();
  const [overview, setOverview] = useState<DeliverabilityOverview | null>(null);
  const [domains, setDomains] = useState<DomainRecord[]>([]);
  const [selected, setSelected] = useState<DomainHealthRecord | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let mounted = true;
    async function load() {
      try {
        const [nextOverview, nextDomains] = await Promise.all([api.getDeliverabilityOverview(), api.listDomains()]);
        if (!mounted) {
          return;
        }
        setOverview(nextOverview);
        setDomains(nextDomains);
        if (nextDomains[0]) {
          const initialHealth = await api.getDomainHealth(nextDomains[0].id);
          if (mounted) {
            setSelected(initialHealth);
          }
        }
        setError(null);
      } catch (nextError) {
        if (mounted) {
          setError(nextError instanceof Error ? nextError.message : "Failed to load deliverability");
        }
      } finally {
        if (mounted) {
          setLoading(false);
        }
      }
    }
    void load();
    return () => {
      mounted = false;
    };
  }, [api]);

  return (
    <div className="space-y-6">
      <PageHeader
        eyebrow="Deliverability"
        title="Health indicators for domains and traffic quality"
        description="zxMail v2 surfaces bounce, deferred, rejection, and DNS health indicators transparently. It does not claim inbox placement or seed-test results."
      />
      {loading ? (
        <div className="grid gap-4">
          <LoadingSkeleton className="h-40" />
          <LoadingSkeleton className="h-64" />
        </div>
      ) : error || !overview ? (
        <ErrorState description={error || "Deliverability overview could not be loaded."} retryLabel="Retry" onRetry={() => window.location.reload()} />
      ) : (
        <>
          <DeliverabilityScoreCard overview={overview} />
          <div className="grid gap-4 xl:grid-cols-[0.9fr_1.1fr]">
            <SectionCard title="Tracked domains" description="Select a domain to inspect the latest DNS and traffic health snapshot.">
              <DataTable
                rows={domains}
                getRowKey={(domain) => domain.id}
                onRowClick={async (domain) => setSelected(await api.getDomainHealth(domain.id))}
                emptyState={<EmptyState title="No domains yet" description="Add and verify a domain before deliverability health snapshots can be generated." />}
                columns={[
                  {
                    key: "name",
                    header: "Domain",
                    render: (domain) => <p className="font-semibold">{domain.name}</p>,
                  },
                  {
                    key: "verified",
                    header: "Verified",
                    render: (domain) => <p className="text-sm text-[var(--muted)]">{domain.verified ? "Yes" : "Pending"}</p>,
                  },
                  {
                    key: "actions",
                    header: "Actions",
                    render: (domain) => (
                      <Button
                        variant="secondary"
                        onClick={async (event) => {
                          event.stopPropagation();
                          setSelected(await api.recheckDomain(domain.id));
                        }}
                      >
                        Recheck
                      </Button>
                    ),
                  },
                ]}
              />
            </SectionCard>
            {selected ? (
              <DomainHealthChecklist health={selected} />
            ) : (
              <EmptyState title="Pick a domain" description="Select a domain from the table to inspect SPF, DKIM, DMARC, quota, and rate indicators." />
            )}
          </div>
        </>
      )}
    </div>
  );
}
