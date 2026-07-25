# OpenAI Ads API Client

A standard-library Python client that turns the OpenAI Ads adapter contract into a runnable integration boundary.

- Mock mode works immediately with bundled fixtures.
- Live Advertiser API mode is read-only.
- Conversions API integration supports local validation and remote `validate_only` requests.
- No campaign mutation path is included.

Start here:

```bash
python3 scripts/openai_ads_client.py --mode mock account
```

Then read `TOOL.md` before connecting an Ads Manager account.
