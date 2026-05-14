import { cn } from "@/lib/utils";

export function Stepper({
  steps,
  current,
}: {
  steps: Array<{ id: number; label: string; detail?: string }>;
  current: number;
}) {
  return (
    <div className="grid gap-3 md:grid-cols-3">
      {steps.map((step) => {
        const completed = current > step.id;
        const active = current === step.id;
        return (
          <div
            key={step.id}
            className={cn(
              "rounded-[22px] border px-4 py-4 transition",
              active
                ? "border-[rgba(23,105,255,0.22)] bg-[rgba(23,105,255,0.08)]"
                : completed
                  ? "border-[rgba(8,127,91,0.18)] bg-[rgba(8,127,91,0.06)]"
                  : "border-[var(--border)] bg-white/70",
            )}
          >
            <div className="flex items-center gap-3">
              <div
                className={cn(
                  "flex h-8 w-8 items-center justify-center rounded-full text-sm font-semibold",
                  active
                    ? "bg-[var(--primary)] text-white"
                    : completed
                      ? "bg-[var(--success)] text-white"
                      : "bg-[#eaf0f8] text-[var(--muted)]",
                )}
              >
                {step.id}
              </div>
              <div>
                <p className="text-sm font-semibold text-[var(--foreground)]">{step.label}</p>
                {step.detail ? <p className="text-xs text-[var(--muted)]">{step.detail}</p> : null}
              </div>
            </div>
          </div>
        );
      })}
    </div>
  );
}
