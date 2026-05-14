"use client";

import { useState } from "react";
import { Button } from "@/components/shared/button";
import { useToast } from "@/components/providers/toast-provider";

export function CopyButton({
  value,
  label = "Copy",
}: {
  value: string;
  label?: string;
}) {
  const [copied, setCopied] = useState(false);
  const { pushToast } = useToast();

  async function handleCopy() {
    await navigator.clipboard.writeText(value);
    setCopied(true);
    pushToast({
      title: "Copied to clipboard",
      description: label === "Copy" ? "Value copied." : `${label} copied.`,
      tone: "success",
    });
    window.setTimeout(() => setCopied(false), 1200);
  }

  return (
    <Button variant="secondary" className="px-3 py-2 text-xs" onClick={handleCopy}>
      {copied ? "Copied" : label}
    </Button>
  );
}
