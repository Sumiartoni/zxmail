"use client";

import { useEffect, useState } from "react";
import { PaymentStatusBadge } from "@/components/billing/payment-status-badge";
import { useAuth } from "@/components/providers/auth-provider";
import { SectionCard } from "@/components/shared/section-card";
import { ErrorState } from "@/components/ui/error-state";
import { LoadingSkeleton } from "@/components/ui/loading-skeleton";
import { PageHeader } from "@/components/ui/page-header";
import { formatDateTime, formatShortDate } from "@/lib/utils";
import type { OrganizationRecord, SubscriptionView } from "@/types/zxmail";

export function SettingsClient() {
  const { api, session } = useAuth();
  const [organization, setOrganization] = useState<OrganizationRecord | null>(null);
  const [subscription, setSubscription] = useState<SubscriptionView | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let mounted = true;
    async function load() {
      try {
        const [nextOrganization, nextSubscription] = await Promise.all([
          api.getOrganization(),
          api.getSubscription(),
        ]);
        if (mounted) {
          setOrganization(nextOrganization);
          setSubscription(nextSubscription);
          setError(null);
        }
      } catch (nextError) {
        if (mounted) {
          setError(nextError instanceof Error ? nextError.message : "Failed to load settings");
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
        eyebrow="Settings"
        title="Organization and account posture"
        description="Customers can still log in during suspension or past-due billing, but SMTP issuance and high-risk actions may be restricted until the account is healthy again."
      />
      {loading ? (
        <div className="grid gap-4">
          <LoadingSkeleton className="h-40" />
          <LoadingSkeleton className="h-40" />
        </div>
      ) : error ? (
        <ErrorState description={error} retryLabel="Retry" onRetry={() => window.location.reload()} />
      ) : (
        <div className="grid gap-4 xl:grid-cols-2">
          <SectionCard title="Account identity" description="Browser auth uses secure HttpOnly cookies and does not store JWTs in local storage.">
            <dl className="grid gap-4">
              <div>
                <dt className="eyebrow">Signed in as</dt>
                <dd className="mt-2 text-lg font-semibold text-[var(--foreground)]">{session?.user.email}</dd>
              </div>
              <div>
                <dt className="eyebrow">Organization</dt>
                <dd className="mt-2 text-lg font-semibold text-[var(--foreground)]">{organization?.name || "Unassigned"}</dd>
              </div>
              <div>
                <dt className="eyebrow">Created</dt>
                <dd className="mt-2 text-sm text-[var(--muted)]">{formatDateTime(organization?.created_at)}</dd>
              </div>
            </dl>
          </SectionCard>
          <SectionCard title="Billing posture" description="Manual payment and admin approval remain the source of truth for plan activation.">
            {subscription ? (
              <div className="grid gap-4">
                <div>
                  <p className="eyebrow">Plan</p>
                  <p className="mt-2 text-lg font-semibold">{subscription.plan.name}</p>
                </div>
                <div className="flex flex-wrap gap-3">
                  <PaymentStatusBadge value={subscription.subscription.status} />
                  <PaymentStatusBadge value={(subscription.payment_status || "not_required") as never} />
                </div>
                <p className="text-sm text-[var(--muted)]">
                  Current period ends {formatShortDate(subscription.subscription.current_period_end)}.
                </p>
              </div>
            ) : (
              <p className="text-sm leading-7 text-[var(--muted)]">
                No active subscription is assigned yet. Contact an administrator to enable a plan.
              </p>
            )}
          </SectionCard>
        </div>
      )}
    </div>
  );
}
