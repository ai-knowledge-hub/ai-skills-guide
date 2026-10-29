# Catalogue rebranding · 29 October 2026

Tracking: epic [#75](https://github.com/ai-knowledge-hub/ai-skills-guide/issues/75), foundations #76, shell #77, Skills slice #78, homepage #79, rollout #80 and verification #81. Branch: `codex/catalogue-rebrand`.

## Selected foundation

The owner selected the independent publication's Op Art/editorial language as the shared product-family base on 24 October. The brand studio's `brand/GUIDELINES.md` and `brand/tokens.json` version 0.1.0 record that decision. Source hashes are captured in `apps/web/brand/baseline.json`; `tokens.reference.json` preserves the machine-readable source. These are portable audit records, not runtime imports from a sibling checkout.

Use Arial/Helvetica/sans-serif, system monospace for code, bold headings, exact light/dark surface roles, calm reading zones and purposeful optical artwork. No new font files, logos, names or general-purpose colour accents have been introduced. `lib/brand.ts` centralises the working identity, catalogue descriptor, module navigation and verified family/source links.

The current publication type and palette still match the captured studio baseline. Its later compressed WebP delivery copies are reused byte-for-byte and recorded in `public/brand/provenance.json`; originals remain intact in the publication. This manifest does not assert missing generation/rights evidence or authorise redistributing an asset pack.

## Composition for review

Visual thesis: bold optical geometry gives the catalogue an entrance; quiet typography and solid surfaces keep browsing and installation clear.

Content plan: branded entrance and Explore skills action; short orientation; the four catalogue destinations with registry-derived counts; selected skills/plugins; usability guidance; recently added entries; family/source links.

Interaction thesis: stable titles and artwork, clear hover/focus cues, saved reading theme, native mobile disclosure navigation, keyboard-operable checkbox filters and truthful copy feedback. Artwork is intentionally static in this review slice. There is no hidden animation loop or WebGL dependency. Motion may be introduced only with a selected source-matched contract, pause/reduced-motion/fallback/lifecycle checks and stable text.

The catalogue frontispiece adapts the publication Library's community artwork. It is a shared browsing treatment rather than a new mapping of Skills, Agents, Plugins and Tools to the publication's editorial categories. The title positions and responsive layouts are catalogue adaptations, not changes to the approved publication entrance.

## Implementation boundaries

Registry entries, availability/readiness classifications, package contents, CLI/install strings, route IDs and package links remain unchanged. Existing inferred classifications remain explicitly inferred. Operational readiness work stays under #33/#52 and cannot be claimed complete through visual redesign.

The shared shell and neutral component styles necessarily reach all routes. Other module templates retain their specialised data/sections. Their final visual rollout remains subject to review of the homepage, Skills catalogue and representative Skill detail. No deployment or remote push is included.

## Verification

Run from `apps/web`:

```sh
pnpm exec tsc --noEmit
pnpm lint
pnpm test:e2e
pnpm build
```

Avoid running build and dev against the same `.next` directory simultaneously. Browser smoke uses its configured port 3000; the visual review server uses port 3050. Stop the visual dev server before a build or the smoke run.

Local screenshots belong in ignored `output/playwright/`. Verification results and outstanding review work are recorded below after checks complete.

## First review build: results

- TypeScript: `pnpm exec tsc --noEmit` passes; the production build also validates types.
- Lint: `pnpm lint` passes without warnings/errors.
- Browser suite: all 12 tests pass with `PLAYWRIGHT_BROWSER_CHANNEL=chrome pnpm test:e2e` (16.1 seconds). This includes the original seven smoke tests, exact clipboard contents, combined filters/deep links/reload/Back, reset/empty recovery, theme persistence, denied storage/clipboard and 320px reflow/keyboard menu checks.
- Host limitation: the pinned Playwright 1.41.2 Chromium build crashes before browser startup on this machine. The optional `PLAYWRIGHT_BROWSER_CHANNEL=chrome` override uses installed Chrome with a temporary test profile; CI's default managed Chromium configuration is preserved. No dependency upgrade was needed.
- A navigation assertion originally raced completion of the detail transition. Waiting for the destination URL before Back resolves that test race; no separate custom history store was retained.
- Production build: `pnpm build` passes, generates 76 static pages and retains the four server-rendered filterable catalogues.
- Source integrity: all four imported WebP copies match their recorded SHA-256 values; registry/package/CLI sources were not edited.
- In-app browser: inspected desktop and 390px mobile, dark/light. Actual clipboard value matches the displayed Codex command. Static artwork needs no pause/failure lifecycle controls. Final screenshots are local, ignored review evidence under `output/playwright/`.

This is a local review build. Homepage/catalogue artwork adaptation is proposed for this product; it is not recorded as owner-approved. Final visual rollout, motion if selected, and full release verification remain under #79–#81. Nothing has been pushed or deployed.
