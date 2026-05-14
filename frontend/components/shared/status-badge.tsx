import type { CredentialStatus, DomainStatus, LogStatus } from "@/types/zxmail";
import { cn, formatStatusLabel } from "@/lib/utils";

type StatusValue = LogStatus | CredentialStatus | DomainStatus | "active" | "released";

const styles: Record<StatusValue, string> = {
  accepted: "bg-[rgba(23,105,255,0.08)] text-[var(--info)]",
  delivered: "bg-[var(--success-soft)] text-[var(--success)]",
  bounced: "bg-[var(--danger-soft)] text-[var(--danger)]",
  deferred: "bg-[var(--warning-soft)] text-[var(--warning)]",
  rejected: "bg-[rgba(200,58,84,0.12)] text-[var(--danger)]",
  enabled: "bg-[var(--success-soft)] text-[var(--success)]",
  limited: "bg-[var(--warning-soft)] text-[var(--warning)]",
  disabled: "bg-[#eef2f7] text-[#54657d]",
  verified: "bg-[var(--success-soft)] text-[var(--success)]",
  pending: "bg-[var(--warning-soft)] text-[var(--warning)]",
  active: "bg-[var(--success-soft)] text-[var(--success)]",
  released: "bg-[#eef2f7] text-[#54657d]",
};

export function StatusBadge({ value }: { value: StatusValue }) {
  return (
    <span
      className={cn(
        "inline-flex rounded-full px-3 py-1 text-[11px] font-semibold uppercase tracking-[0.16em]",
        styles[value],
      )}
    >
      {formatStatusLabel(value)}
    </span>
  );
}
