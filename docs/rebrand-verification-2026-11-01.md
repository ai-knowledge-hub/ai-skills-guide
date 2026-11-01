# Final rebrand verification · 1 November 2026

<!-- markdownlint-disable MD013 -->

Tracking: #81, parent #75. Baseline: merged `dev` at `d096399` (PR #95).
Branch: `codex/rebrand-final-verification`. This records verified behavior and
review evidence for the existing publication-family design, not new visual
approval or live-provider readiness. No production deployment was performed.

## Findings resolved

| Finding | Result and regression |
| --- | --- |
| A saved light theme reverted to dark on direct missing-entry pages | Reproduced against the production build. `SiteHeader` now restores the saved preference before paint on route entry, retaining the existing early head script and storage-failure fallback. Both themes are checked across four missing module entries and the global 404, with Home recovery. |
| Desktop catalogues downloaded the unused compact artwork as well as the rail | Reproduced in actual resource entries. The desktop pseudo-element now clears its background image. Eight module/layout cases check only the active artwork is requested, avoiding 150,080 unnecessary bytes per cold desktop catalogue load. Crop, geometry and source bytes are unchanged. |
| No favicon was declared | The publication's existing 216-byte SVG is copied unchanged, declared in metadata and added to asset provenance. This introduces no new mark or logo design. |
| Route restoration overwrote a session choice when storage was readable but writes failed | Storage initializes the in-memory theme once. Toggles and cross-tab storage events update that session value, which later client-side routes reapply before paint. Failed persistence no longer restores a stale saved value. The initial missing-entry hydration fix remains covered. |

The React check retained separate route-restoration and storage-listener effects,
cleanup for the existing listener, guarded browser storage and unchanged
navigation semantics. There is no new dependency, provider call or runtime
compatibility service.

The storage follow-up models three confirmed writers in `SiteHeader`: initial
restore, local toggle and cross-tab events. The selected theme and the rendered
toggle label are the observable oracle; stale readable storage must not override
a later local choice. The added tests cover both failed-write directions (2/2),
catalogue/detail/Back transitions, reload restoring the persisted value, and one
real two-tab change followed by navigation. `performance.timeOrigin` confirms the
first transition retains the document. This bounds the session guarantee to
client-side navigation; native links and reloads create a new document.
Both final failed-write cases fail on the reviewed implementation at the first
client-side route change and pass with the fix. The follow-up production build
passes lint/types; all 32 browser tests pass in 21 seconds, including the original
ten missing-route/theme cases. The earlier screenshot matrix was not recaptured
for this state-management-only change.

## Verification results

| Boundary | Evidence |
| --- | --- |
| Build, types and lint | Node 20.19.4 production build passes, including lint/type checks and 76 generated static pages. The four filterable catalogue routes retain the existing server-rendered behavior. |
| Functional regression suite | 32 browser tests pass against `next start`, using local Chrome. This includes exact controlled clipboard contents, failure feedback, URL/filter/reload/Back behavior, both themes, read/write storage failure, real cross-tab synchronization, keyboard navigation, open-menu viewport bounds, rail/footer containment, static readiness and version-note display. |
| Route/theme/reflow matrix | 11 representative routes/states × 6 widths × 2 themes = 132 views. Zero page overflows, heading-bound failures, theme mismatches or unnamed visible controls in the bounded check. |
| Actual browser zoom | Native Chrome Guest controls visibly reported 200% and 400%. At 400%, keyboard Skip to content reached search; Domain selection changed the URL to `?category=adtech`, and Escape returned focus to Domain. Zoom was reset to 100% and the Guest window closed. This was native zoom, not a device-scale-factor or pinch-zoom substitute. |
| Focus and keyboard | Separate keyboard-only flows exercised skip/search/filter selection/Escape, mobile menu navigation, and Copy. Focus-visible outline was present; copied text exactly equalled the displayed command. |
| Decorative artwork failure | Four home/catalogue × dark/light cases aborted WebP requests. Titles, primary navigation and search remained usable, without overflow. |
| Reduced motion | Home and catalogue cases report the reduced-motion query active, zero animations and zero-duration ordinary transitions. The selected artwork is static; renderer pause/seam/hidden-loop controls are not applicable. |
| Without JavaScript | Native links navigated Home → Skills → a detail page; all three install command texts remained readable. Interactive filtering, theme switching and automatic Copy require JavaScript and are not claimed to work without it. |
| Loading baseline | Phone Home selects `entrance-480.webp` (35,548 bytes); desktop Home selects `entrance-1600.webp` (165,094 bytes). Compact/rail catalogue assets remain the approved compressed originals. No artwork animations were observed. Max sampled CLS was 0 through network-idle capture; this is a local sample, not field performance data. |
| Asset identity | All six delivered brand files match their recorded hashes, lengths and original publication bytes. Runtime/build code has no dependency on the sibling publication checkout. |
| Readiness and technical content | Detail evidence retains instruction/helper separation, Not verified executable claims, setup requirements, limitations and maintainer-reviewed version notes. Rebrand verification grants no credential, execution or compatibility authority. |

The viewport widths are 320, 390, 768, 1280, 1440 and 1920 CSS pixels.
The matrix includes Home; all four catalogue families; representative Skill,
Agent, Plugin and Tool details; empty results; and a missing-entry page.
Routes, assets and the bounded results are recorded in
[measurements.json](reviews/rebrand-2026-11-01/measurements.json).

The screenshot matrix was refreshed after the theme and favicon fixes. The final
non-rendered background cleanup preserves that geometry and was verified by the
subsequent 29-test production run (26 seconds), including eight artwork-request
cases. Native zoom was checked separately against the same layout geometry.

Configured text-token contrast spot checks were 18.89:1 / 10.12:1 for primary /
secondary dark text and 17.03:1 / 5.79:1 for light text. On the fixed white title
plates, the corresponding values were 17.93:1 / 6.10:1. These checks do not
constitute a complete accessibility conformance audit.

## Visual evidence

Contact sheets are browser-rendered arrangements of unaltered page screenshots.
They are review artifacts, not production artwork. Original viewport captures
remain in ignored `output/playwright/rebrand-final-20261101/` locally.

- [Desktop dark](reviews/rebrand-2026-11-01/contact-1440-dark.jpg)
- [Desktop light](reviews/rebrand-2026-11-01/contact-1440-light.jpg)
- [Phone dark](reviews/rebrand-2026-11-01/contact-390-dark.jpg)
- [Phone light](reviews/rebrand-2026-11-01/contact-390-light.jpg)
- [Technical detail disclosures in both themes](reviews/rebrand-2026-11-01/contact-panels.jpg)

The owner selected the publication foundation and requested the viewport-edge,
full-height rail in PR #82. This pass preserves that decision, the static
entrance and catalogue motifs, working identity, content, routes and package
links. It does not infer a new visual sign-off from passing technical checks.

## Reproduce

From `apps/web`, with Node 20 first on PATH:

```sh
pnpm build
pnpm start --hostname 127.0.0.1 --port 3000
```

In a second terminal at the same directory:

```sh
PLAYWRIGHT_BROWSER_CHANNEL=chrome pnpm exec playwright test
```

The existing Playwright configuration reuses the production server on port 3000.
Do not run `next dev` and `next build` against the same `.next` tree concurrently.
CI retains its managed Chromium default; Chrome is the established local fallback.

Optional matrix capture uses the Playwright skill's CLI. Run from an evidence
output directory, open the intended review origin first, then load the portable
capture entry point (replace the path with the checkout's absolute path):

```sh
PWCLI="$HOME/.codex/skills/playwright/scripts/playwright_cli.sh"
"$PWCLI" -s=rebrand81 open http://127.0.0.1:3000 --browser chrome --headed
"$PWCLI" -s=rebrand81 run-code --filename /path/to/checkout/apps/web/scripts/capture-rebrand-evidence.js
```

The script returns measurements and writes viewport PNGs in the CLI session's
working directory. Requested and observed themes are recorded independently;
otherwise the 404 defect could disappear from a report by being grouped under
the wrong theme. Native zoom was checked separately using browser controls.

## Closeout map and limits

| Issue | Evidence / remaining action |
| --- | --- |
| #76 foundations | Tokens, system fonts, identity, asset provenance and source-byte checks are implemented and rechecked. |
| #77 shell and themes | Shared navigation/footer, active nested routes, keyboard menu and storage fallback are rechecked; this pass fixes error-route theme restoration. |
| #78 Skills slice | Real filters, empty/recovery paths, copy, technical text and narrow layouts are covered. |
| #79 homepage | Static responsive entrance, readable primary action, server-rendered links and failed-image behavior are checked; no new animation or artwork choice is proposed. |
| #80 other modules | All module templates build; representative module-specific sections, limits, commands and links remain visible in the evidence. |
| #81 verification | Technical evidence is complete locally. PR/CI review and integration remain; the parent epic should link this record rather than the old local-only checkpoint. |
| #75 epic | Reconcile ticket status after the verification PR is reviewed/merged; owner visual acceptance stays separate from automated results. |
| #52 runtime release proof | Remains separate. No real provider credentials, live sandbox calls or cross-platform clean-client operational certification were exercised here. |

This pass uses Chrome, not a full Safari/Firefox or assistive-technology matrix.
The production server locally reports a 404 for Vercel's platform-provided
analytics script; that service is unavailable off Vercel and was not changed.
Expected missing-entry document responses remain 404. Asset/layout metrics cover
the named local scenarios, not all networks or all catalogue content permutations.
