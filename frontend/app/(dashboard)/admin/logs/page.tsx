import { LogsTable } from "@/components/logs/logs-table";

export default function AdminLogsPage() {
  return (
    <LogsTable
      title="Admin event explorer"
      description="Operator view across domains and credentials, with the same filter surface used by customer logs plus organization-wide visibility."
    />
  );
}
