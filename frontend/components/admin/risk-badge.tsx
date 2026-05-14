export function RiskBadge({ score }: { score: number }) {
  const tone =
    score >= 80
      ? "bg-[var(--success-soft)] text-[var(--success)]"
      : score >= 60
        ? "bg-[var(--warning-soft)] text-[var(--warning)]"
        : "bg-[var(--danger-soft)] text-[var(--danger)]";

  return (
    <span className={`inline-flex rounded-full px-3 py-1 text-[11px] font-semibold uppercase tracking-[0.16em] ${tone}`}>
      Score {score}
    </span>
  );
}
