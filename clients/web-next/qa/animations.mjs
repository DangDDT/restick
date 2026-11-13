// Targeted animation-factory scenario (mock server). Usage: node qa/animations.mjs [baseUrl]
import { chromium } from "playwright";

const BASE = process.argv[2] ?? "http://localhost:4500";
const results = [];
const log = (name, ok, note = "") => {
  results.push({ name, ok });
  console.log(`${ok ? "PASS" : "FAIL"} ${name} ${note}`);
};
const step = async (name, fn) => {
  try {
    const note = await fn();
    log(name, true, typeof note === "string" ? note : "");
  } catch (error) {
    log(name, false, String(error.message).split("\n")[0].slice(0, 200));
  }
};
const scenario = (name) =>
  fetch(`${BASE}/__mock/scenario`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({ name }),
  });

const browser = await chromium.launch();
async function ctx(width, height) {
  const context = await browser.newContext({
    viewport: { width, height },
    deviceScaleFactor: width < 500 ? 2 : 1,
    permissions: ["clipboard-read", "clipboard-write"],
  });
  const page = await context.newPage();
  const errors = [];
  page.on("pageerror", (error) => errors.push(String(error)));
  page.on("console", (message) => {
    // Failed responses (409 no_changes, 410) are protocol answers, not errors.
    if (message.type() === "error" && !message.text().startsWith("Failed to load resource")) {
      errors.push(message.text());
    }
  });
  return { context, page, errors };
}
async function login(page) {
  await page.goto(`${BASE}/login`);
  await page.fill("input[type=password]", "test");
  await page.click("button[type=submit]");
  await page.waitForURL("**/start");
  await page.waitForSelector("#main");
  await page.waitForLoadState("networkidle");
}
const clip = Buffer.alloc(2 * 1024 * 1024, 7);

await scenario("default");
{
  const { context, page, errors } = await ctx(1440, 900);
  await login(page);

  await step("detail: seeded draft matrix shows two unpublished changes", async () => {
    await page.goto(`${BASE}/packs/detail?id=pk-ready-1`);
    await page.waitForLoadState("networkidle");
    const count = page.locator("[data-qa=unpublished-count]");
    await count.waitFor();
    if ((await count.innerText()).trim() !== "2") throw new Error(`count=${await count.innerText()}`);
    if ((await page.locator("[data-testid=animate-button-1]").count()) !== 1) throw new Error("animate link missing");
    if ((await page.locator("[data-testid=publish-animations]").count()) !== 1) throw new Error("publish button missing");
  });

  await step("detail: per-sticker animate link opens the animate page", async () => {
    await page.locator("[data-testid=animate-button-1]").click();
    await page.waitForURL("**/packs/animate?id=pk-ready-1&index=1");
    await page.waitForSelector("#main h1");
    await page.waitForLoadState("networkidle");
    if ((await page.locator("[data-testid=animate-toggle] [role=switch]").count()) !== 1) {
      throw new Error("toggle missing");
    }
  });

  await step("animate: motion text compiles into a prompt", async () => {
    await page.getByLabel("Mô tả chuyển động").fill("vẫy tay chào thật nhanh");
    const compiled = page.waitForResponse(
      (res) => res.url().endsWith("/api/v1/admin/animations/compile") && res.ok(),
      { timeout: 5000 },
    );
    await page.getByRole("button", { name: "Biên dịch prompt" }).click();
    await compiled;
    const prompt = page.getByLabel("Prompt chuyển động");
    await prompt.waitFor();
    const value = await prompt.inputValue();
    if (!value.includes("vẫy tay chào")) throw new Error(`prompt=${value.slice(0, 60)}`);
  });

  await step("animate: prompt copy reports success", async () => {
    await page.locator("[data-testid=animate-prompt-copy] button").click();
    await page.getByText("Đã sao chép.", { exact: true }).first().waitFor({ timeout: 5000 });
  });

  await step("animate: white reference renders, downloads and copies as PNG", async () => {
    const image = page.locator("img[src*='/stickers/1/on-white.png']");
    await image.waitFor();
    if (!(await image.evaluate((img) => img.complete && img.naturalWidth > 0))) throw new Error("image not loaded");
    const download = page.locator("a[download$='-white.png']");
    if (!(await download.getAttribute("href")).endsWith("/packs/pk-ready-1/stickers/1/on-white.png")) {
      throw new Error("download href wrong");
    }
    await page.getByRole("button", { name: "Sao chép ảnh" }).click();
    await page.getByText("Đã sao chép ảnh tham chiếu.", { exact: true }).waitFor({ timeout: 5000 });
  });

  await step("animate: real clip upload reaches ready through polling", async () => {
    await page.locator("input[type=file]").setInputFiles({ name: "clip.mp4", mimeType: "video/mp4", buffer: clip });
    const uploaded = page.waitForResponse(
      (res) => res.request().method() === "POST" && /\/stickers\/1\/animation$/.test(new URL(res.url()).pathname),
      { timeout: 10000 },
    );
    await page.locator("[data-testid=animate-upload] button").click();
    const response = await uploaded;
    if (response.status() !== 202) throw new Error(`upload=${response.status()}`);
    await page.waitForFunction(
      () => {
        const toggle = document.querySelector("[data-testid=animate-toggle] [role=switch]");
        return toggle instanceof HTMLButtonElement && !toggle.disabled;
      },
      undefined,
      { timeout: 20000 },
    );
  });

  await step("animate: toggle off and on PATCHes the draft", async () => {
    const toggle = page.locator("[data-testid=animate-toggle] [role=switch]");
    if ((await toggle.getAttribute("aria-checked")) !== "true") throw new Error("expected enabled draft");
    const patched = page.waitForResponse(
      (res) => res.request().method() === "PATCH" && res.ok(),
      { timeout: 5000 },
    );
    await toggle.click();
    await patched;
    await page.waitForFunction(
      () => document.querySelector("[data-testid=animate-toggle] [role=switch]")?.getAttribute("aria-checked") === "false",
      undefined,
      { timeout: 5000 },
    );
    const again = page.waitForResponse((res) => res.request().method() === "PATCH" && res.ok(), { timeout: 5000 });
    await toggle.click();
    await again;
    await page.waitForFunction(
      () => document.querySelector("[data-testid=animate-toggle] [role=switch]")?.getAttribute("aria-checked") === "true",
      undefined,
      { timeout: 5000 },
    );
  });

  await step("detail: batch publish lists actual changed variants and applies them", async () => {
    await page.goto(`${BASE}/packs/detail?id=pk-ready-1`);
    await page.waitForLoadState("networkidle");
    await page.locator("[data-testid=publish-animations] button").click();
    const dialog = page.locator("dialog[open]");
    await dialog.waitFor();
    if ((await dialog.locator("li").count()) !== 2) throw new Error(`variants=${await dialog.locator("li").count()}`);
    const started = page.waitForResponse(
      (res) => res.url().endsWith("/publish-animations") && res.status() === 202,
      { timeout: 5000 },
    );
    await dialog.getByRole("button", { name: "Phát hành animation" }).click();
    await started;
    await page.waitForFunction(
      () => document.querySelector("[data-qa=unpublished-count]")?.textContent.trim() === "0",
      undefined,
      { timeout: 20000 },
    );
    if ((await dialog.count()) !== 0) throw new Error("publish dialog still open");
  });

  await step("detail: publishing again surfaces the no_changes error inline", async () => {
    await page.locator("[data-testid=publish-animations] button").click();
    const dialog = page.locator("dialog[open]");
    await dialog.waitFor();
    const rejected = page.waitForResponse(
      (res) => res.url().endsWith("/publish-animations") && res.status() === 409,
      { timeout: 5000 },
    );
    await dialog.getByRole("button", { name: "Phát hành animation" }).click();
    await rejected;
    await dialog.locator("[role=alert]").first().waitFor({ timeout: 5000 });
    const alert = await dialog.locator("[role=alert]").first().innerText();
    if (!alert.includes("Không có thay đổi")) throw new Error(alert);
    await page.keyboard.press("Escape");
    await page.waitForSelector("dialog[open]", { state: "detached" });
  });

  await step("no console or page errors in the animation flow", async () => {
    if (errors.length) throw new Error(errors[0]);
  });
  await context.close();
}

// Responsive surfaces: no horizontal overflow and no console errors.
for (const [width, height] of [
  [390, 844],
  [1440, 900],
]) {
  const { context, page, errors } = await ctx(width, height);
  await login(page);
  for (const route of ["/packs/detail?id=pk-ready-1", "/packs/animate?id=pk-ready-1&index=0"]) {
    await step(`${width}px ${route}: no horizontal overflow`, async () => {
      await page.goto(`${BASE}${route}`);
      await page.waitForLoadState("networkidle");
      await page.waitForSelector("#main h1");
      const overflow = await page.evaluate(
        () => document.documentElement.scrollWidth - document.documentElement.clientWidth,
      );
      if (overflow > 0) throw new Error(`overflow=${overflow}px`);
    });
  }
  await step(`${width}px animation surfaces: no console errors`, async () => {
    if (errors.length) throw new Error(errors[0]);
  });
  await context.close();
}

await browser.close();
const failed = results.filter((result) => !result.ok);
console.log(`\n${results.length - failed.length}/${results.length} passed`);
process.exit(failed.length ? 1 : 0);
