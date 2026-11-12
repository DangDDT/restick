// Interaction, dark-theme, state and axe sweep. Usage: node qa/interact.mjs [baseUrl]
import { chromium } from "playwright";
import { readFileSync, mkdirSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";
const here = dirname(fileURLToPath(import.meta.url));
const BASE = process.argv[2] ?? "http://localhost:4500";
const OUT = join(here, "../../../docs/redesign/screens/states"); mkdirSync(OUT, { recursive: true });
const axeSrc = readFileSync(join(here, "node_modules/axe-core/axe.min.js"), "utf8");
const ROUTES = ["/start", "/factory", "/library", "/characters", "/packs", "/packs/new", "/packs/detail?id=pk-draft-1", "/keys", "/api-docs"];
const results = [];
const log = (name, ok, note = "") => { results.push({ name, ok, note }); console.log(`${ok ? "PASS" : "FAIL"} ${name} ${note}`); };
const step = async (name, fn) => { try { const note = await fn(); log(name, true, note ?? ""); } catch (e) { log(name, false, String(e.message).split("\n")[0].slice(0, 160)); } };
const scenario = (name) => fetch(`${BASE}/__mock/scenario`, { method: "POST", headers: { "content-type": "application/json" }, body: JSON.stringify({ name }) });
// 1x1 PNG: enough for the mock's image checks, and Playwright sets the file from memory.
const PNG = Buffer.from(
  "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg==",
  "base64",
);
// Grant saves are PUTs; waiting for the response (not the optimistic count text)
// keeps the assertions honest and never closes the dialog mid-save.
const GRANT_PUT = (res) => res.request().method() === "PUT" && /\/api\/v1\/admin\/api-keys\/[^/]+\/packs$/.test(new URL(res.url()).pathname) && res.ok();
async function setGrant(p, option, grant, expectedChecked) {
  const saved = p.waitForResponse(GRANT_PUT, { timeout: 5000 });
  if (grant) await option.check(); else await option.uncheck();
  await saved;
  const dialog = p.locator("dialog[open]");
  await dialog.getByText("Đã lưu quyền pack.").waitFor({ timeout: 5000 });
  const checked = await dialog.locator("[data-qa=key-pack-option]:checked").count();
  if (checked !== expectedChecked) throw new Error(`checked=${checked} expected=${expectedChecked}`);
}
async function waitRowPackCount(p, name, expected) {
  await p.waitForFunction(
    ([rowName, want]) => {
      const row = [...document.querySelectorAll("tbody tr")].find((tr) => tr.textContent.includes(rowName));
      return row?.querySelector("[data-qa=key-pack-count]")?.textContent.trim() === want;
    },
    [name, expected],
    { timeout: 5000 },
  );
}

const browser = await chromium.launch();
async function ctx(w, h, scheme = "light") {
  const c = await browser.newContext({ viewport: { width: w, height: h }, colorScheme: scheme, deviceScaleFactor: w < 500 ? 2 : 1 });
  const p = await c.newPage(); const errs = [];
  p.on("pageerror", (e) => errs.push(String(e)));
  return { c, p, errs };
}
async function login(p) {
  await p.goto(`${BASE}/login`); await p.fill("input[type=password]", "test"); await p.click("button[type=submit]"); await p.waitForURL("**/start"); await p.waitForSelector("#main"); await p.waitForLoadState("networkidle");
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
  await step("factory: 10 AI provider links open in a new tab", async () => { await p.goto(`${BASE}/factory`); await p.waitForSelector("a[href='https://chatgpt.com']"); const r = await p.$$eval("a[target=_blank][rel*=noopener]", (a) => a.filter((x) => !x.href.includes("localhost")).length); if (r < 10) throw new Error("links: " + r); });
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

// 3b. design sheets, sheet-based pack creation, per-key grants
{ const { c, p } = await ctx(1440, 900); await login(p);
  await step("factory: upload a design sheet with provider and prompt snapshot", async () => {
    await p.goto(`${BASE}/factory`); await p.waitForLoadState("networkidle");
    await p.getByRole("button", { name: /Mèo Mi – chào buổi sáng/ }).first().click();
    const sheets = p.locator("[data-qa=design-sheets]");
    await sheets.waitFor();
    await sheets.locator("[data-qa=sheet-file]").setInputFiles({ name: "qa-sheet.png", mimeType: "image/png", buffer: PNG });
    await sheets.locator("[data-qa=sheet-provider]").selectOption("gemini");
    await sheets.locator("[data-qa=sheet-note]").fill("QA lần 1");
    await sheets.getByRole("button", { name: "Tải sheet lên" }).click();
    const tile = sheets.locator("[data-qa=sheet-tile][data-provider=gemini]").filter({ hasText: "qa-sheet.png" });
    await tile.waitFor({ timeout: 5000 });
    if (!(await tile.innerText()).includes("Gemini")) throw new Error("provider badge missing");
    await tile.locator("summary").click();
    if (!(await tile.locator("pre").innerText()).trim()) throw new Error("prompt snapshot empty");
    await p.screenshot({ path: join(OUT, "factory-sheets-1440.png") });
  });
  await step("factory: sheet delete asks confirmation, then removes it", async () => {
    const tile = p.locator("[data-qa=sheet-tile]").filter({ hasText: "qa-sheet.png" });
    await tile.getByRole("button", { name: /Xóa qa-sheet\.png/ }).click();
    await p.waitForSelector("dialog[open]");
    await p.locator("dialog[open] button:has-text('Hủy')").first().click();
    await p.waitForSelector("dialog[open]", { state: "detached" });
    if ((await tile.count()) !== 1) throw new Error("sheet vanished on cancel");
    await tile.getByRole("button", { name: /Xóa qa-sheet\.png/ }).click();
    await p.waitForSelector("dialog[open]");
    await p.locator("dialog[open]").getByRole("button", { name: "Xóa", exact: true }).click();
    await tile.waitFor({ state: "detached", timeout: 5000 });
  });
  await step("packs/new: design panel, required sheet, empty-design hint", async () => {
    await p.goto(`${BASE}/packs/new`); await p.waitForLoadState("networkidle");
    const form = p.locator("main form").first();
    const designSelect = form.getByLabel("Thiết kế đã lưu");
    await designSelect.selectOption("design-2");
    const panel = p.locator("[data-qa=design-panel]");
    await panel.waitFor();
    const panelText = await panel.innerText();
    if (!panelText.includes("Gấu Bông ngày Tết")) throw new Error("design name missing");
    if (!panelText.includes("Nhân vật:")) throw new Error("compiled prompt missing");
    if ((await p.locator("[data-qa=sheet-option]").count()) < 2) throw new Error("sheet gallery missing");
    await form.getByLabel("Tên pack").fill("QA sheet pack");
    await form.getByLabel("Slug").fill("qa-sheet-pack");
    await form.getByLabel("Tác giả").fill("QA Studio");
    await form.locator("button[type=submit]").click();
    const sheetAlert = p.locator("form [role=alert]");
    await sheetAlert.waitFor();
    if (!(await sheetAlert.first().innerText()).includes("Chọn một sheet")) throw new Error("no sheet validation");
    await designSelect.selectOption("design-3");
    const empty = p.locator("[data-qa=sheet-empty]");
    await empty.waitFor();
    if (!(await empty.innerText()).includes("Thiết kế chưa có sheet")) throw new Error("empty-design hint missing");
    if ((await empty.locator("a[href='/factory']").count()) === 0) throw new Error("design link missing");
    await designSelect.selectOption("design-2");
    await p.locator("[data-qa=sheet-option]").first().waitFor();
    if ((await p.locator("[data-qa=sheet-option][aria-pressed=true]").count()) !== 0) throw new Error("selection not reset on design switch");
    await p.locator("[data-qa=sheet-option]").first().click();
    if ((await p.locator("[data-qa=sheet-option][aria-pressed=true]").count()) !== 1) throw new Error("sheet not selected");
    await form.locator("button[type=submit]").click();
    await p.waitForURL("**/packs/detail?id=**");
    await p.waitForLoadState("networkidle");
    await p.locator("[data-qa=pack-sheet-image]").waitFor({ timeout: 5000 });
    await p.locator("[data-qa=design-panel]").waitFor({ timeout: 5000 });
  });
  await step("pack detail: change the sheet through PUT", async () => {
    const before = await p.locator("[data-qa=pack-sheet-image]").getAttribute("src");
    await p.getByRole("button", { name: "Đổi sheet" }).click();
    await p.waitForSelector("dialog[open] [data-qa=sheet-picker]");
    await p.locator("dialog[open] [data-qa=sheet-option][aria-pressed=false]").first().click();
    await p.locator("dialog[open] button:has-text('Lưu')").click();
    await p.waitForSelector("dialog[open]", { state: "detached" });
    await p.waitForTimeout(400);
    const after = await p.locator("[data-qa=pack-sheet-image]").getAttribute("src");
    if (after === before) throw new Error("sheet unchanged");
  });
  await step("pack detail: sheet change reports a running job", async () => {
    await p.goto(`${BASE}/packs/detail?id=pk-draft-1`); await p.waitForLoadState("networkidle");
    await p.getByRole("button", { name: "Đổi sheet" }).click();
    await p.waitForSelector("dialog[open] [data-qa=sheet-picker]");
    await p.locator("dialog[open] [data-qa=sheet-option][aria-pressed=false]").first().click();
    await p.locator("dialog[open] button:has-text('Lưu')").click();
    const alert = p.locator("dialog[open] [role=alert]");
    await alert.waitFor();
    if (!(await alert.first().innerText()).includes("job")) throw new Error("job_active message: " + (await alert.first().innerText()));
    await p.keyboard.press("Escape");
  });
  await step("keys: new key grants packs, reload keeps them, count updates", async () => {
    await p.goto(`${BASE}/keys`); await p.waitForLoadState("networkidle");
    await p.getByLabel("Tên khóa").fill("QA grant key");
    await p.getByRole("button", { name: "Tạo khóa" }).click();
    let dialog = p.locator("dialog[open]");
    await dialog.locator("[data-qa=key-pack-access]").waitFor();
    const options = dialog.locator("[data-qa=key-pack-option]");
    if ((await options.count()) < 3) throw new Error("packs missing");
    await setGrant(p, options.nth(0), true, 1);
    await setGrant(p, options.nth(1), true, 2);
    await p.screenshot({ path: join(OUT, "keys-access-1440.png") });
    await p.keyboard.press("Escape");
    await p.waitForSelector("dialog[open]", { state: "detached" });
    await p.reload(); await p.waitForLoadState("networkidle");
    const row = p.locator("tbody tr", { hasText: "QA grant key" }).first();
    await row.getByRole("button", { name: "Quyền pack" }).click();
    dialog = p.locator("dialog[open]");
    await dialog.locator("[data-qa=key-pack-option]:checked").first().waitFor();
    if ((await dialog.locator("[data-qa=key-pack-option]:checked").count()) !== 2) throw new Error("grants not persisted");
    await setGrant(p, dialog.locator("[data-qa=key-pack-option]:checked").first(), false, 1);
    // Clearing the filtered list is an explicit empty set, not a missing body.
    const cleared = p.waitForResponse(GRANT_PUT, { timeout: 5000 });
    await dialog.getByRole("button", { name: "Bỏ chọn tất cả" }).click();
    await cleared;
    await dialog.getByText("Đã lưu quyền pack.").waitFor({ timeout: 5000 });
    if ((await dialog.locator("[data-qa=key-pack-option]:checked").count()) !== 0) throw new Error("clear did not persist");
    await setGrant(p, dialog.locator("[data-qa=key-pack-option]").first(), true, 1);
    await p.keyboard.press("Escape");
    await p.waitForSelector("dialog[open]", { state: "detached" });
    await waitRowPackCount(p, "QA grant key", "1");
  });
  await step("keys: rotation carries the granted packs", async () => {
    await p.locator("tbody tr").first().getByRole("button", { name: "Xoay khóa" }).click();
    await p.waitForSelector("dialog[open]");
    await p.locator("dialog[open]").getByRole("button", { name: "Xoay khóa", exact: true }).click();
    const dialog = p.locator("dialog[open]");
    await dialog.locator("[data-qa=key-pack-access]").waitFor();
    await dialog.locator("[data-qa=key-pack-option]:checked").first().waitFor();
    if ((await dialog.locator("[data-qa=key-pack-option]:checked").count()) !== 1) throw new Error("rotated key lost its grant");
    const raw = await dialog.locator("textarea").inputValue();
    const prefix = raw.slice(0, 10);
    await p.keyboard.press("Escape");
    await p.waitForSelector("dialog[open]", { state: "detached" });
    // The replacement key is the first row and must carry the grant from the old key.
    await p.waitForFunction(
      ([pfx]) => {
        const row = document.querySelector("tbody tr");
        return Boolean(row?.textContent.includes(pfx)) && row?.querySelector("[data-qa=key-pack-count]")?.textContent.trim() === "1";
      },
      [prefix],
      { timeout: 5000 },
    );
  });
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
