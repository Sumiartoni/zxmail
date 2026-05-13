import Link from "next/link";
import { SectionCard } from "@/components/shared/section-card";

export default function Home() {
  return (
    <main className="mesh min-h-screen px-6 py-10 md:px-10">
      <div className="mx-auto flex max-w-6xl flex-col gap-8">
        <section className="panel overflow-hidden p-8 md:p-12">
          <div className="grid gap-8 lg:grid-cols-[1.25fr_0.75fr]">
            <div className="space-y-6">
              <p className="eyebrow">zxMail Production v1</p>
              <h1 className="max-w-3xl text-4xl font-semibold tracking-[-0.05em] md:text-6xl">
                Clean SaaS dashboard for self-hosted transactional email, built around Postal.
              </h1>
              <p className="max-w-2xl text-lg leading-8 text-[var(--muted)]">
                The frontend is scoped only to Production v1: auth, organizations,
                domain onboarding, DNS verification, credentials, logs, suppressions,
                quota visibility, and admin/customer operations.
              </p>
              <div className="flex flex-wrap gap-3">
                <Link
                  href="/login"
                  className="rounded-full bg-[var(--accent)] px-5 py-3 text-sm font-semibold text-white transition hover:bg-[#9f411e]"
                >
                  Sign in
                </Link>
                <Link
                  href="/dashboard"
                  className="rounded-full border border-[var(--line)] bg-white/70 px-5 py-3 text-sm font-semibold transition hover:bg-white"
                >
                  Customer dashboard
                </Link>
                <Link
                  href="/admin"
                  className="rounded-full border border-[var(--line)] bg-white/70 px-5 py-3 text-sm font-semibold transition hover:bg-white"
                >
                  Admin dashboard
                </Link>
              </div>
            </div>

            <SectionCard
              title="Scope boundary"
              description="This frontend intentionally follows the PRD Phase 1 boundary and avoids Phase 2 features."
            >
              <ul className="space-y-3 text-sm leading-7 text-[var(--muted)]">
                <li>No billing or Stripe UI.</li>
                <li>No Kubernetes, multi-node, or IP pool interfaces.</li>
                <li>No seed testing, DMARC aggregate parser, SSO, or Vault.</li>
                <li>No advanced deliverability toolkit beyond logs, bounces, and suppressions.</li>
              </ul>
            </SectionCard>
          </div>
        </section>
      </div>
    </main>
  );
}
