import { DataTable } from "@/components/ui/data-table";
import { EmptyState } from "@/components/ui/empty-state";
import { CodeBlock } from "@/components/ui/code-block";
import { formatDateTime } from "@/lib/utils";
import type { AuditLogRecord } from "@/types/zxmail";

export function AuditLogTable({ records }: { records: AuditLogRecord[] }) {
  return (
    <DataTable
      rows={records}
      getRowKey={(record) => record.id}
      emptyState={<EmptyState title="No audit logs" description="Mutating actions, approvals, and admin controls will appear here." />}
      columns={[
        {
          key: "action",
          header: "Action",
          render: (record) => (
            <div>
              <p className="font-semibold">{record.action}</p>
              <p className="mt-1 text-xs uppercase tracking-[0.16em] text-[var(--muted)]">{record.target_type}</p>
            </div>
          ),
        },
        {
          key: "actor",
          header: "Actor",
          render: (record) => <p className="text-sm text-[var(--muted)]">{record.actor_email || "system"}</p>,
        },
        {
          key: "created_at",
          header: "Created",
          render: (record) => <p className="text-sm text-[var(--muted)]">{formatDateTime(record.created_at)}</p>,
        },
        {
          key: "metadata",
          header: "Details",
          render: (record) => <CodeBlock>{JSON.stringify(record.metadata, null, 2)}</CodeBlock>,
        },
      ]}
    />
  );
}
