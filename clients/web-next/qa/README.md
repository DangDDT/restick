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
