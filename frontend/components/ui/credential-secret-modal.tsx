import { CopyButton } from "@/components/shared/copy-button";
import { Modal } from "@/components/shared/modal";
import { CodeBlock } from "@/components/ui/code-block";
import type { CredentialSecretResponse } from "@/types/zxmail";

export function CredentialSecretModal({
  credential,
  open,
  onClose,
}: {
  credential: CredentialSecretResponse | null;
  open: boolean;
  onClose: () => void;
}) {
  return (
    <Modal
      open={open}
      title="SMTP secret shown once"
      description="Copy this password immediately. After this modal closes, zxMail will not return the same secret again."
      onClose={onClose}
      size="lg"
    >
      {credential ? (
        <div className="space-y-5">
          <div className="grid gap-4 md:grid-cols-2">
            <div className="rounded-[22px] border border-[var(--border)] bg-[#fbfdff] p-4">
              <div className="flex items-center justify-between gap-3">
                <p className="eyebrow">Username</p>
                <CopyButton value={credential.smtp.username} />
              </div>
              <CodeBlock className="mt-3">{credential.smtp.username}</CodeBlock>
            </div>
            <div className="rounded-[22px] border border-[var(--border)] bg-[#fbfdff] p-4">
              <div className="flex items-center justify-between gap-3">
                <p className="eyebrow">Password</p>
                <CopyButton value={credential.secret} />
              </div>
              <CodeBlock className="mt-3">{credential.secret}</CodeBlock>
            </div>
          </div>
          <div className="rounded-[22px] border border-[rgba(184,92,0,0.16)] bg-[rgba(184,92,0,0.06)] p-4 text-sm leading-7 text-[var(--foreground)]">
            Store this secret in your application or secret manager now. Do not expect it to appear again in logs, browser storage, or later credential views.
          </div>
        </div>
      ) : null}
    </Modal>
  );
}
