import { familySites } from "@/lib/brand";
import type { ModuleKey } from "@/lib/registry";

export type CompanionResource = {
  module: ModuleKey;
  id: string;
  context: string;
  version?: string;
};

export type CompanionGuide = {
  site: "publication" | "harness";
  path: string;
  title: string;
  topic: string;
  summary: string;
  review: { reviewedAt: string; source: string };
  resources: readonly CompanionResource[];
};

// Maintainer-curated reading relationships, reviewed 1 November 2026.
// These are editorial connections, independent of runtime readiness/version notes.
// The Harness companion was reviewed against its current page on 12 November.
// Keep release-specific examples pinned; a reading link is not execution evidence.
export const companionGuides: readonly CompanionGuide[] = [
  {
    site: "publication",
    path: "/library/agent-architect-playbook-building-ai-skills-marketing-adtech/",
    title: "The Agent Architect’s Playbook: Designing Marketing Work and Reusable Skills",
    topic: "Design the workflow",
    summary: "Define the outcome, evidence and procedure. Explore a weekly-review skill and a scorecard for assessing its outputs.",
    review: { reviewedAt: "2026-11-01", source: "https://ai-knowledge-hub.org/library/agent-architect-playbook-building-ai-skills-marketing-adtech/" },
    resources: [
      { module: "skills", id: "marketing/meta-google-weekly-performance-review", context: "A paid-media review procedure to adapt alongside the guide’s weekly-reporting example." },
      { module: "skills", id: "marketing/ai-output-eval-scorecard", context: "A framework for reviewing the accuracy, clarity and usefulness of generated work." }
    ]
  },
  {
    site: "publication",
    path: "/library/muscles-of-the-machine-tools-mcp-cli-first-principles-marketing/",
    title: "The Muscles of the Machine: Designing the Marketing Agent’s Tool Layer",
    topic: "Connect the tools",
    summary: "Make data access explicit and bounded. Inspect two read-only MCP implementations with their own setup requirements.",
    review: { reviewedAt: "2026-11-01", source: "https://ai-knowledge-hub.org/library/muscles-of-the-machine-tools-mcp-cli-first-principles-marketing/" },
    resources: [
      { module: "tools", id: "analytics/ga4-mcp-connector", context: "A related implementation of scoped, bounded analytics reporting; it is not the guide’s illustrative marketing-report command." },
      { module: "tools", id: "warehouse/bigquery-mcp-query-runner", context: "A related query implementation with explicit scope, parameters and cost limits; deployment needs separate configuration." }
    ]
  },
  {
    site: "publication",
    path: "/library/building-secure-marketing-agents-ad-platform-authentication-architecture/",
    title: "Secure Marketing Agents: A Reference Architecture for DV360 and Google Ads Automation",
    topic: "Bound the authority",
    summary: "Separate proposals, permission and execution. These skills and the executor are named companions in the article.",
    review: { reviewedAt: "2026-11-01", source: "https://ai-knowledge-hub.org/library/building-secure-marketing-agents-ad-platform-authentication-architecture/" },
    resources: [
      { module: "skills", id: "security/ad-platform-agent-auth-review", context: "Review credential handling, account scope and the separation of read and write access." },
      { module: "skills", id: "agentops/ad-platform-policy-gate-designer", context: "Specify approval responsibilities, action limits, deny rules and audit evidence." },
      { module: "tools", id: "adtech/ad-platform-executor-template", context: "Inspect the executor’s grant checks, state preconditions and receipts. Reading the architecture does not authorize live changes." }
    ]
  },
  {
    site: "harness",
    path: "/learn/ai-engineering/verify-small-code-change",
    title: "Verify a small code change with an agent",
    topic: "Verify a code change",
    summary: "Plan a small repair, check broken and working inputs, and record the evidence before accepting the change.",
    review: { reviewedAt: "2026-11-12", source: "https://harness.ai-knowledge-hub.org/learn/ai-engineering/verify-small-code-change" },
    resources: [
      { module: "plugins", id: "engineering/code-maintenance-plugin", version: "0.2.0", context: "Use the guide’s before-and-after checks to review a repair. Its Code Maintenance 0.2.0 example prints a plan; it does not run the checks or authorize a merge." }
    ]
  }
];

export function companionGuideUrl(guide: Pick<CompanionGuide, "site" | "path">) {
  return new URL(guide.path, familySites[guide.site].url).href;
}

export function companionResourceHref(resource: Pick<CompanionResource, "module" | "id">) {
  return `/${resource.module === "tools" ? "tools-mcp" : resource.module}/${resource.id}`;
}

export function matchesCompanionResource(resource: CompanionResource, module: ModuleKey, id: string, version: string) {
  return resource.module === module && resource.id === id && (resource.version === undefined || resource.version === version);
}

export function relatedGuidesForResource(module: ModuleKey, id: string, version: string) {
  return companionGuides.flatMap(guide => {
    const resource = guide.resources.find(resource => matchesCompanionResource(resource, module, id, version));
    return resource ? [{ ...guide, context: resource.context }] : [];
  });
}
