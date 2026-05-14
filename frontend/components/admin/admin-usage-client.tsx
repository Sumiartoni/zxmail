"use client";

import { useEffect, useState } from "react";
import { useAuth } from "@/components/providers/auth-provider";
import { useToast } from "@/components/providers/toast-provider";
import { Button } from "@/components/shared/button";
import { SectionCard } from "@/components/shared/section-card";
import { DataTable } from "@/components/ui/data-table";
import { EmptyState } from "@/components/ui/empty-state";
import { ErrorState } from "@/components/ui/error-state";
import { LoadingSkeleton } from "@/components/ui/loading-skeleton";
import { PageHeader } from "@/components/ui/page-header";
import { formatNumber } from "@/lib/utils";
import type { OrganizationRecord, UsageOverview } from "@/types/zxmail";

type UsageRow = OrganizationRecord & { usage?: UsageOverview };

export function AdminUsageClient() {
  const { api } = useAuth();
  const { pushToast } = useToast();
  const [rows, setRows] = useState<UsageRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  async function load() {
    const organizations = await api.listOrganizations();
    const usageRows = await Promise.all(
      organizations.map(async (organization) => ({
        ...organization,
        usage: await api.getOrganizationUsage(organization.id),
      })),
    );
    setRows(usageRows);
  }

  useEffect(() => {
    let mounted = true;
    (async () => {
      try {
        const organizations = await api.listOrganizations();
        const usageRows = await Promise.all(
          organizations.map(async (organization) => ({
            ...organization,
            usage: await api.getOrganizationUsage(organization.id),
          })),
        );
        if (mounted) {
          setRows(usageRows);
          setError(null);
        }
      } catch (nextError) {
        if (mounted) {
          setError(nextError instanceof Error ? nextError.message : "Failed to load usage");
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
      <PageHeader eyebrow="Usage" title="Organization-level usage and quota overrides" description="Accepted volume remains the source of truth. Admin overrides change advisory quota posture while direct-to-Postal sending still limits hard pre-send enforcement." />
      {loading ? (
        <LoadingSkeleton className="h-80" />
      ) : error ? (
        <ErrorState description={error} retryLabel="Retry" onRetry={() => window.location.reload()} />
      ) : (
        <>
          <DataTable
            rows={rows}
            getRowKey={(row) => row.id}
            emptyState={<EmptyState title="No usage data" description="Organizations and webhook events must exist before usage is visible." />}
            columns={[
              {
                key: "organization",
                header: "Organization",
                render: (row) => <p className="font-semibold">{row.name}</p>,
              },
              {
                key: "accepted",
                header: "Accepted",
                render: (row) => <p className="text-sm text-[var(--muted)]">{formatNumber(row.usage?.accepted_month || 0)}</p>,
              },
              {
                key: "delivered",
                header: "Delivered",
                render: (row) => <p className="text-sm text-[var(--muted)]">{formatNumber(row.usage?.delivered_month || 0)}</p>,
              },
              {
                key: "status",
                header: "Quota state",
                render: (row) => <p className="text-sm font-semibold text-[var(--foreground)]">{row.usage?.status || "unknown"}</p>,
              },
              {
                key: "actions",
                header: "Actions",
                render: (row) => (
                  <div className="flex gap-2">
                    <Button
                      variant="secondary"
                      onClick={async () => {
                        await api.resetOrganizationUsage(row.id);
                        await load();
                        pushToast({ tone: "success", title: "Usage reset completed" });
                      }}
                    >
                      Reset
                    </Button>
                  </div>
                ),
              },
            ]}
          />

          <SectionCard title="Override note" description="Quota override editing is supported through the API and can be layered into this screen without changing the underlying contracts.">
            <p className="text-sm leading-7 text-[var(--muted)]">
              This screen already reads per-organization usage and executes safe resets. Quota override editing can be expanded next without changing routes or data shape.
            </p>
          </SectionCard>
        </>
      )}
    </div>
  );
}
