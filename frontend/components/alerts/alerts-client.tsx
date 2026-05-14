"use client";

import { useEffect, useState } from "react";
import { AlertCenter } from "@/components/alerts/alert-center";
import { useAuth } from "@/components/providers/auth-provider";
import { ErrorState } from "@/components/ui/error-state";
import { LoadingSkeleton } from "@/components/ui/loading-skeleton";
import { PageHeader } from "@/components/ui/page-header";
import type { AlertRecord } from "@/types/zxmail";

export function AlertsClient() {
  const { api } = useAuth();
  const [alerts, setAlerts] = useState<AlertRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let mounted = true;
    api
      .listAlerts()
      .then((nextAlerts) => {
        if (mounted) {
          setAlerts(nextAlerts);
          setError(null);
        }
      })
      .catch((nextError) => {
        if (mounted) {
          setError(nextError instanceof Error ? nextError.message : "Failed to load alerts");
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
        eyebrow="Alerts"
        title="Customer alert center"
        description="Bounce spikes, rejected traffic, and DNS degradation alerts are surfaced here so customers can react before deliverability worsens."
      />
      {loading ? (
        <LoadingSkeleton className="h-72" />
      ) : error ? (
        <ErrorState description={error} retryLabel="Retry" onRetry={() => window.location.reload()} />
      ) : (
        <AlertCenter alerts={alerts} />
      )}
    </div>
  );
}
