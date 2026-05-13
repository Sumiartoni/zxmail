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

  return (
    <div className="mx-auto flex min-h-screen max-w-[1440px] gap-6 px-4 py-6 md:px-6">
      <aside className="panel hidden w-80 shrink-0 flex-col justify-between p-5 xl:flex">
        <div>
          <div className="rounded-[28px] bg-[linear-gradient(135deg,#fdf4e0_0%,#f7e5c8_100%)] p-5">
            <p className="eyebrow">zxMail</p>
            <h2 className="mt-2 text-2xl font-semibold tracking-[-0.04em]">
              Production v1 dashboard
            </h2>
            <p className="mt-3 text-sm leading-7 text-[var(--muted)]">
              Auth, domains, credentials, logs, suppressions, quota state, and
              Postal-backed operator controls only.
            </p>
          </div>

          <div className="mt-6 space-y-5">
            {navigationSections.map((section) => (
              <div key={section.title}>
                <p className="eyebrow mb-2">{section.title}</p>
                <nav className="space-y-2">
                  {section.items.map((item) => {
                    const active = isActivePath(pathname, item.href);
                    return (
                      <Link
                        key={item.href}
                        href={item.href}
                        className={`block rounded-2xl border px-4 py-3 transition ${
                          active
                            ? "border-[var(--accent)] bg-[#fff2e8] text-[var(--ink)]"
                            : "border-transparent text-[var(--muted)] hover:border-[var(--line)] hover:bg-white/75 hover:text-[var(--ink)]"
                        }`}
                      >
                        <span className="block text-sm font-semibold">{item.label}</span>
                        <span className="mt-1 block text-xs uppercase tracking-[0.16em]">
                          {item.note}
                        </span>
                      </Link>
                    );
                  })}
                </nav>
              </div>
            ))}
          </div>
        </div>

        <div className="rounded-3xl border border-[var(--line)] bg-white/70 p-4 text-sm leading-7 text-[var(--muted)]">
          <p className="font-semibold text-[var(--ink)]">Operational note</p>
          <p className="mt-2">
            SMTP hostname must remain DNS only in Cloudflare. Passwords are shown
            once. Pre-send quota enforcement stays limited until zxMail adds an
            SMTP gateway in front of Postal.
          </p>
        </div>
      </aside>

      <div className="flex-1">
        <header className="panel mb-6 overflow-hidden p-4 md:p-5">
          <div className="flex flex-col gap-4 xl:flex-row xl:items-center xl:justify-between">
            <div>
              <div className="flex flex-wrap items-center gap-2">
                <span className="eyebrow">Production v1</span>
                {previewMode ? (
                  <span className="rounded-full bg-[#fff0dc] px-3 py-1 text-xs font-semibold uppercase tracking-[0.15em] text-[#8a4d12]">
                    Preview mode
                  </span>
                ) : null}
                {session?.user.role ? (
                  <span className="rounded-full bg-[var(--surface-strong)] px-3 py-1 text-xs font-semibold uppercase tracking-[0.15em] text-[var(--muted)]">
                    {session.user.role}
                  </span>
                ) : null}
              </div>
              <h1 className="mt-3 text-2xl font-semibold tracking-[-0.04em]">
                {hydrated && session
                  ? `Signed in as ${session.user.email}`
                  : "Control plane for Postal-backed SMTP operations"}
              </h1>
              <p className="mt-2 max-w-3xl text-sm leading-7 text-[var(--muted)]">
                Responsive SaaS layout for domain onboarding, credential issuance,
                webhook-driven logs, suppressions, and operator administration.
              </p>
            </div>

            <div className="flex flex-wrap gap-3">
              <Link
                href="/domains/new"
                className="inline-flex items-center justify-center rounded-full bg-[var(--accent)] px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-[#9f411e]"
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
                  className="inline-flex items-center justify-center rounded-full border border-[var(--line)] bg-white/70 px-4 py-2.5 text-sm font-semibold transition hover:bg-white"
                >
                  Sign in
                </Link>
              )}
            </div>
          </div>

          <div className="mt-4 flex gap-3 overflow-x-auto xl:hidden">
            {navigationSections.flatMap((section) => section.items).map((item) => {
              const active = isActivePath(pathname, item.href);
              return (
                <Link
                  key={item.href}
                  href={item.href}
                  className={`shrink-0 rounded-full border px-4 py-2 text-sm font-semibold transition ${
                    active
                      ? "border-[var(--accent)] bg-[#fff2e8]"
                      : "border-[var(--line)] bg-white/70"
                  }`}
                >
                  {item.label}
                </Link>
              );
            })}
          </div>
        </header>

        {children}
      </div>
    </div>
  );
}
