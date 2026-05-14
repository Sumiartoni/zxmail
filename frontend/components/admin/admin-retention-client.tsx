"use client";

import { useEffect, useState } from "react";
import { RetentionPolicyForm } from "@/components/admin/retention-policy-form";
import { useAuth } from "@/components/providers/auth-provider";
import { useToast } from "@/components/providers/toast-provider";
import { ErrorState } from "@/components/ui/error-state";
import { LoadingSkeleton } from "@/components/ui/loading-skeleton";
import { PageHeader } from "@/components/ui/page-header";
import type { CleanupResult, RetentionPolicy } from "@/types/zxmail";

export function AdminRetentionClient() {
  const { api } = useAuth();
  const { pushToast } = useToast();
  const [policies, setPolicies] = useState<RetentionPolicy[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  async function load() {
    setPolicies(await api.listRetentionPolicies());
  }

  useEffect(() => {
    let mounted = true;
    (async () => {
      try {
        const nextPolicies = await api.listRetentionPolicies();
        if (mounted) {
          setPolicies(nextPolicies);
          setError(null);
        }
      } catch (nextError) {
        if (mounted) {
          setError(nextError instanceof Error ? nextError.message : "Failed to load retention policies");
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
      <PageHeader eyebrow="Retention" title="Retention and cleanup policy" description="Default send log retention is 90 days. Cleanup remains dry-run capable and never touches invoices, payments, or audit logs." />
      {loading ? (
        <LoadingSkeleton className="h-80" />
      ) : error ? (
        <ErrorState description={error} retryLabel="Retry" onRetry={() => window.location.reload()} />
      ) : (
        <RetentionPolicyForm
          policies={policies}
          onSave={async (organizationID, retentionDays) => {
            await api.updateRetentionPolicy(organizationID, retentionDays);
            await load();
            pushToast({ tone: "success", title: "Retention updated" });
          }}
          onCleanup={async (dryRun): Promise<CleanupResult> => {
            const result = await api.runRetentionCleanup(dryRun);
            pushToast({
              tone: "info",
              title: dryRun ? "Dry-run completed" : "Cleanup completed",
              description: `${result.matched_logs} logs matched across ${result.organizations} organizations.`,
            });
            await load();
            return result;
          }}
        />
      )}
    </div>
  );
}
