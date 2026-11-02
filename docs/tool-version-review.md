# Implemented-tool version notes

Reviewed 31 October 2026 for the static catalogue. These are facts from the
retained packages, not live provider checks, SDK certification or a claim that
a vendor currently supports every packaged API target. No provider calls were
made. A changed provider contract calls for a focused review and, where needed,
a new package release; elapsed time alone does not invalidate these notes.

## Reviewed releases

Each annotation is bound to its exact `tools` package ID and release version.
GA4's existing note remains. Six more notes cover the remaining implemented
tools. No SDK version is supplied where the package uses only runtime built-ins.

| Package | Release | Useful version information | Retained files supporting the note |
| --- | --- | --- | --- |
| `analytics/ga4-mcp-connector` | 0.2.0 | Data API v1beta; Node.js 22+; no Google SDK | Existing annotation links the retained `tool.yaml`. |
| `warehouse/bigquery-mcp-query-runner` | 0.2.0 | BigQuery v2; Node.js 22+; no Google SDK | `tool.yaml`, `scripts/bigquery_mcp_server.mjs`, `scripts/bigquery_auth_driver.mjs` |
| `ads/meta-ads-mcp-connector` | 0.2.0 | Marketing / Graph API v24.0, other versions rejected; Node.js 22+; no Meta SDK | `tool.yaml`, `scripts/meta_ads_mcp_server.mjs`, `scripts/meta_ads_auth_driver.mjs` |
| `adtech/openai-ads-api-client` | 0.2.0 | Advertiser `/v1`, Conversions `/v1/events`; Python 3.9+; no OpenAI SDK | `tool.yaml`, `scripts/openai_ads_client.py`, `scripts/validate_conversion_events.py`, `scripts/openai_ads_auth_driver.py` |
| `adtech/ad-platform-executor-template` | 0.2.0 | Google Ads v25, DV360 v4, other versions rejected; Node.js 22+; Control Plane 0.2.1; no provider SDK | `tool.yaml`, `scripts/ad_platform_executor_core.mjs`, `scripts/ad_platform_executor_server.mjs` |
| `agentops/agent-control-plane-server` | 0.2.1 | Node.js 22+; local MCP control plane, no external provider API or SDK | `tool.yaml`, `scripts/control_plane_core.mjs`, `scripts/agent_control_plane_server.mjs` |
| `adtech/conversion-event-reconciler` | 0.1.0 | Python 3, minimum minor unspecified; local JSON processing, no provider API or SDK | `tool.yaml`, `README.md`, `scripts/reconcile_events.py` |

For the six added notes, the manifest, README and all packaged scripts were
read from `releases/tools-mcp/<id>/<version>/package.tar.gz` and compared byte for
byte with `tools-mcp/<id>` at commit
`bdc8eccba177fe9da6ab38e23da88dd3869500a9`. They match. Each annotation links
that immutable source directory so readers can inspect the implementation and
manifest together. These links describe the retained release, not a moving
development branch. Release archives, sidecars and manifests remain unchanged.

## Scope and limitations

- BigQuery and Meta notes preserve the read-only query/reporting boundaries
  and the need for configured credentials and policy.
- OpenAI Ads notes distinguish local mock use, credentialed reads and
  non-persisting conversion validation. API paths are code targets, not a
  statement of live account access or vendor availability.
- The governed executor is implemented despite its legacy `-template` ID.
  Its restricted operations and control-plane dependency remain explicit.
- The control plane coordinates local authority; it does not perform provider
  mutations. The reconciler processes supplied event data and keeps its
  **Not verified** classification. Reviewing content does not promote readiness.
- `adtech/openai-ads-adapter-template` 0.2.0 is still template-only and receives
  no implemented-provider compatibility claim. Content-only skills, agents and
  plugins are not assigned inferred platform versions.

The existing shared detail panel publishes the notes, review date, source and
limitations during the static build. Exact release matching, missing metadata,
all reviewed detail pages and the reconciler's unchanged classification are
covered by the browser smoke suite. Runtime authentication and evidence-age
policy remain separate under #83.
