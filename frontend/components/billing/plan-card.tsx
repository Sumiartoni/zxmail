import { Button } from "@/components/shared/button";
import { PaymentStatusBadge } from "@/components/billing/payment-status-badge";
import { formatNumber } from "@/lib/utils";
import type { PlanRecord } from "@/types/zxmail";

export function PlanCard({
  plan,
  featuredLabel,
  actionLabel,
  onAction,
}: {
  plan: PlanRecord;
  featuredLabel?: string;
  actionLabel?: string;
  onAction?: () => void;
}) {
  return (
    <article className="rounded-[26px] border border-[var(--border)] bg-white p-5 shadow-[var(--shadow-sm)]">
      <div className="flex items-start justify-between gap-4">
        <div>
          <p className="eyebrow">{plan.code}</p>
          <h3 className="mt-2 text-xl font-semibold tracking-[-0.04em] text-[var(--foreground)]">
            {plan.name}
          </h3>
          <p className="mt-2 text-sm leading-7 text-[var(--muted)]">{plan.description}</p>
        </div>
        <div className="flex flex-col items-end gap-2">
          <PaymentStatusBadge value={plan.active ? "active" : "canceled"} />
          {featuredLabel ? (
            <span className="rounded-full bg-[rgba(23,105,255,0.08)] px-3 py-1 text-[11px] font-semibold uppercase tracking-[0.16em] text-[var(--primary)]">
              {featuredLabel}
            </span>
          ) : null}
        </div>
      </div>
      <div className="mt-5 flex items-end justify-between gap-4">
        <div>
          <p className="text-3xl font-semibold tracking-[-0.05em] text-[var(--foreground)]">
            Rp {formatNumber(plan.price_monthly)}
          </p>
          <p className="mt-1 text-xs uppercase tracking-[0.16em] text-[var(--muted)]">
            billed monthly
          </p>
        </div>
        {onAction && actionLabel ? <Button onClick={onAction}>{actionLabel}</Button> : null}
      </div>
      <dl className="mt-5 grid gap-3 md:grid-cols-2">
        <div className="rounded-[18px] bg-[#f7fbff] px-4 py-3">
          <dt className="text-xs uppercase tracking-[0.16em] text-[var(--muted)]">Monthly quota</dt>
          <dd className="mt-1 text-sm font-semibold text-[var(--foreground)]">
            {plan.monthly_quota ? formatNumber(plan.monthly_quota) : "Unlimited"}
          </dd>
        </div>
        <div className="rounded-[18px] bg-[#f7fbff] px-4 py-3">
          <dt className="text-xs uppercase tracking-[0.16em] text-[var(--muted)]">Per-minute cap</dt>
          <dd className="mt-1 text-sm font-semibold text-[var(--foreground)]">
            {plan.per_minute_quota ? formatNumber(plan.per_minute_quota) : "Unlimited"}
          </dd>
        </div>
        <div className="rounded-[18px] bg-[#f7fbff] px-4 py-3">
          <dt className="text-xs uppercase tracking-[0.16em] text-[var(--muted)]">Daily quota</dt>
          <dd className="mt-1 text-sm font-semibold text-[var(--foreground)]">
            {plan.daily_quota ? formatNumber(plan.daily_quota) : "Unlimited"}
          </dd>
        </div>
        <div className="rounded-[18px] bg-[#f7fbff] px-4 py-3">
          <dt className="text-xs uppercase tracking-[0.16em] text-[var(--muted)]">Trial</dt>
          <dd className="mt-1 text-sm font-semibold text-[var(--foreground)]">{plan.trial_days} days</dd>
        </div>
      </dl>
    </article>
  );
}
