"use client";

import { useEffect, useState } from "react";
import { PageHero } from "@/components/shared/page-hero";
import { SectionCard } from "@/components/shared/section-card";
import { StatusBadge } from "@/components/shared/status-badge";
import { useAuth } from "@/components/providers/auth-provider";
import { DataTable } from "@/components/ui/data-table";
import { EmptyState } from "@/components/ui/empty-state";
import { ErrorState } from "@/components/ui/error-state";
import { formatDateTime } from "@/lib/utils";
import type { SuppressionRecord } from "@/types/zxmail";

export function SuppressionsClient() {
  const { api } = useAuth();
  const [suppressions, setSuppressions] = useState<SuppressionRecord[]>([]);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let mounted = true;

    async function loadSuppressions() {
      try {
        const nextSuppressions = await api.listSuppressions();
        if (mounted) {
          setSuppressions(nextSuppressions);
        }
      } catch (nextError) {
        if (mounted) {
          setError(
            nextError instanceof Error ? nextError.message : "Failed to load suppressions",
          );
        }
      } finally {
        if (mounted) {
          setLoading(false);
        }
      }
    }

    loadSuppressions();
    return () => {
      mounted = false;
    };
  }, [api]);

  return (
    <div className="space-y-6">
      <PageHero
        eyebrow="Suppressions"
        title="Protect future sends from risky or blocked recipients"
        description="Postal bounce processing can create or update suppressions automatically. This page keeps that recipient safety list clean and searchable."
      />

      {error ? <ErrorState description={error} /> : null}

      <SectionCard
        title="Suppression list"
        description="Production v1 keeps this list intentionally narrow: bounce-sourced and manual suppressions only."
      >
        <DataTable
          rows={suppressions}
          getRowKey={(row) => row.id}
          emptyState={
            !loading ? (
              <EmptyState
                title="No suppressions"
                description="Hard bounces and manual recipient blocks will appear here when they exist."
              />
            ) : null
          }
          columns={[
            {
              key: "recipient",
              header: "Recipient",
              render: (item) => <span className="font-medium">{item.recipient}</span>,
            },
            {
              key: "source",
              header: "Source",
              render: (item) => (
                <span className="text-xs uppercase tracking-[0.16em] text-[var(--muted)]">
                  {item.source}
                </span>
              ),
            },
            {
              key: "reason",
              header: "Reason",
              render: (item) => <span className="text-[var(--muted)]">{item.reason || "No reason"}</span>,
            },
            {
              key: "status",
              header: "Status",
              render: (item) => <StatusBadge value={item.active ? "active" : "released"} />,
            },
            {
              key: "created",
              header: "Created",
              render: (item) => <span className="text-[var(--muted)]">{formatDateTime(item.created_at)}</span>,
            },
          ]}
        />
      </SectionCard>
    </div>
  );
}
