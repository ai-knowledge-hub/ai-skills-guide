// Publication owns the shared roles and canonical family destinations.
// Checked against its public home page on 12 November 2026. Keep this local:
// browsing Skills must not depend on either sibling site being available.
export const familySites = {
  publication: { label: "Publication", url: "https://ai-knowledge-hub.org/", role: "Independent research, explanations and practical experiments." },
  skills: { label: "Skills", url: "https://skills.ai-knowledge-hub.org/", role: "Reusable instructions, templates, packages and integrations." },
  harness: { label: "Harness", url: "https://harness.ai-knowledge-hub.org/", role: "Engineering guides, visualizers and worked examples." }
} as const;

export const brand = {
  name: "AI Knowledge Hub",
  siteName: familySites.skills.label,
  descriptor: familySites.skills.role,
  description: "Open-source skills, agent templates, plugins and tools for marketing, engineering, security and agent operations. Inspect the requirements, choose a runtime and build on the source.",
  publication: familySites.publication.url.replace(/\/$/, ""),
  repository: "https://github.com/ai-knowledge-hub/ai-skills-guide",
  themeKey: "skills-hub-theme"
} as const;

export const moduleNavigation = [
  { href: "/skills", label: "Skills" },
  { href: "/agents", label: "Agents" },
  { href: "/plugins", label: "Plugins" },
  { href: "/tools-mcp", label: "Tools & MCP" }
] as const;
