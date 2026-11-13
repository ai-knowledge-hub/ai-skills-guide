# Native Codex skill discovery

<!-- markdownlint-disable MD013 -->

This is the next bounded slice of #52: establish that a real Codex client reports
the exact installed skill, separately from package delivery. The subject is
`engineering/implementation-strategy@0.1.0`, installed from the public HTTPS
registry. The adopting client is pinned to **Codex CLI 0.168.0**.

The check calls the native app-server `initialize` and `skills/list` methods. It
does not start a model turn, invoke a skill, configure a provider, use credentials,
change catalogue readiness, publish a CLI or modify a real user installation.
Claude, plugins, MCP adoption and other client versions remain separate cases.

## Independent evidence and authority

| Boundary | Authority and enforcement | Confidence |
| --- | --- | --- |
| Installer identity | Existing released-client checksum and tag/workflow-bound signed-provenance verification runs before execution. An explicit source-built installer is labelled `candidate`, with no released-client fallback. | Confirmed by the existing verifier and retained run |
| Package identity | Exact version plus independently pinned archive digest; same-origin HTTPS registry/artifact checks, historical manifest validation, exact installed bytes and complete receipt verification reuse the published-install checker. | Confirmed by the retained run |
| Adopting client | Official release archives have reviewed platform-specific SHA-256 pins in `scripts/native_codex.py`. Cached archives must match the same pin before extraction or execution. Binary digest, reported version and runtime home are recorded. These checksum pins are not a separate Codex build attestation. | Confirmed for the recorded macOS client; other host pins await execution |
| Native discovery | Codex must report `implementation-strategy`, the exact namespaced `SKILL.md` path, the expected `user`/`repo` scope, enabled state and no load errors. File existence, an installer receipt or a zero process exit cannot satisfy this oracle. | Confirmed by client responses |
| Side effects | Installer and Codex receive a newly constructed child environment and disposable home/repository. No inherited credential, managed-config or runtime-home environment reaches either child. Each scan must leave the installed package unchanged. | Confirmed for the exercised child context; not an OS-wide filesystem/network sandbox |
| Broader usability | Discovery does not establish instruction adherence, model quality, provider authentication, desktop UI selection or workflow execution. | Unproven and outside this slice |

The installed tree and receipt are durable delivery evidence. The client-reported
list is the native discovery oracle. Neither is live execution authority under
[operational readiness v3](../shared/contracts/operational-readiness-v3.md).

The official [skill locations](https://learn.chatgpt.com/docs/build-skills) and
[app-server protocol](https://learn.chatgpt.com/docs/app-server) informed the test.
The recorded responses, rather than the current documentation or the installer's
path resolver, determine the result for this pinned client.

## Finite case and transition model

Each case starts in a fresh home and a `git init` repository outside the checkout.
Spaces in the temporary path exercise argument handling. No extra discovery roots
are injected. `HOME` in the **child environment** is the disposable home;
the invoking shell and maintainer's environment are unchanged.

| Case | Installer arguments and expected target | Discovery scope |
| --- | --- | --- |
| `default-user` | Omit `--target` and `CODEX_HOME`; require `$HOME/.codex/skills/engineering/implementation-strategy/SKILL.md` | `user` |
| `explicit-user` | Pass `$HOME/.agents/skills` as `--target` | `user` |
| `explicit-repository` | Pass `$REPOSITORY/.agents/skills` as `--target` | `repo` |
| `codex-home` | Override `CODEX_HOME` with an existing disposable directory, omit `--target`; require `$CODEX_HOME/skills` | `user` |
| `unregistered-target` | Pass an arbitrary directory outside discovery roots | No discovered skill, despite verified delivery |

The four positive cases each exercise five states:

1. Before installation, the subject is absent.
2. After remote installation and independent byte/receipt verification, the exact
   name/path/scope is visible and enabled.
3. After renaming `SKILL.md`, the subject is absent.
4. After replacing it with malformed metadata, the subject remains absent and
   Codex reports an error for that exact file.
5. After restoring the original published bytes, discovery and byte/receipt
   verification succeed again.

Each case keeps one app-server process alive and uses `forceReload: true` for
every scan, exercising cache invalidation across these transitions. Bundled
system skills may appear; they cannot substitute for the subject. Unexpected
non-system skills or load errors fail the check. The unregistered-target control
has two states: absent before installation and still absent after verified
installation. Total: **22 native discovery probes in five isolated cases**.

The initial path concern was a compatibility hypothesis, not a confirmed defect.
Codex 0.168.0 discovers the existing legacy default, the explicit `.agents`
locations and the existing `CODEX_HOME` override. The installer resolver is
unchanged; this evidence does not promise support in all client versions.

## Run locally

Requirements: macOS or Linux on a pinned supported architecture, Python 3.12+,
Go, Git, and GitHub CLI with attestation support for a released installer.
The native client is downloaded into the disposable workspace, never installed
over the maintainer's client.

```sh
CGO_ENABLED=0 go build -o bin/published-install-verifier ./cmd/published-install-verifier
python3 scripts/check-native-codex-discovery.py \
  --cli-release v0.4.0-alpha.1 \
  --report output/native-codex/released.json
```

For an explicitly labelled installer candidate:

```sh
make cli-build-test
python3 scripts/check-native-codex-discovery.py \
  --cli bin/skills-hub-test \
  --report output/native-codex/candidate.json
```

An optional `--codex-archive /absolute/path/to/codex-<platform>.tar.gz` uses a
cached official archive, still requiring the hard-coded platform digest.
There is no arbitrary native binary or `latest` override. An explicitly reviewed
`--registry-url` may select another HTTPS deployment, while the package version
and artifact digest remain fixed.

The JSON report includes the installer source/tag, CLI and verifier digests,
native archive/binary digests and version, registry/archive/manifest hashes,
verified payloads/receipts, initialized runtime home and each probe's native
subject/error records. Temporary paths are normalized to `$WORK`; complete
system-skill inventories are omitted. Failures retain their stage and completed
probes, return a nonzero exit, and never trigger a fallback build. Temporary
homes, installed packages, caches and child processes are cleaned up.

The manually dispatched **Native Codex skill discovery** workflow enables
independent Ubuntu/macOS runs and retains reports even on failure. It must be
merged onto the default branch before normal dispatch. Public-network checks
do not gate every PR; deterministic boundary tests do.

## Regression and failure-space coverage

```sh
python3 -m unittest discover -s scripts/tests -p 'test_native_codex_discovery.py'
```

The coupled high-risk dimensions are installer provenance × package identity,
target/root × namespace × discovery scope, and file state × cached client list.
They are checked at the cheapest boundary that preserves each oracle: the live
native client for discovery, existing package validators for delivery, and local
fixtures for checker faults. There is no concurrency or provider operation in
this finite model.

The 12 deterministic tests cover six positive record-field substitutions,
missing/extra/duplicate records and wrong cwd, stale success in all four absent
states, missing/wrong malformed-file diagnostics, credential/home isolation for
all five targets, archive checksum substitution, cached-archive replacement after
hashing and four unsafe archive layouts,
failure-report retention, exact installer target arguments, all state controls,
and mutation of installed bytes during a scan. Protocol tests exercise all nine
modeled faults: wrong response ID, invalid JSON, non-object JSON, RPC error,
missing result, early exit, timeout, oversized stdout and excessive stderr.
Notifications and partial-line delivery also have a successful control.

RPC requests have a 30-second deadline, a 1 MiB message bound and an 8 MiB session
output bound. Downloaded archives and extracted binaries have separate bounds.
Only `initialize` and `skills/list` requests are allowed by the oracle; a model
turn cannot be started through it. Tests do not download clients or call models.
These are bounded falsifications, not an exhaustive input or mutation score.

## Results and remaining work

The [13 November macOS acceptance record](reviews/native-codex-2026-11-13.md)
and [released-installer JSON](reviews/native-codex-2026-11-13/released-macos.json)
retain the observed result. Linux execution, other client versions, process
restart persistence, nested repository ancestry, simultaneous duplicate names,
symlinked roots, policy-disabled skills, actual skill invocation and Claude/MCP/
plugin adoption remain unproven by this run. Ordinary native startup may have
background behavior; no OS-wide egress monitor or zero-network claim is made.
Provider credentials are neither supplied nor requested.

Issue #52 and epic #33 remain open. This run establishes native discovery for this
exact combination; it does not promote catalogue readiness or complete release
acceptance across hosts.
