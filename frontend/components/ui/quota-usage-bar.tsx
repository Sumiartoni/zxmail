import { formatNumber, getUsageRatio } from "@/lib/utils";

export function QuotaUsageBar({
  label,
  used,
  limit,
}: {
  label: string;
  used: number;
  limit?: number | null;
}) {
  const ratio = getUsageRatio(used, limit);

  return (
    <div className="rounded-[20px] border border-[var(--border)] bg-[#fbfdff] p-4">
      <div className="flex items-center justify-between gap-3">
        <p className="text-sm font-semibold text-[var(--foreground)]">{label}</p>
        <p className="text-xs text-[var(--muted)]">
          {formatNumber(used)} / {limit ? formatNumber(limit) : "Unlimited"}
        </p>
      </div>
      <div className="mt-3 h-2 rounded-full bg-[#e7eef8]">
        <div
          className="h-2 rounded-full bg-[var(--primary)]"
          style={{ width: `${Math.max(ratio, limit ? 6 : 24)}%` }}
        />
      </div>
    </div>
  );
}
