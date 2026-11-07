import type { Metadata } from "next";
import UsabilityBadge from "@/components/UsabilityBadge";
import { notFound } from "next/navigation";
import InstallCommands from "@/components/InstallCommands";
import DetailDisclosure from "@/components/DetailDisclosure";
import InstructionMarkdown from "@/components/InstructionMarkdown";
import { buildModuleInstallSnippet, getEntryById, loadAgentsRegistry } from "@/lib/registry";
import { formatCategoryLabel, formatReadinessLabel } from "@/lib/categoryLabels";
import UsabilityPanel from "@/components/UsabilityPanel";
import { readReleaseFiles } from "@/lib/releaseFiles";

export async function generateMetadata({ params }: { params: { id: string[] } }): Promise<Metadata> {
  const entry = await getEntryById("agents", params.id.join("/"));
  return { title: entry?.name ?? "Entry not found", description: entry?.description };
}

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
  const { role, coordinates, outputs, autonomy_level, approval_boundary } = entry.operational ?? {};
  const autonomy = autonomy_level === "execution-capable-with-approval" ? "Execution with approval" : autonomy_level;
  const limitations = entry.usability.limitations;
  const files = await readReleaseFiles("agents", entry, ["AGENT.md"]);
  const instructions = files?.get("AGENT.md");

  return (
    <main id="main" tabIndex={-1} className="detail-page">
      <nav className="breadcrumb" aria-label="Breadcrumb"><a href="/agents">All Agents</a><span aria-hidden="true">/</span><span>{entry.id}</span></nav>

      <article className="card detail-lead">
        <p className="meta">{formatCategoryLabel(entry.category)}</p>
        <h1>{entry.name}</h1>
        <p>{entry.description}</p>
        {role ? (
          <div className="detail-purpose">
            <h2>Role</h2>
            <p>{role}</p>
          </div>
        ) : null}
        <p className="meta">{entry.tags.join(", ")}</p>
        <div className="detail-use-summary"><UsabilityBadge availability={entry.usability.availability} /><span className="meta">{entry.usability.execution}</span><a href="#usability" className="text-link">Requirements &amp; limits</a></div>
        <DetailDisclosure show="View agent workflow" hide="Hide agent workflow" detail={`v${entry.latest}`}>
          <section>
            <h2>What it coordinates</h2>
            {coordinates?.length ? (
              <ul>{coordinates.map((item) => <li key={item}>{item}</li>)}</ul>
            ) : <p>Coordination is not documented for this release.</p>}
          </section>
          <section>
            <h2>Expected outputs</h2>
            {outputs?.length ? (
              <ul>{outputs.map((item) => <li key={item}>{item}</li>)}</ul>
            ) : <p>Outputs are not documented for this release.</p>}
          </section>
          {autonomy || approval_boundary ? (
            <section>
              <h2>Autonomy and approvals</h2>
              {autonomy ? <p><strong>{autonomy.charAt(0).toUpperCase() + autonomy.slice(1)}</strong></p> : null}
              {approval_boundary ? <p>{approval_boundary}</p> : null}
            </section>
          ) : null}
          {limitations?.length ? (
            <section>
              <h2>Current limits</h2>
              <ul>{limitations.map((item) => <li key={item}>{item}</li>)}</ul>
            </section>
          ) : null}
          {instructions ? (
            <details className="detail-disclosure-nested">
              <summary>Read agent instructions (AGENT.md)</summary>
              <div className="detail-disclosure-document"><InstructionMarkdown source={instructions} /></div>
            </details>
          ) : null}
        </DetailDisclosure>
        <div className="detail-install-lead">
          <h2 className="meta">{entry.usability.availability === "template-only" ? "Reference template" : "Install this agent"}</h2>
          <InstallCommands
            compact
            availability={entry.usability.availability}
            codex={buildModuleInstallSnippet("agents", entry, "codex")}
            claude={buildModuleInstallSnippet("agents", entry, "claude")}
            generic={buildModuleInstallSnippet("agents", entry, "generic")}
          />
        </div>
      </article>

      <section className="detail-grid">
        <UsabilityPanel catalogueCompatibility={entry.catalogueCompatibility} compatibility={entry.compatibility} usability={entry.usability} />
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
          <p><span className="meta">Skill dependencies:</span> {entry.dependencies?.skills?.length ?? 0}</p>
          <p><span className="meta">Agent dependencies:</span> {entry.dependencies?.agents?.length ?? 0}</p>
          <p><span className="meta">Tool dependencies:</span> {entry.dependencies?.tools?.length ?? 0}</p>
          {entry.replaced_by ? <p><span className="meta">Replaced by:</span> {entry.replaced_by}</p> : null}
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
