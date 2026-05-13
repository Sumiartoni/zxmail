import type { CredentialStatus, DomainStatus, LogStatus } from "@/types/zxmail";

type StatusValue = LogStatus | CredentialStatus | DomainStatus | "active" | "released";

const styles: Record<StatusValue, string> = {
  accepted: "bg-[#e6f4ff] text-[#16507a]",
  delivered: "bg-[#e7f7ef] text-[#1c6a4c]",
  bounced: "bg-[#fff0ec] text-[#973b20]",
  deferred: "bg-[#fff7df] text-[#7b5a13]",
  rejected: "bg-[#f9e7e4] text-[#8a2d17]",
  enabled: "bg-[#e7f7ef] text-[#1c6a4c]",
  limited: "bg-[#fff7df] text-[#7b5a13]",
  disabled: "bg-[#efe8db] text-[#615446]",
  verified: "bg-[#e7f7ef] text-[#1c6a4c]",
  pending: "bg-[#fff7df] text-[#7b5a13]",
  active: "bg-[#e7f7ef] text-[#1c6a4c]",
  released: "bg-[#efe8db] text-[#615446]",
};

export function StatusBadge({ value }: { value: StatusValue }) {
  return (
    <span
      className={`inline-flex rounded-full px-3 py-1 text-xs font-semibold uppercase tracking-[0.15em] ${styles[value]}`}
    >
      {value}
    </span>
  );
}
