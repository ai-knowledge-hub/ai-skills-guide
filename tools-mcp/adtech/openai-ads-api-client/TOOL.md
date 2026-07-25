# OpenAI Ads API Client

## Contract

Use mock mode without credentials:

```bash
python3 scripts/openai_ads_client.py --mode mock account
python3 scripts/openai_ads_client.py --mode mock campaigns --limit 20
python3 scripts/openai_ads_client.py --mode mock ads --ad-group-id adgrp_301
python3 scripts/openai_ads_client.py --mode mock insights --scope campaign --entity-id cmpn_101
```

Use live read-only mode after setting an account-scoped key:

```bash
export OPENAI_ADS_API_KEY='...'
python3 scripts/openai_ads_client.py --mode live account
```

Validate conversion events locally:

```bash
python3 scripts/validate_conversion_events.py examples/conversion-events.json --now-ms 1784937600000
```

Add `--remote-validate` only when `OPENAI_ADS_PIXEL_ID` and `OPENAI_ADS_CONVERSIONS_API_KEY` are set. The script overwrites `validate_only` with `true` before sending.

## Exposed operations

- `get_account`
- `list_campaigns`
- `list_ads`
- `get_insights`
- `validate_conversion_events`

## Guardrails

1. Never pass API keys as command arguments or place them in prompts.
2. Advertiser API methods are restricted to GET requests.
3. Conversion batches are limited to 1,000 events.
4. Remote conversion validation cannot persist an event.
5. Route mutations through a policy-gated executor with human approval.

## Official references

- https://developers.openai.com/ads/api-overview
- https://developers.openai.com/ads/api-quickstart
- https://developers.openai.com/ads/api-reference/authentication
- https://developers.openai.com/ads/api-reference/insights
- https://developers.openai.com/ads/conversions-api
