# Governed Ad Platform Executor

## Tools

### `preview_ad_platform_change`

Accepts one closed `ad-platform.change-plan/v2` document and an optional bounded timeout. It validates the executor policy and reads the exact provider-native target. It returns state hashes and whether the approved precondition currently matches. It never calls the control plane, claims a grant, or mutates provider state.

### `execute_approved_change`

Accepts only `{grant, plan, timeout_ms?}`. The plan is closed and supports one provider-native target and one field transition. Before mutation the executor requires:

1. plan lifetime of at most 15 minutes and current, canonical timestamps;
2. an allowlisted sandbox target, or an explicitly enabled live policy;
3. exact equality between the plan target and grant target;
4. an `ad_platform.execute` capability and matching action ID;
5. a grant input digest equal to the canonical complete plan;
6. current provider state equal to the approved `from` value;
7. a successful atomic `validate_execution_grant` claim from the Agent Control Plane.

The executor then records `executing`, applies one fixed provider operation, re-reads state, persists an authenticated receipt, and records the terminal control-plane result. The successful response omits stored pre/post values and returns only their hashes, target identity, receipt reference, and reconciliation flag.

### `reconcile_ambiguous_execution`

Requires the same exact grant and plan. It is available only after a durable claim without a receipt. Current provider state is authoritative:

- exact proposed post-image: persist a reconciled success receipt and terminal action;
- exact pre-image: record a verified no-effect failure;
- any other state: stop with `MANUAL_RECONCILIATION_REQUIRED`.

The tool never repeats the provider mutation.

### `rollback_execution`

Requires a new `ad_platform.rollback` grant. Its input digest must bind:

- the original execution ID;
- the complete authenticated execution-receipt digest;
- the authenticated rollback-patch digest.

Rollback stops if provider state no longer equals the original verified post-image. It claims the new grant, applies the reverse bounded field transition, and verifies the original pre-image before recording success.

### `get_execution`

Returns a redacted authenticated receipt or nonterminal intent. Local state whose HMAC no longer verifies is rejected rather than projected.

## Provider mapping

- Google Ads reads use `googleAds:searchStream`; writes use only fixed v25 mutate services and update masks.
- DV360 reads and patches use only v4 advertiser-scoped line-item paths and fixed update masks.
- Redirects, caller-supplied origins, caller-supplied URLs, arbitrary methods, and arbitrary fields are rejected.
- Noncanonical origins are accepted only for explicit loopback test mode.

## Failure semantics

- Provider rejection known to precede an effect may be recorded as failed.
- Mutation timeout, transport loss, retryable server failure, or process loss after claim is ambiguous and requires reconciliation.
- Duplicate requests return the existing verified receipt.
- Concurrent requests for one provider resource serialize locally; stale sequential plans fail their provider-state precondition.
- A post-write mismatch is a conflict: the executor records it and never overwrites the observed third state automatically.
- Rollback persists authenticated state before claiming its grant and reconciles crashes or ambiguous provider outcomes without replaying an uncertain effect.
- Authority cancellation is a durable two-step transition: authenticated `cancellation_pending`, idempotent control-plane `cancelled`, then local `terminal_no_effect`. Dependency failure or process loss leaves the operation resumable and never projects a local terminal state before the authoritative cancellation exists.

## Guardrails

- Never translate free-form text into a provider mutation.
- Never accept caller-selected URLs, HTTP methods, fields, update masks, accounts, resources, or credentials.
- Never mutate before the provider pre-image matches and the control plane durably claims the exact grant.
- Never replay a claimed mutation whose provider outcome is ambiguous; reconcile current provider state first.
- Never treat a provider acknowledgement as success without a verified post-image and authenticated local receipt.
- Never roll back without a new exact grant or when current state diverges from the original verified post-image.
- Never expose provider tokens, control-plane keys, raw identity envelopes, or receipt contents in MCP responses or diagnostics.
