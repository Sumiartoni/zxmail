export function cn(...values: Array<string | false | null | undefined>) {
  return values.filter(Boolean).join(" ");
}

export function formatDateTime(value?: string | null) {
  if (!value) {
    return "Never";
  }

  return new Intl.DateTimeFormat("en-US", {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(new Date(value));
}

export function formatShortDate(value?: string | null) {
  if (!value) {
    return "Pending";
  }

  return new Intl.DateTimeFormat("en-US", {
    dateStyle: "medium",
  }).format(new Date(value));
}

export function formatNumber(value: number) {
  return new Intl.NumberFormat("en-US").format(value);
}

export function formatPercentage(value: number) {
  return `${Math.round(value)}%`;
}

export function clampNonNegative(value: number) {
  if (Number.isNaN(value)) {
    return 0;
  }

  return Math.max(0, value);
}

export function getUsageRatio(used: number, limit?: number | null) {
  if (!limit || limit <= 0) {
    return 0;
  }

  return Math.min(100, (used / limit) * 100);
}

export function formatStatusLabel(value: string) {
  return value.replaceAll("_", " ");
}
