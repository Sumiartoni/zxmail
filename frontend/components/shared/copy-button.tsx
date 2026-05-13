"use client";

import { useState } from "react";
import { Button } from "@/components/shared/button";

export function CopyButton({
  value,
  label = "Copy",
}: {
  value: string;
  label?: string;
}) {
  const [copied, setCopied] = useState(false);

  async function handleCopy() {
    await navigator.clipboard.writeText(value);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1200);
  }

  return (
    <Button variant="secondary" className="px-3 py-2 text-xs" onClick={handleCopy}>
      {copied ? "Copied" : label}
    </Button>
  );
}
