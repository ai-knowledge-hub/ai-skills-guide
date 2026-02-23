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

## Release notes template

- Scope: CLI + registry baseline and hub web MVP.
- Install: include CLI install/usage commands.
- Site: include hub URL and key routes.
- QA: mention smoke E2E coverage (`/`, `/skills`, `/skills/<sample>`).

## Deployment reference

See `docs/deploy-web-vercel.md` for Vercel project setup and branch
environment configuration.
