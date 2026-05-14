import type { ReactNode } from "react";
import { cn } from "@/lib/utils";

export function CodeBlock({
  label,
  children,
  className,
}: {
  label?: string;
  children: ReactNode;
  className?: string;
}) {
  return (
    <div className={cn("rounded-[20px] border border-[var(--border)] bg-[#0f172a] p-4 text-[#d8e4ff]", className)}>
      {label ? <p className="eyebrow !text-[#8da2c2]">{label}</p> : null}
      <pre className="mt-3 overflow-x-auto whitespace-pre-wrap break-words font-mono text-xs leading-6">
        {children}
      </pre>
    </div>
  );
}
