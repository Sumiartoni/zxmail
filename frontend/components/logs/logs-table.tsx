"use client";

import { useEffect, useState } from "react";
import { Button } from "@/components/shared/button";
import { PageHero } from "@/components/shared/page-hero";
import { StatusBadge } from "@/components/shared/status-badge";
import { useAuth } from "@/components/providers/auth-provider";
import { CodeBlock } from "@/components/ui/code-block";
import { DataTable } from "@/components/ui/data-table";
import { Drawer } from "@/components/ui/drawer";
import { EmptyState } from "@/components/ui/empty-state";
import { ErrorState } from "@/components/ui/error-state";
import { MessageTimeline } from "@/components/ui/message-timeline";
import { SectionCard } from "@/components/shared/section-card";
import { formatDateTime } from "@/lib/utils";
import type { CredentialResponse, DomainRecord, LogsFilterState, SendLog } from "@/types/zxmail";

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

    void loadInitial();
    return () => {
      mounted = false;
    };
  }, [api]);

  async function applyFilters(nextFilters = filters) {
    setLoading(true);
    setError("");
    try {
      const result = await api.listLogs(nextFilters);
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
      <PageHero
        eyebrow="Logs"
        title={title}
        description={description}
      />

      <SectionCard
        title="Filters"
        description="Search accepted, delivered, bounced, deferred, and rejected messages by domain, credential, message ID, recipient, and date range."
      >
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
              <option key={credential.credential.id} value={credential.credential.id}>
                {credential.credential.label || credential.credential.username}
              </option>
            ))}
          </select>
          <input
            className="field"
            placeholder="Message ID"
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
            <option value="">All statuses</option>
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
          <Button onClick={() => void applyFilters()}>Apply filters</Button>
          <Button
            variant="ghost"
            onClick={() => {
              setFilters(initialFilters);
              void applyFilters(initialFilters);
            }}
          >
            Reset
          </Button>
          <span className="text-sm text-[var(--muted)]">{total} matching events</span>
        </div>
      </SectionCard>

      {error ? <ErrorState description={error} /> : null}

      <SectionCard
        title="Message events"
        description="Open any row to inspect the message timeline, addresses, reason, and sanitized raw event payload."
      >
        <DataTable
          rows={logs}
          getRowKey={(row) => row.id}
          onRowClick={setSelectedLog}
          emptyState={
            !loading ? (
              <EmptyState
                title="No matching events"
                description="Try a broader date range or clear some filters to find message activity."
              />
            ) : null
          }
          columns={[
            {
              key: "message",
              header: "Message",
              render: (log) => (
                <div>
                  <p className="font-semibold">{log.subject || "No subject"}</p>
                  <p className="mt-1 font-mono text-xs text-[var(--muted)]">
                    {log.message_id_header || log.postal_message_id || "No message id"}
                  </p>
                </div>
              ),
            },
            {
              key: "recipient",
              header: "Recipient",
              render: (log) => (
                <div>
                  <p>{log.to_addr}</p>
                  <p className="mt-1 text-xs text-[var(--muted)]">{log.domain_name || "Unknown domain"}</p>
                </div>
              ),
            },
            {
              key: "status",
              header: "Status",
              className: "w-[140px]",
              render: (log) => <StatusBadge value={log.status} />,
            },
            {
              key: "time",
              header: "Time",
              className: "w-[180px]",
              render: (log) => <span className="text-[var(--muted)]">{formatDateTime(log.created_at)}</span>,
            },
          ]}
        />
      </SectionCard>

      <Drawer
        open={Boolean(selectedLog)}
        title={selectedLog?.subject || "Message detail"}
        description="This drawer shows frontend-safe message metadata plus the current raw event payload returned by the API."
        onClose={() => setSelectedLog(null)}
      >
        {selectedLog ? (
          <div className="space-y-5">
            <div className="flex flex-wrap items-center gap-3">
              <StatusBadge value={selectedLog.status} />
              <span className="text-sm text-[var(--muted)]">{formatDateTime(selectedLog.created_at)}</span>
            </div>

            <MessageTimeline currentStatus={selectedLog.status} />

            <div className="rounded-[22px] border border-[var(--border)] bg-[#fbfdff] p-4 text-sm leading-7 text-[var(--muted)]">
              <p><span className="font-semibold text-[var(--foreground)]">From:</span> {selectedLog.from_addr}</p>
              <p><span className="font-semibold text-[var(--foreground)]">To:</span> {selectedLog.to_addr}</p>
              <p><span className="font-semibold text-[var(--foreground)]">Domain:</span> {selectedLog.domain_name || "Unknown"}</p>
              <p><span className="font-semibold text-[var(--foreground)]">Credential:</span> {selectedLog.credential_name || "Unknown"}</p>
              <p><span className="font-semibold text-[var(--foreground)]">Message ID:</span> {selectedLog.message_id_header || selectedLog.postal_message_id || "Unknown"}</p>
              {selectedLog.reason ? (
                <p><span className="font-semibold text-[var(--foreground)]">Reason:</span> {selectedLog.reason}</p>
              ) : null}
            </div>

            <CodeBlock label="Raw event">
              {JSON.stringify(selectedLog.raw_event, null, 2)}
            </CodeBlock>
          </div>
        ) : null}
      </Drawer>
    </div>
  );
}
