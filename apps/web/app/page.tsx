import Link from "next/link";
import { loadRegistry, uniqueValues } from "@/lib/registry";

export default async function HomePage() {
  const registry = await loadRegistry();
  const skills = registry.skills;
  const categories = uniqueValues(skills.map((s) => s.category));
  const tags = uniqueValues(skills.flatMap((s) => s.tags));

  return (
    <main>
      <div className="nav">
        <span className="pill">AI Knowledge Hub</span>
        <Link href="/skills" className="pill">Browse Skills</Link>
      </div>

      <section className="hero">
        <article className="card">
          <span className="kicker">Open Skill Registry</span>
          <h1 className="display">
            Build
            <br />
            agents
            <br />
            with <span className="accent">real skills.</span>
          </h1>
          <p>
            Discover reusable AI skill packages, copy runtime-specific install
            commands, and contribute new workflows via pull requests.
          </p>
          <div className="actions">
            <Link href="/skills" className="button button--accent">
              Explore catalog
            </Link>
            <a
              href="https://github.com/ai-knowledge-hub/ai-skills-guide"
              className="button button--secondary"
            >
              View repository
            </a>
          </div>
          <div className="tags">
            <span className="tag">{skills.length} skills</span>
            <span className="tag">Registry v{registry.registry_version}</span>
            <span className="tag">Generated {registry.generated_at.slice(0, 10)}</span>
          </div>
        </article>
        <article className="card">
          <h2>Catalog Snapshot</h2>
          <p><span className="meta">Categories:</span> {categories.length}</p>
          <p><span className="meta">Tags:</span> {tags.length}</p>
          <p><span className="meta">Runtimes:</span> codex, claude, generic</p>
          <div className="actions">
            <Link href="/skills" className="button button--secondary">Open full catalog</Link>
          </div>
        </article>
      </section>

      <section className="grid">
        {skills.slice(0, 6).map((skill) => (
          <Link key={skill.id} href={`/skills/${skill.id}`} className="card">
            <p className="meta">{skill.category}</p>
            <h3>{skill.name}</h3>
            <p>{skill.description}</p>
            <div className="tags">
              {skill.tags.slice(0, 3).map((tag) => (
                <span key={tag} className="tag">{tag}</span>
              ))}
            </div>
          </Link>
        ))}
      </section>
    </main>
  );
}
