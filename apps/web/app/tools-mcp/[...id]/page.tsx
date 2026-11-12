import type { Metadata } from "next";
import UsabilityBadge from "@/components/UsabilityBadge";
import { notFound } from "next/navigation";
import InstallCommands from "@/components/InstallCommands";
import InstallationDetails from "@/components/InstallationDetails";
import ResourceFirstUse from "@/components/ResourceFirstUse";
import { loadResourceFirstUse } from "@/lib/resourceFirstUse";
import DetailDisclosure from "@/components/DetailDisclosure";
import InstructionMarkdown from "@/components/InstructionMarkdown";
import { buildModuleInstallSnippet, getEntryById, loadToolsRegistry } from "@/lib/registry";
import { formatCategoryLabel } from "@/lib/categoryLabels";
import UsabilityPanel from "@/components/UsabilityPanel";
import ReviewStatus from "@/components/ReviewStatus";
import { resourceFormLabels } from "@/lib/cataloguePresentation";
import RelatedReading from "@/components/RelatedReading";
import { relatedPublicationGuides } from "@/lib/publicationLinks";

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
  const relatedGuides = relatedPublicationGuides("tools", entry.id);
  const firstUse = await loadResourceFirstUse("tools", entry);
  const isReference = entry.usability.availability === "template-only";
  const { connected_system, access_level, capabilities, trust_boundary, approval_boundary, auth_required } = entry.operational ?? {};
  const access = access_level === "read-write" ? "Read and write"
    : access_level === "read-only" ? "Read only"
    : access_level;
  const accessSummary = [access, connected_system].filter(Boolean).join(" · ");

  return (
    <main id="main" tabIndex={-1} className="detail-page">
      <nav className="breadcrumb" aria-label="Breadcrumb"><a href="/tools-mcp">All Tools &amp; MCP</a><span aria-hidden="true">/</span><span>{entry.id}</span></nav>

      <article className="card detail-lead">
        <p className="meta">{formatCategoryLabel(entry.category)}</p>
        <h1>{entry.name}</h1>
        <p>{entry.description}</p>
        {accessSummary ? (
          <div className="detail-purpose">
            <h2>{isReference ? "Intended access" : "Access"}</h2>
            <p>{accessSummary}</p>
          </div>
        ) : null}
        <div className="detail-use-summary"><UsabilityBadge availability={entry.usability.availability} /><span className="meta">{resourceFormLabels[entry.usability.execution]}</span><a href="#usability" className="text-link">Requirements &amp; limits</a>{relatedGuides.length ? <a href="#related-reading" className="text-link">Related reading</a> : null}</div>
        <ResourceFirstUse entry={entry} content={firstUse}>
          {isReference ? (
            <DetailDisclosure show="Read reference guide" hide="Hide reference guide" detail={`TOOL.md · v${entry.latest}`}>
              {firstUse.files?.get("TOOL.md") ? <InstructionMarkdown source={firstUse.files.get("TOOL.md")!} /> : <p>Reference guide preview is unavailable here. Open the release package for the full content.</p>}
            </DetailDisclosure>
          ) : <a className="button" href="#usability">Review setup requirements</a>}
        </ResourceFirstUse>
        <DetailDisclosure show="View capabilities and access" hide="Hide capabilities and access" detail={`v${entry.latest}`}>
          <section>
            <h2>{isReference ? "Intended capabilities" : "What it can do"}</h2>
            {capabilities?.length ? (
              <ul>{capabilities.map((capability) => <li key={capability}>{capability}</li>)}</ul>
            ) : <p>Capabilities are not documented for this release.</p>}
          </section>
          {approval_boundary || trust_boundary ? (
            <section>
              <h2>Access limits</h2>
              {approval_boundary ? <p>{approval_boundary}</p> : null}
              {trust_boundary ? <p className="meta">Trust boundary · {trust_boundary}</p> : null}
            </section>
          ) : null}
          {auth_required?.length ? (
            <section>
              <h2>Credentials and setup</h2>
              <ul>{auth_required.map((requirement) => <li key={requirement}>{requirement}</li>)}</ul>
            </section>
          ) : null}
        </DetailDisclosure>
        <InstallationDetails title={isReference ? "Reference template" : "Install tool files"}>
          <InstallCommands
            compact
            availability={entry.usability.availability}
            execution={entry.usability.execution}
            codex={buildModuleInstallSnippet("tools", entry, "codex")}
            claude={buildModuleInstallSnippet("tools", entry, "claude")}
            generic={buildModuleInstallSnippet("tools", entry, "generic")}
          />
        </InstallationDetails>
      </article>

      <section className="detail-grid resource-evidence">
        <UsabilityPanel catalogueCompatibility={entry.catalogueCompatibility} compatibility={entry.compatibility} usability={entry.usability} />
        <RelatedReading guides={relatedGuides} module="tools" id={entry.id} />
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
            {entry.replaced_by ? <p><span className="meta">Replaced by:</span> {entry.replaced_by}</p> : null}
          </article>
          <article className="card detail-panel">
            <h2>Dependencies &amp; Setup</h2>
            <p><span className="meta">MCP servers:</span> {entry.dependencies?.mcp_servers?.length ? entry.dependencies.mcp_servers.join(", ") : "Not documented"}</p>
            <p><span className="meta">APIs:</span> {entry.dependencies?.apis?.length ? entry.dependencies.apis.join(", ") : "Not documented"}</p>
            <p><span className="meta">Local tools:</span> {entry.dependencies?.tools?.length ? entry.dependencies.tools.join(", ") : "Not documented"}</p>
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
