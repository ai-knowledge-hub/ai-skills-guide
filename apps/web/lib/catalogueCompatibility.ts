import type { ModuleKey } from "./registry";

// Editorial annotations are tied to a release, never runtime admission.
export type CatalogueCompatibility = {
  module: ModuleKey;
  id: string;
  version: string;
  reviewed: string;
  details: string[];
  limitations: string[];
  source: { label: string; url: string };
};

// Update these details through a normal reviewed change. A date records the
// editorial review; it does not assert a live provider or credential check.
export const catalogueCompatibility: CatalogueCompatibility[] = [
  {
    module: "tools",
    id: "analytics/ga4-mcp-connector",
    version: "0.2.0",
    reviewed: "2026-10-31",
    details: [
      "Google Analytics Data API: v1beta",
      "Runtime: Node.js 22 or newer",
      "Uses Node's built-in HTTP functionality; no Google SDK is packaged."
    ],
    limitations: ["Requires a configured GA4 property and read-only credentials."],
    source: {
      label: "Packaged connector manifest",
      url: "https://github.com/ai-knowledge-hub/ai-skills-guide/blob/278121a55d86a88fecf1bea0adf8a75cde210a6e/releases/tools-mcp/analytics/ga4-mcp-connector/0.2.0/tool.yaml"
    }
  }
];

export function findCatalogueCompatibility(module: ModuleKey, id: string, version: string): CatalogueCompatibility | undefined {
  return catalogueCompatibility.find((item) => item.module === module && item.id === id && item.version === version);
}
