import type { Metadata } from "next";
import UsabilityBadge from "@/components/UsabilityBadge";
import { notFound } from "next/navigation";
import { buildInstallSnippet, getSkillById, loadSkillsRegistry } from "@/lib/registry";
import InstallCommands from "@/components/InstallCommands";
import { formatCategoryLabel, formatReadinessLabel } from "@/lib/categoryLabels";
import UsabilityPanel from "@/components/UsabilityPanel";

export async function generateMetadata({ params }: { params: { id: string[] } }): Promise<Metadata> {
  const entry = await getSkillById(params.id.join("/"));
  return { title: entry?.name ?? "Entry not found", description: entry?.description };
}

export async function generateStaticParams() {
  const registry = await loadSkillsRegistry();
  return registry.skills.map((skill) => ({ id: skill.id.split("/") }));
}

export default async function SkillDetailPage({ params }: { params: { id: string[] } }) {
  const skillId = params.id.join("/");
  const skill = await getSkillById(skillId);
  if (!skill) {
    notFound();
  }

  return (
    <main id="main" tabIndex={-1} className="detail-page">
      <nav className="breadcrumb" aria-label="Breadcrumb"><a href="/skills">All Skills</a><span aria-hidden="true">/</span><span>{skill.id}</span></nav>

      <article className="card detail-lead">
        <p className="meta">{formatCategoryLabel(skill.category)}</p>
        <h1>{skill.name}</h1>
        <p>{skill.description}</p>
        <p className="meta">{skill.tags.join(", ")}</p>
        <div className="detail-use-summary"><UsabilityBadge availability={skill.usability.availability} /><span className="meta">{skill.usability.execution}</span><a href="#usability" className="text-link">Requirements &amp; limits</a></div>
        <div className="detail-install-lead">
          <h2 className="meta">Install this skill</h2>
          <InstallCommands
            compact
            codex={buildInstallSnippet(skill, "codex")}
            claude={buildInstallSnippet(skill, "claude")}
            generic={buildInstallSnippet(skill, "generic")}
          />
        </div>
      </article>

      <section className="detail-grid">
        <UsabilityPanel usability={skill.usability} />
        <article className="card detail-panel">
          <h2>Operational Summary</h2>
          <p><span className="meta">Use when:</span> {skill.operational?.use_when ?? "Not documented"}</p>
          <p><span className="meta">Execution mode:</span> {skill.operational?.execution_mode ?? "Not documented"}</p>
          <p><span className="meta">Approval boundary:</span> {skill.operational?.approval_boundary ?? "Not documented"}</p>
        </article>
        <article className="card detail-panel">
          <h2>Status</h2>
          <p><span className="meta">Readiness:</span> {formatReadinessLabel(skill.readiness)}</p>
          <p><span className="meta">Security reviewed:</span> {skill.security_reviewed ? "yes" : "no"}</p>
          <p><span className="meta">Lifecycle:</span> {skill.deprecated ? "Deprecated" : "Active"}</p>
        </article>
        <article className="card detail-panel">
          <h2>Runtime &amp; Dependencies</h2>
          <p><span className="meta">ID:</span> {skill.id}</p>
          <p><span className="meta">Runtimes:</span> {skill.runtimes.join(", ")}</p>
          <p><span className="meta">Tool dependencies:</span> {skill.dependencies?.tools?.length ?? 0}</p>
          <p><span className="meta">API dependencies:</span> {skill.dependencies?.apis?.length ?? 0}</p>
          {skill.replaced_by ? <p><span className="meta">Replaced by:</span> {skill.replaced_by}</p> : null}
        </article>
        <article className="card detail-panel">
          <h2>Dependencies</h2>
          <p><span className="meta">Tools:</span> {skill.dependencies?.tools?.length ? skill.dependencies.tools.join(", ") : "Not documented"}</p>
          <p><span className="meta">APIs:</span> {skill.dependencies?.apis?.length ? skill.dependencies.apis.join(", ") : "Not documented"}</p>
        </article>
        <article className="card detail-panel">
          <h2>Outputs</h2>
          {skill.operational?.outputs?.length ? (
            <ul>
              {skill.operational.outputs.map((item) => (
                <li key={item}>{item}</li>
              ))}
            </ul>
          ) : (
            <p>Not documented.</p>
          )}
        </article>
      </section>
    </main>
  );
}
