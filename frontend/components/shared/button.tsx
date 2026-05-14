import type { ButtonHTMLAttributes, ReactNode } from "react";
import { cn } from "@/lib/utils";

type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: "primary" | "secondary" | "ghost" | "danger";
  children: ReactNode;
};

export function Button({
  variant = "primary",
  className = "",
  children,
  ...props
}: ButtonProps) {
  const styles = {
    primary:
      "border border-transparent bg-[var(--primary)] text-[var(--primary-foreground)] shadow-[0_14px_32px_rgba(23,105,255,0.24)] hover:bg-[#1458d6]",
    secondary:
      "border border-[var(--border)] bg-white text-[var(--foreground)] hover:bg-[#f8fbff]",
    ghost:
      "border border-transparent bg-transparent text-[var(--muted)] hover:border-[var(--border)] hover:bg-white/85 hover:text-[var(--foreground)]",
    danger:
      "border border-[rgba(200,58,84,0.15)] bg-[rgba(200,58,84,0.08)] text-[var(--danger)] hover:bg-[rgba(200,58,84,0.12)]",
  };

  return (
    <button
      className={cn(
        "inline-flex items-center justify-center gap-2 rounded-[16px] px-4 py-2.5 text-sm font-semibold transition duration-150 disabled:cursor-not-allowed disabled:opacity-55",
        styles[variant],
        className,
      )}
      {...props}
    >
      {children}
    </button>
  );
}
