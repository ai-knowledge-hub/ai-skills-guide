# Versioning and Release

- Use semantic tags for stable skill snapshots.
- Prefer additive updates; avoid breaking output schemas without notes.
- Keep changelogs in PR descriptions.
- Mark deprecated skills clearly in folder README or `SKILL.md`.

## Release hardening checklist

1. CI green on `dev`:
   - skill structure validation
   - manifest schema validation
   - registry freshness check
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

## Automated release cut (recommended)

From a clean `main` branch, run:

```bash
make release-cut VERSION=v0.2.0-alpha.2
```

What this does:
- verifies clean working tree and `main` branch
- fast-forwards `main` from `origin/main`
- runs skill validation and registry generation checks
- runs web install, lint, and build checks
- creates and pushes the tag

Optional:

```bash
RUN_E2E=1 make release-cut VERSION=v0.2.0-alpha.2
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

- Scope: CLI + registry baseline and hub web MVP.
- Install: include CLI install/usage commands.
- Site: include hub URL and key routes.
- QA: mention smoke E2E coverage (`/`, `/skills`, `/skills/<sample>`).

## Deployment reference

See `docs/deploy-web-vercel.md` for Vercel project setup and branch
environment configuration.
