import type { ReactNode } from "react";

type SectionCardProps = {
  title: string;
  description: string;
  children: ReactNode;
};

export function SectionCard({
  title,
  description,
  children,
}: SectionCardProps) {
  return (
    <section className="panel p-6">
      <div className="mb-5">
        <h2 className="text-xl font-semibold tracking-[-0.03em]">{title}</h2>
        <p className="mt-2 text-sm leading-7 text-[var(--muted)]">
          {description}
        </p>
      </div>
      {children}
    </section>
  );
}
