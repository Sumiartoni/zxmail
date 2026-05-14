import { Button } from "@/components/shared/button";
import { DataTable } from "@/components/ui/data-table";
import { EmptyState } from "@/components/ui/empty-state";
import { formatDateTime } from "@/lib/utils";
import type { AlertRecord } from "@/types/zxmail";

function AlertSeverityBadge({ value }: { value: AlertRecord["severity"] }) {
  const styles = {
    info: "bg-[rgba(23,105,255,0.08)] text-[var(--info)]",
    warning: "bg-[var(--warning-soft)] text-[var(--warning)]",
    critical: "bg-[var(--danger-soft)] text-[var(--danger)]",
  };

  return (
    <span className={`inline-flex rounded-full px-3 py-1 text-[11px] font-semibold uppercase tracking-[0.16em] ${styles[value]}`}>
      {value}
    </span>
  );
}

export function AlertCenter({
  alerts,
  onResolve,
  allowResolve = false,
}: {
  alerts: AlertRecord[];
  onResolve?: (alertID: string) => void;
  allowResolve?: boolean;
}) {
  return (
    <DataTable
      rows={alerts}
      getRowKey={(alert) => alert.id}
      emptyState={
        <EmptyState
          title="No alerts"
          description="Open system and deliverability alerts will appear here when thresholds or DNS health checks need attention."
        />
      }
      columns={[
        {
          key: "severity",
          header: "Severity",
          render: (alert) => <AlertSeverityBadge value={alert.severity} />,
        },
        {
          key: "alert",
          header: "Alert",
          render: (alert) => (
            <div>
              <p className="font-semibold">{alert.title}</p>
              <p className="mt-1 text-sm text-[var(--muted)]">{alert.message}</p>
            </div>
          ),
        },
        {
          key: "status",
          header: "Status",
          render: (alert) => (
            <span className="text-sm font-semibold capitalize text-[var(--foreground)]">{alert.status}</span>
          ),
        },
        {
          key: "created_at",
          header: "Created",
          render: (alert) => <p className="text-sm text-[var(--muted)]">{formatDateTime(alert.created_at)}</p>,
        },
        {
          key: "actions",
          header: "Actions",
          className: "w-[160px]",
          render: (alert) =>
            allowResolve && alert.status === "open" ? (
              <Button variant="secondary" onClick={() => onResolve?.(alert.id)}>
                Resolve
              </Button>
            ) : null,
        },
      ]}
    />
  );
}
