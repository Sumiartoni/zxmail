import { DataTable } from "@/components/ui/data-table";
import { EmptyState } from "@/components/ui/empty-state";
import { PaymentStatusBadge } from "@/components/billing/payment-status-badge";
import { formatNumber, formatShortDate } from "@/lib/utils";
import type { InvoiceRecord } from "@/types/zxmail";

export function InvoiceTable({
  invoices,
  action,
}: {
  invoices: InvoiceRecord[];
  action?: (invoice: InvoiceRecord) => React.ReactNode;
}) {
  return (
    <DataTable
      rows={invoices}
      getRowKey={(invoice) => invoice.id}
      emptyState={
        <EmptyState
          title="No invoices yet"
          description="Manual payment invoices will appear here after a subscription is assigned."
        />
      }
      columns={[
        {
          key: "invoice",
          header: "Invoice",
          render: (invoice) => (
            <div>
              <p className="font-semibold">{invoice.invoice_number}</p>
              <p className="mt-1 text-xs uppercase tracking-[0.16em] text-[var(--muted)]">
                {formatShortDate(invoice.issued_at)}
              </p>
            </div>
          ),
        },
        {
          key: "status",
          header: "Status",
          render: (invoice) => <PaymentStatusBadge value={invoice.status} />,
        },
        {
          key: "amount",
          header: "Amount",
          render: (invoice) => (
            <div>
              <p className="font-semibold">Rp {formatNumber(invoice.amount)}</p>
              <p className="mt-1 text-xs text-[var(--muted)]">Due {formatShortDate(invoice.due_at)}</p>
            </div>
          ),
        },
        {
          key: "period",
          header: "Period",
          render: (invoice) => (
            <p className="text-sm text-[var(--muted)]">
              {formatShortDate(invoice.period_start)} to {formatShortDate(invoice.period_end)}
            </p>
          ),
        },
        {
          key: "actions",
          header: "Actions",
          className: "w-[160px]",
          render: (invoice) => action?.(invoice) ?? null,
        },
      ]}
    />
  );
}
