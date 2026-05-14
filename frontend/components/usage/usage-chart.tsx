import { MetricCard } from "@/components/ui/metric-card";
import { formatNumber } from "@/lib/utils";
import type { UsageOverview } from "@/types/zxmail";

export function UsageChart({ usage }: { usage: UsageOverview }) {
  return (
    <section className="grid gap-4 md:grid-cols-2 xl:grid-cols-5">
      <MetricCard label="Accepted today" value={formatNumber(usage.accepted_today)} note="Webhook accepted events today." />
      <MetricCard label="Accepted month" value={formatNumber(usage.accepted_month)} note="Source of truth for billing and quota." />
      <MetricCard label="Delivered" value={formatNumber(usage.delivered_month)} note="Delivered events in current month." />
      <MetricCard label="Bounced" value={formatNumber(usage.bounced_month)} note="Bounce count tracked from Postal webhooks." />
      <MetricCard label="Overage" value={formatNumber(usage.overage_count)} note="Overage records if plan quota is exceeded." />
    </section>
  );
}
