# Conversion Event Reconciler

## Contract
Accept a JSON document containing `browser_events`, `server_events`, and optional `outcomes`.

Run:

```bash
python3 scripts/reconcile_events.py examples/input.json
```

## Behavior
- Deduplicate browser and server events with the same `event_name` and `event_id`.
- Preserve source provenance.
- Flag missing event IDs and conflicting values.
- Join canonical events to order, CRM, refund, cancellation, or support outcomes by `order_id`.
- Report platform-facing conversions separately from independently verified outcomes.

## Guardrails
Input fixtures must use pseudonymous IDs. Do not put access tokens, raw emails, or payment credentials in event payloads.
