"use client";

import { useEffect, useState } from "react";
import { MetricCard } from "@/components/dashboard/metric-card";
import { PageHero } from "@/components/shared/page-hero";
import { SectionCard } from "@/components/shared/section-card";
import { StatusBadge } from "@/components/shared/status-badge";
import { useAuth } from "@/components/providers/auth-provider";
import type { SystemHealth } from "@/types/zxmail";

export function AdminSystemClient() {
  const { api } = useAuth();
  const [health, setHealth] = useState<SystemHealth | null>(null);

  useEffect(() => {
    let mounted = true;

    async function loadHealth() {
      try {
        const nextHealth = await api.health();
        if (mounted) {
          setHealth(nextHealth);
        }
      } catch {}
    }

    loadHealth();
    return () => {
      mounted = false;
    };
  }, [api]);

  return (
    <div className="space-y-6">
      <PageHero
        eyebrow="Admin system"
        title="System posture, health checks, and Production v1 guardrails"
        description="Operationally useful without drifting into Phase 2: API and datastore health, Postal integration caveats, Cloudflare SMTP warning, and quota-enforcement constraints."
      />

      <section className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <MetricCard
          label="API"
          value={health?.api === "healthy" ? "Healthy" : "Pending"}
          note="Derived from health endpoints or preview system state."
        />
        <MetricCard
          label="PostgreSQL"
          value={health?.postgres === "healthy" ? "Healthy" : "Pending"}
          note="Backs organizations, domains, credentials, logs, suppressions, and audit logs."
        />
        <MetricCard
          label="Redis"
          value={health?.redis === "healthy" ? "Healthy" : "Pending"}
          note="Used for minute-window rate state and queue-oriented extensions."
        />
        <MetricCard
          label="Postal wiring"
          value={health?.postal === "manual-check" ? "Manual" : "Ready"}
          note="Credential and server creation remain explicit placeholders until contracts are confirmed."
        />
      </section>

      <div className="grid gap-4 xl:grid-cols-[1fr_1fr]">
        <SectionCard
          title="Current service state"
          description="These badges reflect the health summary surfaced by the API layer or preview mode adapter."
        >
          <div className="space-y-4">
            {health ? (
              <>
                <div className="flex items-center justify-between rounded-2xl border border-[var(--line)] bg-white/70 p-4">
                  <span className="font-semibold">API readiness</span>
                  <StatusBadge value={health.api === "healthy" ? "verified" : "pending"} />
                </div>
                <div className="flex items-center justify-between rounded-2xl border border-[var(--line)] bg-white/70 p-4">
                  <span className="font-semibold">Redis minute windows</span>
                  <StatusBadge value={health.redis === "healthy" ? "verified" : "pending"} />
                </div>
                <div className="flex items-center justify-between rounded-2xl border border-[var(--line)] bg-white/70 p-4">
                  <span className="font-semibold">PostgreSQL counters</span>
                  <StatusBadge
                    value={health.postgres === "healthy" ? "verified" : "pending"}
                  />
                </div>
              </>
            ) : null}
          </div>
        </SectionCard>

        <SectionCard
          title="Operator notes"
          description="Scope and implementation boundaries from the PRD that the UI should reinforce."
        >
          <ul className="space-y-3 text-sm leading-7 text-[var(--muted)]">
            {health?.notes.map((note) => (
              <li key={note}>{note}</li>
            ))}
            <li>No billing UI, subscriptions, IP pools, seed testing, or multi-node controls are included here.</li>
            <li>SMTP passwords are revealed once and cleared from in-memory modal state when closed.</li>
          </ul>
        </SectionCard>
      </div>
    </div>
  );
}
