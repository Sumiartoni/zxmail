"use client";

import { useEffect, useState } from "react";
import { Button } from "@/components/shared/button";
import { SectionCard } from "@/components/shared/section-card";
import { StatusBadge } from "@/components/shared/status-badge";
import { formatDateTime } from "@/lib/utils";
import type { CredentialResponse, DomainRecord, LogsFilterState, SendLog } from "@/types/zxmail";
import { useAuth } from "@/components/providers/auth-provider";

const initialFilters: LogsFilterState = {
  domain_id: "",
  credential_id: "",
  message_id: "",
  recipient: "",
  status: "",
  from: "",
  to: "",
  date_from: "",
  date_to: "",
  limit: 25,
  offset: 0,
};

export function LogsTable({
  title,
  description,
}: {
  title: string;
  description: string;
}) {
  const { api } = useAuth();
  const [domains, setDomains] = useState<DomainRecord[]>([]);
  const [credentials, setCredentials] = useState<CredentialResponse[]>([]);
  const [filters, setFilters] = useState<LogsFilterState>(initialFilters);
  const [logs, setLogs] = useState<SendLog[]>([]);
  const [selectedLog, setSelectedLog] = useState<SendLog | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [total, setTotal] = useState(0);

  useEffect(() => {
    let mounted = true;

    async function loadInitial() {
      try {
        const [nextDomains, nextCredentials, result] = await Promise.all([
          api.listDomains(),
          api.listCredentials(),
          api.listLogs(initialFilters),
        ]);
        if (!mounted) {
          return;
        }
        setDomains(nextDomains);
        setCredentials(nextCredentials);
        setLogs(result.logs);
        setSelectedLog(result.logs[0] ?? null);
        setTotal(result.total);
      } catch (nextError) {
        if (mounted) {
          setError(nextError instanceof Error ? nextError.message : "Failed to load logs");
        }
      } finally {
        if (mounted) {
          setLoading(false);
        }
      }
    }

    loadInitial();
    return () => {
      mounted = false;
    };
  }, [api]);

  async function applyFilters() {
    setLoading(true);
    setError("");
    try {
      const result = await api.listLogs(filters);
      setLogs(result.logs);
      setSelectedLog(result.logs[0] ?? null);
      setTotal(result.total);
    } catch (nextError) {
      setError(nextError instanceof Error ? nextError.message : "Failed to load logs");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="space-y-6">
      <SectionCard title={title} description={description}>
        <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
          <select
            className="field"
            value={filters.domain_id}
            onChange={(event) =>
              setFilters((current) => ({ ...current, domain_id: event.target.value }))
            }
          >
            <option value="">All domains</option>
            {domains.map((domain) => (
              <option key={domain.id} value={domain.id}>
                {domain.name}
              </option>
            ))}
          </select>
          <select
            className="field"
            value={filters.credential_id}
            onChange={(event) =>
              setFilters((current) => ({
                ...current,
                credential_id: event.target.value,
              }))
            }
          >
            <option value="">All credentials</option>
            {credentials.map((credential) => (
              <option
                key={credential.credential.id}
                value={credential.credential.id}
              >
                {credential.credential.label || credential.credential.username}
              </option>
            ))}
          </select>
          <input
            className="field"
            placeholder="Message ID or Postal ID"
            value={filters.message_id}
            onChange={(event) =>
              setFilters((current) => ({ ...current, message_id: event.target.value }))
            }
          />
          <input
            className="field"
            placeholder="Recipient"
            value={filters.recipient}
            onChange={(event) =>
              setFilters((current) => ({ ...current, recipient: event.target.value }))
            }
          />
          <select
            className="field"
            value={filters.status}
            onChange={(event) =>
              setFilters((current) => ({ ...current, status: event.target.value }))
            }
          >
            <option value="">Any status</option>
            <option value="accepted">Accepted</option>
            <option value="delivered">Delivered</option>
            <option value="bounced">Bounced</option>
            <option value="deferred">Deferred</option>
            <option value="rejected">Rejected</option>
          </select>
          <input
            className="field"
            placeholder="From address"
            value={filters.from}
            onChange={(event) =>
              setFilters((current) => ({ ...current, from: event.target.value }))
            }
          />
          <input
            className="field"
            type="date"
            value={filters.date_from}
            onChange={(event) =>
              setFilters((current) => ({ ...current, date_from: event.target.value }))
            }
          />
          <input
            className="field"
            type="date"
            value={filters.date_to}
            onChange={(event) =>
              setFilters((current) => ({ ...current, date_to: event.target.value }))
            }
          />
        </div>

        <div className="mt-4 flex flex-wrap items-center gap-3">
          <Button onClick={applyFilters}>Apply filters</Button>
          <Button
            variant="ghost"
            onClick={() => {
              setFilters(initialFilters);
              setSelectedLog(null);
            }}
          >
            Reset filters
          </Button>
          <span className="text-sm text-[var(--muted)]">{total} matching events</span>
        </div>
      </SectionCard>

      {error ? (
        <div className="rounded-3xl border border-[#d8ad9f] bg-[#fff1ec] px-4 py-3 text-sm text-[#8d2d11]">
          {error}
        </div>
      ) : null}

      <div className="grid gap-4 xl:grid-cols-[1.2fr_0.8fr]">
        <SectionCard
          title="Event stream"
          description="Search by domain, credential, recipient, status, or message identifier."
        >
          <div className="overflow-x-auto">
            <table className="min-w-full text-left text-sm">
              <thead className="text-[var(--muted)]">
                <tr>
                  <th className="pb-3 font-medium">Message</th>
                  <th className="pb-3 font-medium">Recipient</th>
                  <th className="pb-3 font-medium">Status</th>
                  <th className="pb-3 font-medium">Time</th>
                </tr>
              </thead>
              <tbody>
                {logs.map((log) => (
                  <tr
                    key={log.id}
                    className="cursor-pointer border-t border-[var(--line)]"
                    onClick={() => setSelectedLog(log)}
                  >
                    <td className="py-4 pr-4">
                      <div className="font-semibold text-[var(--ink)]">
                        {log.subject || "No subject"}
                      </div>
                      <div className="mt-1 font-mono text-xs text-[var(--muted)]">
                        {log.message_id_header || log.postal_message_id}
                      </div>
                    </td>
                    <td className="py-4 pr-4">
                      <div>{log.to_addr}</div>
                      <div className="mt-1 text-xs text-[var(--muted)]">
                        {log.domain_name || "Unknown domain"}
                      </div>
                    </td>
                    <td className="py-4 pr-4">
                      <StatusBadge value={log.status} />
                    </td>
                    <td className="py-4">{formatDateTime(log.created_at)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
            {!loading && logs.length === 0 ? (
              <div className="rounded-2xl border border-dashed border-[var(--line)] px-4 py-8 text-center text-sm text-[var(--muted)]">
                No events matched the current filter set.
              </div>
            ) : null}
          </div>
        </SectionCard>

        <SectionCard
          title="Event detail"
          description="Inspect the selected Postal event with reason and raw payload."
        >
          {selectedLog ? (
            <div className="space-y-4">
              <div className="flex flex-wrap items-center gap-3">
                <StatusBadge value={selectedLog.status} />
                <span className="text-sm text-[var(--muted)]">
                  {formatDateTime(selectedLog.created_at)}
                </span>
              </div>
              <div className="space-y-2 text-sm leading-7 text-[var(--muted)]">
                <p>
                  <span className="font-semibold text-[var(--ink)]">From:</span>{" "}
                  {selectedLog.from_addr}
                </p>
                <p>
                  <span className="font-semibold text-[var(--ink)]">To:</span>{" "}
                  {selectedLog.to_addr}
                </p>
                <p>
                  <span className="font-semibold text-[var(--ink)]">Domain:</span>{" "}
                  {selectedLog.domain_name || "Unknown"}
                </p>
                <p>
                  <span className="font-semibold text-[var(--ink)]">Credential:</span>{" "}
                  {selectedLog.credential_name || "Unknown"}
                </p>
                {selectedLog.reason ? (
                  <p>
                    <span className="font-semibold text-[var(--ink)]">Reason:</span>{" "}
                    {selectedLog.reason}
                  </p>
                ) : null}
              </div>

              <div className="rounded-3xl border border-[var(--line)] bg-[#f8f3ea] p-4">
                <p className="eyebrow">Raw event</p>
                <pre className="mt-3 overflow-x-auto text-xs leading-6 text-[var(--muted)]">
                  {JSON.stringify(selectedLog.raw_event, null, 2)}
                </pre>
              </div>
            </div>
          ) : (
            <div className="rounded-2xl border border-dashed border-[var(--line)] px-4 py-8 text-sm text-[var(--muted)]">
              Select an event from the table to inspect its timeline payload.
            </div>
          )}
        </SectionCard>
      </div>
    </div>
  );
}
