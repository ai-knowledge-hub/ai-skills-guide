import Link from "next/link";
import CatalogueArtwork from "@/components/CatalogueArtwork";
import { loadAgentsRegistry, loadPluginsRegistry, loadSkillsRegistry, loadToolsRegistry } from "@/lib/registry";
import { brand, familySites } from "@/lib/brand";
import { companionGuides, companionGuideUrl, companionResourceHref, matchesCompanionResource } from "@/lib/publicationLinks";
import UsabilityBadge from "@/components/UsabilityBadge";

export default async function HomePage() {
  const [skillsRegistry, agentsRegistry, toolsRegistry, pluginsRegistry] = await Promise.all([
    loadSkillsRegistry(), loadAgentsRegistry(), loadToolsRegistry(), loadPluginsRegistry()
  ]);
  const entries = {
    skills: new Map(skillsRegistry.skills.map(entry => [entry.id, entry])),
    tools: new Map(toolsRegistry.skills.map(entry => [entry.id, entry])),
    agents: new Map(agentsRegistry.skills.map(entry => [entry.id, entry])),
    plugins: new Map(pluginsRegistry.skills.map(entry => [entry.id, entry]))
  };
  const guides = companionGuides.map(guide => ({
    ...guide,
    resources: guide.resources.flatMap(resource => {
      const entry = entries[resource.module].get(resource.id);
      if (!entry) throw new Error(`Companion relationship points to missing ${resource.module} entry: ${resource.id}`);
      return matchesCompanionResource(resource, resource.module, entry.id, entry.latest) ? [{ ...resource, entry }] : [];
    })
  })).filter(guide => guide.resources.length);
  const resourceLabels = { skills: "Skill", agents: "Agent", plugins: "Plugin", tools: "Tool" };
  const modules = [
    { href: "/skills", name: "Skills", count: skillsRegistry.skills.length, description: "Reusable procedures for recurring work." },
    { href: "/agents", name: "Agents", count: agentsRegistry.skills.length, description: "Templates for coordinating roles, skills and tools." },
    { href: "/plugins", name: "Plugins", count: pluginsRegistry.skills.length, description: "Installable bundles and reference compositions." },
    { href: "/tools-mcp", name: "Tools & MCP", count: toolsRegistry.skills.length, description: "Integrations with explicit setup and access requirements." }
  ];

  return (
    <main id="main" tabIndex={-1} className="catalogue-main companion-main">
      <CatalogueArtwork />
      <div className="companion-content">
        <header className="companion-intro">
          <p className="eyebrow">Skills · The open-source catalogue</p>
          <h1>Skills for<br />practical AI work.</h1>
          <p>{brand.descriptor} Inspect the source, adapt a resource or configure it for your own work. Check its requirements and evidence before you use it.</p>
          <div className="actions">
            <Link href="/skills" className="button">Explore skills</Link>
          </div>
        </header>

        <nav className="companion-modules" aria-label="Browse the catalogue">
          {modules.map((module, index) => (
            <Link className="companion-module" href={module.href} key={module.href}>
              <span className="meta module-number" aria-hidden="true">0{index + 1}</span>
              <h2>{module.name}</h2>
              <p>{module.description}</p>
              <span className="meta path-count">{module.count} entries</span>
            </Link>
          ))}
        </nav>

        <section className="publication-paths" aria-labelledby="publication-paths-title">
          <div className="section-head">
            <h2 id="publication-paths-title">From reading to practice</h2>
            <a href={`${brand.publication}/library/`} className="text-link">Publication library</a>
          </div>
          <p>Optional reading and learning guides with related resources to inspect and adapt. Each resource keeps its own setup requirements and limits.</p>
          {guides.map(guide => (
            <article className="publication-path" key={companionGuideUrl(guide)}>
              <div className="publication-path-reading">
                <p className="eyebrow">{familySites[guide.site].label} · {guide.topic}</p>
                <h3><a href={companionGuideUrl(guide)}>{guide.title}</a></h3>
                <p>{guide.summary}</p>
              </div>
              <ul className="publication-path-resources" aria-label={`Resources for ${guide.topic.toLowerCase()}`}>
                {guide.resources.map(resource => (
                  <li key={`${resource.module}:${resource.id}`}>
                    <Link href={companionResourceHref(resource)}>
                      <span className="meta">{resourceLabels[resource.module]}</span>
                      <span className="publication-resource-name">{resource.entry.name}</span>
                      <UsabilityBadge availability={resource.entry.usability.availability} />
                    </Link>
                  </li>
                ))}
              </ul>
            </article>
          ))}
        </section>

        <section className="companion-use-guide" aria-labelledby="use-the-catalog">
          <h2 id="use-the-catalog">Know what you are installing</h2>
          <p>Review maturity and operational usability are separate signals. Check each resource’s requirements and limits before using it.</p>
          <dl>
            <div><dt><UsabilityBadge availability="documentation-only" /></dt><dd>Instructions or guidance. Packaged executable helpers have their own status.</dd></div>
            <div><dt><UsabilityBadge availability="template-only" /></dt><dd>A reference to adapt before it can execute.</dd></div>
            <div><dt><UsabilityBadge availability="not-verified" /></dt><dd>Current operational evidence for your environment is not shown. Inspect implementation and installation details.</dd></div>
            <div><dt><UsabilityBadge availability="setup-required" /></dt><dd>Credentials, bindings or local policy are required.</dd></div>
            <div><dt><UsabilityBadge availability="usable-now" /></dt><dd>Read the entry for the exact scope supported by its evidence.</dd></div>
          </dl>
        </section>
      </div>
    </main>
  );
}
