"use client";

import { useEffect, useState } from "react";
import { PageHero } from "@/components/shared/page-hero";
import { SectionCard } from "@/components/shared/section-card";
import { StatusBadge } from "@/components/shared/status-badge";
import { useAuth } from "@/components/providers/auth-provider";
import { formatDateTime } from "@/lib/utils";
import type { SuppressionRecord } from "@/types/zxmail";

export function SuppressionsClient() {
  const { api } = useAuth();
  const [suppressions, setSuppressions] = useState<SuppressionRecord[]>([]);
  const [error, setError] = useState("");

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
        title="Protect future sends from hard-bounced or manually blocked recipients"
        description="Postal bounce processing can create or update suppression entries. This page gives customers and operators a clean view of active recipient blocks."
      />

      {error ? (
        <div className="rounded-3xl border border-[#d8ad9f] bg-[#fff1ec] px-4 py-3 text-sm text-[#8d2d11]">
          {error}
        </div>
      ) : null}

      <SectionCard
        title="Suppression list"
        description="Phase 1 keeps this list intentionally narrow: bounce-sourced and manual suppressions only."
      >
        <div className="overflow-x-auto">
          <table className="min-w-full text-left text-sm">
            <thead className="text-[var(--muted)]">
              <tr>
                <th className="pb-3 font-medium">Recipient</th>
                <th className="pb-3 font-medium">Source</th>
                <th className="pb-3 font-medium">Reason</th>
                <th className="pb-3 font-medium">Status</th>
                <th className="pb-3 font-medium">Created</th>
              </tr>
            </thead>
            <tbody>
              {suppressions.map((item) => (
                <tr key={item.id} className="border-t border-[var(--line)]">
                  <td className="py-4 pr-4 font-medium">{item.recipient}</td>
                  <td className="py-4 pr-4 uppercase tracking-[0.14em] text-[var(--muted)]">
                    {item.source}
                  </td>
                  <td className="py-4 pr-4 text-[var(--muted)]">{item.reason || "No reason"}</td>
                  <td className="py-4 pr-4">
                    <StatusBadge value={item.active ? "active" : "released"} />
                  </td>
                  <td className="py-4 text-[var(--muted)]">
                    {formatDateTime(item.created_at)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </SectionCard>
    </div>
  );
}
