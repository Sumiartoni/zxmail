"use client";

import { useEffect, useState } from "react";
import { AuditLogTable } from "@/components/admin/audit-log-table";
import { useAuth } from "@/components/providers/auth-provider";
import { ErrorState } from "@/components/ui/error-state";
import { LoadingSkeleton } from "@/components/ui/loading-skeleton";
import { PageHeader } from "@/components/ui/page-header";
import type { AuditLogRecord } from "@/types/zxmail";

export function AdminAuditLogsClient() {
  const { api } = useAuth();
  const [records, setRecords] = useState<AuditLogRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let mounted = true;
    api.listAuditLogs()
      .then((nextRecords) => {
        if (mounted) {
          setRecords(nextRecords);
          setError(null);
        }
      })
      .catch((nextError) => {
        if (mounted) {
          setError(nextError instanceof Error ? nextError.message : "Failed to load audit logs");
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
      <PageHeader eyebrow="Audit logs" title="Admin audit trail" description="Mutating billing, quota, suspension, and credential actions should all leave a sanitized audit trail here." />
      {loading ? (
        <LoadingSkeleton className="h-80" />
      ) : error ? (
        <ErrorState description={error} retryLabel="Retry" onRetry={() => window.location.reload()} />
      ) : (
        <AuditLogTable records={records} />
      )}
    </div>
  );
}
