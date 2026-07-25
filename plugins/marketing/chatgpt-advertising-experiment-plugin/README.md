# ChatGPT Advertising Experiment Plugin

This bundle turns the ChatGPT advertising experiment pack into an installable workflow. It separates three things that are easy to confuse:

- **Usable now:** planning skills, the local event reconciler, mock fixtures, and the deterministic mock lab.
- **Usable after setup:** live read-only OpenAI Ads account, campaign, ad, and insights retrieval; remote conversion validation.
- **Template only:** the write-capable ad-platform executor. Replace and review it before live mutation work.

## Install

```bash
./bin/skills-hub install \
  --module plugins \
  --entry marketing/chatgpt-advertising-experiment-plugin@0.1.0 \
  --runtime codex
```

The installer places bundled skills, agents, and tools in their native runtime directories. Plugin config, hooks, examples, and the mock runner stay in the plugin directory.

## First run: no credentials

From the installed plugin directory:

```bash
python3 scripts/run_mock_lab.py
```

The command reads the installed mock Ads API fixtures, validates a conversion batch, reconciles platform events against business outcomes, and writes `output/mock-evidence-bundle.json`. It makes no network calls.

Use that evidence bundle with `adtech/chatgpt-ads-experiment-supervisor` in `mock-run` mode.

## Live read-only setup

Create an account-scoped API key in OpenAI Ads Manager, store it outside agent context, and run:

```bash
export OPENAI_ADS_API_KEY='...'
python3 ../../../tools-mcp/adtech/openai-ads-api-client/scripts/openai_ads_client.py --mode live account
```

For remote conversion validation, also configure `OPENAI_ADS_PIXEL_ID` and `OPENAI_ADS_CONVERSIONS_API_KEY`. The bundled validator always sends `validate_only: true`.

## Live writes

No live OpenAI Ads writer ships in this plugin. The included executor is an architecture template. A production implementation must add:

- explicit operation allowlists
- bounded budget and targeting deltas
- two-step approval
- current-state validation
- idempotency and retry policy
- immutable audit records and rollback evidence

## Official OpenAI Ads references

- https://developers.openai.com/ads/api-overview
- https://developers.openai.com/ads/api-quickstart
- https://developers.openai.com/ads/conversions-api
