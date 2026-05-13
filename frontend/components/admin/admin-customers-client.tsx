"use client";

import { useEffect, useState } from "react";
import { Button } from "@/components/shared/button";
import { Modal } from "@/components/shared/modal";
import { PageHero } from "@/components/shared/page-hero";
import { SectionCard } from "@/components/shared/section-card";
import { useAuth } from "@/components/providers/auth-provider";
import { formatDateTime } from "@/lib/utils";
import type { OrganizationRecord } from "@/types/zxmail";

type CreateCustomerForm = {
  name: string;
  owner_email: string;
  owner_password: string;
};

const initialForm: CreateCustomerForm = {
  name: "",
  owner_email: "",
  owner_password: "",
};

export function AdminCustomersClient() {
  const { api } = useAuth();
  const [organizations, setOrganizations] = useState<OrganizationRecord[]>([]);
  const [modalOpen, setModalOpen] = useState(false);
  const [form, setForm] = useState<CreateCustomerForm>(initialForm);
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    let mounted = true;

    async function loadOrganizations() {
      try {
        const nextOrganizations = await api.listOrganizations();
        if (mounted) {
          setOrganizations(nextOrganizations);
        }
      } catch (nextError) {
        if (mounted) {
          setError(
            nextError instanceof Error ? nextError.message : "Failed to load customers",
          );
        }
      }
    }

    loadOrganizations();
    return () => {
      mounted = false;
    };
  }, [api]);

  async function submitOrganization() {
    setSubmitting(true);
    setError("");
    try {
      const nextOrganization = await api.createOrganization(form);
      setOrganizations((current) => [nextOrganization, ...current]);
      setForm(initialForm);
      setModalOpen(false);
    } catch (nextError) {
      setError(
        nextError instanceof Error ? nextError.message : "Failed to create customer organization",
      );
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="space-y-6">
      <PageHero
        eyebrow="Admin customers"
        title="Create and review customer organizations"
        description="Admins can provision customer organizations and hand off the initial owner credentials while keeping all customer data scoped to its owning organization."
        actions={<Button onClick={() => setModalOpen(true)}>Create customer</Button>}
      />

      {error ? (
        <div className="rounded-3xl border border-[#d8ad9f] bg-[#fff1ec] px-4 py-3 text-sm text-[#8d2d11]">
          {error}
        </div>
      ) : null}

      <SectionCard
        title="Customer organizations"
        description="Matches the Production v1 organization model already wired in the backend."
      >
        <div className="overflow-x-auto">
          <table className="min-w-full text-left text-sm">
            <thead className="text-[var(--muted)]">
              <tr>
                <th className="pb-3 font-medium">Organization</th>
                <th className="pb-3 font-medium">Owner email</th>
                <th className="pb-3 font-medium">Created</th>
              </tr>
            </thead>
            <tbody>
              {organizations.map((organization) => (
                <tr key={organization.id} className="border-t border-[var(--line)]">
                  <td className="py-4 pr-4 font-semibold">{organization.name}</td>
                  <td className="py-4 pr-4 text-[var(--muted)]">
                    {organization.owner_email}
                  </td>
                  <td className="py-4 text-[var(--muted)]">
                    {formatDateTime(organization.created_at)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </SectionCard>

      <Modal
        open={modalOpen}
        title="Create customer organization"
        description="This form matches the backend admin endpoint for organization creation."
        onClose={() => setModalOpen(false)}
      >
        <div className="grid gap-4">
          <input
            className="field"
            placeholder="Organization name"
            value={form.name}
            onChange={(event) => setForm((current) => ({ ...current, name: event.target.value }))}
          />
          <input
            className="field"
            type="email"
            placeholder="Owner email"
            value={form.owner_email}
            onChange={(event) =>
              setForm((current) => ({ ...current, owner_email: event.target.value }))
            }
          />
          <input
            className="field"
            type="password"
            placeholder="Temporary owner password"
            value={form.owner_password}
            onChange={(event) =>
              setForm((current) => ({ ...current, owner_password: event.target.value }))
            }
          />
        </div>
        <div className="mt-6 flex flex-wrap gap-3">
          <Button
            disabled={
              submitting ||
              !form.name.trim() ||
              !form.owner_email.trim() ||
              !form.owner_password.trim()
            }
            onClick={submitOrganization}
          >
            {submitting ? "Creating..." : "Create organization"}
          </Button>
          <Button variant="ghost" onClick={() => setModalOpen(false)}>
            Cancel
          </Button>
        </div>
      </Modal>
    </div>
  );
}
