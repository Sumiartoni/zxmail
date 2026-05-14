"use client";

import type { ReactNode } from "react";
import { cn } from "@/lib/utils";

export function Drawer({
  open,
  title,
  description,
  onClose,
  children,
}: {
  open: boolean;
  title: string;
  description?: string;
  onClose: () => void;
  children: ReactNode;
}) {
  if (!open) {
    return null;
  }

  return (
    <div className="fixed inset-0 z-[80] flex justify-end bg-[rgba(7,12,24,0.44)] backdrop-blur-sm">
      <div className={cn("flex h-full w-full max-w-2xl flex-col border-l border-[var(--border)] bg-white shadow-[var(--shadow-lg)]")}>
        <div className="flex items-start justify-between gap-4 border-b border-[var(--border)] px-5 py-5">
          <div>
            <h2 className="text-2xl font-semibold tracking-[-0.04em]">{title}</h2>
            {description ? <p className="mt-2 text-sm leading-7 text-[var(--muted)]">{description}</p> : null}
          </div>
          <button
            type="button"
            onClick={onClose}
            className="rounded-[14px] border border-[var(--border)] px-3 py-2 text-xs font-semibold uppercase tracking-[0.15em] text-[var(--muted)]"
          >
            Close
          </button>
        </div>
        <div className="hide-scrollbar flex-1 overflow-y-auto px-5 py-5">{children}</div>
      </div>
    </div>
  );
}
