# Restick Studio redesign: roadmap and coverage

Direction: [DIRECTION.md](DIRECTION.md). Product: [../../PRODUCT.md](../../PRODUCT.md). Tokens/components: [../../DESIGN.md](../../DESIGN.md).
Evidence: `docs/redesign/screens/before|after/` (10 routes x 1440/1280/768/390, plus `report.json`). Tooling: `clients/web-next/qa/README.md` (fixture mock API, never the real instance).

## Gate legend
- **Rendered** = loaded in headless Chromium against the fixture mock at 4 widths, 0 failed requests, 0 detector findings.
- **Interacted / audited** = states, keyboard, contrast actually exercised. Not yet done unless ticked.

## Verification summary (latest run)
- `node qa/shoot.mjs after`: 10 routes x 4 widths, 0 failed requests, 0 horizontal overflow, 0 detector findings.
- `node qa/interact.mjs`: **77/77 steps passed**, including axe-core (WCAG 2.0/2.1/2.2 A+AA) with zero violations on 8 routes in light and dark at 1440 and 390.
- Bugs found and fixed by this sweep: `/keys` 768px overflow (sr-only inside non-relative scroller); ⌘K palette and factory fragment picker did not receive typing (React `autoFocus` fires before native `<dialog>` opens; `Dialog` now honours `data-autofocus`); two magenta primaries on `/factory` (now Copy only).

## Routes
| Route | Implemented | Rendered 4 widths | Interacted / audited |
|---|---|---|---|
| shell (rail, tabs, ⌘K, theme) | yes | yes | palette, theme persistence, skip link, axe |
| `/login` | yes | yes | wrong-password alert |
| `/` splash | yes | n/a (redirect) | no |
| `/factory` | yes | yes | axe, mobile sticky bar viewport check, dark |
| `/library` | yes | yes | create drawer + Esc, axe, dark |
| `/characters` | yes | yes | axe, dark (forms not exercised) |
| `/packs` | yes | yes | empty/error/slow states, axe |
| `/packs/new` | yes | yes | empty-submit validation, axe |
| `/packs/detail` draft / ready / revoked | yes | draft only (ready/revoked shot at 1440) | delete confirm, axe |
| `/keys` | yes | yes | create + one-time secret, disable confirm, axe |
| `/api-docs` | frame only | yes | axe |
| 404 | yes | yes | no |

## Not verified / open
1. Real-backend behaviour (all checks use the fixture mock; `/v1/*` not mocked).
2. Character create/edit/upload, pack source upload, job `failed`/`interrupted` views, rotate-key flow: implemented, not exercised.
3. Screen-reader pass and real-device (iOS Safari) check.
4. Two unused-variable lint warnings in page sources.
5. `/api-docs` inner document is Go-served `/docs`, outside this frontend.
