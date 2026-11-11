# Restick Studio redesign: direction record

Mode: **Operate** (task surface). Build path: code-led. Branch `redesign/studio-ui` (from `feature/single-operator-sticker-factory`, the only branch that contains `/factory`, `/library`, `/characters`).

## Product mechanism, scene, rut

- Mechanism: a one-operator sticker **production line**: fragments → prompt → external AI sheet → cut/outline → pack → API-key client.
- Scene: one owner at a desk **or on a phone**, switching between Studio and an AI image tool, judging artwork colour, copying prompts, uploading sheets.
- Cultural home: Vietnamese print-shop and sticker-sheet culture (die-cut sheets, job tickets "phiếu lệnh", proof tables), not the Zalo store grid.
- The rut (kept out of the list): a violet SaaS admin with card grids, and its opposite, a node-canvas Figma clone.

## Seven grounded candidates (ordered by resonance)

1. Kiss-cut backing sheet: stickers peel from a liner, cut lines in spot ink.
2. Print-proof table: crop marks, registration targets, colour bars on neutral grey.
3. Pantone-style spot-colour chip book.
4. Photographic contact sheet with grease-pencil marks.
5. Convention/zine circle catalogue grid (thumbnail whole-cell packing).
6. **Production traveler: a factory job ticket that follows a job station to station, stamped at each stage.** (assigned by the roll)
7. Dark graphite developer console with hairline seams.

`concept-seed` assigned index **6**. Build: *Production traveler on a proofing table*, internal name **Proof Table**.

## Three directions put forward

| # | Direction | Verdict |
|---|-----------|---------|
| A | **Proof Table** (assigned #6, fused with #2 registration discipline): neutral proofing ground, hairline rules, job numbers, stage stamps, kiss-cut-magenta selection | **Built.** Only one that makes the idea→pack pipeline the navigation grammar and keeps sticker art colour-true. |
| B | Graphite panel console (challenger 3, dark-first) | Competitive on dense desktop work; loses on mobile and colour-true artwork judging. Kept: **theme parity**, hairline seams over shadows, isolation of destructive actions. |
| C | Newsprint circle-catalogue (challenger 4) | Declined as a world (costume risk); donated discipline: **whole-cell thumbnail packing** and "table code" ids for library, packs, sticker grids. |

Challengers 1 (split-flap), 2 (orienteering), 5 (ticket wallet) lost both axes (audience identification, product clarity). From the ticket wallet we keep one discipline: a **stamp, not a toast** for state (validated/void).

## Direction contract

THESIS: A sticker is a print job. Studio is the job traveler and proof table where it is composed, checked, cut, and shipped. Refuses the card-grid SaaS dashboard: structure comes from ruled sections, job numbers, and station order.

OWN-WORLD: Neutral proofing ground (paper-grey light, booth-grey dark, chroma ≤ 0.01 so sticker colours read true). One accent, **cut-line magenta**, reserved for the active station, primary action, and kiss-cut selection outline (dashed, offset). Hairline 1px rules, 4px control radius, 8px panel radius, shadows only on overlays. Bricolage Grotesque (headings), Be Vietnam Pro (UI/body, Vietnamese-native), JetBrains Mono tabular for ids/hex/counts only. Status is an inked rectangular **stamp**, never a pastel pill. Artwork sits in crop-mark frames.

STORY: Operator always knows the station (Soạn → Tải sheet → Cắt → Pack → Key) and the next move. Frequent actions one gesture or one keystroke (⌘K palette). Work is never hidden or lost.

FIRST VIEWPORT (`/factory`, desktop): left rail of stations; ruled workspace with saved-design strip, the composition (character, palette, fragments per scope) on the left, live prompt output pinned right with Copy as the single magenta primary; station strip on top. Mobile: bottom tab bar, composition as stacked steps, prompt in a sticky bottom sheet with Copy.

SIGNATURE: kiss-cut selection (dashed magenta outline that "draws" itself in 180ms) on any selected artefact, plus sticker grids that peel-reveal in stagger once per mount. Reduced motion: instant.

FORM: Operate/production traveler, list position 6, seed reference: `seed: be01007b6f4564f6`.

## Decisions made autonomously (per brief: no per-decision approval)

- Redesign base is the feature branch, not `main` (see PRODUCT.md Operating Context).
- Both themes ship (tokens), default follows `prefers-color-scheme`.
- `/api-docs` embeds Go-served `/docs` in an iframe: chrome and framing are redesigned; the inner document is outside the Next frontend. Recorded as a known limit.
