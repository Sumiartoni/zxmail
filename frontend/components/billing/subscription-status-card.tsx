import { PaymentStatusBadge } from "@/components/billing/payment-status-badge";
import { SectionCard } from "@/components/shared/section-card";
import { formatShortDate } from "@/lib/utils";
import type { SubscriptionView } from "@/types/zxmail";

export function SubscriptionStatusCard({
  subscription,
}: {
  subscription: SubscriptionView | null;
}) {
  if (!subscription) {
    return (
      <SectionCard title="Subscription" description="No subscription has been assigned to this organization yet.">
        <p className="text-sm leading-7 text-[var(--muted)]">
          Billing is gateway-agnostic in v2. An admin still needs to assign a plan before
          manual payment and invoice tracking become active.
        </p>
      </SectionCard>
    );
  }

  return (
    <SectionCard
      title={`${subscription.plan.name} subscription`}
      description="Plan, payment, and expiry state all affect SMTP issuance and quota posture."
    >
      <div className="grid gap-4 md:grid-cols-3">
        <div className="rounded-[20px] bg-[#f7fbff] px-4 py-4">
          <p className="text-xs uppercase tracking-[0.16em] text-[var(--muted)]">Subscription</p>
          <div className="mt-3">
            <PaymentStatusBadge value={subscription.subscription.status} />
          </div>
        </div>
        <div className="rounded-[20px] bg-[#f7fbff] px-4 py-4">
          <p className="text-xs uppercase tracking-[0.16em] text-[var(--muted)]">Payment</p>
          <div className="mt-3">
            <PaymentStatusBadge value={(subscription.payment_status || "not_required") as never} />
          </div>
        </div>
        <div className="rounded-[20px] bg-[#f7fbff] px-4 py-4">
          <p className="text-xs uppercase tracking-[0.16em] text-[var(--muted)]">Current period end</p>
          <p className="mt-3 text-lg font-semibold text-[var(--foreground)]">
            {formatShortDate(subscription.subscription.current_period_end)}
          </p>
        </div>
      </div>
    </SectionCard>
  );
}
