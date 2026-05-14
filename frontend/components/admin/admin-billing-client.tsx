"use client";

import { useEffect, useState } from "react";
import { PlanCard } from "@/components/billing/plan-card";
import { useAuth } from "@/components/providers/auth-provider";
import { PageHeader } from "@/components/ui/page-header";
import { ErrorState } from "@/components/ui/error-state";
import { LoadingSkeleton } from "@/components/ui/loading-skeleton";
import { EmptyState } from "@/components/ui/empty-state";
import type { PlanRecord } from "@/types/zxmail";

export function AdminBillingClient() {
  const { api } = useAuth();
  const [plans, setPlans] = useState<PlanRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let mounted = true;
    api.listPlans()
      .then((nextPlans) => {
        if (mounted) {
          setPlans(nextPlans);
          setError(null);
        }
      })
      .catch((nextError) => {
        if (mounted) {
          setError(nextError instanceof Error ? nextError.message : "Failed to load plans");
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
      <PageHeader
        eyebrow="Admin billing"
        title="Published plan catalog"
        description="v2 billing is intentionally gateway-agnostic. Midtrans, Xendit, or other providers can be added later behind the same payment abstraction without changing customer-facing semantics."
      />
      {loading ? (
        <LoadingSkeleton className="h-80" />
      ) : error ? (
        <ErrorState description={error} retryLabel="Retry" onRetry={() => window.location.reload()} />
      ) : plans.length > 0 ? (
        <div className="grid gap-4 xl:grid-cols-3">
          {plans.map((plan) => (
            <PlanCard key={plan.id} plan={plan} />
          ))}
        </div>
      ) : (
        <EmptyState title="No plans configured" description="Create plans through the API or seed flow before customer subscriptions can be assigned." />
      )}
    </div>
  );
}
