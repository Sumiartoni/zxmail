type MetricCardProps = {
  label: string;
  value: string;
  note: string;
};

export function MetricCard({ label, value, note }: MetricCardProps) {
  return (
    <article className="panel p-5">
      <p className="eyebrow">{label}</p>
      <div className="mt-3 text-4xl font-semibold tracking-[-0.05em]">
        {value}
      </div>
      <p className="mt-3 text-sm leading-7 text-[var(--muted)]">{note}</p>
    </article>
  );
}
