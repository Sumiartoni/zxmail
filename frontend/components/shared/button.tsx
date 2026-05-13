import type { ButtonHTMLAttributes, ReactNode } from "react";

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
      "bg-[var(--accent)] text-white shadow-[0_10px_30px_rgba(187,77,35,0.24)] hover:bg-[#9f411e]",
    secondary:
      "border border-[var(--line)] bg-white/75 text-[var(--ink)] hover:bg-white",
    ghost:
      "border border-transparent bg-transparent text-[var(--muted)] hover:border-[var(--line)] hover:bg-white/65 hover:text-[var(--ink)]",
    danger:
      "border border-[#d2a293] bg-[#fff2ee] text-[#8d2d11] hover:bg-[#ffe6df]",
  };

  return (
    <button
      className={`inline-flex items-center justify-center rounded-full px-4 py-2.5 text-sm font-semibold transition disabled:cursor-not-allowed disabled:opacity-55 ${styles[variant]} ${className}`}
      {...props}
    >
      {children}
    </button>
  );
}
