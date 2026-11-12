# QA tooling for the studio UI

Test-only: a mock of the admin API + static export, and a screenshot sweep. Not part of the app build.

- Build the app first: `npx next build` in `clients/web-next` (writes `out/`), or set `STATIC_DIR`.
- Mock: `cd clients/web-next/qa && npm run mock` (port 4500; `PORT=` and `STATIC_DIR=` override; password `test`).
- Scenarios: `curl -X POST localhost:4500/__mock/scenario -H 'Content-Type: application/json' -d '{"name":"default"}'`
  with `default`, `empty` (lists empty), `error` (admin API 500), `slow` (1.5s latency), `loggedout` (session 401). Each switch resets the data.
- Sweep: `node qa/shoot.mjs before|after http://localhost:4500` (run from `clients/web-next`, after `npm i` and `npx playwright install chromium` in `qa/`).
- Output: `docs/redesign/screens/<label>/<route>-<width>.png` (40 shots: 10 routes x 4 widths) and `report.json`
  with console errors, failed requests (status >= 400 or requestfailed), and horizontal document overflow per shot.
- The sweep logs in with `POST /api/v1/admin/session`; the `/login` route is shot logged out. It refuses 127.0.0.1:8080 (the real instance).
- Seeded pack ids: `pk-draft-1` (running job), `pk-ready-1`, `pk-revoked-1`. Detail route: `/packs/detail?id=pk-draft-1`.
- Regenerate the pristine baseline: `git archive 2a4f07b clients/web-next | tar -x -C <scratch>`, `cp -cR <worktree>/clients/web-next/node_modules <scratch>/clients/web-next/` (a symlink breaks Turbopack), `npx next build`, then `STATIC_DIR=<scratch>/clients/web-next/out node qa/mock-server.mjs "" 4501`.

## Interaction / a11y sweep
`node qa/interact.mjs [baseUrl]` (mock running): login error, ⌘K palette, theme persistence, skip link, key create/confirm, library drawer, validation, pack states, dark + mobile viewport shots, empty/error/slow scenarios, axe-core WCAG 2.x AA on 8 routes x light/dark x 1440/390. Shots go to `docs/redesign/screens/states/`.

Sheets and grants flows in the same sweep:
- Factory: open a saved design, upload a PNG sheet with provider `gemini` and a note, check the provider badge and the prompt snapshot, cancel then confirm the delete.
- New pack: pick a design (design panel + sheet gallery), submit is blocked until a sheet is chosen, a design with no sheets shows the upload hint, switching designs drops the selection, then create the pack and check the sheet image and design panel on the detail page.
- Pack detail: change the sheet through `PUT` and confirm the image changes; a running job reports the `job_active` error.
- Keys: grant two packs from the one-time created-key dialog, reload and confirm the grants, untick one and confirm `pack_count` in the row, rotate the key and confirm the replacement keeps the grant.

## Character/reference composition (targeted checks)

Run these checks against a fresh disposable real API and the current static
export at **1440px and 390px**; the mock is useful for interaction/error states,
not compiler correctness or private-image/clipboard proof. No all-route sweep,
new runner, dependency install or AI-provider submission is needed for this task.
Use the existing `real-proxy.mjs` with explicit free loopback ports if Chromium
needs localhost cookie/Origin support. Keep the production auth policy intact;
never use `.localdata*`, real secrets or stored user designs as test fixtures.

1. Prepare a character with three actual PNG/JPEG/WebP images. In `/factory`,
   select that profile: no reference radio is initially checked. Compile and
   Save without a reference each show required feedback; Suggest remains usable.
   Pick A, compile, then select B without compiling: prompt and compiled preview
   stay paired with A. Compile B, save, reload and load: both remain B. Detach B
   from the profile: its old compiled private image still opens/copies, but a new
   compile/save rejects stale membership. Changing/removing the character clears
   the editor's image choice, never silently selects another image.
2. Hold each compile/save/suggest/delete request (browser network interception,
   released after inspection). Every editor input and New/saved-design load is
   locked; the other request actions cannot run concurrently. Release a failed
   response: prior prompt/image are unchanged. New or successful delete clears
   both. Check the desktop panel and mobile prompt dialog agree.
3. Open saved archived profiles via GET, not the active-only character list.
   A profile with images still permits an explicit valid choice. An image-less
   archived profile keeps the old prompt available and explains active prepared
   profile/no-character choices, with no inaccessible archived editor link.
   For a legacy snapshot without reference, the old prompt remains readable,
   no compiled image is invented, and compile/save require an explicit choice.
4. With no character, use a 1×3 layout and three separate dish descriptions plus
   exact labels **Bún bò Huế**, **Phở Hà Nội**, **Bánh xèo miền Tây**. Compile,
   save/reload and Suggest retain all descriptions/labels. Shrink/expand the
   grid: retained labels survive and new cells are empty. Also exercise mixed
   empty labels and quoted/multiline Vietnamese labels; a label-only cell must
   fail validation. Inspect the real returned prompt for separate drawing/exact
   text guidance, no same-subject/global-no-text contradiction. No claim about
   real AI lettering accuracy follows from this check.
5. Before selection, each palette card exposes name + four role/hex swatches.
   Tab/arrow/Space choose a palette; choose **Không dùng bảng màu** to clear it.
   A saved selected identity palette stays visible/disabled with a character;
   clear it or use the existing identity-conflict removal action to continue.
   Check visible focus and absence of horizontal document overflow.
6. Copy compiled A/B/C separately. Read actual clipboard `image/png`, decode it,
   compare dimensions and pixels with the displayed private source (JPEG decode
   tolerance is appropriate), not only MIME/nonempty bytes. Deny clipboard
   permission, simulate unsupported binary clipboard, return private-image HTTP
   401, then return invalid image bytes: each must show an error, no success and
   no URL/text copied; the manual private-image link remains. Unauthenticated
   private-image access must still return 401. Never click provider submission.

Useful selectors (scope to visible desktop panel or open dialog to avoid hidden
desktop duplicates):

- `input[type=radio][name=composition-reference]` (value is the private asset ID).
- `fieldset` with legend **Bảng màu**, then radio by palette name or `value=""`.
- Textareas labeled **Mô tả hình cần vẽ**, **Chữ trên sticker (tùy chọn)**,
  and readonly **Prompt đã biên dịch**.
- Region **Ảnh của prompt đã biên dịch**, its `img`, **Sao chép ảnh** button,
  private-image link, and status/alert banners.
- Mobile **Xem prompt** opens the same frozen result in the dialog.

Mock fixtures: `design-food` carries the three labels; `design-legacy` has an
active character without a frozen reference; `design-3` uses an archived profile
with an image; `design-archived-empty` uses an archived image-less profile.
The first two original designs deliberately retain historical identity conflicts
to exercise explicit cleanup. Mock compile/save validate reference membership,
grid/cell content and identity conflicts; Suggest keeps labels/reference/text
and pins without demanding a reference. Character archive preserves GET and
reference assets; association updates reject unknown assets before mutation.
Demo `ref-mi`/`ref-bong`/`ref-bo` are SVG artwork, deliberately not binary-copy
success fixtures: upload real PNG/JPEG/WebP for that check, and use the real API
for acceptance. Switching mock scenarios resets data; use a one-request delay or
failure interception to test result preservation without resetting the context.

The unchanged extraction smoke is independent: use a throwaway 1×3 white sheet
with connected dish-like shapes, caption ribbons and diacritic-like marks, then
decode the real processor's three WebPs and assert each ribbon/mark color stays
in the corresponding output. Do not alter the processor or retain the synthetic
script. Existing controller evidence can be reused; do not rerun for UI changes.

