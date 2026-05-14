import { StatusBadge } from "@/components/shared/status-badge";
import { cn } from "@/lib/utils";

const orderedStatuses = ["accepted", "delivered", "bounced", "deferred", "rejected"] as const;

export function MessageTimeline({ currentStatus }: { currentStatus: string }) {
  return (
    <div className="rounded-[22px] border border-[var(--border)] bg-[#fbfdff] p-4">
      <p className="eyebrow">Timeline</p>
      <div className="mt-4 flex flex-wrap gap-3">
        {orderedStatuses.map((status, index) => {
          const active = currentStatus === status;
          const reached =
            orderedStatuses.indexOf(currentStatus as (typeof orderedStatuses)[number]) >= index;

          return (
            <div key={status} className="flex items-center gap-3">
              <div className="flex flex-col items-center gap-2">
                <div
                  className={cn(
                    "h-3 w-3 rounded-full",
                    active
                      ? "bg-[var(--primary)]"
                      : reached
                        ? "bg-[var(--success)]"
                        : "bg-[#d7e1ef]",
                  )}
                />
                <StatusBadge value={status} />
              </div>
              {index < orderedStatuses.length - 1 ? (
                <div className="hidden h-px w-10 bg-[#d7e1ef] md:block" />
              ) : null}
            </div>
          );
        })}
      </div>
    </div>
  );
}
