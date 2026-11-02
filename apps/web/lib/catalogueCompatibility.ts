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
  },
  {
    module: "tools",
    id: "warehouse/bigquery-mcp-query-runner",
    version: "0.2.0",
    reviewed: "2026-10-31",
    details: [
      "Packaged API target: Google BigQuery API v2",
      "Runtime: Node.js 22 or newer",
      "Uses Node's built-in HTTP functionality; no Google SDK is packaged."
    ],
    limitations: [
      "Requires BigQuery credentials and a policy defining allowed projects, datasets, locations and query budgets.",
      "Accepts one read-only SELECT or WITH GoogleSQL query; scripts, DML and DDL are rejected."
    ],
    source: {
      label: "Reviewed BigQuery release implementation",
      url: "https://github.com/ai-knowledge-hub/ai-skills-guide/tree/bdc8eccba177fe9da6ab38e23da88dd3869500a9/tools-mcp/warehouse/bigquery-mcp-query-runner"
    }
  },
  {
    module: "tools",
    id: "ads/meta-ads-mcp-connector",
    version: "0.2.0",
    reviewed: "2026-10-31",
    details: [
      "Packaged API target: Meta Marketing / Graph API v24.0",
      "Runtime: Node.js 22 or newer",
      "Uses Node's built-in HTTP functionality; no Meta SDK is packaged."
    ],
    limitations: [
      "This release requires graph_api_version v24.0; other versions are rejected.",
      "Read-only Insights reporting requires app credentials, a token with ads_read and business_management permissions, and allowed business and ad-account IDs.",
      "Reporting fields, breakdowns and request budgets are restricted by the package and configured policy."
    ],
    source: {
      label: "Reviewed Meta Ads release implementation",
      url: "https://github.com/ai-knowledge-hub/ai-skills-guide/tree/bdc8eccba177fe9da6ab38e23da88dd3869500a9/tools-mcp/ads/meta-ads-mcp-connector"
    }
  },
  {
    module: "tools",
    id: "adtech/openai-ads-api-client",
    version: "0.2.0",
    reviewed: "2026-10-31",
    details: [
      "Packaged API targets: Advertiser API /v1 and Ads Conversions API /v1/events",
      "Runtime: Python 3.9 or newer",
      "Uses the Python standard library; no OpenAI SDK is packaged."
    ],
    limitations: [
      "Mock mode runs locally. Live Advertiser reads require an account-scoped Ads API key; optional remote conversion validation needs separate pixel and Conversions API credentials.",
      "Advertiser operations are GET-only. Conversion requests force validate_only; persisted conversions and campaign mutations are excluded.",
      "Automatic pagination, retries and rate-limit backoff are not included."
    ],
    source: {
      label: "Reviewed OpenAI Ads release implementation",
      url: "https://github.com/ai-knowledge-hub/ai-skills-guide/tree/bdc8eccba177fe9da6ab38e23da88dd3869500a9/tools-mcp/adtech/openai-ads-api-client"
    }
  },
  {
    module: "tools",
    id: "adtech/ad-platform-executor-template",
    version: "0.2.0",
    reviewed: "2026-10-31",
    details: [
      "Packaged API targets: Google Ads API v25 and Display & Video 360 API v4",
      "Runtime: Node.js 22 or newer; requires Agent Control Plane 0.2.1",
      "Uses Node's built-in HTTP functionality; no Google Ads or DV360 SDK is packaged."
    ],
    limitations: [
      "This implemented executor retains a legacy template package ID. Other provider API versions are rejected by this release.",
      "Supports only bounded campaign status/budget, ad-group status/CPC and DV360 line-item status/max-average-CPM updates.",
      "Provider credentials, policy and an exact control-plane grant are required for mutations. Ambiguous outcomes require reconciliation; rollback requires a separate grant."
    ],
    source: {
      label: "Reviewed governed executor release implementation",
      url: "https://github.com/ai-knowledge-hub/ai-skills-guide/tree/bdc8eccba177fe9da6ab38e23da88dd3869500a9/tools-mcp/adtech/ad-platform-executor-template"
    }
  },
  {
    module: "tools",
    id: "agentops/agent-control-plane-server",
    version: "0.2.1",
    reviewed: "2026-10-31",
    details: [
      "Runtime: Node.js 22 or newer",
      "Local MCP control plane using Node built-ins; no external provider API or SDK dependency."
    ],
    limitations: [
      "Requires configured identities, keys, policies and a private local filesystem store.",
      "Coordinates authority and records on one filesystem host; it does not execute provider mutations or provide multi-host consensus."
    ],
    source: {
      label: "Reviewed control plane release implementation",
      url: "https://github.com/ai-knowledge-hub/ai-skills-guide/tree/bdc8eccba177fe9da6ab38e23da88dd3869500a9/tools-mcp/agentops/agent-control-plane-server"
    }
  },
  {
    module: "tools",
    id: "adtech/conversion-event-reconciler",
    version: "0.1.0",
    reviewed: "2026-10-31",
    details: [
      "Runtime: Python 3; the release does not specify a minimum minor version.",
      "Uses the Python standard library with local JSON input and output; no provider API or SDK dependency."
    ],
    limitations: [
      "Reconciles supplied event exports or fixtures; it does not collect live events or upload conversions to a provider.",
      "This editorial review does not change the package's Not verified classification."
    ],
    source: {
      label: "Reviewed conversion reconciler release implementation",
      url: "https://github.com/ai-knowledge-hub/ai-skills-guide/tree/bdc8eccba177fe9da6ab38e23da88dd3869500a9/tools-mcp/adtech/conversion-event-reconciler"
    }
  }
];

export function findCatalogueCompatibility(module: ModuleKey, id: string, version: string): CatalogueCompatibility | undefined {
  return catalogueCompatibility.find((item) => item.module === module && item.id === id && item.version === version);
}
