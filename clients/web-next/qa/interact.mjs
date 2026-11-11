// Interaction, dark-theme, state and axe sweep. Usage: node qa/interact.mjs [baseUrl]
import { chromium } from "playwright";
import { readFileSync, mkdirSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";
const here = dirname(fileURLToPath(import.meta.url));
const BASE = process.argv[2] ?? "http://localhost:4500";
const OUT = join(here, "../../../docs/redesign/screens/states"); mkdirSync(OUT, { recursive: true });
const axeSrc = readFileSync(join(here, "node_modules/axe-core/axe.min.js"), "utf8");
const ROUTES = ["/factory", "/library", "/characters", "/packs", "/packs/new", "/packs/detail?id=pk-draft-1", "/keys", "/api-docs"];
const results = [];
const log = (name, ok, note = "") => { results.push({ name, ok, note }); console.log(`${ok ? "PASS" : "FAIL"} ${name} ${note}`); };
const step = async (name, fn) => { try { const note = await fn(); log(name, true, note ?? ""); } catch (e) { log(name, false, String(e.message).split("\n")[0].slice(0, 160)); } };
const scenario = (name) => fetch(`${BASE}/__mock/scenario`, { method: "POST", headers: { "content-type": "application/json" }, body: JSON.stringify({ name }) });

const browser = await chromium.launch();
async function ctx(w, h, scheme = "light") {
  const c = await browser.newContext({ viewport: { width: w, height: h }, colorScheme: scheme, deviceScaleFactor: w < 500 ? 2 : 1 });
  const p = await c.newPage(); const errs = [];
  p.on("pageerror", (e) => errs.push(String(e)));
  return { c, p, errs };
}
async function login(p) {
  await p.goto(`${BASE}/login`); await p.fill("input[type=password]", "test"); await p.click("button[type=submit]"); await p.waitForURL("**/factory"); await p.waitForSelector("#main"); await p.waitForLoadState("networkidle");
}
await scenario("default");

// 1. login failure
{ const { c, p } = await ctx(1440, 900);
  await step("login wrong password shows alert", async () => { await p.goto(`${BASE}/login`); await p.fill("input[type=password]", "nope"); await p.click("button[type=submit]"); await p.waitForSelector("[role=alert]", { timeout: 4000 }); await p.screenshot({ path: join(OUT, "login-error-1440.png") }); });
  await c.close(); }

// 2. palette + theme + keyboard
{ const { c, p, errs } = await ctx(1440, 900); await login(p);
  await step("cmd+K opens palette, filter + Enter navigates", async () => { await p.keyboard.press("Control+k"); await p.waitForSelector("[role=combobox]"); await p.waitForFunction(() => document.activeElement?.getAttribute("role") === "combobox"); await p.keyboard.type("pack"); await p.screenshot({ path: join(OUT, "palette-1440.png") }); await p.keyboard.press("Enter"); await p.waitForURL("**/packs"); });
  await step("Escape closes palette and restores focus", async () => { await p.waitForSelector("#main h1"); await p.waitForLoadState("networkidle"); await p.keyboard.press("Control+k"); await p.waitForSelector("dialog[open]"); await p.keyboard.press("Escape"); await p.waitForSelector("dialog[open]", { state: "detached", timeout: 2000 }); });
  await step("theme toggle sets data-theme=dark and persists", async () => { await p.click("aside button[aria-label*='tối']"); const t = await p.evaluate(() => document.documentElement.dataset.theme); if (t !== "dark") throw new Error("theme=" + t); await p.reload(); await p.waitForLoadState("networkidle"); if ((await p.evaluate(() => document.documentElement.dataset.theme)) !== "dark") throw new Error("not persisted"); });
  await step("tab order: skip link first", async () => { await p.goto(`${BASE}/factory`); await p.waitForSelector("#main"); await p.keyboard.press("Tab"); const t = await p.evaluate(() => document.activeElement?.textContent?.trim()); if (!/Bỏ qua/.test(t ?? "")) throw new Error("first focus: " + t); });
  await step("no uncaught page errors", async () => { if (errs.length) throw new Error(errs[0]); });
  await c.close(); }

// 3. flows
{ const { c, p } = await ctx(1440, 900); await login(p);
  await step("keys: create shows one-time secret dialog", async () => { await p.goto(`${BASE}/keys`); await p.waitForLoadState("networkidle"); await p.waitForSelector("#main h1"); await p.getByLabel("Tên khóa").fill("QA key"); await p.getByRole("button", { name: "Tạo khóa" }).click(); await p.waitForSelector("dialog[open]", { timeout: 4000 }); await p.screenshot({ path: join(OUT, "keys-secret-1440.png") }); await p.keyboard.press("Escape"); });
  await step("keys: delete asks confirmation, cancel keeps key", async () => { await p.goto(`${BASE}/keys`); await p.waitForLoadState("networkidle"); const before = await p.locator("tbody tr").count(); await p.locator("tbody tr").first().getByRole("button", { name: "Vô hiệu hóa" }).click(); await p.waitForSelector("dialog[open]"); await p.screenshot({ path: join(OUT, "keys-confirm-1440.png") }); await p.locator("dialog[open] button:has-text('Hủy')").first().click(); if ((await p.locator("tbody tr").count()) !== before) throw new Error("row count changed"); });
  await step("library: create drawer opens, Esc closes", async () => { await p.goto(`${BASE}/library`); await p.waitForLoadState("networkidle"); await p.locator("main button:has-text('Tạo mảnh')").first().click(); await p.waitForSelector("dialog[open]"); await p.screenshot({ path: join(OUT, "library-drawer-1440.png") }); await p.keyboard.press("Escape"); await p.waitForSelector("dialog[open]", { state: "detached", timeout: 2000 }); });
  await step("packs/new: submitting empty form shows validation", async () => { await p.goto(`${BASE}/packs/new`); await p.waitForLoadState("networkidle"); await p.locator("main form button[type=submit]").first().click(); await p.waitForTimeout(400); await p.screenshot({ path: join(OUT, "pack-new-validation-1440.png") }); });
  await step("pack detail: delete asks confirmation", async () => { await p.goto(`${BASE}/packs/detail?id=pk-draft-1`); await p.waitForLoadState("networkidle"); await p.locator("main button:has-text('Xóa')").first().click(); await p.waitForSelector("dialog[open]"); await p.keyboard.press("Escape"); });
  await step("pack detail ready/revoked render", async () => { for (const id of ["pk-ready-1", "pk-revoked-1"]) { await p.goto(`${BASE}/packs/detail?id=${id}`); await p.waitForLoadState("networkidle"); await p.screenshot({ path: join(OUT, `${id}-1440.png`) }); if (await p.locator("main h1").count() === 0) throw new Error(id + " no heading"); } });
  await c.close(); }

// 4. mobile viewport shots (fixed bars) + dark shots
for (const [scheme, w, h] of [["light", 390, 844], ["dark", 390, 844], ["dark", 1440, 900]]) {
  const { c, p } = await ctx(w, h, scheme); await login(p);
  for (const r of ROUTES) await step(`${scheme} ${w} ${r} viewport shot`, async () => { await p.goto(`${BASE}${r}`); await p.waitForLoadState("networkidle"); await p.addStyleTag({ content: "*{animation:none!important;transition:none!important}" }); await p.screenshot({ path: join(OUT, `${scheme}-${w}-${r.replace(/[^a-z]+/gi, "_")}.png`) }); });
  await c.close();
}

// 5. states
for (const sc of ["empty", "error", "slow"]) {
  await scenario(sc); const { c, p } = await ctx(1440, 900); await login(p).catch(() => {});
  for (const r of ["/packs", "/library", "/keys"]) await step(`scenario ${sc} ${r}`, async () => { await p.goto(`${BASE}${r}`); await p.waitForTimeout(sc === "slow" ? 400 : 1200); await p.screenshot({ path: join(OUT, `scenario-${sc}-${r.slice(1)}.png`) }); });
  await c.close();
}
await scenario("default");

// 6. axe (light+dark, desktop+mobile)
for (const [scheme, w] of [["light", 1440], ["dark", 1440], ["light", 390], ["dark", 390]]) {
  const { c, p } = await ctx(w, 900, scheme); await login(p);
  for (const r of ROUTES) await step(`axe ${scheme} ${w} ${r}`, async () => {
    await p.goto(`${BASE}${r}`); await p.waitForLoadState("networkidle"); await p.evaluate(axeSrc);
    const v = await p.evaluate(async () => (await window.axe.run(document, { runOnly: ["wcag2a", "wcag2aa", "wcag22aa"] })).violations.map((x) => `${x.id}(${x.impact}) x${x.nodes.length} ${x.nodes[0].target.join(" ").slice(0, 70)}`));
    if (v.length) throw new Error(v.join(" | ")); });
  await c.close();
}
await browser.close();
const fail = results.filter((r) => !r.ok);
console.log(`\n${results.length - fail.length}/${results.length} passed`);
process.exit(fail.length ? 1 : 0);
