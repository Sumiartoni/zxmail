import { LogsTable } from "@/components/logs/logs-table";

export default function LogsPage() {
  return (
    <LogsTable
      title="Delivery log explorer"
      description="Search accepted, delivered, bounced, deferred, and rejected events by message ID, recipient, credential, domain, or date range."
    />
  );
}
