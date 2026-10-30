"use client";

import { useEffect, useRef, useState } from "react";
import type { RegistryEntry } from "@/lib/registry";

type InstallCommandsProps = {
  codex: string;
  claude: string;
  generic: string;
  availability: RegistryEntry["usability"]["availability"];
  compact?: boolean;
};

type RuntimeKey = "codex" | "claude" | "generic";

export default function InstallCommands({ codex, claude, generic, availability, compact = false }: InstallCommandsProps) {
  const [copied, setCopied] = useState<RuntimeKey | null>(null);
  const [failed, setFailed] = useState<RuntimeKey | null>(null);
  const timeout = useRef<ReturnType<typeof setTimeout> | null>(null);
  const attempt = useRef(0);

  useEffect(() => () => {
    attempt.current += 1;
    if (timeout.current) clearTimeout(timeout.current);
  }, []);

  async function copy(runtime: RuntimeKey, value: string) {
    const current = ++attempt.current;
    if (timeout.current) clearTimeout(timeout.current);
    setCopied(null);
    setFailed(null);
    try {
      await navigator.clipboard.writeText(value);
      if (current !== attempt.current) return;
      setCopied(runtime);
      timeout.current = setTimeout(() => setCopied(null), 2000);
    } catch {
      if (current !== attempt.current) return;
      setCopied(null);
      setFailed(runtime);
    }
  }

  const rows: Array<{ key: RuntimeKey; title: string; command: string }> = [
    { key: "codex", title: "Install (Codex)", command: codex },
    { key: "claude", title: "Install (Claude)", command: claude },
    { key: "generic", title: "Install (Generic)", command: generic }
  ];

  if (availability === "template-only") {
    return (
      <article className={compact ? "install-inline" : "card detail-panel install-card"}>
        <h3>Reference template</h3>
        <p>This entry cannot be installed into an operational runtime. Copy its source from the repository only as a scaffold for implementation and review.</p>
      </article>
    );
  }

  return (
    <article className={compact ? "install-inline" : "card detail-panel install-card"}>
      {rows.map((row) => (
        <section key={row.key} className="install-item">
          <div className="install-head">
            {compact ? <h3>{row.title}</h3> : <h2>{row.title}</h2>}
            <button
              type="button"
              className="button button--secondary copy-button"
              aria-label={`Copy ${row.key} installation command`}
              onClick={() => copy(row.key, row.command)}
            >
              {copied === row.key ? "Copied" : "Copy"}
            </button>
          </div>
          <pre className="install" tabIndex={0} aria-label={`${row.key} installation command`}>{row.command}</pre>
          <p className="copy-status" role="status">{failed === row.key ? "Clipboard unavailable. Select and copy the command above." : copied === row.key ? "Command copied." : ""}</p>
        </section>
      ))}
    </article>
  );
}
