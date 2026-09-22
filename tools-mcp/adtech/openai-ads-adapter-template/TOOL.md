# OpenAI Ads Adapter Template (deprecated)

> Replaced by `adtech/openai-ads-api-client`. This document describes the old
> contract only so existing consumers can migrate without guessing.

## Purpose
Provide a mock-first, read-only provider boundary for OpenAI Ads account, campaign, ad, insight, and product-feed data.

## Proposed operations
- `get_account`
- `list_campaigns`
- `get_campaign`
- `list_ads`
- `get_campaign_insights`
- `validate_product_feed`

## Binding rules
1. Keep API keys in the adapter environment, never in agent context.
2. Scope every request to the configured advertiser account.
3. Validate responses against local contracts before returning them.
4. Start with the fixtures in `examples/`.
5. Route any future write operation through `adtech/ad-platform-executor-template`; do not add direct model-to-platform writes here.

## Guardrails
- Keep the adapter read-only.
- Never expose credentials to the agent.
- Do not infer that mock schemas match current production schemas.

## Migration

1. Replace dependency ID `adtech/openai-ads-adapter-template` with
   `adtech/openai-ads-api-client`.
2. Replace direct fixture reads with the client mock commands documented in its
   `TOOL.md`.
3. For live reads, bind `OPENAI_ADS_API_KEY` through the runtime-owned
   environment and run the packaged authentication readiness flow.
4. Keep all campaign writes outside the client and behind the governed
   ad-platform executor.

The historical template must not be reactivated or treated as operational
evidence for the replacement package.
