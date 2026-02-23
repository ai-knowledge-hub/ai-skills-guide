import { notFound } from "next/navigation";
import Link from "next/link";
import { buildInstallSnippet, getSkillById, loadRegistry } from "@/lib/registry";
import InstallCommands from "@/components/InstallCommands";

export async function generateStaticParams() {
  const registry = await loadRegistry();
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
        <Link href="/" className="pill">Home</Link>
        <Link href="/skills" className="pill">Catalog</Link>
        <span className="pill">{skill.id}</span>
      </div>

      <article className="card">
        <p className="meta">{skill.category}</p>
        <h1>{skill.name}</h1>
        <p>{skill.description}</p>
        <div className="tags">
          {skill.tags.map((tag) => (
            <span key={tag} className="tag">{tag}</span>
          ))}
        </div>
      </article>

      <section className="detail-grid">
        <article className="card detail-panel">
          <h2>Metadata</h2>
          <p><span className="meta">ID:</span> {skill.id}</p>
          <p><span className="meta">Latest:</span> {skill.latest}</p>
          <p><span className="meta">Runtimes:</span> {skill.runtimes.join(", ")}</p>
          <p><span className="meta">Deprecated:</span> {String(skill.deprecated)}</p>
          {skill.replaced_by ? <p><span className="meta">Replaced by:</span> {skill.replaced_by}</p> : null}
        </article>
        <InstallCommands
          codex={buildInstallSnippet(skill, "codex")}
          claude={buildInstallSnippet(skill, "claude")}
          generic={buildInstallSnippet(skill, "generic")}
        />
      </section>
    </main>
  );
}
