# A quieter practical companion

<!-- markdownlint-disable MD013 -->

This iteration follows the completed rebrand (#75–#81). It is based on merged
`dev` at `96cc5f9` and is separate from the open published-install build in #97.
The owner requested a quieter landing page, the first supplied folded-field
artwork, a narrower rail, and links between articles and relevant skills/tools.

## Composition and content

The visual thesis is a calm reading field with one narrow, high-contrast optical
edge. Shared typography, paper/ink themes, navigation and the existing working
identity remain. The publication's full-width tunnel entrance is replaced by a
compact introduction and immediate routes into all four catalogue modules.

Content order is introduction/action → registry-derived module routes → three
curated reading paths → a concise explanation of availability labels. The older
featured-resource arrays and homepage-only hero styles are removed. The curated
reading paths replace that selection mechanism; package contents remain intact.

Interaction follows ordinary links, visible focus/hover cues and the existing
sticky rail. The selected artwork, text and figures remain static. This calmer
brief does not add an entrance animation, a new renderer or a motion dependency.

Desktop uses `clamp(220px, 24vw, 380px)` for the shared home/catalogue rail,
contained by `main` so it stops before the footer. Following the owner's
2 November crop selection, its full-height crop uses `81% center` registration
to feature the right-hand oval swirl rather than the middle folds. The source
artwork and delivery files are unchanged. At 1440 CSS pixels the rail is
approximately 346px, increased from the initial 259px crop,
compared with approximately 410px at a 900px-high viewport in the prior design.
Compact home layouts use a 100px artwork band (140px at tablet widths); catalogue
pages retain their compact heading treatment with the new artwork.

## Selected artwork

The first owner-supplied attachment is preserved byte-for-byte as
`apps/web/brand/artwork/folded-field-original.png` (1672 × 941; 1,979,031 bytes;
SHA-256 `699561fd7d949d9c49b12bf8e1c83b9ae90b880bc28072ad6bbdb8b0b7d10ac2`).
Its decoded pixels match the art chat's `05-embedded-folded-field` original,
although the two PNG byte encodings differ. The source record retains the known
generation prompt, tool, original hashes and authoring method.

The two full-frame WebP delivery copies use proportional resizing and quality 84:
960px / 113,958 bytes and 1600px / 197,244 bytes. CSS performs display cropping;
the raw figure contours and optical geometry were not redrawn or animated.
The PNG source is outside `public`, so it is not downloaded by the page.
Earlier publication artwork remains archived in the repository but is no longer
referenced by active page styles. There is no build/runtime dependency on the
sibling art or publication checkouts.

## Curated article relationships

`apps/web/lib/publicationLinks.ts` is the single editorial mapping. It contains
three published guides and seven explicit `(module, package ID)` relationships.
Home resolves current names and badges through the existing registry loaders;
a missing referenced package fails the build rather than producing a dead link.
Mapped Skill/Tool detail pages expose a Related reading shortcut and section.
Unmapped entries receive neither. These links are independent of compatibility
notes, release selection, evidence expiry and runtime admission.

The destinations returned HTTP 200 on 1 November 2026 and were reviewed alongside
the publication source and the referenced package instructions/contracts:

| Article | Catalogue relationship and basis |
| --- | --- |
| [The Agent Architect's Playbook](https://ai-knowledge-hub.org/library/agent-architect-playbook-building-ai-skills-marketing-adtech/) | The guide develops a weekly performance-review procedure and evaluation requirements. Related examples: `marketing/meta-google-weekly-performance-review` and `marketing/ai-output-eval-scorecard`. The packaged review is an adaptation example, not a claim that its code is the guide's illustrative skill verbatim. |
| [The Muscles of the Machine](https://ai-knowledge-hub.org/library/muscles-of-the-machine-tools-mcp-cli-first-principles-marketing/) | The guide covers bounded retrieval, explicit contracts, caller scope, cost limits and MCP/CLI delivery. GA4 and BigQuery are related read-only implementations to inspect. They are not presented as the article's illustrative `marketing report` command or a ready-made deployment. |
| [Secure Marketing Agents](https://ai-knowledge-hub.org/library/building-secure-marketing-agents-ad-platform-authentication-architecture/) | The article's Companion material explicitly identifies the authentication-review skill, policy-gate skill and executor contract. Their current catalogue pages are linked with their setup/availability information intact. |

The publication repository and live article pages are not changed by this PR.
The catalogue landing page connects each guide to its related resources; the
resource pages link back to the guide and explain the relationship. Additional
article-side deep links can be edited in the publication separately.

To maintain a relationship, review the article and package purpose, record a
specific connection, and update the module-qualified mapping. Topic similarity
does not establish operational readiness. Dates record editorial review and do
not introduce an expiry rule. All page data is rendered from repository content;
there is no article scraper, live API lookup or new server service.

## Verification

Commands use Node 20.19.4 from `apps/web`:

```sh
pnpm build
pnpm start --hostname 127.0.0.1 --port 3000
PLAYWRIGHT_BROWSER_CHANNEL=chrome pnpm exec playwright test
```

The existing browser suite now follows all seven landing-page resource links,
operates the Related reading shortcut, checks the reviewed article destination,
and preserves Documentation / Setup required labels. It also checks an unmapped
entry, includes Home in rail/footer containment, and checks that Home and all
four catalogues request only the new artwork appropriate to their layout.
External article endpoints are checked separately from deterministic PR tests.

Final production results on 1 November 2026:

- Build, lint and types pass; 76 static pages generated.
- All 33 browser tests pass against `next start` in 27.2 seconds.
- 10 representative routes/states × 6 widths × 2 themes = 120 views, with zero
  sampled horizontal overflows, clipped headings, theme mismatches or offscreen
  home primary actions. Widths: 320, 390, 768, 1024, 1440 and 1920 CSS pixels.
- Each of the 60 home/catalogue matrix views requests exactly one active artwork
  asset. The separate regression covers both active layouts for all five routes.
- Four home/catalogue × theme cases remain usable with artwork requests blocked
  and reduced motion active (zero animations). Native no-JavaScript navigation
  follows Home → resource → Related reading, with all three install command texts
  still present. Keyboard Skip to content reaches Explore skills with visible
  2px focus.
- All three article destinations were independently checked live (HTTP 200).
  Resource/reading navigation in PR tests remains deterministic and local.

Review captures:

- [Desktop dark](reviews/companion-landing-2026-11-01/home-1440-dark.jpg) · [Desktop light](reviews/companion-landing-2026-11-01/home-1440-light.jpg)
- [Phone dark](reviews/companion-landing-2026-11-01/home-390-dark.jpg) · [Phone light](reviews/companion-landing-2026-11-01/home-390-light.jpg)
- [Reading paths](reviews/companion-landing-2026-11-01/home-paths-1440-dark.jpg) · [Related-reading panel](reviews/companion-landing-2026-11-01/related-reading-1440-light.jpg)
- [Catalogue dark](reviews/companion-landing-2026-11-01/skills-1440-dark.jpg) · [Catalogue light](reviews/companion-landing-2026-11-01/skills-1440-light.jpg)
- [Measurements and bounded edge checks](reviews/companion-landing-2026-11-01/measurements.json)

The React check retained server-rendered content, parallel independent registry
loads, module-qualified relationships and current registry-derived badges. It
adds no client effect, dependency, live content request or theme-state writer.

Chrome coverage is bounded; it is not a complete assistive-technology or
cross-browser audit. Owner review of the rendered composition remains the PR
review step. No provider verification or production promotion occurs.

## Oval crop follow-up — 2 November 2026

The owner selected the oval swirl shown in their screenshot. The shared desktop
rail now uses `81% center` instead of the original `44% center`, bringing that
part of the existing artwork into view on Home and all four catalogue pages.
The owner subsequently approved widening the shared desktop rail from 18vw
(280px maximum) to 24vw (380px maximum), closer to the publication Library's
proportions. At 1440px this increases the visible artwork from about 259px to
346px. Scaling, compact layouts, source pixels and delivery hashes are unchanged.

The production build, lint/types and three existing browser regressions pass
(open filter menus, rail/footer containment and active-layout artwork requests).
A separate local Chrome check covers five routes × two themes × eight viewport
sizes: 1000×768, 1024×768, 1440×900, 1600×900, 1920×1080, 390×844, 768×1024 and
1024×1440. All 80 views reflow without horizontal or filter-grid overflow; all
50 desktop cases keep the selected oval's source focal point inside the rail's
central half. No page exceptions were observed.

[Home, light](reviews/oval-artwork-2026-11-02/home-1440-light.jpg) · [Skills, dark](reviews/oval-artwork-2026-11-02/skills-1440-dark.jpg)

## Shared Library frame — 2 November 2026

The owner approved aligning the companion's frame with the publication Library.
The masthead now spans the viewport, with a 27px desktop wordmark and the four
catalogue navigation links. “The open-source catalogue” moves into Home's
introduction. Home retains its smaller “Put the ideas to work” heading.

Catalogue pages use a larger white title plate, overlapping the desktop rail by
48px. Introductions, filters, result counts, entries and the footer share the
reading edge. The footer border spans the viewport; its content follows the
rail on catalogue routes and retains the normal page gutter on detail pages.
Compact and portrait catalogues use a full-width artwork band with a 32px title
overlap and a solid reading surface below it. The band is 180px tall on phones
and 240px on larger compact layouts; Home retains its shallow existing band.

The selected desktop oval crop (`81% center`), 24vw rail (380px maximum), artwork
bytes, module navigation, filters, install commands, article relationships and
readiness information remain in place. This changes layout and introductory
placement; it adds no runtime request, dependency or theme-state writer.

Verification against the local Node 20 production build:

- Build, lint/types and Markdown/diff checks pass.
- All 33 existing browser tests pass, including keyboard navigation, readable
  but write-blocked theme storage, filters, copy feedback and article links.
  The rail/footer regression also passes after the final compact-footer fix.
- 80 artwork/reflow views cover Home and the four catalogues in both themes;
  the selected oval remains inside the rail's central half in all 50 rail views.
- 144 additional views cover Home, the four catalogues and BigQuery details at
  12 viewport sizes, from 320px through 1920px, including wide portrait screens.
  No sampled horizontal overflow or clipped heading text; catalogue
  introductions, filters, entries and footer retain their shared reading edge.
- The original artwork and both WebP hashes are unchanged. React review retains
  server-rendered content, the existing theme/menu logic and semantic links.

Chrome verification is bounded; this is not a cross-browser accessibility audit.

[Home, light](reviews/oval-artwork-2026-11-02/home-1440-light.jpg) · [Skills, dark](reviews/oval-artwork-2026-11-02/skills-1440-dark.jpg) · [Phone catalogue](reviews/oval-artwork-2026-11-02/tools-390-light.jpg) · [Footer](reviews/oval-artwork-2026-11-02/footer-1440-light.jpg)
