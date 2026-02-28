# Roadmap

## Status

- v0.1 delivered:
  - 6 practical skills from the guide
  - shared conventions and validation
  - CI and contribution templates
- v0.2 delivered:
  - stable `skill.yaml` and `registry/index.json` contracts
  - deterministic registry generation and validation in CI
  - runtime-aware CLI install paths (`codex`, `claude`, `generic`)
- v0.3 delivered (Hub POC):
  - public catalog is live at `https://skills.ai-knowledge-hub.org`
  - Next.js hub routes and install UX are live in `apps/web`
  - web lint/build + Playwright smoke checks are in CI
  - release-cut and tag-publish automation are in place
- Catalog expansion in progress:
  - current catalog includes marketing, adtech, QA, and BI-oriented skills
  - BI-related additions now include:
    - `adtech/weekly-performance-review-bi`
    - `adtech/dashboard-generator`
    - `adtech/dashboard-qa-checker`
    - `adtech/executive-narrative-writer`
- Current focus: v0.4 trust, governance, and release maturity.

## v0.2 (Foundation)

### Goals

- Define stable skill and registry contracts.
- Keep repository contribution flow simple (GitHub PR first).
- Prepare CLI and website to consume the same registry model.

### Deliverables

- `skill.yaml` schema (machine-validated).
- `index.json` registry schema (machine-validated).
- CI checks for schema validation and package integrity.
- Runtime-aware CLI installs (`codex`, `claude`, `generic`).

### Exit Criteria

- Every non-legacy skill includes valid `skill.yaml`.
- Registry index can be generated deterministically from source.
- CLI can install a skill from local source using schema metadata.

## v0.3 (Hub POC)

### Goals

- Launch a public catalog website with install UX.
- Keep cost near zero and reduce infrastructure complexity.
- Enable discoverability of marketing/adtech skills.

### Deliverables

- Modular monorepo layout for web, CLI, and shared schemas.
- Website MVP:
  - browse and search skills
  - skill detail pages
  - runtime-specific install commands
- Static registry hosting (no backend required for initial launch).
- Marketing taxonomy:
  - ads-ops
  - creative-ops
  - measurement
  - seo-sem
  - lifecycle-crm
  - compliance

### Exit Criteria

- Users can discover skills and copy install commands.
- Registry updates are visible on the website after publish.
- External contributor can submit a skill PR end-to-end.
- Status: Delivered.

## v0.4 (Hub Beta)

### Goals

- Improve trust, quality signals, and operational safety.
- Introduce optional managed features only when needed.

### Deliverables

- Verified badges and quality scoring signals.
- Version history and deprecation migration UX.
- CLI remote install/update from registry URL.
- BI workflows hardening:
  - dashboard generation quality checks
  - executive narrative consistency checks
  - weekly BI review reliability baselines
- Optional backend + database only if product needs exceed static flow.

### Exit Criteria

- Stable release workflow for skills and CLI.
- Clear governance for review, deprecation, and compatibility.
- Consistent release cadence with milestone-quality tags.

## Decision Gates

1. Before backend/database:
   - Confirm static registry + static website is insufficient.
2. Before authentication:
   - Confirm non-GitHub contribution or user accounts are required.
3. Before paid infrastructure:
   - Confirm usage volume exceeds free-tier constraints.
