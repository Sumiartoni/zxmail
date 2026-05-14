import { Button } from "@/components/shared/button";

export function ErrorState({
  title = "Something went wrong",
  description,
  retryLabel,
  onRetry,
}: {
  title?: string;
  description: string;
  retryLabel?: string;
  onRetry?: () => void;
}) {
  return (
    <div className="rounded-[24px] border border-[rgba(200,58,84,0.18)] bg-[rgba(200,58,84,0.06)] px-5 py-5">
      <h3 className="text-base font-semibold text-[var(--foreground)]">{title}</h3>
      <p className="mt-2 text-sm leading-7 text-[var(--muted)]">{description}</p>
      {onRetry && retryLabel ? (
        <div className="mt-4">
          <Button variant="danger" onClick={onRetry}>
            {retryLabel}
          </Button>
        </div>
      ) : null}
    </div>
  );
}
