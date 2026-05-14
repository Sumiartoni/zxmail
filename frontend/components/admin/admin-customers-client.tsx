"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { RiskBadge } from "@/components/admin/risk-badge";
import { useAuth } from "@/components/providers/auth-provider";
import { DataTable } from "@/components/ui/data-table";
import { EmptyState } from "@/components/ui/empty-state";
import { ErrorState } from "@/components/ui/error-state";
import { LoadingSkeleton } from "@/components/ui/loading-skeleton";
import { PageHeader } from "@/components/ui/page-header";
import type { OrganizationRecord, RiskRecord } from "@/types/zxmail";

export function AdminCustomersClient() {
  const { api } = useAuth();
  const [organizations, setOrganizations] = useState<OrganizationRecord[]>([]);
  const [risk, setRisk] = useState<Record<string, RiskRecord>>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let mounted = true;
    Promise.all([api.listOrganizations(), api.listRiskOrganizations()])
      .then(([nextOrganizations, nextRisk]) => {
        if (!mounted) {
          return;
        }
        setOrganizations(nextOrganizations);
        setRisk(Object.fromEntries(nextRisk.map((item) => [item.organization_id, item])));
        setError(null);
      })
      .catch((nextError) => {
        if (mounted) {
          setError(nextError instanceof Error ? nextError.message : "Failed to load organizations");
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
        eyebrow="Organizations"
        title="Customer organizations and risk posture"
        description="Customers remain isolated to their own organization, while admins can inspect billing, deliverability, quota, and suspension state in one place."
      />

      {loading ? (
        <LoadingSkeleton className="h-80" />
      ) : error ? (
        <ErrorState description={error} retryLabel="Retry" onRetry={() => window.location.reload()} />
      ) : (
        <DataTable
          rows={organizations}
          getRowKey={(organization) => organization.id}
          emptyState={<EmptyState title="No customers yet" description="Organizations created by admins will appear here." />}
          columns={[
            {
              key: "name",
              header: "Organization",
              render: (organization) => (
                <div>
                  <p className="font-semibold">{organization.name}</p>
                  <p className="mt-1 text-sm text-[var(--muted)]">{organization.owner_email}</p>
                </div>
              ),
            },
            {
              key: "risk",
              header: "Risk",
              render: (organization) =>
                risk[organization.id] ? <RiskBadge score={risk[organization.id].risk_score} /> : <span className="text-sm text-[var(--muted)]">Pending</span>,
            },
            {
              key: "payment",
              header: "Payment",
              render: (organization) => <p className="text-sm text-[var(--muted)]">{risk[organization.id]?.payment_status || "Unknown"}</p>,
            },
            {
              key: "detail",
              header: "Detail",
              render: (organization) => (
                <Link href={`/admin/organizations/${organization.id}`} className="text-sm font-semibold text-[var(--primary)]">
                  Open detail
                </Link>
              ),
            },
          ]}
        />
      )}
    </div>
  );
}
