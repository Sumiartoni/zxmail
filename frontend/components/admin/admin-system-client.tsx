"use client";

import { useEffect, useState } from "react";
import { HealthStatusCard } from "@/components/ui/health-status-card";
import { useAuth } from "@/components/providers/auth-provider";
import { ErrorState } from "@/components/ui/error-state";
import { LoadingSkeleton } from "@/components/ui/loading-skeleton";
import { PageHeader } from "@/components/ui/page-header";
import { SectionCard } from "@/components/shared/section-card";
import type { AdminSystemHealth, QueueHealth } from "@/types/zxmail";

export function AdminSystemClient() {
  const { api } = useAuth();
  const [health, setHealth] = useState<AdminSystemHealth | null>(null);
  const [queues, setQueues] = useState<QueueHealth | null>(null);
  const [postal, setPostal] = useState<{ reachable: boolean; note: string } | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let mounted = true;
    Promise.all([api.getAdminSystemHealth(), api.getQueueHealth(), api.getPostalHealth()])
      .then(([nextHealth, nextQueues, nextPostal]) => {
        if (mounted) {
          setHealth(nextHealth);
          setQueues(nextQueues);
          setPostal(nextPostal);
          setError(null);
        }
      })
      .catch((nextError) => {
        if (mounted) {
          setError(nextError instanceof Error ? nextError.message : "Failed to load system health");
        }
      })
      .finally(() => {
        if (mounted) {
          setLoading(false);
        }
      });
    return () => {
      mounted = false;
    };
  }, [api]);

  return (
    <div className="space-y-6">
      <PageHeader
        eyebrow="System"
        title="Operational readiness"
        description="Public health stays minimal, while admin system health exposes dependency notes, Postal reachability, and worker activity without leaking sensitive configuration."
      />

      {loading ? (
        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-5">
          {Array.from({ length: 5 }).map((_, index) => (
            <LoadingSkeleton key={index} className="h-44" />
          ))}
        </div>
      ) : error || !health || !queues || !postal ? (
        <ErrorState description={error || "System health could not be loaded."} retryLabel="Retry" onRetry={() => window.location.reload()} />
      ) : (
        <>
          <section className="grid gap-4 md:grid-cols-2 xl:grid-cols-5">
            <HealthStatusCard label="PostgreSQL" value={health.postgres as never} note={health.notes.postgres || "Primary persistence for identities, logs, and billing."} />
            <HealthStatusCard label="Redis" value={health.redis as never} note={health.notes.redis || "Used for throttling, rate advisory counters, and worker support."} />
            <HealthStatusCard label="Postal API" value={health.postal as never} note={postal.note || health.notes.postal || "Postal reachability and webhook contract."} />
            <HealthStatusCard label="Worker" value={health.worker as never} note={health.notes.worker || "Scheduled jobs drive retention, snapshots, and resets."} />
            <HealthStatusCard label="Queue posture" value={health.queue as never} note={queues.note} />
          </section>

          <SectionCard title="Queue detail" description="Production Ready v2 still uses scheduled workers before a dedicated SMTP gateway queue is introduced.">
            <div className="grid gap-4 md:grid-cols-3">
              <div className="rounded-[20px] bg-[#f7fbff] px-4 py-4">
                <p className="eyebrow">Mode</p>
                <p className="mt-2 text-lg font-semibold">{queues.mode}</p>
              </div>
              <div className="rounded-[20px] bg-[#f7fbff] px-4 py-4">
                <p className="eyebrow">Pending</p>
                <p className="mt-2 text-lg font-semibold">{queues.pending}</p>
              </div>
              <div className="rounded-[20px] bg-[#f7fbff] px-4 py-4">
                <p className="eyebrow">In progress</p>
                <p className="mt-2 text-lg font-semibold">{queues.in_progress}</p>
              </div>
            </div>
          </SectionCard>
        </>
      )}
    </div>
  );
}
