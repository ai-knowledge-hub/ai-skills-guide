# Governed Ad Platform Executor

A launchable MCP executor for a deliberately small Google Ads and Display & Video 360 mutation surface. It is the effect-boundary companion to `agentops/agent-control-plane-server`: the control plane owns identity, policy, approval, and single-use grants; this package owns provider-state revalidation, serialized effects, provider receipts, reconciliation, and rollback verification.

## Authority boundary

- A model may propose a closed change plan or request a read-only preview.
- The executor policy independently allowlists provider-native account, resource, object, operation, environment, and deadline.
- Only the Agent Control Plane may attest the signed executor identity and authorize a mutation.
- A mutation grant must bind the exact canonical plan digest, action ID, capability, and provider-native target.
- The executor re-reads current state before atomically claiming the grant. A stale pre-image stops before the provider mutation.
- An executor claim is single use. Duplicate or ambiguous work is reconciled from provider state instead of replayed.
- Rollback is a new effect and therefore requires its own grant bound to the authenticated original receipt and rollback patch.

No tool accepts natural-language mutation instructions, raw URLs, HTTP methods, arbitrary update masks, account names, or caller-selected provider endpoints.

## Supported operations

| Provider | Resource | Bounded fields |
| --- | --- | --- |
| Google Ads API v25 | campaign | status |
| Google Ads API v25 | campaign budget | amount in micros |
| Google Ads API v25 | ad group | status, CPC bid in micros |
| Display & Video 360 API v4 | line item | entity status, maximum average CPM bid in micros |

Provider-native numeric IDs remain authoritative. The policy and grant must name the same account, resource type, and resource ID as the plan.

## Setup

Requires Node.js 22 or newer and an installed Agent Control Plane 0.2.1 package. Configure these runtime-owned bindings:

- `AD_PLATFORM_PROVIDER_CREDENTIAL`: a JSON access-token envelope. Google Ads may also include `developer_token` and `login_customer_id`.
- `AD_PLATFORM_EXECUTOR_POLICY`: the closed sandbox/live policy shown in `config/executor-policy.example.json`.
- `AD_PLATFORM_EXECUTOR_STORE`: absolute private directory for authenticated intents and receipts.
- `AD_PLATFORM_EXECUTOR_AUDIT_KEY`: independent random key of at least 32 bytes.
- `CONTROL_PLANE_PACKAGE_ROOT`: absolute root of the installed Agent Control Plane package.
- `CONTROL_PLANE_PACKAGE_INTEGRITY`: runtime-owned JSON pin containing the exact `0.2.1` install runtime, tree SHA-256, and runtime-contract SHA-256 from `.skills-hub-install.json`. The executor recomputes the installed tree before every control-plane launch.
- `CONTROL_PLANE_STORE`, `CONTROL_PLANE_IDENTITY`, `CONTROL_PLANE_IDENTITY_PUBLIC_KEY`, `CONTROL_PLANE_GRANT_ROLE_KEY`, and `CONTROL_PLANE_AUDIT_KEY`: the separately configured executor profile. `CONTROL_PLANE_GRANT_ROLE_KEY` must contain only the grant public key.

The executor launches the exact Agent Control Plane entrypoint from the verified package root with a minimal environment. Provider credentials are never passed to that subprocess. The packaged auth driver verifies the control-plane identity and audit chain, then performs a read-only provider-target probe.

```sh
node scripts/ad_platform_executor_server.mjs --healthcheck
node scripts/ad_platform_executor_server.mjs --smoke
node scripts/ad_platform_executor_server.mjs
```

Register `.mcp.json` only after `skills-hub auth configure` and `skills-hub auth status` succeed for the executor profile.

## Effect lifecycle

```text
closed plan + signed short-lived grant
  → policy and exact-binding validation
  → per-resource lock
  → current provider-state read
  → stale-state check
  → authenticated local intent
  → atomic Agent Control Plane claim
  → current grant, policy, approval, identity, and scope revalidation
  → executing observation
  → bounded provider mutation
  → current provider-state verification
  → authenticated immutable receipt
  → terminal Agent Control Plane observation
```

The local intent is written before claiming authority. If current authority ends before an effect, the executor first persists `cancellation_pending`, records the exact cancellation idempotently in the control plane, and only then marks the local intent `terminal_no_effect`. A restart resumes that sequence without revalidating authority or touching the provider. A crash or timeout after the claim never causes automatic provider replay. `reconcile_ambiguous_execution` reads provider state and records either the exact proposed post-image or the verified absence of effect. Any third state requires manual reconciliation.

## Rollback and conflict limits

Post-write verification that observes an unexpected third state is a conflict. The executor records it and stops; it never restores the old value automatically because doing so could overwrite a newer authorized provider change.

`rollback_execution` requires a new `ad_platform.rollback` grant bound to the original authenticated receipt and must remain allowed by current executor policy. It persists an authenticated rollback intent before claiming the grant, records the effect-start boundary, and reconciles restarts against the original post-image, restored pre-image, or a conflicting third state. A successful provider response is not enough; the executor re-reads and verifies the original pre-image before recording success.

## Evidence boundary

CI uses deterministic provider simulations and exercises authority substitution, dependency substitution, stale state, duplicates, concurrency, timeouts, ambiguous effects, crash points, deleted or tampered local state, post-write conflicts, and rollback recovery. Live sandbox mutation evidence is intentionally separate and is not claimed by this release.
