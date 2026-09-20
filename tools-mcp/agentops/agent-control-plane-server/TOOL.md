# Agent Control Plane MCP Server

## Lifecycle

```text
signed runtime identity
  → check_policy(exact proposal)
  → optional request_approval
  → separate administrator decision
  → authorize_action
  → signed short-lived execution grant
  → external governed executor
  → validate_execution_grant atomically claims the grant immediately before effect
  → record_agent_action(optional executing observation → terminal reconciliation)
```

Every state transition is tenant-scoped, idempotent, append-only, and represented in the hash-chained audit log.

## `check_policy`

Accepts an action ID, run ID, capability and version, risk tier, provider-native target, and SHA-256 input hash. Actor and tenant are never accepted from tool input. The server verifies the proposal against the signed runtime identity and active policy.

The returned immutable decision includes the policy ID, version, digest, matched rules, effect, policy-derived effective risk, and approval requirements. All rules for the capability and provider-account target participate regardless of the proposal's requested risk label: deny overrides approval, approval overrides allow, and the highest declared risk plus shortest applicable approval TTL wins. An unmatched action is denied.

## `request_approval`

Accepts only a `require_approval` decision owned by the current runtime identity. The immutable request copies the exact proposal and policy binding and expires no later than either the rule TTL or policy expiry.

Approval decisions are not MCP tools. They require a separately authenticated administrator process with one of the policy-declared approver roles.

## `get_approval`

Returns the authoritative request and its immutable decision, if present. It cannot read another actor's request.

## `authorize_action`

Revalidates runtime identity, scope, current policy digest, and any required approval. It returns a signed grant valid for at most 60 seconds. Policy replacement invalidates earlier decisions and approvals.

The grant is an input to a governed executor, not proof that an external effect occurred.

## `validate_execution_grant`

Requires a separately signed `executor_runtime` identity with `CONTROL_PLANE_GRANT_ROLE_KEY` bound to the grant public key. The same binding carries the private key only in an `agent_runtime` issuer profile; role-aware startup rejects that key type for executors. Immediately before the provider effect the executor revalidates the grant signature and authenticated persisted grant, executor tenant/capability/account scope, current policy digest, approval status and expiry, and grant expiry. It then atomically persists a single-use execution claim. Concurrent or later claims fail closed. This durable claim is the effect-boundary handoff for governed executors.

## `record_agent_action`

Records the executor-observed lifecycle and requires both the authenticated single-use claim and the same separately signed `executor_runtime` identity whose tenant, capability, and provider-account scope cover the grant. The proposing `agent_runtime` identity cannot call this operation successfully. An optional `executing` observation may be followed by exactly one terminal state: `executed`, `failed`, or `cancelled`. A terminal record may also reconcile a crash after the provider effect but before an `executing` observation. Exact retries are idempotent; conflicting duplicates and terminal resurrection fail closed.

No action payload, raw provider receipt, or provider credential is stored—only hashes, immutable target identity, authority references, claim identity, and status.

## `verify_audit_chain`

Checks sequence continuity, tenant identity, previous hashes, event hashes, and the separately signed durable head. It returns the number of verified events and current head hash without returning sensitive event content.

## Administration

`scripts/control_plane_admin.mjs` is deliberately outside native MCP discovery. It supports:

- `mint-identity`
- `install-policy`
- `decide-approval`

Policy administration requires `policy_admin`; approval decisions require a role named by the matching policy rule.

## Guardrails

- Never accept actor, tenant, role, capability authority, or account scope from tool arguments.
- Never expose policy installation or approval decisions through the governed MCP process.
- Never issue a grant from stale policy, expired identity, expired approval, or a mismatched target or input hash.
- Never perform a provider effect without winning the durable single-use grant claim.
- Never treat a grant or approval as evidence that a provider effect occurred.
- Never store provider credentials or raw action inputs and outputs in control-plane records.
- Never repair a committed audit mutation as though it were an interrupted transaction.
