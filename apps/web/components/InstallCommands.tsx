"use client";

import { useState } from "react";

type InstallCommandsProps = {
  codex: string;
  claude: string;
  generic: string;
};

type RuntimeKey = "codex" | "claude" | "generic";

export default function InstallCommands({ codex, claude, generic }: InstallCommandsProps) {
  const [copied, setCopied] = useState<RuntimeKey | null>(null);

  async function copy(runtime: RuntimeKey, value: string) {
    try {
      await navigator.clipboard.writeText(value);
      setCopied(runtime);
      window.setTimeout(() => setCopied((prev) => (prev === runtime ? null : prev)), 1400);
    } catch {
      setCopied(null);
    }
  }

  const rows: Array<{ key: RuntimeKey; title: string; command: string }> = [
    { key: "codex", title: "Install (Codex)", command: codex },
    { key: "claude", title: "Install (Claude)", command: claude },
    { key: "generic", title: "Install (Generic)", command: generic }
  ];

  return (
    <article className="card detail-panel install-card">
      {rows.map((row) => (
        <section key={row.key} className="install-item">
          <div className="install-head">
            <h2>{row.title}</h2>
            <button
              type="button"
              className="button button--secondary copy-button"
              onClick={() => copy(row.key, row.command)}
            >
              {copied === row.key ? "Copied" : "Copy"}
            </button>
          </div>
          <pre className="install">{row.command}</pre>
        </section>
      ))}
    </article>
  );
}
