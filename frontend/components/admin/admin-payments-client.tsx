"use client";

import { useEffect, useState } from "react";
import { PaymentStatusBadge } from "@/components/billing/payment-status-badge";
import { useAuth } from "@/components/providers/auth-provider";
import { useToast } from "@/components/providers/toast-provider";
import { Button } from "@/components/shared/button";
import { DataTable } from "@/components/ui/data-table";
import { EmptyState } from "@/components/ui/empty-state";
import { ErrorState } from "@/components/ui/error-state";
import { LoadingSkeleton } from "@/components/ui/loading-skeleton";
import { PageHeader } from "@/components/ui/page-header";
import { formatDateTime, formatNumber } from "@/lib/utils";
import type { PaymentRecord } from "@/types/zxmail";

export function AdminPaymentsClient() {
  const { api } = useAuth();
  const { pushToast } = useToast();
  const [payments, setPayments] = useState<PaymentRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  async function load() {
    setPayments(await api.listPayments());
  }

  useEffect(() => {
    let mounted = true;
    (async () => {
      try {
        const nextPayments = await api.listPayments();
        if (mounted) {
          setPayments(nextPayments);
          setError(null);
        }
      } catch (nextError) {
        if (mounted) {
          setError(nextError instanceof Error ? nextError.message : "Failed to load payments");
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
      <PageHeader eyebrow="Payments" title="Manual payment approvals" description="Approve or reject manual bank transfer and QRIS submissions without exposing payment secrets or gateway credentials in the browser." />
      {loading ? (
        <LoadingSkeleton className="h-80" />
      ) : error ? (
        <ErrorState description={error} retryLabel="Retry" onRetry={() => window.location.reload()} />
      ) : (
        <DataTable
          rows={payments}
          getRowKey={(payment) => payment.id}
          emptyState={<EmptyState title="No payments yet" description="Payment submissions will appear after subscriptions start generating invoices." />}
          columns={[
            {
              key: "provider",
              header: "Provider",
              render: (payment) => (
                <div>
                  <p className="font-semibold">{payment.provider_code}</p>
                  <p className="mt-1 text-xs text-[var(--muted)]">{formatDateTime(payment.submitted_at)}</p>
                </div>
              ),
            },
            {
              key: "status",
              header: "Status",
              render: (payment) => <PaymentStatusBadge value={payment.status} />,
            },
            {
              key: "amount",
              header: "Amount",
              render: (payment) => <p className="font-semibold">Rp {formatNumber(payment.amount)}</p>,
            },
            {
              key: "actions",
              header: "Actions",
              render: (payment) =>
                payment.status === "pending" ? (
                  <div className="flex gap-2">
                    <Button
                      variant="secondary"
                      onClick={async () => {
                        await api.approvePayment(payment.id);
                        await load();
                        pushToast({ tone: "success", title: "Payment approved" });
                      }}
                    >
                      Approve
                    </Button>
                    <Button
                      variant="danger"
                      onClick={async () => {
                        await api.rejectPayment(payment.id);
                        await load();
                        pushToast({ tone: "success", title: "Payment rejected" });
                      }}
                    >
                      Reject
                    </Button>
                  </div>
                ) : null,
            },
          ]}
        />
      )}
    </div>
  );
}
