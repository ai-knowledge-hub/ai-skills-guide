# Roadmap

## Status

- v0.1 delivered:
  - 6 practical skills from the guide
  - shared conventions and validation
  - CI and contribution templates
- v0.3 in progress:
  - Hub website MVP routes are live in `apps/web`
  - Playwright smoke E2E is wired into CI
  - reusable `adtech/playwright-agentic-e2e` QA skill added
- Current focus: evolve from companion repo to public skills hub.

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

## v0.4 (Hub Beta)

### Goals

- Improve trust, quality signals, and operational safety.
- Introduce optional managed features only when needed.

### Deliverables

- Verified badges and quality scoring signals.
- Version history and deprecation migration UX.
- CLI remote install/update from registry URL.
- Optional backend + database only if product needs exceed static flow.

### Exit Criteria

- Stable release workflow for skills and CLI.
- Clear governance for review, deprecation, and compatibility.

## Decision Gates

1. Before backend/database:
   - Confirm static registry + static website is insufficient.
2. Before authentication:
   - Confirm non-GitHub contribution or user accounts are required.
3. Before paid infrastructure:
   - Confirm usage volume exceeds free-tier constraints.
