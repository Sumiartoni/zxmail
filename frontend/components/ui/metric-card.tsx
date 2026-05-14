import type { ReactNode } from "react";

export function MetricCard({
  label,
  value,
  note,
  accent,
}: {
  label: string;
  value: string;
  note: string;
  accent?: ReactNode;
}) {
  return (
    <article className="rounded-[24px] border border-[rgba(199,211,227,0.7)] bg-[var(--card)] p-5 shadow-[var(--shadow-sm)]">
      <div className="flex items-start justify-between gap-3">
        <p className="eyebrow">{label}</p>
        {accent}
      </div>
      <div className="mt-3 text-4xl font-semibold tracking-[-0.06em] text-[var(--foreground)]">
        {value}
      </div>
      <p className="mt-3 text-sm leading-7 text-[var(--muted)]">{note}</p>
    </article>
  );
}
