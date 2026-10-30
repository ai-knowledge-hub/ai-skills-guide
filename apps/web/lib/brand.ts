export const brand = {
  name: "AI Knowledge Hub",
  descriptor: "Skills, agents & tools for practical AI work.",
  description: "Open-source skills, agent templates, plugins and tools for marketing, engineering, security and agent operations. Inspect the requirements, choose a runtime and build on the source.",
  publication: "https://ai-knowledge-hub.org",
  repository: "https://github.com/ai-knowledge-hub/ai-skills-guide",
  themeKey: "skills-hub-theme"
} as const;

export const moduleNavigation = [
  { href: "/skills", label: "Skills" },
  { href: "/agents", label: "Agents" },
  { href: "/plugins", label: "Plugins" },
  { href: "/tools-mcp", label: "Tools & MCP" }
] as const;
