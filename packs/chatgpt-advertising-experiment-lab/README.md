# ChatGPT Advertising Experiment Lab

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
| `adtech/openai-ads-adapter-template` | Start with read-only mock OpenAI Ads contracts and fixtures. |
| `adtech/chatgpt-ads-experiment-supervisor` | Coordinate the end-to-end experiment. |

## Reused foundations

- `marketing/creative-operating-system-supervisor`
- `marketing/ai-output-eval-scorecard`
- `agentops/ad-platform-policy-gate-designer`
- `adtech/ad-platform-control-plane-supervisor`
- `adtech/ad-platform-executor-template`

## Recommended first run

1. Install the supervisor and its dependencies.
2. Start in `mock-run` mode.
3. Use the Northstar Meals task-map and event fixtures.
4. Run the conversion reconciler.
5. Compare platform-reported conversions with settled and refunded outcomes.
6. Add the missing retention signal before drawing a performance conclusion.

```text
Use ChatGPT Ads Experiment Supervisor in mock-run mode.
Run the Northstar Meals experiment using bundled fixtures.
Keep paid, earned, and executable evidence separate.
Stop if the measurement ledger cannot independently verify conversion quality.
```

## Live-system boundary

The pack does not ship a live OpenAI Ads client and does not grant write authority. Confirm current official API access and schemas before replacing fixtures. Route all future writes through the existing policy-gated executor with human approval and rollback evidence.
