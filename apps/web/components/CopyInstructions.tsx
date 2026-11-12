"use client";

import { useEffect, useRef, useState } from "react";

export default function CopyInstructions({ source, filename }: { source: string; filename: "SKILL.md" | "AGENT.md" }) {
  const [canCopy, setCanCopy] = useState(false);
  const [status, setStatus] = useState<"idle" | "copied" | "failed">("idle");
  const timeout = useRef<ReturnType<typeof setTimeout> | null>(null);
  const attempt = useRef(0);

  useEffect(() => {
    setCanCopy(true);
    setStatus("idle");
    return () => {
      attempt.current += 1;
      if (timeout.current) clearTimeout(timeout.current);
    };
  }, [source]);

  async function copy() {
    const current = ++attempt.current;
    if (timeout.current) clearTimeout(timeout.current);
    setStatus("idle");
    try {
      await navigator.clipboard.writeText(source);
      if (current !== attempt.current) return;
      setStatus("copied");
      timeout.current = setTimeout(() => setStatus("idle"), 2000);
    } catch {
      if (current !== attempt.current) return;
      setStatus("failed");
    }
  }

  return (
    <div className="instruction-copy">
      <div className="instruction-copy-actions">
        <p className="copy-status" role="status">
          {status === "copied" ? "Full instructions copied." : status === "failed" ? "Clipboard unavailable. Select and copy the source below." : ""}
        </p>
        <button
          type="button"
          className="button button--secondary copy-button"
          aria-label={`Copy full ${filename} text`}
          disabled={!canCopy}
          onClick={copy}
        >
          {status === "copied" ? "Copied" : "Copy full text"}
        </button>
      </div>
      <noscript>
        <p className="meta">The copy button requires JavaScript. Open the source text to select the complete file.</p>
        <details className="detail-disclosure-nested">
          <summary>Read source text ({filename})</summary>
          <textarea className="instruction-copy-source" aria-label={`${filename} source text`} readOnly rows={10} value={source} />
        </details>
      </noscript>
      {status === "failed" ? (
        <textarea
          className="instruction-copy-source"
          aria-label={`${filename} source text`}
          readOnly
          rows={10}
          value={source}
          onFocus={(event) => event.currentTarget.select()}
        />
      ) : null}
    </div>
  );
}
