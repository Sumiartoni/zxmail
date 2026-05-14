import { cn, formatStatusLabel } from "@/lib/utils";

type PaymentLikeStatus =
  | "pending"
  | "approved"
  | "rejected"
  | "failed"
  | "draft"
  | "issued"
  | "paid"
  | "void"
  | "trialing"
  | "active"
  | "past_due"
  | "expired"
  | "suspended"
  | "canceled"
  | "not_required";

const styles: Record<PaymentLikeStatus, string> = {
  pending: "bg-[var(--warning-soft)] text-[var(--warning)]",
  approved: "bg-[var(--success-soft)] text-[var(--success)]",
  rejected: "bg-[var(--danger-soft)] text-[var(--danger)]",
  failed: "bg-[var(--danger-soft)] text-[var(--danger)]",
  draft: "bg-[#eef2f7] text-[#54657d]",
  issued: "bg-[rgba(23,105,255,0.08)] text-[var(--info)]",
  paid: "bg-[var(--success-soft)] text-[var(--success)]",
  void: "bg-[#eef2f7] text-[#54657d]",
  trialing: "bg-[rgba(23,105,255,0.08)] text-[var(--info)]",
  active: "bg-[var(--success-soft)] text-[var(--success)]",
  past_due: "bg-[var(--warning-soft)] text-[var(--warning)]",
  expired: "bg-[var(--danger-soft)] text-[var(--danger)]",
  suspended: "bg-[rgba(200,58,84,0.12)] text-[var(--danger)]",
  canceled: "bg-[#eef2f7] text-[#54657d]",
  not_required: "bg-[#eef2f7] text-[#54657d]",
};

export function PaymentStatusBadge({ value }: { value: PaymentLikeStatus }) {
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
