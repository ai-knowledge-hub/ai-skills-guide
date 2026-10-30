# OpenAI Ads Adapter Template (deprecated)

This package is retained only to preserve release history. It is replaced by
`adtech/openai-ads-api-client`, the single supported executable path for mock
and live read-only OpenAI Ads access.

Do not add new plugin, agent, or pack references to this template. Existing
references should be changed to `adtech/openai-ads-api-client`; its mock command
is the direct replacement for these fixtures:

```bash
python3 scripts/openai_ads_client.py --mode mock account
```

Official starting point: https://developers.openai.com/ads
