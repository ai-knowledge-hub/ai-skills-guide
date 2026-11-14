# Catalogue task acceptance — 12 November 2026

Issue: [#119](https://github.com/ai-knowledge-hub/ai-skills-guide/issues/119). Branch: `feature/catalogue-task-verification`, based on merged main `c4e80764109ca92acab0b9eb969da2a4bc5b6b27`.

The bounded expert/browser walkthrough passed after two fixes. It preserves the masthead, artwork, normal desktop/phone composition and plain link labels. This report records observable task outcomes; it is not evidence from a reader study.

## Findings fixed

1. **Zoom reading space.** Native Chrome 400% zoom left the two-row sticky masthead occupying most of the short layout viewport. At 320×180 and 1440×450, scrolling still left the header's bottom at 118px and 128px respectively. At 500px viewport height or less the masthead now scrolls with the page, anchor offsets and the artwork rail use the available space, mobile navigation can extend with the document, and filter panels fit the short viewport. Normal-height sticky behavior is retained.
2. **No-JavaScript copy affordance and source fidelity.** Expanded skill and agent instructions showed an enabled copy button without a working handler. Copy controls now remain disabled until hydration. Independent review also caught that selecting rendered Markdown would omit frontmatter, Markdown syntax and link destinations. A no-script note now points to a collapsed native source-text disclosure containing a read-only textarea with the exact released file. The test compares its complete value to SKILL.md/AGENT.md from each retained archive and checks keyboard selection of the entire source. Rendered instructions remain readable. Existing successful-copy feedback, rejection fallback and stale-attempt handling are retained.

The three focused baseline probes failed against the previous production build for the expected causes: two header geometries and an enabled no-JavaScript copy button. They pass after the fixes. A reduced-motion test initially targeted a non-focusable label; the final test targets the native disclosure summary instead. No product change or timeout increase was made for that test error.

## Evidence and bounded domain

The project inventory was generated with the failure-space discovery script (`/private/tmp/skills119-domain-inventory.md`). It includes dependency-store noise, so the model below uses the inspected web code and normative contracts rather than treating the inventory counts as application coverage.

| Confirmed boundary | Authority and observable outcome |
| --- | --- |
| Resource identity/content | Retained releases, exact module/ID/version and registry loaders supply content. Detail instructions, installation commands and optional companions must retain that identity. |
| Readiness and compatibility | `shared/contracts/operational-readiness-v3.md` and `catalogue-compatibility-v1.md` distinguish editorial declarations from current operational evidence. Labels and sample output cannot authorize execution or promote readiness. |
| Reader interaction state | Catalogue URL parameters own filter/history state; local storage holds an optional theme preference. Disclosure state and hydration are local browser state. Recovery must preserve browsing and readable content. |
| Clipboard boundary | `writeText` can resolve or reject. Feedback must match the result, and denial must leave selectable source available. No-JavaScript controls must not promise a working copy action. |
| External destinations | Family and companion anchors are optional. No provider operation, native-app discovery or external-site availability is established by following a catalogue link. |

These are confirmed from the shared contracts, `CatalogClient`, `SiteHeader`, `CopyInstructions`, `InstallCommands`, `ResourceFirstUse`, the companion mapping, and the existing release/evidence tests. Actual reader comprehension, assistive-technology behavior and native-agent adoption remain unproven by this exercise.

The failure model selects the interactions most likely to mislead or block a reader: resource form × supported first action; exact release × evidence scope; search/filter × detail/Back; clipboard outcome × readable fallback; layout height × sticky navigation/focus; width/theme × open controls; and script/motion preference × native disclosure behavior. Backend concurrency and provider effects are outside this content-only web change.

## Reader-task outcomes

| Required task | Observed outcome and executable evidence |
| --- | --- |
| Identify Skills on direct entry | Existing family-navigation cases identify Skills, distinguish its local Catalogue landmark from Publication/Harness, and preserve ordinary links without JavaScript. The expanded layout matrix includes direct entry to Home, all four catalogues and four details. |
| Find and use weekly-review instructions | New task searches the Skills catalogue, applies a runtime filter, enters weekly review, identifies inputs/output, opens and copies the procedure, injects clipboard rejection, compares the selectable fallback to the captured full source, checks the unverified helper, and returns to retained search/theme state. Existing exact-release copy tests remain passing. |
| Compare a template and a bundle | New task searches Plugins, inspects Performance Reporting's reference composition and absence of install commands, returns to search, then inspects Code Maintenance's sample, installed-directory requirement, generic target, bundled skill location, advisory hook and optional guide. A printed plan remains distinct from execution or merge authority. |
| Distinguish samples, live reads and unavailable writes | New task inspects the advertising plugin's offline fixture command and separate live-read credentials, confirms unresolved Mock lab/Live read evidence and unavailable Live write, then finds GA4 and selects its Claude command with Node 22 and native-discovery limits. No command or provider operation is executed. |
| Use an optional companion | Existing companion tests retain the exact Code Maintenance 0.2.0 guide and its reason, fence other versions, and show no related panel/jump link for unmapped resources in all four modules. The independent seven-pair Publication journey remains passing. |
| Recover from empty and missing results | New layout states include empty Skills results and a missing Plugin detail. Existing filter clearing, deep-link, reload, Back, theme/storage failure and missing-entry recovery cases remain passing. |
| Keyboard, zoom, motion and no-script reading | New checks cover short-window menu/filter reachability, no-script instruction controls for both SKILL.md and AGENT.md, reduced-motion navigation to an agent workflow and the layout/control states below. Existing copy failures, runtime choices, Escape/focus recovery and no-script navigation remain passing. Native zoom was separately exercised as described below. |
| Preserve package/compatibility invariants | Existing exact-release annotation, projection, retained-preview and installation-closure tests pass. Generated registries, retained release bytes, contracts and schemas have no diff. |

## Finite coverage and native zoom

- **66/66 selected route/width/theme states:** 11 routes × 320/390/1440px × dark/light, each at 900px height. Routes are Home; Skills, Agents, Plugins and Tools & MCP catalogues; weekly-review skill, weekly supervisor agent, Code Maintenance plugin and OpenAI Ads API client details; empty Skills results; missing Plugin detail.
- **24/24 selected open command states:** one generic command on each of four details across all six width/theme combinations, with inline fit checked. Skill and agent instructions are also expanded in those detail states.
- **24/24 selected catalogue filter states:** Runtime opens and returns focus on Escape across four catalogues and six width/theme combinations. The existing suite separately covers Domain/Tag/Runtime positioning, selections and resize boundaries.
- **44/44 selected mobile menu states:** open and Escape/focus recovery on all 11 routes at both phone widths in both themes. Horizontal document overflow and bounds of visible family, theme, filter, copy and recovery actions are checked.
- **2/2 selected short-layout geometries** and **1/1 short-window final-filter-action case** pass. The latter uses actual Tab order to reach Done, checks that its center is unobscured, then restores focus with Escape.
- **2/2 no-script instruction cases** expose disabled copy controls, readable guardrails and selectable exact source with frontmatter/Markdown preserved. **1/1 selected clipboard rejection** produces truthful feedback and the full selectable source. These are bounded dependency probes, not an exhaustive failure matrix.

Native Chrome zoom was operated through its actual keyboard/browser controls in a disposable Guest window. The toolbar visibly confirmed **200%** and **400%**; this was not CSS `zoom`, device scale or viewport emulation. At 400%, the fixed Home reading-space issue was rechecked, the mobile menu's final Tools & MCP link was reached by Tab and activated, and the Runtime filter's final Done action remained visible with its focus outline. At 200%, weekly-review instructions, copy-button focus, generic adapter choice and command-region focus were inspected. The command was also inspected at 400% and reflowed within the reading width. Native screenshots are attached to the implementation chat. Zoom was restored to 100% and the Guest window closed.

## Commands, build and captures

- `bash scripts/validate-skills.sh`, `go run ./cmd/registry-builder`, `bash scripts/validate-manifests.sh`: passed; generated data inspected with no registry/release/contract/schema diff.
- Web `pnpm lint`, `pnpm exec tsc --noEmit`, `pnpm build`: passed; **76 production pages** generated.
- `PLAYWRIGHT_BROWSER_CHANNEL=chrome pnpm test:e2e`: **84 passed (1.1m)** against `pnpm start --port 3000` after the review fix. The 14 new task cases run alongside the 70 existing tests, including exact retained-source equality and complete keyboard selection for both no-script instruction types.
- Tool versions: Node 25.8.2, pnpm 10.33.0, Next.js 14.2.11, TypeScript 5.5.4, Playwright 1.41.2, automated Chrome 157.0.8103.51. The native zoom check used installed Chrome through the app UI.
- Manually inspected 13 selected production captures: Home; all four catalogues; expanded agent instructions; plugin sample; tool evidence limits; open mobile menu; selected command; empty results; missing-entry recovery; and no-script instructions. After review, also inspected the exact raw-source view at 320px (`no-js-exact-source-320.png`), with no document overflow. Widths/themes and initial measurements are in `/private/tmp/skills119-captures.log`; images are in `/private/tmp/skills119-browser-review/`. An early headless Home capture had a duplicated painted header; an independent browser recheck showed the normal single masthead. This capture artifact was not treated as a product failure.
- Command logs are `/private/tmp/skills119-{before-fixes,focused,nojs-final,e2e,e2e-final,lint,types,validation,build}.log`. The first two retain the baseline failures and the corrected keyboard-test trigger as well as their verification history; `e2e-final` includes the exact-source review fix. Local Vercel Analytics retains its existing script 404; no application error was observed in normal browsing. Requested missing-entry routes return their expected not-found responses.

## Limits and handoff

This finite set is not exhaustive across every package, device, browser, zoom level or assistive technology. It establishes expert/automated task completion and inspected presentation, not how an unassisted visitor understands the site. Native VoiceOver/NVDA testing and reader research were not performed. Chrome's pinned Playwright-managed browser, Firefox and Safari were not run in this acceptance pass.

Package download/installation, actual Codex/Claude discovery, credential configuration, live providers and model evaluation were not exercised; #33/#52 retain that work. No new live sibling-site verification was necessary: targets/reasons were checked deterministically, with #113/#118's separately recorded destination reviews retained. No deployment, release, readiness promotion or analytics backend was added. The existing untracked planning document remains untouched. The production server is left on `http://127.0.0.1:3000` for local review.
