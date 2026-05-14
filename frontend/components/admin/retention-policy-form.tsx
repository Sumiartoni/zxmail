"use client";

import { useState } from "react";
import { Button } from "@/components/shared/button";
import { DataTable } from "@/components/ui/data-table";
import { EmptyState } from "@/components/ui/empty-state";
import type { CleanupResult, RetentionPolicy } from "@/types/zxmail";

export function RetentionPolicyForm({
  policies,
  onSave,
  onCleanup,
}: {
  policies: RetentionPolicy[];
  onSave: (organizationID: string, retentionDays: number) => Promise<void>;
  onCleanup: (dryRun: boolean) => Promise<CleanupResult>;
}) {
  const [editing, setEditing] = useState<Record<string, number>>({});
  const [cleanupResult, setCleanupResult] = useState<CleanupResult | null>(null);

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap gap-3">
        <Button
          variant="secondary"
          onClick={async () => {
            const result = await onCleanup(true);
            setCleanupResult(result);
          }}
        >
          Dry-run cleanup
        </Button>
        <Button
          variant="danger"
          onClick={async () => {
            const result = await onCleanup(false);
            setCleanupResult(result);
          }}
        >
          Run cleanup now
        </Button>
      </div>

      {cleanupResult ? (
        <div className="rounded-[22px] border border-[var(--border)] bg-white px-4 py-4 text-sm text-[var(--muted)]">
          Cleanup {cleanupResult.dry_run ? "dry-run" : "execution"} matched {cleanupResult.matched_logs} logs across {cleanupResult.organizations} organizations and deleted {cleanupResult.deleted_logs} logs.
        </div>
      ) : null}

      <DataTable
        rows={policies}
        getRowKey={(policy) => policy.organization_id}
        emptyState={<EmptyState title="No retention policies" description="Organizations will inherit the default log retention policy until configured." />}
        columns={[
          {
            key: "name",
            header: "Organization",
            render: (policy) => <p className="font-semibold">{policy.name}</p>,
          },
          {
            key: "days",
            header: "Retention days",
            render: (policy) => (
              <input
                type="number"
                min={1}
                className="w-28 rounded-[14px] border border-[var(--border)] bg-white px-3 py-2 text-sm"
                value={editing[policy.organization_id] ?? policy.retention_days}
                onChange={(event) =>
                  setEditing((current) => ({
                    ...current,
                    [policy.organization_id]: Number(event.target.value),
                  }))
                }
              />
            ),
          },
          {
            key: "actions",
            header: "Actions",
            render: (policy) => (
              <Button
                variant="secondary"
                onClick={() => onSave(policy.organization_id, editing[policy.organization_id] ?? policy.retention_days)}
              >
                Save
              </Button>
            ),
          },
        ]}
      />
    </div>
  );
}
