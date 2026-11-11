# Restick Studio redesign: roadmap and coverage

Direction: [DIRECTION.md](DIRECTION.md). Product: [../../PRODUCT.md](../../PRODUCT.md). Tokens/components: [../../DESIGN.md](../../DESIGN.md).
Evidence: `docs/redesign/screens/before|after/` (10 routes x 1440/1280/768/390, plus `report.json`). Tooling: `clients/web-next/qa/README.md` (fixture mock API, never the real instance).

## Gate legend
- **Rendered** = loaded in headless Chromium against the fixture mock at 4 widths, 0 failed requests, 0 detector findings.
- **Interacted / audited** = states, keyboard, contrast actually exercised. Not yet done unless ticked.

## Routes
| Route | Implemented | Rendered 4 widths | Interacted / audited |
|---|---|---|---|
| shell (rail, tabs, ⌘K, theme) | yes | yes (via every route) | no |
| `/login` | yes | yes | no |
| `/` splash | yes | n/a (redirect) | no |
| `/factory` | yes | yes | no |
| `/library` | yes | yes | no |
| `/characters` | yes | yes | no |
| `/packs` | yes | yes | no |
| `/packs/new` | yes | yes | no |
| `/packs/detail` (draft) | yes | yes | no |
| `/packs/detail` ready / revoked / failed job | yes | **not shot** | no |
| `/keys` | yes | yes (768px reports document overflow) | no |
| `/api-docs` | frame only | yes | no |
| 404 | yes | yes | no |

## Open items (next session starts here)
1. Fix `/keys` 768px horizontal overflow.
2. Exercise: dialogs/drawers, forms and validation, delete/revoke confirms, key secret reveal, palette keyboard flow, mock scenarios `empty`/`error`/`slow`/`loggedout`.
3. Dark theme visual pass on every route.
4. Contrast and keyboard audit (Impeccable `audit`), then `critique`, `polish`, `harden`, `adapt` passes.
5. Viewport (not full-page) check of mobile sticky prompt bar on `/factory` against the bottom tab bar.
6. Two unused-variable lint warnings (`stickers`, `data`) in page sources.
7. Known limit: `/api-docs` inner document is Go-served `/docs`, outside this frontend.
