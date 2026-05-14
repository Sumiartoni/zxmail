"use client";

import { useState } from "react";
import { Button } from "@/components/shared/button";
import { SectionCard } from "@/components/shared/section-card";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";

export function AdminActionPanel({
  suspended,
  onSuspend,
  onUnsuspend,
  onDisableCredentials,
}: {
  suspended: boolean;
  onSuspend: () => Promise<void>;
  onUnsuspend: () => Promise<void>;
  onDisableCredentials: () => Promise<void>;
}) {
  const [dialog, setDialog] = useState<null | "suspend" | "unsuspend" | "disable">(null);

  async function handleConfirm() {
    if (dialog === "suspend") {
      await onSuspend();
    }
    if (dialog === "unsuspend") {
      await onUnsuspend();
    }
    if (dialog === "disable") {
      await onDisableCredentials();
    }
    setDialog(null);
  }

  return (
    <SectionCard title="Admin actions" description="Destructive actions are confirmation-gated and audit logged.">
      <div className="flex flex-wrap gap-3">
        {suspended ? (
          <Button variant="secondary" onClick={() => setDialog("unsuspend")}>
            Unsuspend organization
          </Button>
        ) : (
          <Button variant="danger" onClick={() => setDialog("suspend")}>
            Suspend organization
          </Button>
        )}
        <Button variant="danger" onClick={() => setDialog("disable")}>
          Disable all credentials
        </Button>
      </div>
      <ConfirmDialog
        open={dialog !== null}
        title={
          dialog === "disable"
            ? "Disable all credentials?"
            : dialog === "unsuspend"
              ? "Unsuspend this organization?"
              : "Suspend this organization?"
        }
        description="This action is intended for payment, abuse, or operational incidents and will be recorded in audit logs."
        confirmLabel="Confirm"
        onClose={() => setDialog(null)}
        onConfirm={() => {
          void handleConfirm();
        }}
      />
    </SectionCard>
  );
}
