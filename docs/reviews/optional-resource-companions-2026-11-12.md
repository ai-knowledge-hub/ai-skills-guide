# Optional resource companions — 12 November 2026

Issue: [#118](https://github.com/ai-knowledge-hub/ai-skills-guide/issues/118). Branch: `feature/optional-resource-companions`, based on main `2725bad08eb1cb67f032a9c7eaae50509c2a2737`.

## Reader outcome and scope

The existing editorial mapping now supports Publication and Harness destinations for all four resource modules. Home distinguishes the source site, and detail pages offer a plain optional link with its task benefit. Pages without a reviewed relationship have no empty panel or jump link. The seven existing Publication/resource relationships are preserved; no relationship is generated from tags or required for browsing, reading instructions or installing files.

The accepted new relationship is [Harness: Verify a small code change with an agent](https://harness.ai-knowledge-hub.org/learn/ai-engineering/verify-small-code-change) with `plugins:engineering/code-maintenance-plugin@0.2.0`. Its guide was updated on 10 November and describes broken/valid inputs, before-and-after checks and evidence for the reviewer. It explicitly uses the plugin's 0.2.0 first-use example, which prints a plan rather than running the proposed checks or authorizing merge. The link context preserves that distinction. Matching a different release hides this version-specific companion rather than carrying its claims forward. Existing publication relationships remain conceptual, with their original review provenance.

The guide returned HTTP 200 at its declared canonical URL and opened in the browser with the expected title. All three existing Publication destinations also returned HTTP 200 with the expected canonical URLs and article titles. Public-page verification was separate from deterministic PR tests; no forms, downloads or provider operations were invoked. Skills does not request these pages at runtime. No sibling content or deployment was changed.

## Validation

- Structure validation, registry generation and manifest validation passed. Generated registries, retained releases, schemas and readiness contracts have no diff.
- Web lint, TypeScript and production build passed; the production build generated 76 pages.
- `PLAYWRIGHT_BROWSER_CHANNEL=chrome pnpm test:e2e` against the production server: **70 passed (42.4s)**. Five new checks cover curated provenance and real resource identities, exact-release matching and module namespaces, Home-to-plugin-to-guide navigation, unmapped resources in each module, and phone/keyboard/no-JavaScript access. The independent seven-pair Publication journey and existing evidence/installation regressions still pass.
- Inspected the plugin related-reading section at 1440px, 390px and 320px in dark mode and at 320px in light mode; inspected the new Home relationship at desktop in light mode. No horizontal overflow or clipped links were observed. Source labels, context and ordinary links retain the existing hover and keyboard focus treatment, with no decorative arrows.
- Toolchain: Node 25.8.2, pnpm 10.30.1, Next.js 14.2.11, TypeScript 5.5.4, Playwright 1.41.2, local Chrome 157.0.8103.51.

Logs are `/private/tmp/skills118-{validation,lint,types,build,e2e}.log`. Browser captures and snapshots are retained under `/private/tmp/skills118-browser-review/`. The production preview runs at `http://127.0.0.1:3000`. The existing local Vercel Analytics script returns 404; no application error was observed.

This is local expert/browser verification, not reader research, assistive-technology certification, native-client adoption or provider proof. Broader integrated acceptance remains #119. The existing untracked planning document is preserved. No push, merge or deployment is included.
