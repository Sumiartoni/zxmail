import { CopyButton } from "@/components/shared/copy-button";
import { StatusBadge } from "@/components/shared/status-badge";
import { CodeBlock } from "@/components/ui/code-block";
import type { DNSCheck, DNSRequirement } from "@/types/zxmail";

export function DNSRecordCard({
  record,
  check,
  verified,
}: {
  record: DNSRequirement;
  check?: DNSCheck;
  verified: boolean;
}) {
  return (
    <article className="rounded-[24px] border border-[var(--border)] bg-white p-5 shadow-[var(--shadow-sm)]">
      <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
        <div>
          <div className="flex flex-wrap items-center gap-3">
            <span className="eyebrow">{record.type}</span>
            <StatusBadge value={verified ? "verified" : check?.found ? "verified" : "pending"} />
          </div>
          <h3 className="mt-3 text-lg font-semibold tracking-[-0.03em]">{record.name}</h3>
          <p className="mt-2 text-sm leading-7 text-[var(--muted)]">{record.note}</p>
          {check ? (
            <p className="mt-2 text-xs text-[var(--muted)]">
              {check.found ? `Resolver found ${check.found_value || "a matching value"}.` : "Resolver has not found this required value yet."}
            </p>
          ) : null}
        </div>
        <div className="flex gap-2">
          <CopyButton value={record.name} label="Copy name" />
          <CopyButton value={record.value} label="Copy value" />
        </div>
      </div>
      <CodeBlock label="Record value" className="mt-4">
        {record.value}
      </CodeBlock>
    </article>
  );
}
