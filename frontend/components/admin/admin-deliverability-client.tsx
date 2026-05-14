"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { RiskBadge } from "@/components/admin/risk-badge";
import { DeliverabilityScoreCard } from "@/components/deliverability/deliverability-score-card";
import { useAuth } from "@/components/providers/auth-provider";
import { DataTable } from "@/components/ui/data-table";
import { EmptyState } from "@/components/ui/empty-state";
import { ErrorState } from "@/components/ui/error-state";
import { LoadingSkeleton } from "@/components/ui/loading-skeleton";
import { PageHeader } from "@/components/ui/page-header";
import { formatPercentage } from "@/lib/utils";
import type { DeliverabilityOverview, RiskRecord } from "@/types/zxmail";

export function AdminDeliverabilityClient() {
  const { api } = useAuth();
  const [overview, setOverview] = useState<DeliverabilityOverview | null>(null);
  const [risk, setRisk] = useState<RiskRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let mounted = true;
    Promise.all([api.getAdminDeliverabilityOverview(), api.listRiskOrganizations()])
      .then(([nextOverview, nextRisk]) => {
        if (mounted) {
          setOverview(nextOverview);
          setRisk(nextRisk);
          setError(null);
        }
      })
      .catch((nextError) => {
        if (mounted) {
          setError(nextError instanceof Error ? nextError.message : "Failed to load deliverability");
        }
      })
      .finally(() => {
        if (mounted) {
          setLoading(false);
        }
      });
    return () => {
      mounted = false;
    };
  }, [api]);

  return (
    <div className="space-y-6">
      <PageHeader eyebrow="Deliverability" title="Deliverability indicators across all customers" description="This surface focuses on transparent health scoring, bounce posture, and organizations that may need operational intervention." />
      {loading ? (
        <LoadingSkeleton className="h-80" />
      ) : error || !overview ? (
        <ErrorState description={error || "Deliverability overview unavailable."} retryLabel="Retry" onRetry={() => window.location.reload()} />
      ) : (
        <>
          <DeliverabilityScoreCard overview={overview} />
          <DataTable
            rows={risk}
            getRowKey={(row) => row.organization_id}
            emptyState={<EmptyState title="No deliverability risk yet" description="Risk signals will appear after traffic and domain health snapshots are generated." />}
            columns={[
              {
                key: "organization",
                header: "Organization",
                render: (row) => <p className="font-semibold">{row.name}</p>,
              },
              {
                key: "risk",
                header: "Risk score",
                render: (row) => <RiskBadge score={row.risk_score} />,
              },
              {
                key: "bounce",
                header: "Bounce rate",
                render: (row) => <p className="text-sm text-[var(--muted)]">{formatPercentage(row.bounce_rate * 100)}</p>,
              },
              {
                key: "detail",
                header: "Detail",
                render: (row) => (
                  <Link href={`/admin/organizations/${row.organization_id}`} className="text-sm font-semibold text-[var(--primary)]">
                    Review org
                  </Link>
                ),
              },
            ]}
          />
        </>
      )}
    </div>
  );
}
