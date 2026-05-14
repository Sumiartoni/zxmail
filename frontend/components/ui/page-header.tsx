import type { ReactNode } from "react";

export function PageHeader({
  eyebrow,
  title,
  description,
  actions,
}: {
  eyebrow?: string;
  title: string;
  description?: string;
  actions?: ReactNode;
}) {
  return (
    <section className="rounded-[28px] border border-[rgba(199,211,227,0.7)] bg-[linear-gradient(135deg,rgba(255,255,255,0.96),rgba(239,245,252,0.96))] px-6 py-6 shadow-[var(--shadow-sm)] md:px-7 md:py-7">
      <div className="flex flex-col gap-5 lg:flex-row lg:items-end lg:justify-between">
        <div className="max-w-3xl">
          {eyebrow ? <p className="eyebrow">{eyebrow}</p> : null}
          <h1 className="mt-3 text-3xl font-semibold tracking-[-0.05em] text-[var(--foreground)] md:text-[2rem]">
            {title}
          </h1>
          {description ? (
            <p className="mt-3 text-sm leading-7 text-[var(--muted)] md:text-[15px]">
              {description}
            </p>
          ) : null}
        </div>
        {actions ? <div className="flex flex-wrap gap-3">{actions}</div> : null}
      </div>
    </section>
  );
}
