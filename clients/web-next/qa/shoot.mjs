// Screenshot sweep + QA report for the studio UI.
//
// Usage: node qa/shoot.mjs <before|after> [baseUrl]   (default http://localhost:4500)
// Writes docs/redesign/screens/<label>/<route>-<width>.png and report.json
// under the repo root. Authed routes log in first, through a fetch() run inside
// the page itself (so the browser sends Origin and the cookie like the app does).
// The /login route is shot logged out, and only there is a session 401 expected.

import { mkdirSync, writeFileSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { chromium } from "playwright";

const here = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(here, "../../..");

const [label = "", baseArg = "http://localhost:4500"] = process.argv.slice(2);
if (!["before", "after"].includes(label)) {
  console.error("usage: node qa/shoot.mjs <before|after> [baseUrl]");
  process.exit(2);
}
const base = new URL(baseArg);
// Safety: the real instance holds production data. Never point this at it.
if (base.port === "8080" && ["127.0.0.1", "localhost"].includes(base.hostname)) {
  console.error("refusing to run against the real instance (127.0.0.1:8080)");
  process.exit(2);
}
const origin = base.origin;

// Placeholder shown by RequireAuth while the session check runs (lib/i18n-messages.ts "loading").
const LOADING_TEXT = "Đang tải…";

const ROUTES = [
  { slug: "login", path: "/login", loggedOut: true },
  { slug: "factory", path: "/factory" },
  { slug: "library", path: "/library" },
  { slug: "characters", path: "/characters" },
  { slug: "packs", path: "/packs" },
  { slug: "packs-new", path: "/packs/new" },
  { slug: "packs-detail", path: "/packs/detail?id=pk-draft-1" },
  { slug: "keys", path: "/keys" },
  { slug: "api-docs", path: "/api-docs" },
  { slug: "not-found", path: "/this-route-does-not-exist" },
];

const VIEWPORTS = [
  { width: 1440, height: 900, dsf: 1 },
  { width: 1280, height: 800, dsf: 1 },
  { width: 768, height: 1024, dsf: 1 },
  { width: 390, height: 844, dsf: 2 },
];

const outDir = path.join(repoRoot, "docs/redesign/screens", label);
mkdirSync(outDir, { recursive: true });

// Mock-only control endpoint; null against any other server.
async function mockScenario() {
  try {
    const res = await fetch(`${origin}/__mock/scenario`);
    if (!res.ok) return null;
    return (await res.json()).scenario ?? null;
  } catch {
    return null;
  }
}

const browser = await chromium.launch();
const entries = [];
const scenario = await mockScenario();

for (const route of ROUTES) {
  for (const vp of VIEWPORTS) {
    const context = await browser.newContext({
      viewport: { width: vp.width, height: vp.height },
      deviceScaleFactor: vp.dsf,
      reducedMotion: "reduce",
    });
    const page = await context.newPage();

    if (!route.loggedOut) {
      // Same-origin fetch from the page: sends Origin and stores the cookie in this context.
      await page.goto(`${origin}/login`, { waitUntil: "networkidle" });
      const status = await page.evaluate(async (password) => {
        const r = await fetch("/api/v1/admin/session", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ password }),
          credentials: "same-origin",
        });
        return r.status;
      }, "test");
      if (status !== 204) throw new Error(`login failed: ${status}`);
    }

    const consoleErrors = [];
    const failed = [];
    const expected = [];
    page.on("console", (msg) => {
      if (msg.type() === "error") consoleErrors.push(msg.text());
    });
    page.on("pageerror", (err) => consoleErrors.push(`pageerror: ${err.message}`));
    page.on("response", (res) => {
      const status = res.status();
      if (status < 400) return;
      const url = res.url();
      // The page's own document status is reported as document_status instead.
      if (res.request().isNavigationRequest() && url === `${origin}${route.path}`) return;
      // Only the logged-out /login route may probe the session and get 401.
      if (route.loggedOut && status === 401 && url.includes("/api/v1/admin/session")) {
        expected.push({ url, status });
      } else {
        failed.push({ url, status });
      }
    });
    page.on("requestfailed", (req) => {
      failed.push({ url: req.url(), failure: req.failure()?.errorText ?? "failed" });
    });

    const nav = await page.goto(`${origin}${route.path}`, { waitUntil: "networkidle" });
    const documentStatus = nav?.status() ?? null;
    await page.addStyleTag({
      content: "*,*::before,*::after{animation:none!important;transition:none!important;caret-color:transparent!important}",
    });
    await page.evaluate(() => document.fonts.ready);
    await page.waitForLoadState("networkidle");
    await page.waitForTimeout(300);

    const finalUrl = page.url();
    const finalPath = new URL(finalUrl).pathname;
    const loadingPlaceholder = await page.getByText(LOADING_TEXT, { exact: true }).count() > 0;
    // An authed route that ends on /login means the auth guard fired.
    const authedRedirectedToLogin = !route.loggedOut && finalPath === "/login";

    const overflow = await page.evaluate(() => ({
      scrollWidth: document.documentElement.scrollWidth,
      innerWidth: window.innerWidth,
      overflow: document.documentElement.scrollWidth > window.innerWidth,
    }));

    const file = `${route.slug}-${vp.width}.png`;
    await page.screenshot({ path: path.join(outDir, file), fullPage: true, animations: "disabled" });

    entries.push({
      route: route.slug,
      path: route.path,
      viewport: vp.width,
      height: vp.height,
      deviceScaleFactor: vp.dsf,
      screenshot: `docs/redesign/screens/${label}/${file}`,
      document_status: documentStatus,
      final_url: finalUrl,
      authed_redirected_to_login: authedRedirectedToLogin,
      loading_placeholder_visible: loadingPlaceholder,
      console_errors: [...new Set(consoleErrors)],
      failed_requests: failed,
      expected_401: expected,
      horizontal_overflow: overflow,
    });
    console.log(
      `${route.slug.padEnd(12)} ${String(vp.width).padEnd(5)} status=${documentStatus} final=${finalPath} ` +
        `loading=${loadingPlaceholder} overflow=${overflow.overflow} errors=${consoleErrors.length} failed=${failed.length}`,
    );
    await context.close();
  }
}

await browser.close();

const summary = {
  routes: ROUTES.length,
  screenshots: entries.length,
  routes_with_console_errors: entries.filter((e) => e.console_errors.length).length,
  routes_with_failed_requests: entries.filter((e) => e.failed_requests.length).length,
  routes_with_horizontal_overflow: entries.filter((e) => e.horizontal_overflow.overflow).length,
  auth_redirects: entries.filter((e) => e.authed_redirected_to_login).length,
  loading_placeholders: entries.filter((e) => e.loading_placeholder_visible).length,
};
const report = { label, baseUrl: origin, scenario, generatedAt: new Date().toISOString(), summary, entries };
writeFileSync(path.join(outDir, "report.json"), JSON.stringify(report, null, 2) + "\n");
console.log(`wrote ${entries.length} screenshots and report.json to ${outDir}`);
console.log(JSON.stringify(summary));
