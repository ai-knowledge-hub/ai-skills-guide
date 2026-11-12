"use client";

import { useEffect, useId, useRef, useState } from "react";
import type { ModuleKey, RegistryEntry } from "@/lib/registry";
import { executionRuntimesFor, installationTarget, installationModuleTarget, installerRelease, type InstallationGuidance, type RuntimeKey } from "@/lib/installation";
import { availabilityDescriptions, installationExplanation } from "@/lib/cataloguePresentation";

type InstallCommandsProps = {
  codex: string;
  claude: string;
  generic: string;
  module: ModuleKey;
  entry: Pick<RegistryEntry, "id" | "runtimes" | "usability" | "execution">;
  guidance: InstallationGuidance;
  compact?: boolean;
};

export default function InstallCommands({ codex, claude, generic, module, entry, guidance, compact = false }: InstallCommandsProps) {
  const { availability, execution } = entry.usability;
  const runtimeGroup = useId();
  const [canCopy, setCanCopy] = useState(false);
  const [copied, setCopied] = useState<RuntimeKey | null>(null);
  const [failed, setFailed] = useState<RuntimeKey | null>(null);
  const timeout = useRef<ReturnType<typeof setTimeout> | null>(null);
  const attempt = useRef(0);

  useEffect(() => {
    setCanCopy(true);
    return () => {
      attempt.current += 1;
      if (timeout.current) clearTimeout(timeout.current);
    };
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
        <p>{availabilityDescriptions[availability]}</p>
      </article>
    );
  }

  if (guidance.unavailable.length || guidance.platforms?.length === 0) {
    return <article className={compact ? "install-inline" : "card detail-panel install-card"}>
      <h3>Review installation requirements</h3>
      <p>{guidance.unavailable.length ? "The full release requirements could not be read here. Inspect the release package and requirements above before constructing an installation command." : "The release and its bundled dependencies declare no common supported installer host. Review their platform requirements before installation."}</p>
    </article>;
  }

  const platformLabels: Record<string, string> = { linux: "Linux", macos: "macOS", windows: "Windows" };
  const excludedHosts = guidance.platforms ? Object.keys(platformLabels).filter((value) => !guidance.platforms!.includes(value)) : [];

  return (
    <article className={compact ? "install-inline" : "card detail-panel install-card"}>
      <p className="install-explanation">{installationExplanation(execution)}</p>
      <p className="install-hosts">{guidance.platforms ? <><strong>Declared installation hosts · {guidance.platforms.map((value) => platformLabels[value]).join(", ")}</strong>. This includes bundled dependency restrictions.{excludedHosts.length ? <> {excludedHosts.map((value) => platformLabels[value]).join(", ")} {excludedHosts.length === 1 ? "is" : "are"} not supported for this release.</> : null}</> : "No installer host restriction is declared for these files. Confirm native app discovery separately."}</p>
      <section className="installer-setup">
        <h3>1. Get the installer</h3>
        <p>Use the <a href={installerRelease.url}>installation-only CLI {installerRelease.version}</a>. The commands below assume <code>skills-hub</code> is on your PATH; no repository checkout is needed.</p>
        <details className="detail-disclosure-nested">
          <summary>Download and prepare skills-hub</summary>
          <p>Choose your operating system and processor. Download <a href={`${installerRelease.assets}/SHA256SUMS`}>SHA256SUMS</a> and compare the downloaded file&apos;s SHA-256 with its matching entry before running it. The release also includes <a href={`${installerRelease.assets}/skills-hub_${installerRelease.version}_provenance.json`}>build provenance</a>.</p>
          <ul>{[
            ["macOS", "darwin", "Intel", "Apple silicon"],
            ["Linux", "linux", "x86-64", "ARM64"],
            ["Windows", "windows", "x86-64", "ARM64"],
          ].map(([label, os, amd64, arm64]) => (
            <li key={os}>{label} · {[ ["amd64", amd64], ["arm64", arm64] ].map(([arch, cpu], index) => (
              <span key={arch}>{index ? " · " : ""}<a href={`${installerRelease.assets}/skills-hub_${installerRelease.version}_${os}_${arch}${os === "windows" ? ".exe" : ""}`}>{cpu}</a></span>
            ))}</li>
          ))}</ul>
          <p>Rename the binary to <code>skills-hub</code> on macOS/Linux or <code>skills-hub.exe</code> on Windows, and add its folder to your PATH. On macOS/Linux, make it executable with <code>chmod +x ./skills-hub</code>. Check it with <code>skills-hub help</code>. Alternatively, replace <code>skills-hub</code> in each command with your binary&apos;s absolute path.</p>
          <p className="meta">Published binaries and package adapter declarations are separate from verified operation in a specific host or app version. <a href="https://github.com/ai-knowledge-hub/ai-skills-guide/issues/52">Current verification coverage and native discovery limits</a>.</p>
          <details className="detail-disclosure-nested">
            <summary>Developer checkout alternative</summary>
            <p>With Git, Go 1.22+ and Make installed, run these commands from the folder where you keep projects. This builds an unreleased local installer candidate.</p>
            <pre tabIndex={0} aria-label="Developer installer setup"><code>{"git clone https://github.com/ai-knowledge-hub/ai-skills-guide.git\ncd ai-skills-guide\nmake cli-build\n./bin/skills-hub help"}</code></pre>
            <p>Stay in the checkout root and replace <code>skills-hub</code> below with <code>./bin/skills-hub</code>. Keep <code>--source remote</code> to install published packages. The CLI&apos;s default <code>--source local</code> reads mutable checkout files for development.</p>
          </details>
        </details>
      </section>
      <section className="install-runtime-choices" aria-labelledby={`${runtimeGroup}-heading`}>
        <h3 id={`${runtimeGroup}-heading`}>2. Choose an agent app</h3>
        <p>Open one supported adapter to see its command. Run it from any folder; the generic example uses a target relative to that folder.</p>
        <noscript><p>Select and copy command text directly. Copy buttons require JavaScript.</p></noscript>
        {rows.filter((row) => guidance.adapters.includes(row.key)).map((row) => {
          const environments = executionRuntimesFor(guidance, row.key);
          const environmentLabels = environments.map((environment) => environment === "node22" ? "Node.js 22 or newer"
            : environment === "python3" ? "the package's required Python 3 version" : environment);
          return (
            <details key={row.key} className="install-runtime-option" {...{ name: runtimeGroup }}>
              <summary>{row.key === "generic" ? "Generic / another app" : row.key === "codex" ? "Codex" : "Claude"}</summary>
              <section className="install-item">
                <p>Package location · <code>{installationTarget(module, entry.id, row.key)}</code></p>
                {row.key === "generic" ? <p>Choose your intended module directory in <code>--target</code> before running. The installer appends the package ID beneath it.</p> : <p className="meta">{row.key === "codex" ? "CODEX_HOME can change this root." : "CLAUDE_HOME or CLAUDE_CODE_HOME can change this root."} Confirm that your app discovers this location; <a href="https://github.com/ai-knowledge-hub/ai-skills-guide/issues/52">default-path compatibility remains under verification</a>.</p>}
                {guidance.bundledModules.length ? <p className="install-closure-effects">For the shown root, also writes bundled components under {guidance.bundledModules.map((child) => installationModuleTarget(child, row.key)).join(", ")}. {module === "agents" || row.key !== "generic" ? "Review the generated adapter registration/configuration files before connecting the package to your app." : "Connect this plugin directory and its components to your app manually."}</p> : null}
                {environments.length ? <p>For this command, have {environmentLabels.join(" and ")} available first. <code>--execution-runtime</code> declares those environments; it does not install them.</p> : null}
                <div className="install-head">
                  <h4>{row.title}</h4>
                  <button
                    type="button"
                    className="button button--secondary copy-button"
                    aria-label={`Copy ${row.key} installation command`}
                    disabled={!canCopy}
                    onClick={() => copy(row.key, row.command)}
                  >
                    {copied === row.key ? "Copied" : "Copy"}
                  </button>
                </div>
                <pre className="install" tabIndex={0} role="region" aria-label={`${row.key} installation command`}><code>{row.command}</code></pre>
                <p className="copy-status" role="status">{failed === row.key ? "Clipboard unavailable. Select and copy the command above." : copied === row.key ? "Command copied." : ""}</p>
                <p>After file installation, run <code>skills-hub help setup</code> and follow the requirements and first-use instructions above. The installer reports the destination; use that installed package folder for documented samples.</p>
              </section>
            </details>
          );
        })}
        {!rows.some((row) => guidance.adapters.includes(row.key)) ? <p>No common installation adapter is declared for this release and its bundled dependencies.</p> : null}
      </section>
    </article>
  );
}
