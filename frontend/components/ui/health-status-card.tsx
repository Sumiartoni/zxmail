import { StatusBadge } from "@/components/shared/status-badge";

export function HealthStatusCard({
  label,
  value,
  note,
}: {
  label: string;
  value: "healthy" | "degraded" | "ready" | "manual-check";
  note: string;
}) {
  const badgeValue =
    value === "healthy" || value === "ready"
      ? "verified"
      : value === "manual-check"
        ? "pending"
        : "rejected";

  return (
    <div className="rounded-[24px] border border-[var(--border)] bg-white p-5 shadow-[var(--shadow-sm)]">
      <div className="flex items-center justify-between gap-3">
        <div>
          <p className="eyebrow">{label}</p>
          <p className="mt-2 text-lg font-semibold text-[var(--foreground)]">{value}</p>
        </div>
        <StatusBadge value={badgeValue} />
      </div>
      <p className="mt-3 text-sm leading-7 text-[var(--muted)]">{note}</p>
    </div>
  );
}
