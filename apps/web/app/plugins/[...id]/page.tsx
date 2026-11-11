import type { Metadata } from "next";
import UsabilityBadge from "@/components/UsabilityBadge";
import { notFound } from "next/navigation";
import InstallCommands from "@/components/InstallCommands";
import PluginContents from "@/components/PluginContents";
import { buildModuleInstallSnippet, getEntryById, loadPluginsRegistry } from "@/lib/registry";
import { formatCategoryLabel } from "@/lib/categoryLabels";
import UsabilityPanel from "@/components/UsabilityPanel";
import ReviewStatus from "@/components/ReviewStatus";
import { resourceFormLabels } from "@/lib/cataloguePresentation";
import { findCapabilityNote } from "@/lib/capabilityNotes";

const REPO_BLOB_BASE = "https://github.com/ai-knowledge-hub/ai-skills-guide/blob/main";

export async function generateMetadata({ params }: { params: { id: string[] } }): Promise<Metadata> {
  const entry = await getEntryById("plugins", params.id.join("/"));
  return { title: entry?.name ?? "Entry not found", description: entry?.description };
}

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
    <main id="main" tabIndex={-1} className="detail-page">
      <nav className="breadcrumb" aria-label="Breadcrumb"><a href="/plugins">All Plugins</a><span aria-hidden="true">/</span><span>{entry.id}</span></nav>

      <article className="card detail-lead">
        <p className="meta">{formatCategoryLabel(entry.category)}</p>
        <h1>{entry.name}</h1>
        <p>{entry.description}</p>
        <p className="meta">{entry.tags.join(", ")}</p>
        <div className="detail-use-summary"><UsabilityBadge availability={entry.usability.availability} /><span className="meta">{resourceFormLabels[entry.usability.execution]}</span><a href="#usability" className="text-link">Requirements &amp; limits</a></div>
        <PluginContents entry={entry} />
        <div className="detail-install-lead">
          <h2 className="meta">{entry.usability.availability === "template-only" ? "Reference composition" : "Install plugin files"}</h2>
          <InstallCommands
            compact
            availability={entry.usability.availability}
            execution={entry.usability.execution}
            codex={buildModuleInstallSnippet("plugins", entry, "codex")}
            claude={buildModuleInstallSnippet("plugins", entry, "claude")}
            generic={buildModuleInstallSnippet("plugins", entry, "generic")}
          />
          {entry.usability.availability !== "template-only" ? (
            <>
              <p className="meta">Bundled skills, agents and tools are placed in the selected runtime directories. Hooks stay inside the plugin directory. Codex and Claude targets also receive a generated plugin manifest.</p>
            </>
          ) : null}
          {!entry.security_reviewed ? (
            <p className="meta">No security review is declared for this release. Review bundled components, required secrets and approval rules before live use.</p>
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
        <UsabilityPanel catalogueCompatibility={entry.catalogueCompatibility} compatibility={entry.compatibility} usability={entry.usability} />
        {entry.capability_readiness?.length ? (
          <article className="card detail-panel">
            <h2>Capability readiness</h2>
            {entry.capability_readiness.map((capability) => {
              const note = findCapabilityNote("plugins", entry.id, entry.latest, capability.id);
              return (
                <section key={capability.id}>
                  <h3>{capability.name}</h3>
                  <p>
                    {note ? <strong>{note.label}</strong> : <UsabilityBadge availability={capability.availability} />}
                    {!note ? <> · <span className="meta">{capability.access === "offline" ? "Sample fixtures" : capability.access === "read-only" ? "Read only" : "Read and write"}</span></> : null}
                  </p>
                  <p>{note?.explanation ?? capability.description}</p>
                  {note ? <p className="meta">Recorded classification · {capability.availability} · <a href={note.source.url}>{note.source.label}</a></p> : null}
                  {capability.requires_setup?.length ? (
                    <ul>
                      {capability.requires_setup.map((requirement) => <li key={requirement}>{requirement}</li>)}
                    </ul>
                  ) : null}
                  {capability.limitations?.length ? (
                    <ul>
                      {capability.limitations.map((limitation) => <li key={limitation}>{limitation}</li>)}
                    </ul>
                  ) : null}
                </section>
              );
            })}
          </article>
        ) : null}
        <ReviewStatus entry={entry} />
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
          {entry.requires?.secrets?.length ? (
            <p><span className="meta">Secret names:</span> {entry.requires.secrets.join(", ")}</p>
          ) : null}
          {entry.requires?.approvals?.length ? (
            <p><span className="meta">Approval rules:</span> {entry.requires.approvals.join(", ")}</p>
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
