import Link from "next/link";
import { loadAgentsRegistry, loadSkillsRegistry, loadToolsRegistry, uniqueValues } from "@/lib/registry";
import { FEATURED_SKILL_IDS } from "@/lib/home";
import {
  formatCategoryFamily,
  formatCategoryLabel,
  formatReadinessLabel
} from "@/lib/categoryLabels";

export default async function HomePage() {
  const [skillsRegistry, agentsRegistry, toolsRegistry] = await Promise.all([
    loadSkillsRegistry(),
    loadAgentsRegistry(),
    loadToolsRegistry()
  ]);
  const skills = skillsRegistry.skills;
  const agents = agentsRegistry.skills;
  const tools = toolsRegistry.skills;
  const categories = uniqueValues(skills.map((s) => s.category));
  const tags = uniqueValues(skills.flatMap((s) => s.tags));
  const featuredSkills = FEATURED_SKILL_IDS
    .map((id) => skills.find((skill) => skill.id === id))
    .filter((skill): skill is (typeof skills)[number] => Boolean(skill));
  const newestSkills = [...skills]
    .sort((a, b) => {
      const aDate = new Date(
        a.versions.find((version) => version.version === a.latest)?.released_at ?? 0
      ).getTime();
      const bDate = new Date(
        b.versions.find((version) => version.version === b.latest)?.released_at ?? 0
      ).getTime();

      if (bDate !== aDate) {
        return bDate - aDate;
      }

      return a.id.localeCompare(b.id);
    })
    .slice(0, 3);

  return (
    <main>
      <div className="nav">
        <span className="pill">AI Knowledge Hub</span>
        <Link href="/skills" className="pill">Skills</Link>
        <Link href="/agents" className="pill">Agents</Link>
        <Link href="/tools-mcp" className="pill">Tools &amp; MCP</Link>
      </div>

      <section className="hero">
        <article className="card hero-mission">
          <span className="kicker">Open Skills Infrastructure</span>
          <h1 className="display">
            Build
            <br />
            agents
            <br />
            with <span className="accent">real skills.</span>
          </h1>
          <p>
            AI Knowledge Hub is an open, runtime-agnostic skills platform for
            marketing, adtech, engineering, security, and agent operations.
          </p>
          <p>
            We publish reusable skill packages with guardrails, tests, and
            install paths so teams can stop rebuilding the same automations,
            review loops, and harness policies in silos.
          </p>
        </article>
        <article className="card">
          <h2>Catalog Snapshot</h2>
          <p><span className="meta">Skills categories:</span> {categories.length}</p>
          <p><span className="meta">Skills tags:</span> {tags.length}</p>
          <p><span className="meta">Runtimes:</span> codex, claude, generic</p>
          <div className="actions snapshot-actions">
            <Link href="/skills" className="button button--accent">
              Explore skills
            </Link>
            <Link href="/agents" className="button button--secondary">
              Browse agents
            </Link>
            <Link href="/tools-mcp" className="button button--secondary">
              Browse tools
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
            <span className="tag">{agents.length} agents</span>
            <span className="tag">{tools.length} tools</span>
            <span className="tag">Skills registry v{skillsRegistry.registry_version}</span>
            <span className="tag">Generated {skillsRegistry.generated_at.slice(0, 10)}</span>
          </div>
          <div className="new-alpha">
            <p className="meta">Newest skills</p>
            <ul>
              {newestSkills.map((skill) => (
                <li key={skill.id}>
                  <Link href={`/skills/${skill.id}`}>{skill.name}</Link>
                </li>
              ))}
            </ul>
          </div>
        </article>
      </section>

      <section className="featured-section">
        <div className="section-head">
          <h2>Featured Skills</h2>
          <p className="meta">Sample cards from the full catalog.</p>
          <Link href="/skills" className="button button--secondary">
            View all {skills.length} skills
          </Link>
        </div>
      </section>

      <section className="grid featured-grid">
        {featuredSkills.map((skill) => (
          <Link key={skill.id} href={`/skills/${skill.id}`} className="card catalog-card">
            <div className="catalog-card-head">
              <span className="catalog-pack-badge">{formatCategoryFamily(skill.category)}</span>
              <div className="catalog-status-badges">
                <span className={`catalog-status-badge is-${skill.readiness}`}>
                  {formatReadinessLabel(skill.readiness)}
                </span>
                {skill.security_reviewed ? (
                  <span className="catalog-status-badge is-reviewed-detail">Security</span>
                ) : null}
              </div>
            </div>
            <h3>{skill.name}</h3>
            <p className="meta catalog-card-category">{formatCategoryLabel(skill.category)}</p>
            <p>{skill.description}</p>
            <div className="tags">
              {skill.tags.slice(0, 2).map((tag) => (
                <span key={tag} className="tag">{tag}</span>
              ))}
              {skill.tags.length > 2 ? (
                <span className="tag tag--muted">+{skill.tags.length - 2}</span>
              ) : null}
            </div>
          </Link>
        ))}
      </section>
    </main>
  );
}
