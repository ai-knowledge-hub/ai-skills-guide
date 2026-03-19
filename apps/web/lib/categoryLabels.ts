const CATEGORY_LABELS: Record<string, string> = {
  "marketing-tools/ads-ops": "Marketing Tools / Ads Ops",
  "marketing-tools/creative-ops": "Marketing Tools / Creative Ops",
  "marketing-tools/measurement": "Marketing Tools / Measurement",
  "marketing-tools/seo-sem": "Marketing Tools / SEO & SEM",
  "marketing-tools/lifecycle-crm": "Marketing Tools / Lifecycle CRM",
  "marketing-tools/compliance": "Marketing Tools / Compliance",
  "adtech/analytics-engineering": "Adtech / Analytics Engineering",
  "adtech/compliance": "Adtech / Compliance",
  "adtech/activation": "Adtech / Activation",
  "adtech/other": "Adtech / Other",
  "engineering/code-maintenance": "Engineering / Code Maintenance",
  "engineering/testing-quality": "Engineering / Testing Quality",
  "engineering/pr-review": "Engineering / PR Review",
  "security/prompt-safety": "Security / Prompt Safety",
  "security/supply-chain": "Security / Supply Chain",
  "security/runtime-hardening": "Security / Runtime Hardening",
  "agentops/harness-evolution": "AgentOps / Harness Evolution",
  "agentops/harness-governance": "AgentOps / Harness Governance",
  "agentops/harness-evaluation": "AgentOps / Harness Evaluation",
  "marketing-agents/performance": "Marketing Agents / Performance",
  "marketing-agents/governance": "Marketing Agents / Governance",
  "adtech-agents/analytics": "Adtech Agents / Analytics",
  "tools-mcp/analytics": "Tools & MCP / Analytics",
  "tools-mcp/ads": "Tools & MCP / Ads",
  "tools-mcp/warehouse": "Tools & MCP / Warehouse"
};

export function formatCategoryLabel(category: string) {
  if (CATEGORY_LABELS[category]) {
    return CATEGORY_LABELS[category];
  }

  return category
    .split("/")
    .map((part) =>
      part
        .split("-")
        .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
        .join(" ")
    )
    .join(" / ");
}

export function formatCategoryFamily(category: string) {
  const [family] = category.split("/");
  if (family === "marketing-tools") return "Marketing Tools";
  if (family === "adtech") return "Adtech";
  if (family === "engineering") return "Engineering";
  if (family === "security") return "Security";
  if (family === "agentops") return "AgentOps";
  if (family === "marketing-agents") return "Marketing Agents";
  if (family === "adtech-agents") return "Adtech Agents";
  if (family === "tools-mcp") return "Tools & MCP";
  return "Other";
}

export function formatReadinessLabel(readiness: "experimental" | "reviewed" | "deprecated") {
  if (readiness === "reviewed") return "Reviewed";
  if (readiness === "deprecated") return "Deprecated";
  return "Experimental";
}
