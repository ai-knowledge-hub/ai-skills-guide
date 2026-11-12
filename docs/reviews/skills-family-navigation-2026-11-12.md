# Skills identity and family navigation — 12 November 2026

Issue: [#113](https://github.com/ai-knowledge-hub/ai-skills-guide/issues/113). Branch: `feature/skills-family-navigation`, based on merged main `97479e13017b9a14c7709a27d8cc139ae5e7c863`.

## Reader flow

Visitors entering Home, a catalogue or a detail page can identify this site as Skills, read its reusable-resource role, browse the four local modules and optionally visit Publication or Harness. The AI Knowledge Hub masthead, artwork assets and catalogue composition are retained.

Family labels, canonical URLs and short roles live in `apps/web/lib/brand.ts`. The current Skills destination uses `/` so local review and ordinary no-JavaScript navigation stay local. The separate Catalogue landmark retains its module links. Header and footer family links use ordinary anchors; the footer explains the three sites' roles. Home describes inspecting, adapting or configuring a resource without making Publication a prerequisite.

Publication ownership and roles were checked against [its current home page](https://ai-knowledge-hub.org/). Both that URL and [Harness](https://harness.ai-knowledge-hub.org/) opened in the browser at their canonical destinations, with their expected page titles; no forms were submitted. Neither site is fetched at runtime by Skills.

## Validation

- `bash scripts/validate-skills.sh`, `go run ./cmd/registry-builder` and `bash scripts/validate-manifests.sh` passed. Generated registries, retained releases, schemas and readiness contracts have no diff.
- Web `pnpm lint`, `pnpm exec tsc --noEmit` and `pnpm build` passed; production build generated all 76 pages.
- `PLAYWRIGHT_BROWSER_CHANNEL=chrome pnpm test:e2e`: **65 passed**, using the production server at `http://127.0.0.1:3000` (50.9s final run).
- The initial run passed 63/64, with a timeout in the existing six-resource installer case. That case passed in isolation with tracing (12.0s), then passed in the complete final run. No product fix or timeout increase was needed. New viewport checks are separated by theme to bound each test's duration.
- Tool versions: Node 25.8.2, pnpm 10.30.1, Next.js 14.2.11, TypeScript 5.5.4, Playwright 1.41.2, Chrome 157.0.8103.51.

## Browser inspection

Inspected Home, `/skills` and `/skills/marketing/meta-google-weekly-performance-review` at 320px, 390px and 1440px in both themes. Family links stay visible, the current site has a non-colour underline, and the local catalogue menu remains distinct. No document overflow or clipped actions were observed. The automated checks also cover 761px and the sticky artwork/footer boundary.

Keyboard checks cover opening the mobile menu, tabbing to catalogue links, Escape with focus returned to Menu, visible focus on family links, and closing after local navigation. No-JavaScript checks cover direct detail entry, visible family links, the native mobile disclosure and catalogue-to-detail discovery at 320px and desktop. Existing search, filters, Back/reload, themes, copy feedback, publication pairs and compatibility checks pass.

Local captures and viewport measurements are in `/private/tmp/skills113-visuals/`; command logs use `/private/tmp/skills113-*.log`. Early headless desktop captures omitted painted sticky content; fresh captures and an independent browser inspection confirmed the masthead and links render normally. Local Vercel Analytics reports its existing `/_vercel/insights/script.js` 404; no application error was observed.

This is an expert/browser walkthrough, not reader research, screen-reader certification, native-client adoption or provider verification. Broader integrated task acceptance remains #119. No sibling repository, deployment, release or readiness state was changed. The existing untracked planning document is preserved.
