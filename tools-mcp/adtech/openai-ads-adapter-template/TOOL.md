# OpenAI Ads Adapter Template

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

## Status
This repository ships contracts and fixtures, not a live OpenAI Ads client. Confirm current API availability and official schemas before implementing a production binding.
