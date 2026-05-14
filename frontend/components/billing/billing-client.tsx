"use client";

import { useEffect, useState } from "react";
import { PlanCard } from "@/components/billing/plan-card";
import { InvoiceTable } from "@/components/billing/invoice-table";
import { SubscriptionStatusCard } from "@/components/billing/subscription-status-card";
import { useAuth } from "@/components/providers/auth-provider";
import { useToast } from "@/components/providers/toast-provider";
import { SectionCard } from "@/components/shared/section-card";
import { EmptyState } from "@/components/ui/empty-state";
import { ErrorState } from "@/components/ui/error-state";
import { LoadingSkeleton } from "@/components/ui/loading-skeleton";
import { PageHeader } from "@/components/ui/page-header";
import type { InvoiceRecord, PlanRecord, SubscriptionView } from "@/types/zxmail";

export function BillingClient() {
  const { api } = useAuth();
  const { pushToast } = useToast();
  const [plans, setPlans] = useState<PlanRecord[]>([]);
  const [subscription, setSubscription] = useState<SubscriptionView | null>(null);
  const [invoices, setInvoices] = useState<InvoiceRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let mounted = true;

    async function load() {
      try {
        const [nextPlans, nextSubscription, nextInvoices] = await Promise.all([
          api.listPlans(),
          api.getSubscription(),
          api.listInvoices(),
        ]);
        if (!mounted) {
          return;
        }
        setPlans(nextPlans);
        setSubscription(nextSubscription);
        setInvoices(nextInvoices);
        setError(null);
      } catch (nextError) {
        if (mounted) {
          setError(nextError instanceof Error ? nextError.message : "Failed to load billing");
        }
      } finally {
        if (mounted) {
          setLoading(false);
        }
      }
    }

    void load();
    return () => {
      mounted = false;
    };
  }, [api]);

  return (
    <div className="space-y-6">
      <PageHeader
        eyebrow="Billing"
        title="Manual billing, invoices, and package visibility"
        description="Production Ready v2 keeps payments gateway-agnostic. Customers can review their assigned package, invoice history, and current payment state without changing plans directly."
      />

      {loading ? (
        <div className="grid gap-4">
          <LoadingSkeleton className="h-40" />
          <LoadingSkeleton className="h-72" />
          <LoadingSkeleton className="h-72" />
        </div>
      ) : error ? (
        <ErrorState title="Billing unavailable" description={error} retryLabel="Retry" onRetry={() => window.location.reload()} />
      ) : (
        <>
          <SubscriptionStatusCard subscription={subscription} />

          <SectionCard title="Available plans" description="Plan changes still require admin approval in v2.">
            {plans.length > 0 ? (
              <div className="grid gap-4 xl:grid-cols-3">
                {plans.map((plan) => (
                  <PlanCard
                    key={plan.id}
                    plan={plan}
                    featuredLabel={subscription?.plan.id === plan.id ? "current plan" : undefined}
                    actionLabel={subscription?.plan.id === plan.id ? undefined : "Request via admin"}
                    onAction={
                      subscription?.plan.id === plan.id
                        ? undefined
                        : () =>
                            pushToast({
                              tone: "info",
                              title: "Admin approval required",
                              description: "Plan changes remain admin-managed in Production Ready v2.",
                            })
                    }
                  />
                ))}
              </div>
            ) : (
              <EmptyState
                title="No plans published"
                description="An administrator still needs to publish plans before customer billing can start."
              />
            )}
          </SectionCard>

          <SectionCard title="Invoice history" description="Invoices are generated and tracked without exposing payment secrets in the browser.">
            <InvoiceTable invoices={invoices} />
          </SectionCard>
        </>
      )}
    </div>
  );
}
