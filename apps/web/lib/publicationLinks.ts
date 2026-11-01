import { brand } from "@/lib/brand";

export type PublicationResource = {
  module: "skills" | "tools";
  id: string;
  context: string;
};

export type PublicationGuide = {
  slug: string;
  title: string;
  topic: string;
  summary: string;
  resources: readonly PublicationResource[];
};

// Maintainer-curated reading relationships, reviewed 1 November 2026.
// These are editorial connections, independent of runtime readiness/version notes.
export const publicationGuides: readonly PublicationGuide[] = [
  {
    slug: "agent-architect-playbook-building-ai-skills-marketing-adtech",
    title: "The Agent Architect’s Playbook: Designing Marketing Work and Reusable Skills",
    topic: "Design the workflow",
    summary: "Define the outcome, evidence and procedure. Explore a weekly-review skill and a scorecard for assessing its outputs.",
    resources: [
      { module: "skills", id: "marketing/meta-google-weekly-performance-review", context: "A paid-media review procedure to adapt alongside the guide’s weekly-reporting example." },
      { module: "skills", id: "marketing/ai-output-eval-scorecard", context: "A framework for reviewing the accuracy, clarity and usefulness of generated work." }
    ]
  },
  {
    slug: "muscles-of-the-machine-tools-mcp-cli-first-principles-marketing",
    title: "The Muscles of the Machine: Designing the Marketing Agent’s Tool Layer",
    topic: "Connect the tools",
    summary: "Make data access explicit and bounded. Inspect two read-only MCP implementations with their own setup requirements.",
    resources: [
      { module: "tools", id: "analytics/ga4-mcp-connector", context: "A related implementation of scoped, bounded analytics reporting; it is not the guide’s illustrative marketing-report command." },
      { module: "tools", id: "warehouse/bigquery-mcp-query-runner", context: "A related query implementation with explicit scope, parameters and cost limits; deployment needs separate configuration." }
    ]
  },
  {
    slug: "building-secure-marketing-agents-ad-platform-authentication-architecture",
    title: "Secure Marketing Agents: A Reference Architecture for DV360 and Google Ads Automation",
    topic: "Bound the authority",
    summary: "Separate proposals, permission and execution. These skills and the executor are named companions in the article.",
    resources: [
      { module: "skills", id: "security/ad-platform-agent-auth-review", context: "Review credential handling, account scope and the separation of read and write access." },
      { module: "skills", id: "agentops/ad-platform-policy-gate-designer", context: "Specify approval responsibilities, action limits, deny rules and audit evidence." },
      { module: "tools", id: "adtech/ad-platform-executor-template", context: "Inspect the executor’s grant checks, state preconditions and receipts. Reading the architecture does not authorize live changes." }
    ]
  }
];

export function publicationGuideUrl(guide: PublicationGuide) {
  return `${brand.publication}/library/${guide.slug}/`;
}

export function publicationResourceHref(resource: Pick<PublicationResource, "module" | "id">) {
  return `${resource.module === "tools" ? "/tools-mcp" : "/skills"}/${resource.id}`;
}

export function relatedPublicationGuides(module: PublicationResource["module"], id: string) {
  return publicationGuides.filter(guide => guide.resources.some(resource => resource.module === module && resource.id === id));
}
