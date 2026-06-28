# Versioning and Release

- Use semantic tags for stable skill snapshots.
- Prefer additive updates; avoid breaking output schemas without notes.
- Keep changelogs in PR descriptions.
- Mark deprecated skills clearly in folder README or `SKILL.md`.

## Release hardening checklist

1. CI green on `dev`:
   - module structure validation
   - manifest schema validation
   - registry generation for all module indexes
   - web lint + build
   - web Playwright smoke E2E
2. Merge `dev` into `main`.
3. Confirm Vercel production deploy from `main`.
4. Confirm preview deploys from `dev`.

## First alpha tag example

Create and push the baseline alpha tag:

```bash
git checkout main
git pull origin main
git tag v0.2.0-alpha.1
git push origin v0.2.0-alpha.1
```

## GitHub release cut (recommended)

Use the `Cut Release` workflow from the GitHub Actions tab. It is a manual
workflow so maintainers can decide which merged changes deserve a public
release.

Inputs:

- `release_type`: the version bump to cut from the latest `v*` tag.
- `dry_run`: validates and computes the next tag without pushing it.

Recommended flow:

1. Merge `dev` into `main`.
2. Open GitHub Actions.
3. Run `Cut Release` with `dry_run: true`.
4. Confirm the computed tag and checks.
5. Re-run with `dry_run: false`.

The workflow runs validation, rebuilds the registry, checks generated files,
installs the web app, lints it, builds it, and then pushes the computed tag.
The tag push triggers `.github/workflows/release-on-tag.yml`.

## Release type definitions

Use semantic versioning: `vMAJOR.MINOR.PATCH`.

`patch`:

- Small stable fixes.
- Example: `v0.2.0-alpha.1` -> `v0.2.1`.
- Use for bug fixes, docs fixes, CI fixes, and no new capability.

`minor`:

- New stable backward-compatible capability.
- Example: `v0.2.0-alpha.1` -> `v0.3.0`.
- Use for new plugins, skills, agents, catalog features, or install behavior
  that does not break existing users.

`major`:

- Stable breaking release.
- Example: `v0.2.0-alpha.1` -> `v1.0.0`.
- Use when schemas, registry contracts, CLI behavior, or install layout break
  existing consumers.

`patch-alpha`:

- Prerelease patch.
- Example: `v0.2.0-alpha.1` -> `v0.2.1-alpha.1`.
- Use for small fixes that should remain marked experimental.

`minor-alpha`:

- Prerelease minor.
- Example: `v0.2.0-alpha.1` -> `v0.3.0-alpha.1`.
- Use for new capabilities that are still alpha. This is the usual choice for
  catalog/plugin waves.

`major-alpha`:

- Prerelease major.
- Example: `v0.2.0-alpha.1` -> `v1.0.0-alpha.1`.
- Use when preparing breaking changes that are not ready to call stable.

## Local release cut fallback

If GitHub Actions is unavailable, a maintainer can still cut a release from a
clean `main` branch:

```bash
make release-cut VERSION=v0.3.0-alpha.1
```

Optional:

```bash
RUN_E2E=1 make release-cut VERSION=v0.3.0-alpha.1
```

to include local Playwright smoke tests before tagging.

## Automated GitHub release publishing

When a `v*` tag is pushed, GitHub Actions workflow
`.github/workflows/release-on-tag.yml` automatically publishes a release
entry with generated notes.

Tags containing `-alpha`, `-beta`, or `-rc` are marked as prerelease.

## When to cut a release

Use two tracks:

- **Deploy track**: merge to `main` whenever changes are ready.
- **Release track**: push a `v*` tag only for meaningful milestones.

Cut a release when at least one applies:

- new skills are added (especially a grouped wave)
- user-facing behavior changes in web UI, install flow, or registry
- compatibility or governance changes need explicit communication
- external announcement or changelog checkpoint is needed

Do not cut a release for:

- typo-only or formatting-only changes
- small internal refactors with no user-facing impact
- routine maintenance commits that are not milestone-grade

Suggested cadence:

- continue frequent `dev -> main` merges
- cut prereleases (`vX.Y.Z-alpha.N`) on a fixed rhythm (for example every
  1-2 weeks) or when a feature wave is complete

## Release notes template

- Scope: CLI + modular registries and hub web routes.
- Install: include CLI install/usage commands.
- Site: include hub URL and key routes.
- QA: mention smoke E2E coverage (`/`, `/skills`, `/agents`, `/tools-mcp`).

## Deployment reference

See `docs/deploy-web-vercel.md` for Vercel project setup and branch
environment configuration.
