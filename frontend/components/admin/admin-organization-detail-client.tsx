"use client";

import { useEffect, useState } from "react";
import { AdminActionPanel } from "@/components/admin/admin-action-panel";
import { PlanCard } from "@/components/billing/plan-card";
import { PaymentStatusBadge } from "@/components/billing/payment-status-badge";
import { useAuth } from "@/components/providers/auth-provider";
import { useToast } from "@/components/providers/toast-provider";
import { SectionCard } from "@/components/shared/section-card";
import { ErrorState } from "@/components/ui/error-state";
import { LoadingSkeleton } from "@/components/ui/loading-skeleton";
import { PageHeader } from "@/components/ui/page-header";
import { formatDateTime, formatPercentage } from "@/lib/utils";
import type { AdminOrganizationDetail, PlanRecord } from "@/types/zxmail";

export function AdminOrganizationDetailClient({ organizationID }: { organizationID: string }) {
  const { api } = useAuth();
  const { pushToast } = useToast();
  const [detail, setDetail] = useState<AdminOrganizationDetail | null>(null);
  const [plans, setPlans] = useState<PlanRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  async function load() {
    const [nextDetail, nextPlans] = await Promise.all([
      api.getAdminOrganizationDetail(organizationID),
      api.listPlans(),
    ]);
    setDetail(nextDetail);
    setPlans(nextPlans);
  }

  useEffect(() => {
    let mounted = true;
    (async () => {
      try {
        const [nextDetail, nextPlans] = await Promise.all([
          api.getAdminOrganizationDetail(organizationID),
          api.listPlans(),
        ]);
        if (mounted) {
          setDetail(nextDetail);
          setPlans(nextPlans);
          setError(null);
        }
      } catch (nextError) {
        if (mounted) {
          setError(nextError instanceof Error ? nextError.message : "Failed to load organization detail");
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
  }, [api, organizationID]);

  if (loading) {
    return <LoadingSkeleton className="h-96" />;
  }

  if (error || !detail) {
    return <ErrorState description={error || "Organization detail unavailable."} retryLabel="Retry" onRetry={() => window.location.reload()} />;
  }

  return (
    <div className="space-y-6">
      <PageHeader
        eyebrow="Organization detail"
        title={detail.name}
        description="Admins can suspend organizations, disable credentials, and assign plans without deleting historical data."
      />

      <div className="grid gap-4 xl:grid-cols-[1.05fr_0.95fr]">
        <SectionCard title="Current state" description="This snapshot combines billing, quota, and deliverability posture.">
          <div className="grid gap-4 md:grid-cols-2">
            <div className="rounded-[18px] bg-[#f7fbff] px-4 py-4">
              <p className="eyebrow">Subscription</p>
              <div className="mt-3">
                <PaymentStatusBadge value={detail.current_subscription_status as never} />
              </div>
            </div>
            <div className="rounded-[18px] bg-[#f7fbff] px-4 py-4">
              <p className="eyebrow">Payment</p>
              <div className="mt-3">
                <PaymentStatusBadge value={detail.payment_status as never} />
              </div>
            </div>
            <div className="rounded-[18px] bg-[#f7fbff] px-4 py-4">
              <p className="eyebrow">Verified domains</p>
              <p className="mt-3 text-lg font-semibold">{detail.verified_domains}</p>
            </div>
            <div className="rounded-[18px] bg-[#f7fbff] px-4 py-4">
              <p className="eyebrow">Enabled credentials</p>
              <p className="mt-3 text-lg font-semibold">{detail.enabled_credentials}</p>
            </div>
            <div className="rounded-[18px] bg-[#f7fbff] px-4 py-4">
              <p className="eyebrow">Latest send activity</p>
              <p className="mt-3 text-sm text-[var(--muted)]">{formatDateTime(detail.latest_send_activity_at)}</p>
            </div>
            <div className="rounded-[18px] bg-[#f7fbff] px-4 py-4">
              <p className="eyebrow">Bounce rate</p>
              <p className="mt-3 text-lg font-semibold">{formatPercentage(detail.bounce_rate * 100)}</p>
            </div>
          </div>
        </SectionCard>

        <AdminActionPanel
          suspended={detail.suspended}
          onSuspend={async () => {
            await api.suspendOrganization(organizationID, "manual admin action");
            await load();
            pushToast({ tone: "success", title: "Organization suspended" });
          }}
          onUnsuspend={async () => {
            await api.unsuspendOrganization(organizationID);
            await load();
            pushToast({ tone: "success", title: "Organization unsuspended" });
          }}
          onDisableCredentials={async () => {
            await api.disableOrganizationCredentials(organizationID);
            await load();
            pushToast({ tone: "success", title: "Credentials disabled" });
          }}
        />
      </div>

      <SectionCard title="Assign package" description="Plan assignment remains admin-managed in v2 and triggers manual invoice or payment tracking.">
        <div className="grid gap-4 xl:grid-cols-3">
          {plans.map((plan) => (
            <PlanCard
              key={plan.id}
              plan={plan}
              actionLabel="Assign plan"
              onAction={async () => {
                await api.assignSubscription(organizationID, {
                  plan_id: plan.id,
                  payment_provider: plan.payment_methods[0] || "manual_bank_transfer",
                  notes: "Assigned from admin organization detail",
                  start_trial: plan.trial_days > 0,
                });
                await load();
                pushToast({ tone: "success", title: `${plan.name} assigned` });
              }}
            />
          ))}
        </div>
      </SectionCard>
    </div>
  );
}
