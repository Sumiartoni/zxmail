"use client";

import { useEffect, useState } from "react";
import { InvoiceTable } from "@/components/billing/invoice-table";
import { useAuth } from "@/components/providers/auth-provider";
import { useToast } from "@/components/providers/toast-provider";
import { Button } from "@/components/shared/button";
import { ErrorState } from "@/components/ui/error-state";
import { LoadingSkeleton } from "@/components/ui/loading-skeleton";
import { PageHeader } from "@/components/ui/page-header";
import type { InvoiceRecord } from "@/types/zxmail";

export function AdminInvoicesClient() {
  const { api } = useAuth();
  const { pushToast } = useToast();
  const [invoices, setInvoices] = useState<InvoiceRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  async function load() {
    setInvoices(await api.listAdminInvoices());
  }

  useEffect(() => {
    let mounted = true;
    (async () => {
      try {
        const nextInvoices = await api.listAdminInvoices();
        if (mounted) {
          setInvoices(nextInvoices);
          setError(null);
        }
      } catch (nextError) {
        if (mounted) {
          setError(nextError instanceof Error ? nextError.message : "Failed to load invoices");
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
      <PageHeader eyebrow="Invoices" title="Invoice control surface" description="Admins can move invoices to paid or failed while keeping payment evidence and audit logs intact." />
      {loading ? (
        <LoadingSkeleton className="h-80" />
      ) : error ? (
        <ErrorState description={error} retryLabel="Retry" onRetry={() => window.location.reload()} />
      ) : (
        <InvoiceTable
          invoices={invoices}
          action={(invoice) =>
            invoice.status === "issued" ? (
              <div className="flex gap-2">
                <Button
                  variant="secondary"
                  onClick={async () => {
                    await api.markInvoicePaid(invoice.id);
                    await load();
                    pushToast({ tone: "success", title: "Invoice marked paid" });
                  }}
                >
                  Mark paid
                </Button>
                <Button
                  variant="danger"
                  onClick={async () => {
                    await api.markInvoiceFailed(invoice.id);
                    await load();
                    pushToast({ tone: "success", title: "Invoice marked failed" });
                  }}
                >
                  Mark failed
                </Button>
              </div>
            ) : null
          }
        />
      )}
    </div>
  );
}
