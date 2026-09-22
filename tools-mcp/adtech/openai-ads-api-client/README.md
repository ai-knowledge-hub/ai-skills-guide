# OpenAI Ads API Client

A standard-library Python client and the preferred OpenAI Ads package in this
catalog. It replaces `adtech/openai-ads-adapter-template` with a runnable,
bounded integration boundary.

- Mock mode works immediately with bundled fixtures.
- Live Advertiser API mode is read-only.
- Conversions API integration supports local validation and remote `validate_only` requests.
- No campaign mutation path is included.

Start here:

```bash
python3 scripts/openai_ads_client.py --mode mock account
```

Then read `TOOL.md` before connecting an Ads Manager account.

Existing plugin references to `adtech/openai-ads-adapter-template` should be
changed directly to this package. Mock mode covers the old fixture-first path,
so no adapter wrapper is needed.
