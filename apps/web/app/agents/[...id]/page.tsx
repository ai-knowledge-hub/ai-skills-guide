import { notFound } from "next/navigation";
import InstallCommands from "@/components/InstallCommands";
import { buildModuleInstallSnippet, getEntryById, loadAgentsRegistry } from "@/lib/registry";
import { formatCategoryLabel, formatReadinessLabel } from "@/lib/categoryLabels";

export async function generateStaticParams() {
  const registry = await loadAgentsRegistry();
  return registry.skills.map((entry) => ({ id: entry.id.split("/") }));
}

export default async function AgentDetailPage({ params }: { params: { id: string[] } }) {
  const entryId = params.id.join("/");
  const entry = await getEntryById("agents", entryId);
  if (!entry) {
    notFound();
  }
  const latestVersion = entry.versions.find((v) => v.version === entry.latest) ?? entry.versions[0];

  return (
    <main>
      <div className="nav">
        <a href="/" className="pill">Home</a>
        <a href="/agents" className="pill">Agents</a>
        <a href="/plugins" className="pill">Plugins</a>
        <span className="pill">{entry.id}</span>
      </div>

      <article className="card">
        <p className="meta">{formatCategoryLabel(entry.category)}</p>
        <h1>{entry.name}</h1>
        <p>{entry.description}</p>
        <div className="tags">
          {entry.tags.map((tag) => (
            <span key={tag} className="tag">{tag}</span>
          ))}
        </div>
        <div className="detail-install-lead">
          <p className="meta">Install this agent</p>
          <InstallCommands
            compact
            codex={buildModuleInstallSnippet("agents", entry, "codex")}
            claude={buildModuleInstallSnippet("agents", entry, "claude")}
            generic={buildModuleInstallSnippet("agents", entry, "generic")}
          />
        </div>
      </article>

      <section className="detail-grid">
        <article className="card detail-panel">
          <h2>Operational Summary</h2>
          <p><span className="meta">Role:</span> {entry.operational?.role ?? "Not documented"}</p>
          <p><span className="meta">Autonomy level:</span> {entry.operational?.autonomy_level ?? "Not documented"}</p>
          <p><span className="meta">Approval boundary:</span> {entry.operational?.approval_boundary ?? "Not documented"}</p>
        </article>
        <article className="card detail-panel">
          <h2>Status</h2>
          <p><span className={`catalog-status-badge is-${entry.readiness}`}>{formatReadinessLabel(entry.readiness)}</span></p>
          <p><span className="meta">Security reviewed:</span> {entry.security_reviewed ? "yes" : "no"}</p>
          <p><span className="meta">Lifecycle:</span> {entry.deprecated ? "Deprecated" : "Active"}</p>
        </article>
        <article className="card detail-panel">
          <h2>Runtime &amp; Dependencies</h2>
          <p><span className="meta">ID:</span> {entry.id}</p>
          <p><span className="meta">Runtimes:</span> {entry.runtimes.join(", ")}</p>
          <p><span className="meta">Skill dependencies:</span> {entry.dependencies?.skills?.length ?? 0}</p>
          <p><span className="meta">Tool dependencies:</span> {entry.dependencies?.tools?.length ?? 0}</p>
          {entry.replaced_by ? <p><span className="meta">Replaced by:</span> {entry.replaced_by}</p> : null}
        </article>
        <article className="card detail-panel">
          <h2>Coordinates</h2>
          {entry.operational?.coordinates?.length ? (
            <ul>
              {entry.operational.coordinates.map((item) => (
                <li key={item}>{item}</li>
              ))}
            </ul>
          ) : (
            <p>Not documented.</p>
          )}
        </article>
        <article className="card detail-panel">
          <h2>Outputs</h2>
          {entry.operational?.outputs?.length ? (
            <ul>
              {entry.operational.outputs.map((item) => (
                <li key={item}>{item}</li>
              ))}
            </ul>
          ) : (
            <p>Not documented.</p>
          )}
        </article>
        <article className="card detail-panel">
          <h2>Dependencies</h2>
          {entry.dependencies?.skills?.length ? (
            <p><span className="meta">Skills:</span> {entry.dependencies.skills.join(", ")}</p>
          ) : (
            <p><span className="meta">Skills:</span> Not documented</p>
          )}
          {entry.dependencies?.tools?.length ? (
            <p><span className="meta">Tools:</span> {entry.dependencies.tools.join(", ")}</p>
          ) : (
            <p><span className="meta">Tools:</span> Not documented</p>
          )}
          {entry.dependencies?.apis?.length ? (
            <p><span className="meta">APIs:</span> {entry.dependencies.apis.join(", ")}</p>
          ) : null}
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
