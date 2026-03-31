import { loadAgentsRegistry, loadPluginsRegistry, loadSkillsRegistry, loadToolsRegistry } from "@/lib/registry";
import { FEATURED_PLUGIN_IDS, FEATURED_SKILL_IDS } from "@/lib/home";
import { formatCategoryFamily } from "@/lib/categoryLabels";

export default async function HomePage() {
  const [skillsRegistry, agentsRegistry, toolsRegistry] = await Promise.all([
    loadSkillsRegistry(),
    loadAgentsRegistry(),
    loadToolsRegistry()
  ]);
  const pluginsRegistry = await loadPluginsRegistry();
  const skills = skillsRegistry.skills;
  const agents = agentsRegistry.skills;
  const tools = toolsRegistry.skills;
  const plugins = pluginsRegistry.skills;
  const featuredSkills = FEATURED_SKILL_IDS
    .map((id) => skills.find((skill) => skill.id === id))
    .filter((skill): skill is (typeof skills)[number] => Boolean(skill));
  const featuredPlugins = FEATURED_PLUGIN_IDS
    .map((id) => plugins.find((plugin) => plugin.id === id))
    .filter((plugin): plugin is (typeof plugins)[number] => Boolean(plugin));
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
        <a href="/skills" className="pill">Skills</a>
        <a href="/agents" className="pill">Agents</a>
        <a href="/plugins" className="pill">Plugins</a>
        <a href="/tools-mcp" className="pill">Tools &amp; MCP</a>
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
          <h2>Catalog</h2>
          <p className="meta catalog-counts">{skills.length} skills &middot; {agents.length} agents &middot; {plugins.length} plugins &middot; {tools.length} tools</p>
          <div className="actions snapshot-actions">
            <a href="/skills" className="button button--accent">
              Explore skills
            </a>
            <a href="/agents" className="button button--secondary">
              Browse agents
            </a>
            <a href="/plugins" className="button button--secondary">
              Browse plugins
            </a>
            <a href="/tools-mcp" className="button button--secondary">
              Browse tools
            </a>
          </div>
          <div className="new-alpha">
            <p className="meta">Recently added</p>
            <ul>
              {newestSkills.map((skill) => (
                <li key={skill.id}>
                  <a href={`/skills/${skill.id}`}>{skill.name}</a>
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
          <a href="/skills" className="button button--secondary">
            View all {skills.length} skills
          </a>
        </div>
      </section>

      <section className="grid featured-grid">
        {featuredSkills.map((skill) => (
          <a key={skill.id} href={`/skills/${skill.id}`} className="card catalog-card">
            <p className="meta catalog-card-domain">{formatCategoryFamily(skill.category)}</p>
            <h3>{skill.name}</h3>
            <p>{skill.description}</p>
          </a>
        ))}
      </section>

      <section className="featured-section">
        <div className="section-head">
          <h2>Featured Plugins</h2>
          <p className="meta">New non-marketing plugin packs for maintenance, security, and governed harness improvement.</p>
          <a href="/plugins" className="button button--secondary">
            View all {plugins.length} plugins
          </a>
        </div>
      </section>

      <section className="grid featured-grid">
        {featuredPlugins.map((plugin) => (
          <a key={plugin.id} href={`/plugins/${plugin.id}`} className="card catalog-card">
            <p className="meta catalog-card-domain">{formatCategoryFamily(plugin.category)}</p>
            <h3>{plugin.name}</h3>
            <p>{plugin.description}</p>
          </a>
        ))}
      </section>
    </main>
  );
}
