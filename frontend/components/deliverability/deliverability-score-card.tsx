import { SectionCard } from "@/components/shared/section-card";
import { formatPercentage } from "@/lib/utils";
import type { DeliverabilityOverview } from "@/types/zxmail";

export function DeliverabilityScoreCard({ overview }: { overview: DeliverabilityOverview }) {
  return (
    <SectionCard title="Deliverability indicators" description="These are health indicators, not an inbox placement guarantee.">
      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-5">
        <div className="rounded-[20px] bg-[#f7fbff] px-4 py-4">
          <p className="eyebrow">Health score</p>
          <p className="mt-3 text-3xl font-semibold text-[var(--foreground)]">{overview.average_health_score}</p>
        </div>
        <div className="rounded-[20px] bg-[#f7fbff] px-4 py-4">
          <p className="eyebrow">Delivered</p>
          <p className="mt-3 text-2xl font-semibold text-[var(--foreground)]">{formatPercentage(overview.delivered_rate * 100)}</p>
        </div>
        <div className="rounded-[20px] bg-[#f7fbff] px-4 py-4">
          <p className="eyebrow">Bounced</p>
          <p className="mt-3 text-2xl font-semibold text-[var(--foreground)]">{formatPercentage(overview.bounce_rate * 100)}</p>
        </div>
        <div className="rounded-[20px] bg-[#f7fbff] px-4 py-4">
          <p className="eyebrow">Deferred</p>
          <p className="mt-3 text-2xl font-semibold text-[var(--foreground)]">{formatPercentage(overview.deferred_rate * 100)}</p>
        </div>
        <div className="rounded-[20px] bg-[#f7fbff] px-4 py-4">
          <p className="eyebrow">Rejected</p>
          <p className="mt-3 text-2xl font-semibold text-[var(--foreground)]">{formatPercentage(overview.rejected_rate * 100)}</p>
        </div>
      </div>
    </SectionCard>
  );
}
