import Image from "next/image";
import { loadAgentsRegistry, loadPluginsRegistry, loadSkillsRegistry, loadToolsRegistry } from "@/lib/registry";
import { FEATURED_PLUGIN_IDS, FEATURED_SKILL_IDS } from "@/lib/home";
import { formatCategoryFamily } from "@/lib/categoryLabels";
import { brand } from "@/lib/brand";
import UsabilityBadge from "@/components/UsabilityBadge";

export default async function HomePage() {
  const [skillsRegistry, agentsRegistry, toolsRegistry, pluginsRegistry] = await Promise.all([
    loadSkillsRegistry(), loadAgentsRegistry(), loadToolsRegistry(), loadPluginsRegistry()
  ]);
  const skills = skillsRegistry.skills;
  const plugins = pluginsRegistry.skills;
  const featuredSkills = FEATURED_SKILL_IDS.map(id => skills.find(skill => skill.id === id))
    .filter((skill): skill is (typeof skills)[number] => Boolean(skill));
  const featuredPlugins = FEATURED_PLUGIN_IDS.map(id => plugins.find(plugin => plugin.id === id))
    .filter((plugin): plugin is (typeof plugins)[number] => Boolean(plugin));
  const newestSkills = [...skills].sort((a, b) => {
    const date = (entry: typeof a) => new Date(entry.versions.find(v => v.version === entry.latest)?.released_at ?? 0).getTime();
    return date(b) - date(a) || a.id.localeCompare(b.id);
  }).slice(0, 3);
  const modules = [
    { href: "/skills", name: "Skills", count: skills.length, description: "Task-level instructions and local tools for recurring work." },
    { href: "/agents", name: "Agents", count: agentsRegistry.skills.length, description: "Orchestrated templates that connect roles, skills and tools." },
    { href: "/plugins", name: "Plugins", count: plugins.length, description: "Packaged capabilities, hooks and setup guidance." },
    { href: "/tools-mcp", name: "Tools & MCP", count: toolsRegistry.skills.length, description: "Integrations and adapters for connecting to other systems." }
  ];

  return (
    <main id="main" tabIndex={-1} className="home-main">
      <section className="home-entrance" aria-labelledby="home-title">
        <picture>
          <source srcSet="/brand/entrance-480.webp 480w, /brand/entrance-800.webp 800w, /brand/entrance-1600.webp 1600w" sizes="100vw" />
          <Image className="entrance-image" src="/brand/entrance-1600.webp" alt="" width={1600} height={903} unoptimized loading="eager" fetchPriority="high" />
        </picture>
        <div className="entrance-sign">
          <p className="eyebrow">The open-source catalogue</p>
          <h1 id="home-title">AI KNOWLEDGE<br />HUB</h1>
          <p>{brand.descriptor}</p>
          <a href="/skills" className="button">Explore skills</a>
        </div>
      </section>

      <section className="home-section home-intro wrap" aria-labelledby="intro-title">
        <span className="eyebrow">Inspect. Adapt. Build.</span>
        <div><h2 id="intro-title">Find the right<br />building block.</h2>
          <p>Reusable work for marketing, engineering, security and agent operations. Explore the source, check what each package needs, and choose how to use it.</p>
        </div>
      </section>
      <nav className="module-paths wrap" aria-label="Browse the catalogue">
        {modules.map((module, index) => (
          <a className="module-path" href={module.href} key={module.href}>
            <span className="eyebrow">0{index + 1}</span><h3>{module.name}</h3><p>{module.description}</p>
            <span className="path-count">{module.count} entries</span>
          </a>
        ))}
      </nav>

      <section className="featured-section wrap" aria-labelledby="featured-skills">
        <div className="section-head"><h2 id="featured-skills">Featured Skills</h2><a href="/skills" className="text-link">View all {skills.length} skills</a></div>
        <div className="featured-grid">
          {featuredSkills.map(skill => (
            <a key={skill.id} href={`/skills/${skill.id}`} className="card catalog-card">
              <div className="catalog-card-head"><span className="meta catalog-card-domain">{formatCategoryFamily(skill.category)}</span><UsabilityBadge availability={skill.usability.availability} /></div>
              <h3>{skill.name}</h3><p>{skill.description}</p>
            </a>
          ))}
        </div>
      </section>
      <section className="featured-section wrap" aria-labelledby="featured-plugins">
        <div className="section-head"><h2 id="featured-plugins">Featured Plugins</h2><a href="/plugins" className="text-link">View all {plugins.length} plugins</a></div>
        <div className="featured-grid">
          {featuredPlugins.map(plugin => (
            <a key={plugin.id} href={`/plugins/${plugin.id}`} className="card catalog-card">
              <div className="catalog-card-head"><span className="meta catalog-card-domain">{formatCategoryFamily(plugin.category)}</span><UsabilityBadge availability={plugin.usability.availability} /></div>
              <h3>{plugin.name}</h3><p>{plugin.description}</p>
            </a>
          ))}
        </div>
      </section>
      <section className="usability-guide" aria-labelledby="use-the-catalog">
        <div className="home-section wrap">
          <div className="section-head"><h2 id="use-the-catalog">Know what you are installing</h2></div>
          <p>Review maturity and operational usability are separate signals.</p>
          <div className="usability-guide-grid">
            <article><UsabilityBadge availability="usable-now" /><p>Instructions or a local executable you can use after installation.</p></article>
            <article><UsabilityBadge availability="setup-required" /><p>An integration or orchestrator that needs credentials, bindings or local policy.</p></article>
            <article><UsabilityBadge availability="template-only" /><p>A reference contract or scaffold to adapt before it can execute.</p></article>
            <article><UsabilityBadge availability="documentation-only" /><p>A learning pack or architecture guide, outside the installable runtime capabilities.</p></article>
          </div>
        </div>
      </section>
      <section className="home-recent wrap" aria-labelledby="recent-title">
        <h2 id="recent-title">Recently added</h2><ul>{newestSkills.map(skill => <li key={skill.id}><a href={`/skills/${skill.id}`}>{skill.name}</a></li>)}</ul>
      </section>
    </main>
  );
}
