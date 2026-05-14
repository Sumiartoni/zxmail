"use client";

import { useEffect, useState } from "react";
import { useAuth } from "@/components/providers/auth-provider";
import { ErrorState } from "@/components/ui/error-state";
import { LoadingSkeleton } from "@/components/ui/loading-skeleton";
import { PageHeader } from "@/components/ui/page-header";
import { QuotaProgress } from "@/components/usage/quota-progress";
import { UsageChart } from "@/components/usage/usage-chart";
import type { UsageOverview } from "@/types/zxmail";

export function UsageClient() {
  const { api } = useAuth();
  const [usage, setUsage] = useState<UsageOverview | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let mounted = true;
    api
      .getUsage()
      .then((nextUsage) => {
        if (mounted) {
          setUsage(nextUsage);
          setError(null);
        }
      })
      .catch((nextError) => {
        if (mounted) {
          setError(nextError instanceof Error ? nextError.message : "Failed to load usage");
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
        eyebrow="Usage"
        title="Quota and metering"
        description="Usage is sourced from Postal webhook events. Accepted volume remains the source of truth for quota posture, while direct-to-Postal sending still limits hard pre-send enforcement until a future SMTP gateway exists."
      />
      {loading ? (
        <div className="grid gap-4">
          <LoadingSkeleton className="h-32" />
          <LoadingSkeleton className="h-64" />
        </div>
      ) : error || !usage ? (
        <ErrorState description={error || "Usage could not be loaded."} retryLabel="Retry" onRetry={() => window.location.reload()} />
      ) : (
        <>
          <UsageChart usage={usage} />
          <QuotaProgress usage={usage} />
        </>
      )}
    </div>
  );
}
