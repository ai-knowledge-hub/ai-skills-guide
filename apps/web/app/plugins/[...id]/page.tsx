import { notFound } from "next/navigation";
import InstallCommands from "@/components/InstallCommands";
import { buildModuleInstallSnippet, getEntryById, loadPluginsRegistry } from "@/lib/registry";
import { formatCategoryLabel, formatReadinessLabel } from "@/lib/categoryLabels";

const REPO_BLOB_BASE = "https://github.com/ai-knowledge-hub/ai-skills-guide/blob/main";

export async function generateStaticParams() {
  const registry = await loadPluginsRegistry();
  return registry.skills.map((entry) => ({ id: entry.id.split("/") }));
}

export default async function PluginDetailPage({ params }: { params: { id: string[] } }) {
  const entryId = params.id.join("/");
  const entry = await getEntryById("plugins", entryId);
  if (!entry) {
    notFound();
  }
  const latestVersion = entry.versions.find((v) => v.version === entry.latest) ?? entry.versions[0];
  const packageBasePath = `plugins/${entry.id}`;

  return (
    <main>
      <div className="nav">
        <a href="/" className="pill">Home</a>
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
          <p className="meta">Install this plugin</p>
          <InstallCommands
            compact
            codex={buildModuleInstallSnippet("plugins", entry, "codex")}
            claude={buildModuleInstallSnippet("plugins", entry, "claude")}
            generic={buildModuleInstallSnippet("plugins", entry, "generic")}
          />
          <p className="meta">Plugin installs also resolve bundled skills, agents, and tools into their native runtime directories. Packaged hooks stay inside the plugin directory.</p>
          <p className="meta">Codex and Claude installs also generate a runtime-specific plugin manifest inside the installed plugin directory.</p>
          {!entry.security_reviewed ? (
            <p className="meta">This plugin is still experimental. Review bundled components, required secrets, and approval rules before enabling it in a live runtime.</p>
          ) : null}
          {entry.requires?.secrets?.length || entry.requires?.approvals?.length ? (
            <ul>
              {entry.requires?.secrets?.length ? (
                <li>Required secrets: {entry.requires.secrets.join(", ")}</li>
              ) : null}
              {entry.requires?.approvals?.length ? (
                <li>Approval rules: {entry.requires.approvals.join(", ")}</li>
              ) : null}
            </ul>
          ) : null}
        </div>
      </article>

      <section className="detail-grid">
        <article className="card detail-panel">
          <h2>Status</h2>
          <p><span className={`catalog-status-badge is-${entry.readiness}`}>{formatReadinessLabel(entry.readiness)}</span></p>
          <p><span className="meta">Security reviewed:</span> {entry.security_reviewed ? "yes" : "no"}</p>
          <p><span className="meta">Lifecycle:</span> {entry.deprecated ? "Deprecated" : "Active"}</p>
        </article>
        <article className="card detail-panel">
          <h2>Install Summary</h2>
          <p><span className="meta">Installed skills:</span> {entry.includes?.skills?.length ?? 0}</p>
          <p><span className="meta">Installed agents:</span> {entry.includes?.agents?.length ?? 0}</p>
          <p><span className="meta">Installed tools:</span> {entry.includes?.tools?.length ?? 0}</p>
          <p><span className="meta">Packaged hooks:</span> {entry.includes?.hooks?.length ?? 0}</p>
        </article>
        <article className="card detail-panel">
          <h2>Installed Skills</h2>
          {entry.includes?.skills?.length ? (
            <ul>
              {entry.includes.skills.map((skillId) => (
                <li key={skillId}>
                  <a href={`/skills/${skillId}`}>{skillId}</a>
                </li>
              ))}
            </ul>
          ) : (
            <p>No bundled skills declared.</p>
          )}
        </article>
        <article className="card detail-panel">
          <h2>Installed Agents</h2>
          {entry.includes?.agents?.length ? (
            <ul>
              {entry.includes.agents.map((agentId) => (
                <li key={agentId}>
                  <a href={`/agents/${agentId}`}>{agentId}</a>
                </li>
              ))}
            </ul>
          ) : (
            <p>No bundled agents declared.</p>
          )}
        </article>
        <article className="card detail-panel">
          <h2>Installed Tools</h2>
          {entry.includes?.tools?.length ? (
            <ul>
              {entry.includes.tools.map((toolId) => (
                <li key={toolId}>
                  <a href={`/tools-mcp/${toolId}`}>{toolId}</a>
                </li>
              ))}
            </ul>
          ) : (
            <p>No bundled tools declared.</p>
          )}
        </article>
        <article className="card detail-panel">
          <h2>Packaged Hooks</h2>
          {entry.includes?.hooks?.length ? (
            <ul>
              {entry.includes.hooks.map((hookId) => (
                <li key={hookId}>
                  <a href={`${REPO_BLOB_BASE}/${packageBasePath}/hooks/${hookId}.md`}>{hookId}</a>
                </li>
              ))}
            </ul>
          ) : (
            <p>No packaged hooks declared.</p>
          )}
        </article>
        <article className="card detail-panel">
          <h2>Metadata</h2>
          <p><span className="meta">ID:</span> {entry.id}</p>
          <p><span className="meta">Latest:</span> {entry.latest}</p>
          <p><span className="meta">Runtimes:</span> {entry.runtimes.join(", ")}</p>
          {entry.replaced_by ? <p><span className="meta">Replaced by:</span> {entry.replaced_by}</p> : null}
        </article>
        <article className="card detail-panel">
          <h2>Requirements</h2>
          <p><span className="meta">Secrets:</span> {entry.requires?.secrets?.length ?? 0}</p>
          <p><span className="meta">Approvals:</span> {entry.requires?.approvals?.length ?? 0}</p>
          <p><span className="meta">Hooks:</span> {entry.includes?.hooks?.length ?? 0}</p>
          {entry.requires?.secrets?.length ? (
            <p><span className="meta">Secret names:</span> {entry.requires.secrets.join(", ")}</p>
          ) : null}
          {entry.requires?.approvals?.length ? (
            <p><span className="meta">Approval rules:</span> {entry.requires.approvals.join(", ")}</p>
          ) : null}
          {entry.includes?.hooks?.length ? (
            <p><span className="meta">Hook names:</span> {entry.includes.hooks.join(", ")}</p>
          ) : null}
        </article>
        <article className="card detail-panel">
          <h2>Latest Version</h2>
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
        <article className="card detail-panel">
          <h2>Package Files</h2>
          <ul>
            <li><a href={`${REPO_BLOB_BASE}/${packageBasePath}/README.md`}>README.md</a></li>
            <li><a href={`${REPO_BLOB_BASE}/${packageBasePath}/plugin.yaml`}>plugin.yaml</a></li>
            <li><a href={`${REPO_BLOB_BASE}/${packageBasePath}/plugin.json`}>plugin.json</a></li>
            <li><a href={`${REPO_BLOB_BASE}/${packageBasePath}/tests/test-prompts.md`}>tests/test-prompts.md</a></li>
            <li><a href={`${REPO_BLOB_BASE}/${packageBasePath}/examples/overview.md`}>examples/overview.md</a></li>
          </ul>
        </article>
      </section>
    </main>
  );
}
