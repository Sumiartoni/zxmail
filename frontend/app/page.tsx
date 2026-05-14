import Link from "next/link";
import { SectionCard } from "@/components/shared/section-card";

export default function Home() {
  return (
    <main className="mesh min-h-screen px-6 py-10 md:px-10">
      <div className="mx-auto flex max-w-7xl flex-col gap-8">
        <section className="overflow-hidden rounded-[32px] border border-[rgba(199,211,227,0.72)] bg-[rgba(255,255,255,0.88)] p-8 shadow-[var(--shadow-lg)] backdrop-blur md:p-12">
          <div className="grid gap-8 lg:grid-cols-[1.15fr_0.85fr]">
            <div className="space-y-6">
              <p className="eyebrow">zxMail Production Ready v2</p>
              <h1 className="max-w-4xl text-4xl font-semibold tracking-[-0.06em] text-[var(--foreground)] md:text-6xl">
                Modern SaaS control plane for self-hosted transactional email, manual billing, and deliverability operations.
              </h1>
              <p className="max-w-3xl text-lg leading-8 text-[var(--muted)]">
                Clean operator UX for domains, DNS verification, SMTP credentials, delivery logs, suppressions, manual payments, usage posture, and customer administration.
              </p>
              <div className="flex flex-wrap gap-3">
                <Link
                  href="/login"
                  className="inline-flex items-center justify-center rounded-[16px] bg-[var(--primary)] px-5 py-3 text-sm font-semibold text-white shadow-[0_14px_32px_rgba(23,105,255,0.24)] transition hover:bg-[#1458d6]"
                >
                  Sign in
                </Link>
                <Link
                  href="/dashboard"
                  className="inline-flex items-center justify-center rounded-[16px] border border-[var(--border)] bg-white px-5 py-3 text-sm font-semibold transition hover:bg-[#f8fbff]"
                >
                  Customer dashboard
                </Link>
                <Link
                  href="/admin"
                  className="inline-flex items-center justify-center rounded-[16px] border border-[var(--border)] bg-white px-5 py-3 text-sm font-semibold transition hover:bg-[#f8fbff]"
                >
                  Admin dashboard
                </Link>
              </div>
            </div>

            <SectionCard
              title="Production Ready v2 boundary"
              description="This frontend extends the existing PRD scope without opening Phase 3 surfaces."
            >
              <ul className="space-y-3 text-sm leading-7 text-[var(--muted)]">
                <li>No Stripe dependency or hardcoded foreign payment gateway.</li>
                <li>No IP pool automation, multi-node orchestration, or Kubernetes.</li>
                <li>No warm-up automation, seed testing, FBL ingestion, or inbox placement claims.</li>
                <li>No SSO enterprise surface or Postal multi-node controller.</li>
              </ul>
            </SectionCard>
          </div>
        </section>
      </div>
    </main>
  );
}
