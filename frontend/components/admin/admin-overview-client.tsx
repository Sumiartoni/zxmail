"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { RiskBadge } from "@/components/admin/risk-badge";
import { useAuth } from "@/components/providers/auth-provider";
import { MetricCard } from "@/components/ui/metric-card";
import { DataTable } from "@/components/ui/data-table";
import { EmptyState } from "@/components/ui/empty-state";
import { ErrorState } from "@/components/ui/error-state";
import { LoadingSkeleton } from "@/components/ui/loading-skeleton";
import { PageHeader } from "@/components/ui/page-header";
import { formatNumber, formatPercentage } from "@/lib/utils";
import type { AdminOverview, RiskRecord } from "@/types/zxmail";

export function AdminOverviewClient() {
  const { api } = useAuth();
  const [overview, setOverview] = useState<AdminOverview | null>(null);
  const [risk, setRisk] = useState<RiskRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let mounted = true;
    Promise.all([api.getAdminOverview(), api.listRiskOrganizations()])
      .then(([nextOverview, nextRisk]) => {
        if (mounted) {
          setOverview(nextOverview);
          setRisk(nextRisk);
          setError(null);
        }
      })
      .catch((nextError) => {
        if (mounted) {
          setError(nextError instanceof Error ? nextError.message : "Failed to load admin overview");
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
      <PageHeader
        eyebrow="Admin overview"
        title="Global posture for customers, traffic, and payment risk"
        description="Production Ready v2 shifts the admin surface from simple monitoring to real SaaS operations: package state, quota posture, deliverability indicators, and manual payment workflow."
        actions={
          <>
            <Link href="/admin/payments" className="inline-flex items-center justify-center rounded-[16px] bg-[var(--primary)] px-4 py-2.5 text-sm font-semibold text-white shadow-[0_14px_32px_rgba(23,105,255,0.24)]">
              Review payments
            </Link>
            <Link href="/admin/alerts" className="inline-flex items-center justify-center rounded-[16px] border border-[var(--border)] bg-white px-4 py-2.5 text-sm font-semibold">
              Open alerts
            </Link>
          </>
        }
      />

      {loading ? (
        <div className="grid gap-4">
          <LoadingSkeleton className="h-32" />
          <LoadingSkeleton className="h-80" />
        </div>
      ) : error || !overview ? (
        <ErrorState description={error || "Admin overview could not be loaded."} retryLabel="Retry" onRetry={() => window.location.reload()} />
      ) : (
        <>
          <section className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
            <MetricCard label="Accepted volume" value={formatNumber(overview.total_email_sent)} note="Accepted webhook events across all organizations." />
            <MetricCard label="Delivered" value={formatNumber(overview.delivered)} note="Delivered volume in the current aggregate window." />
            <MetricCard label="Open alerts" value={formatNumber(overview.open_alerts)} note="Operational and deliverability alerts requiring review." />
            <MetricCard label="Past-due invoices" value={formatNumber(overview.past_due_payments)} note="Invoices or payment flows that need manual follow-up." />
          </section>

          <DataTable
            rows={risk}
            getRowKey={(item) => item.organization_id}
            emptyState={<EmptyState title="No organizations yet" description="Risk posture will appear after customers, domains, and traffic exist." />}
            columns={[
              {
                key: "name",
                header: "Organization",
                render: (item) => (
                  <div>
                    <p className="font-semibold">{item.name}</p>
                    <p className="mt-1 text-xs uppercase tracking-[0.16em] text-[var(--muted)]">{item.payment_status}</p>
                  </div>
                ),
              },
              {
                key: "risk",
                header: "Risk score",
                render: (item) => <RiskBadge score={item.risk_score} />,
              },
              {
                key: "bounce",
                header: "Bounce rate",
                render: (item) => <p className="text-sm text-[var(--muted)]">{formatPercentage(item.bounce_rate * 100)}</p>,
              },
              {
                key: "status",
                header: "State",
                render: (item) => (
                  <p className="text-sm font-semibold text-[var(--foreground)]">{item.suspended ? "Suspended" : "Active"}</p>
                ),
              },
              {
                key: "detail",
                header: "Detail",
                render: (item) => (
                  <Link href={`/admin/organizations/${item.organization_id}`} className="text-sm font-semibold text-[var(--primary)]">
                    Open detail
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
