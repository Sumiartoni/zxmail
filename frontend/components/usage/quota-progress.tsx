import { PaymentStatusBadge } from "@/components/billing/payment-status-badge";
import { QuotaUsageBar } from "@/components/ui/quota-usage-bar";
import { SectionCard } from "@/components/shared/section-card";
import type { UsageOverview } from "@/types/zxmail";

export function QuotaProgress({ usage }: { usage: UsageOverview }) {
  return (
    <SectionCard title="Quota posture" description="Advisory quota state based on accepted volume and configured plan or admin overrides.">
      <div className="mb-4">
        <PaymentStatusBadge value={usage.status as never} />
      </div>
      <div className="grid gap-3 md:grid-cols-3">
        <QuotaUsageBar label="Daily" used={usage.accepted_today} limit={usage.effective_daily_quota} />
        <QuotaUsageBar label="Monthly" used={usage.accepted_month} limit={usage.effective_monthly_quota} />
        <QuotaUsageBar label="Per-minute cap" used={0} limit={usage.effective_per_minute_quota} />
      </div>
    </SectionCard>
  );
}
