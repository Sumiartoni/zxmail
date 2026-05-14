import { StatusBadge } from "@/components/shared/status-badge";
import { SectionCard } from "@/components/shared/section-card";
import { formatPercentage, formatShortDate } from "@/lib/utils";
import type { DomainHealthRecord } from "@/types/zxmail";

export function DomainHealthChecklist({ health }: { health: DomainHealthRecord }) {
  const checks = [
    { label: "SPF", ok: health.spf_found },
    { label: "DKIM", ok: health.dkim_found },
    { label: "DMARC", ok: health.dmarc_found },
    { label: "MX / bounce note", ok: health.mx_note_found },
    { label: "Quota limited", ok: !health.quota_limited },
  ];

  return (
    <SectionCard
      title={health.domain_name}
      description={`Last checked ${formatShortDate(health.checked_at)}. rDNS remains a manual checklist item via the VPS provider.`}
    >
      <div className="grid gap-3 md:grid-cols-2">
        {checks.map((check) => (
          <div key={check.label} className="flex items-center justify-between rounded-[18px] border border-[var(--border)] bg-white px-4 py-3">
            <p className="text-sm font-semibold text-[var(--foreground)]">{check.label}</p>
            <StatusBadge value={check.ok ? "verified" : "pending"} />
          </div>
        ))}
      </div>
      <div className="mt-4 grid gap-3 md:grid-cols-3">
        <div className="rounded-[18px] bg-[#f7fbff] px-4 py-3">
          <p className="eyebrow">Bounce rate</p>
          <p className="mt-2 text-lg font-semibold">{formatPercentage(health.bounce_rate * 100)}</p>
        </div>
        <div className="rounded-[18px] bg-[#f7fbff] px-4 py-3">
          <p className="eyebrow">Deferred rate</p>
          <p className="mt-2 text-lg font-semibold">{formatPercentage(health.deferred_rate * 100)}</p>
        </div>
        <div className="rounded-[18px] bg-[#f7fbff] px-4 py-3">
          <p className="eyebrow">Rejected rate</p>
          <p className="mt-2 text-lg font-semibold">{formatPercentage(health.rejected_rate * 100)}</p>
        </div>
      </div>
    </SectionCard>
  );
}
