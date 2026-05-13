"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { Button } from "@/components/shared/button";
import { useAuth } from "@/components/providers/auth-provider";

export default function LoginPage() {
  const router = useRouter();
  const { login, previewMode } = useAuth();
  const [email, setEmail] = useState("admin@zxmail.site");
  const [password, setPassword] = useState("secret123");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");

  async function handleSubmit() {
    setSubmitting(true);
    setError("");
    try {
      const session = await login(email, password);
      router.push(session.user.role === "admin" ? "/admin" : "/dashboard");
    } catch (nextError) {
      setError(nextError instanceof Error ? nextError.message : "Login failed");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <main className="mesh flex min-h-screen items-center justify-center px-6 py-10">
      <section className="panel w-full max-w-5xl overflow-hidden p-0">
        <div className="grid lg:grid-cols-[0.95fr_1.05fr]">
          <div className="bg-[linear-gradient(145deg,#1f1d16_0%,#3f2e1b_100%)] px-8 py-10 text-white md:px-10">
            <p className="eyebrow !text-[#f9dcc1]">zxMail Production v1</p>
            <h1 className="mt-4 text-4xl font-semibold tracking-[-0.05em]">
              Sign in to the Postal-backed control plane
            </h1>
            <p className="mt-4 max-w-md text-sm leading-7 text-[#ecd9c8]">
              This dashboard covers auth, domain onboarding, DNS verification,
              credentials, logs, suppressions, quota state, and admin customer management.
            </p>

            <div className="mt-8 space-y-3 text-sm text-[#f2e5d6]">
              <div className="rounded-3xl border border-white/10 bg-white/8 p-4">
                SMTP credentials are shown once and never re-rendered after modal close.
              </div>
              <div className="rounded-3xl border border-white/10 bg-white/8 p-4">
                SMTP records must remain DNS only in Cloudflare. Never proxy them.
              </div>
              <div className="rounded-3xl border border-white/10 bg-white/8 p-4">
                Production v1 excludes billing, multi-node orchestration, IP pools, and seed testing.
              </div>
            </div>
          </div>

          <div className="px-8 py-10 md:px-10">
            <div className="max-w-md">
              <div className="flex flex-wrap items-center gap-2">
                <span className="eyebrow">Authentication</span>
                {previewMode ? (
                  <span className="rounded-full bg-[#fff0dc] px-3 py-1 text-xs font-semibold uppercase tracking-[0.15em] text-[#8a4d12]">
                    Preview mode enabled
                  </span>
                ) : null}
              </div>
              <h2 className="mt-4 text-3xl font-semibold tracking-[-0.04em]">
                Login with JWT-backed session handling
              </h2>
              <p className="mt-3 text-sm leading-7 text-[var(--muted)]">
                When a live API base URL is configured, this form uses
                <span className="font-mono text-xs"> /api/v1/auth/login</span>. In preview
                mode, the frontend keeps a local demo session so the Production v1 UI can
                be explored end-to-end.
              </p>

              <div className="mt-8 grid gap-4">
                <input
                  className="field"
                  type="email"
                  placeholder="Email"
                  value={email}
                  onChange={(event) => setEmail(event.target.value)}
                />
                <input
                  className="field"
                  type="password"
                  placeholder="Password"
                  value={password}
                  onChange={(event) => setPassword(event.target.value)}
                />
              </div>

              {error ? (
                <div className="mt-4 rounded-3xl border border-[#d8ad9f] bg-[#fff1ec] px-4 py-3 text-sm text-[#8d2d11]">
                  {error}
                </div>
              ) : null}

              <div className="mt-6 flex flex-wrap gap-3">
                <Button disabled={submitting} onClick={handleSubmit}>
                  {submitting ? "Signing in..." : "Sign in"}
                </Button>
                <Link
                  href="/dashboard"
                  className="inline-flex items-center justify-center rounded-full border border-[var(--line)] bg-white/70 px-4 py-2.5 text-sm font-semibold transition hover:bg-white"
                >
                  Open dashboard
                </Link>
              </div>
            </div>
          </div>
        </div>
      </section>
    </main>
  );
}
