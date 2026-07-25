# ChatGPT Advertising Experiment Lab

> **Installable path:** Use `marketing/chatgpt-advertising-experiment-plugin` for the bundled skills, agents, tools, configuration, and one-command offline mock lab. This page remains the documentation pack.

A practical pack for testing the advertising shift described in **Who Owns the Pixel? ChatGPT and the Repricing of Advertising**.

Article link: pending publication.

## What this pack proves

Advertising inside an agent-mediated channel needs two control planes:

1. The execution control plane governs what an agent may do.
2. The evidence control plane governs what the system may learn and claim.

## Included entries

| Entry | Purpose |
| --- | --- |
| `marketing/task-map-designer` | Convert customer situations into falsifiable task hypotheses. |
| `adtech/advertising-signal-ownership-auditor` | Map collection, control, visibility, and use of every signal. |
| `marketing/agent-share-of-choice-evaluator` | Evaluate progression through agent-mediated decision stages. |
| `adtech/conversion-event-reconciler` | Deduplicate Pixel and server events and verify first-party outcomes. |
| `adtech/openai-ads-api-client` | Run mock or live read-only OpenAI Ads retrieval and validate conversion batches without saving them. |
| `adtech/openai-ads-adapter-template` | Reference the provider-boundary contract when building another integration. |
| `adtech/chatgpt-ads-experiment-supervisor` | Coordinate the end-to-end experiment. |

## Reused foundations

- `marketing/creative-operating-system-supervisor`
- `marketing/ai-output-eval-scorecard`
- `agentops/ad-platform-policy-gate-designer`
- `adtech/ad-platform-control-plane-supervisor`
- `adtech/ad-platform-executor-template`

## Recommended first run

1. Install `marketing/chatgpt-advertising-experiment-plugin`.
2. Run `python3 scripts/run_mock_lab.py` from the installed plugin directory.
3. Give the generated evidence bundle to the supervisor in `mock-run` mode.
4. Compare platform-reported conversions with settled and refunded outcomes.
5. Add the missing retention signal before drawing a performance conclusion.

```text
Use ChatGPT Ads Experiment Supervisor in mock-run mode.
Run the Northstar Meals experiment using bundled fixtures.
Keep paid, earned, and executable evidence separate.
Stop if the measurement ledger cannot independently verify conversion quality.
```

## Live-system boundary

The installable plugin includes a live read-only OpenAI Ads client and non-persisting conversion validation. It does not grant write authority. Route all future mutations through a reviewed policy-gated executor with human approval and rollback evidence.
