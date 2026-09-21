# Agent Control Plane MCP Server

A launchable, dependency-free local MCP authority for governed agent execution.

The server authenticates one runtime identity, evaluates externally administered tenant policy, creates exact approval requests, issues short-lived signed execution grants, records ordered action outcomes, and verifies a tamper-evident audit chain. It does not call provider mutation APIs or let the governed agent administer policy or approve its own actions.

## Authority boundary

- The model may propose an action and explain why it wants approval.
- The signed runtime identity supplies actor, tenant, capability, and account authority.
- A separately signed executor identity—not the proposing agent—records provider execution lifecycle evidence.
- The active policy deterministically decides allow, require approval, or deny.
- A separately launched administrator command installs policy and records approval decisions.
- A grant is bound to the actor, tenant, exact provider-native target, action ID, capability/version, input hash, policy version/digest, approval, and expiry.
- Provider executors remain separate and must atomically claim the signed grant at their effect boundary before making a provider call.

## Local setup

Requires Node.js 22 or newer. Generate separate Ed25519 identity and execution-grant key pairs, plus an independent random `CONTROL_PLANE_AUDIT_KEY` of at least 32 bytes. Keep both private keys and the audit key in the runtime secret store, and set `CONTROL_PLANE_STORE` to an absolute, private directory. Never bind the grant private key into an executor profile.

```sh
openssl genpkey -algorithm Ed25519 -out identity-private.pem
openssl pkey -in identity-private.pem -pubout -out identity-public.pem
openssl genpkey -algorithm Ed25519 -out grant-private.pem
openssl pkey -in grant-private.pem -pubout -out grant-public.pem
```

The administrator identity minting command receives `CONTROL_PLANE_IDENTITY_PRIVATE_KEY`; runtime servers receive only `CONTROL_PLANE_IDENTITY_PUBLIC_KEY`. `CONTROL_PLANE_GRANT_ROLE_KEY` is explicitly role-dependent: an `agent_runtime` profile binds the issuer private key, while an `executor_runtime` profile binds only the grant public key. Startup validates the signed identity first, derives the required key type from that role, and rejects an executor profile containing a private key.

| Authentication profile | Signed identity role | `CONTROL_PLANE_GRANT_ROLE_KEY` |
| --- | --- | --- |
| Issuer | `agent_runtime` | Grant private key |
| Executor | `executor_runtime` | Grant public key only |

Both profiles use the same five declared binding names in `skills-hub auth configure`; the packaged driver validates the role-specific key type during bootstrap.

Mint the separately scoped identities from the example claims:

```sh
node scripts/control_plane_admin.mjs mint-identity --file config/admin-identity-claims.example.json
node scripts/control_plane_admin.mjs mint-identity --file config/runtime-identity-claims.example.json
node scripts/control_plane_admin.mjs mint-identity --file config/executor-identity-claims.example.json
```

Bind the first result to `CONTROL_PLANE_ADMIN_IDENTITY`, install the policy through the administrator process, and bind the second result to `CONTROL_PLANE_IDENTITY` before starting the issuer MCP server. Set `CONTROL_PLANE_GRANT_ROLE_KEY` to the grant private key for that issuer profile. For the separate executor profile, bind the third identity result and set the same binding name to the grant public key. The administrator process also receives the audit key, but never the grant private key:

```sh
node scripts/control_plane_admin.mjs install-policy --file config/policy.example.json
node scripts/agent_control_plane_server.mjs --healthcheck
node scripts/agent_control_plane_server.mjs
```

When an approval is pending, an authorized human or approval service uses the separate administrator command:

```sh
node scripts/control_plane_admin.mjs decide-approval --file approval-decision.json
```

The proposing MCP client never receives the administrator or executor identity. A governed executor runs its own server/client boundary with the executor identity and only the grant public key.

## Persistence and recovery

All state is tenant-scoped with collision-free storage encoding. Policy versions, activation records, decisions, approval requests/decisions, execution claims, action lifecycle records, transaction receipts, audit events, and a signed audit head are persisted with fsync and atomic publication. Temporary transaction intents are closed-schema, authenticated, and tenant-contained; they are durably removed after their receipt commits, so recovery scans unfinished work rather than complete history. Recovery rejects injected, altered, or path-escaping intents. Each locked operation loads audit evidence once, and every authorization-sensitive domain record is verified against that authenticated index before use.

The audit verifier detects gaps, deletion—including tail deletion—substitution, reordering, cross-tenant events, invalid event hashes, and a missing or forged signed head.

## MCP tools

- `check_policy`
- `request_approval`
- `get_approval`
- `authorize_action`
- `validate_execution_grant`
- `get_execution_claim`
- `revalidate_execution_claim`
- `record_agent_action`
- `verify_audit_chain`

See [TOOL.md](TOOL.md) for lifecycle details.

## Non-goals

- No policy or approval administration through MCP.
- No provider mutation calls.
- No storage of provider credentials or action payloads.
- No claim that an approval proves a provider effect occurred; governed executors supply receipt references, and only their SHA-256 digests are persisted.
