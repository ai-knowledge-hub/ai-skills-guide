import type { Metadata } from "next";
import UsabilityBadge from "@/components/UsabilityBadge";
import { notFound } from "next/navigation";
import InstallCommands from "@/components/InstallCommands";
import { buildModuleInstallSnippet, getEntryById, loadToolsRegistry } from "@/lib/registry";
import { formatCategoryLabel, formatReadinessLabel } from "@/lib/categoryLabels";
import UsabilityPanel from "@/components/UsabilityPanel";

export async function generateMetadata({ params }: { params: { id: string[] } }): Promise<Metadata> {
  const entry = await getEntryById("tools", params.id.join("/"));
  return { title: entry?.name ?? "Entry not found", description: entry?.description };
}

export async function generateStaticParams() {
  const registry = await loadToolsRegistry();
  return registry.skills.map((entry) => ({ id: entry.id.split("/") }));
}

export default async function ToolDetailPage({ params }: { params: { id: string[] } }) {
  const entryId = params.id.join("/");
  const entry = await getEntryById("tools", entryId);
  if (!entry) {
    notFound();
  }
  const latestVersion = entry.versions.find((v) => v.version === entry.latest) ?? entry.versions[0];

  return (
    <main id="main" tabIndex={-1} className="detail-page">
      <nav className="breadcrumb" aria-label="Breadcrumb"><a href="/tools-mcp">All Tools &amp; MCP</a><span aria-hidden="true">/</span><span>{entry.id}</span></nav>

      <article className="card detail-lead">
        <p className="meta">{formatCategoryLabel(entry.category)}</p>
        <h1>{entry.name}</h1>
        <p>{entry.description}</p>
        <p className="meta">{entry.tags.join(", ")}</p>
        <div className="detail-use-summary"><UsabilityBadge availability={entry.usability.availability} /><span className="meta">{entry.usability.execution}</span><a href="#usability" className="text-link">Requirements &amp; limits</a></div>
        <div className="detail-install-lead">
          <h2 className="meta">Install this connector</h2>
          <InstallCommands
            compact
            codex={buildModuleInstallSnippet("tools", entry, "codex")}
            claude={buildModuleInstallSnippet("tools", entry, "claude")}
            generic={buildModuleInstallSnippet("tools", entry, "generic")}
          />
        </div>
      </article>

      <section className="detail-grid">
        <UsabilityPanel usability={entry.usability} />
        <article className="card detail-panel">
          <h2>Operational Summary</h2>
          <p><span className="meta">Connected system:</span> {entry.operational?.connected_system ?? "Not documented"}</p>
          <p><span className="meta">Access level:</span> {entry.operational?.access_level ?? "Not documented"}</p>
          <p><span className="meta">Trust boundary:</span> {entry.operational?.trust_boundary ?? "Not documented"}</p>
          <p><span className="meta">Approval boundary:</span> {entry.operational?.approval_boundary ?? "Not documented"}</p>
        </article>
        <article className="card detail-panel">
          <h2>Status</h2>
          <p><span className="meta">Readiness:</span> {formatReadinessLabel(entry.readiness)}</p>
          <p><span className="meta">Security reviewed:</span> {entry.security_reviewed ? "yes" : "no"}</p>
          <p><span className="meta">Lifecycle:</span> {entry.deprecated ? "Deprecated" : "Active"}</p>
        </article>
        <article className="card detail-panel">
          <h2>Runtime &amp; Dependencies</h2>
          <p><span className="meta">ID:</span> {entry.id}</p>
          <p><span className="meta">Runtimes:</span> {entry.runtimes.join(", ")}</p>
          <p><span className="meta">Auth required:</span> {entry.operational?.auth_required?.length ? entry.operational.auth_required.join(", ") : "Not documented"}</p>
          {entry.replaced_by ? <p><span className="meta">Replaced by:</span> {entry.replaced_by}</p> : null}
        </article>
        <article className="card detail-panel">
          <h2>Dependencies &amp; Setup</h2>
          <p><span className="meta">MCP servers:</span> {entry.dependencies?.mcp_servers?.length ? entry.dependencies.mcp_servers.join(", ") : "Not documented"}</p>
          <p><span className="meta">APIs:</span> {entry.dependencies?.apis?.length ? entry.dependencies.apis.join(", ") : "Not documented"}</p>
          <p><span className="meta">Local tools:</span> {entry.dependencies?.tools?.length ? entry.dependencies.tools.join(", ") : "Not documented"}</p>
        </article>
        <article className="card detail-panel">
          <h2>Capabilities</h2>
          {entry.operational?.capabilities?.length ? (
            <ul>
              {entry.operational.capabilities.map((capability) => (
                <li key={capability}>{capability}</li>
              ))}
            </ul>
          ) : (
            <p>Not documented.</p>
          )}
        </article>
        <article className="card detail-panel">
          <h2>Latest Version</h2>
          <p><span className="meta">Latest:</span> {entry.latest}</p>
          <p><span className="meta">Released:</span> {latestVersion?.released_at.slice(0, 10) ?? "n/a"}</p>
          <p>
            <span className="meta">Manifest:</span>{" "}
            {latestVersion ? <a href={latestVersion.manifest_url}>{latestVersion.manifest_url}</a> : "n/a"}
          </p>
          <p>
            <span className="meta">Artifact:</span>{" "}
            {latestVersion ? <a href={latestVersion.artifact_url}>{latestVersion.artifact_url}</a> : "n/a"}
          </p>
        </article>
      </section>
    </main>
  );
}
