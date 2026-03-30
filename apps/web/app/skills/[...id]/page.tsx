import { notFound } from "next/navigation";
import { buildInstallSnippet, getSkillById, loadSkillsRegistry } from "@/lib/registry";
import InstallCommands from "@/components/InstallCommands";
import { formatCategoryLabel, formatReadinessLabel } from "@/lib/categoryLabels";

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
    <main>
      <div className="nav">
        <a href="/" className="pill">Home</a>
        <a href="/skills" className="pill">Catalog</a>
        <a href="/agents" className="pill">Agents</a>
        <a href="/plugins" className="pill">Plugins</a>
        <a href="/tools-mcp" className="pill">Tools &amp; MCP</a>
        <span className="pill">{skill.id}</span>
      </div>

      <article className="card">
        <p className="meta">{formatCategoryLabel(skill.category)}</p>
        <h1>{skill.name}</h1>
        <p>{skill.description}</p>
        <div className="tags">
          {skill.tags.map((tag) => (
            <span key={tag} className="tag">{tag}</span>
          ))}
        </div>
        <div className="detail-install-lead">
          <p className="meta">Install this skill</p>
          <InstallCommands
            compact
            codex={buildInstallSnippet(skill, "codex")}
            claude={buildInstallSnippet(skill, "claude")}
            generic={buildInstallSnippet(skill, "generic")}
          />
        </div>
      </article>

      <section className="detail-grid">
        <article className="card detail-panel">
          <h2>Operational Summary</h2>
          <p><span className="meta">Use when:</span> {skill.operational?.use_when ?? "Not documented"}</p>
          <p><span className="meta">Execution mode:</span> {skill.operational?.execution_mode ?? "Not documented"}</p>
          <p><span className="meta">Approval boundary:</span> {skill.operational?.approval_boundary ?? "Not documented"}</p>
        </article>
        <article className="card detail-panel">
          <h2>Status</h2>
          <p><span className={`catalog-status-badge is-${skill.readiness}`}>{formatReadinessLabel(skill.readiness)}</span></p>
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
