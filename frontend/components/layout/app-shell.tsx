"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import type { ReactNode } from "react";
import { navigationSections } from "@/lib/navigation";
import { useAuth } from "@/components/providers/auth-provider";
import { Button } from "@/components/shared/button";

function isActivePath(pathname: string, href: string) {
  if (href === "/dashboard") {
    return pathname === href;
  }

  return pathname === href || pathname.startsWith(`${href}/`);
}

export function AppShell({ children }: { children: ReactNode }) {
  const pathname = usePathname();
  const { session, hydrated, logout, previewMode } = useAuth();
  const primaryNav = navigationSections.flatMap((section) => section.items);

  return (
    <div className="flex min-h-screen bg-transparent">
      <aside className="hidden w-[296px] shrink-0 flex-col justify-between border-r border-[rgba(255,255,255,0.08)] bg-[var(--sidebar)] px-5 py-5 text-[var(--sidebar-foreground)] xl:flex">
        <div>
          <div className="rounded-[26px] border border-white/10 bg-[linear-gradient(145deg,rgba(23,105,255,0.18),rgba(255,255,255,0.02))] p-5">
            <div className="flex items-center gap-3">
              <div className="flex h-11 w-11 items-center justify-center rounded-2xl bg-white/10 text-lg font-semibold">
                zx
              </div>
              <div>
                <p className="eyebrow !text-[#9fb4d6]">zxMail</p>
                <p className="text-lg font-semibold tracking-[-0.03em]">Production Ready v2</p>
              </div>
            </div>
            <p className="mt-4 text-sm leading-7 text-[var(--sidebar-muted)]">
              Transactional email control plane for domains, SMTP access, billing, usage, deliverability, and admin operations.
            </p>
          </div>

          <div className="mt-8 space-y-6">
            {navigationSections.map((section) => (
              <div key={section.title}>
                <p className="eyebrow mb-3 !text-[#8ea1bd]">{section.title}</p>
                <nav className="space-y-1.5">
                  {section.items.map((item) => {
                    const active = isActivePath(pathname, item.href);
                    return (
                      <Link
                        key={item.href}
                        href={item.href}
                        className={`group flex items-center gap-3 rounded-[18px] border px-4 py-3 transition ${
                          active
                            ? "border-white/10 bg-white/10 text-white"
                            : "border-transparent text-[var(--sidebar-muted)] hover:border-white/6 hover:bg-white/5 hover:text-white"
                        }`}
                      >
                        <span className={`flex h-9 w-9 items-center justify-center rounded-2xl text-[11px] font-semibold uppercase tracking-[0.16em] ${active ? "bg-white/12 text-white" : "bg-white/6 text-[#aebddb]"}`}>
                          {item.short}
                        </span>
                        <span className="block">
                          <span className="block text-sm font-semibold">{item.label}</span>
                          <span className="mt-1 block text-xs uppercase tracking-[0.16em]">
                            {item.note}
                          </span>
                        </span>
                      </Link>
                    );
                  })}
                </nav>
              </div>
            ))}
          </div>
        </div>

        <div className="rounded-[24px] border border-white/10 bg-white/5 p-4 text-sm leading-7 text-[var(--sidebar-muted)]">
          <p className="font-semibold text-white">Operational note</p>
          <p className="mt-2">
            SMTP hostname must remain DNS only in Cloudflare. Passwords are shown
            once. Pre-send quota enforcement still stays limited until zxMail adds
            an SMTP gateway in front of Postal.
          </p>
        </div>
      </aside>

      <div className="flex-1">
        <div className="mx-auto max-w-[1560px] px-4 py-4 md:px-6 md:py-6">
          <header className="mb-6 rounded-[28px] border border-[rgba(199,211,227,0.72)] bg-[rgba(255,255,255,0.82)] px-5 py-4 shadow-[var(--shadow-sm)] backdrop-blur xl:px-6">
            <div className="flex flex-col gap-4 xl:flex-row xl:items-center xl:justify-between">
              <div>
                <div className="flex flex-wrap items-center gap-2">
                  <span className="eyebrow">zxMail dashboard</span>
                  {previewMode ? (
                    <span className="rounded-full bg-[rgba(23,105,255,0.08)] px-3 py-1 text-[11px] font-semibold uppercase tracking-[0.16em] text-[var(--primary)]">
                      Preview mode
                    </span>
                  ) : null}
                  {session?.user.role ? (
                    <span className="rounded-full bg-[#edf3fb] px-3 py-1 text-[11px] font-semibold uppercase tracking-[0.16em] text-[var(--muted)]">
                      {session.user.role}
                    </span>
                  ) : null}
                </div>
                <h1 className="mt-3 text-2xl font-semibold tracking-[-0.05em] text-[var(--foreground)]">
                  {hydrated && session
                    ? `Signed in as ${session.user.email}`
                    : "Control plane for Postal-backed SMTP operations"}
                </h1>
                <p className="mt-2 max-w-3xl text-sm leading-7 text-[var(--muted)]">
                  Production Ready v2 adds manual billing, usage metering, retention, and deliverability health without forcing a platform rewrite.
                </p>
              </div>

              <div className="flex flex-wrap gap-3">
                <Link
                  href="/domains/new"
                  className="inline-flex items-center justify-center rounded-[16px] bg-[var(--primary)] px-4 py-2.5 text-sm font-semibold text-white shadow-[0_14px_32px_rgba(23,105,255,0.24)] transition hover:bg-[#1458d6]"
                >
                  Add domain
                </Link>
                {session ? (
                  <Button
                    variant="secondary"
                    onClick={() => {
                      void logout();
                    }}
                  >
                    Sign out
                  </Button>
                ) : (
                  <Link
                    href="/login"
                    className="inline-flex items-center justify-center rounded-[16px] border border-[var(--border)] bg-white px-4 py-2.5 text-sm font-semibold transition hover:bg-[#f8fbff]"
                  >
                    Sign in
                  </Link>
                )}
              </div>
            </div>

            <div className="hide-scrollbar mt-4 flex gap-3 overflow-x-auto xl:hidden">
              {primaryNav.map((item) => {
                const active = isActivePath(pathname, item.href);
                return (
                  <Link
                    key={item.href}
                    href={item.href}
                    className={`shrink-0 rounded-[14px] border px-4 py-2 text-sm font-semibold transition ${
                      active
                        ? "border-[rgba(23,105,255,0.12)] bg-[rgba(23,105,255,0.08)] text-[var(--primary)]"
                        : "border-[var(--border)] bg-white text-[var(--foreground)]"
                    }`}
                  >
                    {item.label}
                  </Link>
                );
              })}
            </div>
          </header>

          <main>{children}</main>
        </div>
      </div>
    </div>
  );
}
