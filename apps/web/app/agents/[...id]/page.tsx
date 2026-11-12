import type { Metadata } from "next";
import UsabilityBadge from "@/components/UsabilityBadge";
import { notFound } from "next/navigation";
import InstallCommands from "@/components/InstallCommands";
import InstallationDetails from "@/components/InstallationDetails";
import ResourceFirstUse from "@/components/ResourceFirstUse";
import { loadResourceFirstUse } from "@/lib/resourceFirstUse";
import DetailDisclosure from "@/components/DetailDisclosure";
import InstructionMarkdown from "@/components/InstructionMarkdown";
import CopyInstructions from "@/components/CopyInstructions";
import { buildModuleInstallSnippet, getEntryById, loadAgentsRegistry } from "@/lib/registry";
import { formatCategoryLabel } from "@/lib/categoryLabels";
import UsabilityPanel from "@/components/UsabilityPanel";
import ReviewStatus from "@/components/ReviewStatus";
import { resourceFormLabels } from "@/lib/cataloguePresentation";

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
  const { role, coordinates, autonomy_level, approval_boundary } = entry.operational ?? {};
  const autonomy = autonomy_level === "execution-capable-with-approval" ? "Execution with approval" : autonomy_level;
  const firstUse = await loadResourceFirstUse("agents", entry);
  const instructions = firstUse.files?.get("AGENT.md");

  return (
    <main id="main" tabIndex={-1} className="detail-page">
      <nav className="breadcrumb" aria-label="Breadcrumb"><a href="/agents">All Agents</a><span aria-hidden="true">/</span><span>{entry.id}</span></nav>

      <article className="card detail-lead">
        <p className="meta">{formatCategoryLabel(entry.category)}</p>
        <h1>{entry.name}</h1>
        <p>{entry.description}</p>
        <div className="detail-use-summary"><UsabilityBadge availability={entry.usability.availability} /><span className="meta">{resourceFormLabels[entry.usability.execution]}</span><a href="#usability" className="text-link">Requirements &amp; limits</a></div>
        <ResourceFirstUse entry={entry} content={firstUse}>
          <DetailDisclosure show="View agent workflow" hide="Hide agent workflow" detail={`v${entry.latest}`}>
            {role ? <p>{role}</p> : null}
            <section>
              <h2>What it coordinates</h2>
              {coordinates?.length ? (
                <ul>{coordinates.map((item) => <li key={item}>{item}</li>)}</ul>
              ) : <p>Coordination is not documented for this release.</p>}
            </section>
            {autonomy || approval_boundary ? (
              <section>
                <h2>Autonomy and approvals</h2>
                {autonomy ? <p><strong>{autonomy.charAt(0).toUpperCase() + autonomy.slice(1)}</strong></p> : null}
                {approval_boundary ? <p>{approval_boundary}</p> : null}
              </section>
            ) : null}
            {instructions ? (
              <details className="detail-disclosure-nested">
                <summary>Read agent instructions (AGENT.md)</summary>
                <div className="detail-disclosure-document">
                  <CopyInstructions source={instructions} filename="AGENT.md" />
                  <InstructionMarkdown source={instructions} />
                </div>
              </details>
            ) : null}
          </DetailDisclosure>
        </ResourceFirstUse>
        <InstallationDetails title={entry.usability.availability === "template-only" ? "Reference template" : "Install agent files"}>
          <InstallCommands
            compact
            availability={entry.usability.availability}
            execution={entry.usability.execution}
            codex={buildModuleInstallSnippet("agents", entry, "codex")}
            claude={buildModuleInstallSnippet("agents", entry, "claude")}
            generic={buildModuleInstallSnippet("agents", entry, "generic")}
          />
        </InstallationDetails>
      </article>

      <section className="detail-grid resource-evidence">
        <UsabilityPanel catalogueCompatibility={entry.catalogueCompatibility} compatibility={entry.compatibility} usability={entry.usability} />
      </section>
      <details className="resource-technical">
        <summary>Technical details and release files</summary>
        <section className="detail-grid">
          <ReviewStatus entry={entry} />
          <article className="card detail-panel">
            <h2>Runtime &amp; Dependencies</h2>
            <p><span className="meta">ID:</span> {entry.id}</p>
            <p><span className="meta">Tags:</span> {entry.tags.join(", ")}</p>
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
      </details>
    </main>
  );
}
