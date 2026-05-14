import type { ReactNode } from "react";

type SectionCardProps = {
  title: string;
  description?: string;
  children: ReactNode;
};

export function SectionCard({
  title,
  description,
  children,
}: SectionCardProps) {
  return (
    <section className="rounded-[28px] border border-[rgba(199,211,227,0.72)] bg-[var(--card)] p-6 shadow-[var(--shadow-sm)]">
      <div className="mb-5">
        <h2 className="text-xl font-semibold tracking-[-0.03em] text-[var(--foreground)]">
          {title}
        </h2>
        {description ? (
          <p className="mt-2 text-sm leading-7 text-[var(--muted)]">{description}</p>
        ) : null}
      </div>
      {children}
    </section>
  );
}
