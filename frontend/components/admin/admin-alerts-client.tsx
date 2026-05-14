"use client";

import { useEffect, useState } from "react";
import { AlertCenter } from "@/components/alerts/alert-center";
import { useAuth } from "@/components/providers/auth-provider";
import { useToast } from "@/components/providers/toast-provider";
import { ErrorState } from "@/components/ui/error-state";
import { LoadingSkeleton } from "@/components/ui/loading-skeleton";
import { PageHeader } from "@/components/ui/page-header";
import type { AlertRecord } from "@/types/zxmail";

export function AdminAlertsClient() {
  const { api } = useAuth();
  const { pushToast } = useToast();
  const [alerts, setAlerts] = useState<AlertRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  async function load() {
    setAlerts(await api.listAlerts());
  }

  useEffect(() => {
    let mounted = true;
    (async () => {
      try {
        const nextAlerts = await api.listAlerts();
        if (mounted) {
          setAlerts(nextAlerts);
          setError(null);
        }
      } catch (nextError) {
        if (mounted) {
          setError(nextError instanceof Error ? nextError.message : "Failed to load alerts");
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
      <PageHeader eyebrow="Alerts" title="Admin alert center" description="Resolve delivery degradation, DNS drift, and health alerts while keeping the underlying record intact." />
      {loading ? (
        <LoadingSkeleton className="h-80" />
      ) : error ? (
        <ErrorState description={error} retryLabel="Retry" onRetry={() => window.location.reload()} />
      ) : (
        <AlertCenter
          alerts={alerts}
          allowResolve
          onResolve={async (alertID) => {
            await api.resolveAlert(alertID);
            await load();
            pushToast({ tone: "success", title: "Alert resolved" });
          }}
        />
      )}
    </div>
  );
}
