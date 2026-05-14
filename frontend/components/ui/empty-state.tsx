import type { ReactNode } from "react";

export function EmptyState({
  title,
  description,
  action,
}: {
  title: string;
  description: string;
  action?: ReactNode;
}) {
  return (
    <div className="rounded-[24px] border border-dashed border-[var(--border-strong)] bg-[rgba(255,255,255,0.7)] px-6 py-10 text-center">
      <div className="mx-auto max-w-md">
        <div className="mx-auto h-12 w-12 rounded-2xl bg-[rgba(23,105,255,0.08)]" />
        <h3 className="mt-4 text-lg font-semibold text-[var(--foreground)]">{title}</h3>
        <p className="mt-2 text-sm leading-7 text-[var(--muted)]">{description}</p>
        {action ? <div className="mt-5 flex justify-center">{action}</div> : null}
      </div>
    </div>
  );
}
