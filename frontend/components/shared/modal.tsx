"use client";

import type { ReactNode } from "react";
import { cn } from "@/lib/utils";

export function Modal({
  open,
  title,
  description,
  onClose,
  children,
  size = "lg",
}: {
  open: boolean;
  title: string;
  description?: string;
  onClose: () => void;
  children: ReactNode;
  size?: "md" | "lg" | "xl";
}) {
  if (!open) {
    return null;
  }

  const sizeClass = {
    md: "max-w-xl",
    lg: "max-w-2xl",
    xl: "max-w-4xl",
  }[size];

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-[rgba(7,12,24,0.52)] px-4 py-6 backdrop-blur-sm">
      <div className={cn("panel max-h-[90vh] w-full overflow-y-auto p-6 md:p-7", sizeClass)}>
        <div className="flex items-start justify-between gap-4">
          <div>
            <h2 className="text-2xl font-semibold tracking-[-0.04em] text-[var(--foreground)]">
              {title}
            </h2>
            {description ? (
              <p className="mt-2 max-w-xl text-sm leading-7 text-[var(--muted)]">
                {description}
              </p>
            ) : null}
          </div>
          <button
            type="button"
            onClick={onClose}
            className="rounded-[14px] border border-[var(--border)] px-3 py-2 text-xs font-semibold uppercase tracking-[0.15em] text-[var(--muted)] transition hover:bg-[#f8fbff]"
          >
            Close
          </button>
        </div>
        <div className="mt-6">{children}</div>
      </div>
    </div>
  );
}
