"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { Button } from "@/components/shared/button";
import { useAuth } from "@/components/providers/auth-provider";
import { useToast } from "@/components/providers/toast-provider";

export default function LoginPage() {
  const router = useRouter();
  const { login, previewMode } = useAuth();
  const { pushToast } = useToast();
  const [email, setEmail] = useState("admin@zxmail.site");
  const [password, setPassword] = useState("secret123");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");

  async function handleSubmit() {
    setSubmitting(true);
    setError("");
    try {
      const session = await login(email, password);
      pushToast({
        title: "Signed in",
        description: "Your browser session is now attached to the HttpOnly auth cookie.",
        tone: "success",
      });
      router.push(session.user.role === "admin" ? "/admin" : "/dashboard");
    } catch (nextError) {
      setError(nextError instanceof Error ? nextError.message : "Login failed");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <main className="mesh surface-grid flex min-h-screen items-center justify-center px-6 py-10">
      <section className="w-full max-w-6xl overflow-hidden rounded-[32px] border border-[rgba(199,211,227,0.7)] bg-[rgba(255,255,255,0.86)] shadow-[var(--shadow-lg)] backdrop-blur">
        <div className="grid lg:grid-cols-[1.05fr_0.95fr]">
          <div className="bg-[linear-gradient(155deg,#0b1220_0%,#11203f_60%,#15346b_100%)] px-8 py-10 text-white md:px-10 md:py-12">
            <div className="flex items-center gap-3">
              <div className="flex h-12 w-12 items-center justify-center rounded-2xl bg-white/10 text-lg font-semibold">
                zx
              </div>
              <div>
                <p className="eyebrow !text-[#9db4d8]">zxMail</p>
                <p className="text-lg font-semibold">Production v1</p>
              </div>
            </div>
            <h1 className="mt-8 max-w-xl text-4xl font-semibold tracking-[-0.05em] md:text-[3.35rem]">
              Transactional email operations with a calmer, cleaner control plane.
            </h1>
            <p className="mt-4 max-w-xl text-sm leading-7 text-[#d6e1f2] md:text-[15px]">
              Domain onboarding, DNS verification, SMTP credentials, logs, suppressions, and admin controls for teams shipping email on top of Postal.
            </p>

            <div className="mt-8 grid gap-3">
              <div className="rounded-[24px] border border-white/10 bg-white/6 p-4">
                <p className="text-sm font-semibold">Guided domain onboarding</p>
                <p className="mt-2 text-sm leading-7 text-[#adc0db]">
                  Add a sending domain, copy exact DNS records, verify propagation, then issue SMTP access.
                </p>
              </div>
              <div className="rounded-[24px] border border-white/10 bg-white/6 p-4">
                <p className="text-sm font-semibold">Show-once SMTP secret</p>
                <p className="mt-2 text-sm leading-7 text-[#adc0db]">
                  Credentials reveal the SMTP password only once and avoid storing it in browser storage.
                </p>
              </div>
              <div className="rounded-[24px] border border-white/10 bg-white/6 p-4">
                <p className="text-sm font-semibold">Cloudflare warning built in</p>
                <p className="mt-2 text-sm leading-7 text-[#adc0db]">
                  SMTP and related mail records must stay DNS only. Proxy mode is not supported for mail traffic.
                </p>
              </div>
            </div>
          </div>

          <div className="px-8 py-10 md:px-10 md:py-12">
            <div className="mx-auto max-w-md">
              <div className="flex flex-wrap items-center gap-2">
                <span className="eyebrow">Browser authentication</span>
                {previewMode ? (
                  <span className="rounded-full bg-[rgba(23,105,255,0.08)] px-3 py-1 text-[11px] font-semibold uppercase tracking-[0.16em] text-[var(--primary)]">
                    Preview mode
                  </span>
                ) : null}
              </div>
              <h2 className="mt-4 text-3xl font-semibold tracking-[-0.05em] text-[var(--foreground)]">
                Sign in to your zxMail workspace
              </h2>
              <p className="mt-3 text-sm leading-7 text-[var(--muted)]">
                Live environments use an HttpOnly cookie from <span className="font-mono text-xs">/api/v1/auth/login</span>. Preview mode only exists to inspect the frontend without a backend session.
              </p>

              <div className="mt-8 grid gap-4">
                <label className="grid gap-2">
                  <span className="text-sm font-semibold text-[var(--foreground)]">Email</span>
                  <input
                    className="field"
                    type="email"
                    placeholder="you@company.com"
                    value={email}
                    onChange={(event) => setEmail(event.target.value)}
                  />
                </label>
                <label className="grid gap-2">
                  <span className="text-sm font-semibold text-[var(--foreground)]">Password</span>
                  <input
                    className="field"
                    type="password"
                    placeholder="Your password"
                    value={password}
                    onChange={(event) => setPassword(event.target.value)}
                  />
                </label>
              </div>

              {error ? (
                <div className="mt-4 rounded-[22px] border border-[rgba(200,58,84,0.16)] bg-[rgba(200,58,84,0.07)] px-4 py-3 text-sm text-[var(--foreground)]">
                  {error}
                </div>
              ) : null}

              <div className="mt-6 flex flex-wrap gap-3">
                <Button disabled={submitting} onClick={handleSubmit}>
                  {submitting ? "Signing in..." : "Sign in"}
                </Button>
                <Link
                  href="/dashboard"
                  className="inline-flex items-center justify-center rounded-[16px] border border-[var(--border)] bg-white px-4 py-2.5 text-sm font-semibold transition hover:bg-[#f8fbff]"
                >
                  Explore dashboard
                </Link>
              </div>

              <div className="mt-6 rounded-[22px] border border-[var(--border)] bg-[#fbfdff] p-4 text-sm leading-7 text-[var(--muted)]">
                This login screen is intentionally scoped to Production v1. Billing, IP pools, warm-up automation, and other Phase 2 screens are not exposed here.
              </div>
            </div>
          </div>
        </div>
      </section>
    </main>
  );
}
