import { expect, test, type Page } from "@playwright/test";

async function fixture(page: Page) {
  const errors: string[] = []; const mutations: string[] = [];
  page.on("pageerror", error => errors.push(error.message));
  await page.routeWebSocket(/\/api\/v1\/events/, () => {});
  await page.route("**/api/v1/**", async route => {
    const request = route.request(), url = new URL(request.url());
    if (request.method() !== "GET") mutations.push(url.pathname);
    const data = url.pathname === "/api/v1/terminals" ? { items: [] } :
      url.pathname.endsWith("/entries") ? { project_version: 1, next_cursor: "", items: url.pathname.includes("/other/") ? [] : Array.from({ length: 30 }, (_, i) => ({ name: `file-${i}.ts`, kind: "file", identity: `file-${i}`, size: 19, mtime: "2026-10-02T00:00:00Z" })) } :
      url.pathname.endsWith("/content") ? { kind: "text", content: "const answer = 42;\n", version: { mtime: "2026-10-02T00:00:00Z", size: 19, etag: "fixture", identity: url.searchParams.get("path") } } :
      url.pathname.endsWith("/git-baseline") ? { state: "no_repository", repo_id: "", head: "", content: "", version: null } : null;
    if (!data) throw new Error(`未预期的请求 ${url.pathname}`);
    await route.fulfill({ json: { data, request_id: "fixture" } });
  });
  await page.goto("/tests/fixtures/workbench-interactions.html");
  return { errors, mutations };
}

async function contained(page: Page) {
  await expect.poll(() => page.evaluate(() => ({ width: document.documentElement.scrollWidth, height: document.documentElement.scrollHeight }))).toEqual(await page.evaluate(() => ({ width: innerWidth, height: innerHeight })));
}

test("微圆角面板保持几何边界、可调整分隔器、标签滚动及固定动作", async ({ page }, info) => {
  await page.setViewportSize({ width: 1440, height: 845 });
  const h = await fixture(page);
  await page.getByRole("treeitem", { name: "file-0.ts", exact: true }).click();
  await expect(page.locator(".monaco-editor")).toBeVisible();
  await expect(page.getByRole("button", { name: "资源管理器", exact: true })).toHaveAttribute("aria-pressed", "true");
  const sidebar = page.locator(".workbench-sidebar"), separator = page.locator(".workbench-body > [data-group] > .workbench-separator");
  const before = (await sidebar.boundingBox())!.width;
  await separator.focus(); await page.keyboard.press("ArrowRight");
  await expect.poll(async () => (await sidebar.boundingBox())!.width).toBeGreaterThan(before);
  const bounds = (await separator.boundingBox())!;
  const resized = (await sidebar.boundingBox())!.width;
  await page.mouse.move(bounds.x + bounds.width / 2, bounds.y + 100); await page.mouse.down(); await page.mouse.move(bounds.x + 24, bounds.y + 100); await page.mouse.up();
  await expect.poll(async () => (await sidebar.boundingBox())!.width).toBeGreaterThan(resized);
  await page.getByRole("button", { name: "切换终端面板", exact: true }).click();
  await expect(page.getByTestId("terminal")).toHaveCSS("height", "0px");
  await page.getByRole("button", { name: "切换终端面板", exact: true }).click();
  await expect(page.locator(".terminal-pane")).toBeVisible();
  for (let i = 1; i < 12; i++) await page.getByRole("treeitem", { name: `file-${i}.ts`, exact: true }).click();
  await page.locator(".editor-tabs").evaluate(element => { element.scrollLeft = element.scrollWidth; });
  await expect(page.getByRole("button", { name: "保存文件", exact: true })).toBeVisible();
  await expect(page.getByRole("tab", { name: "file-11.ts", exact: true })).toBeVisible();
  await expect(page.getByRole("treeitem", { name: "file-11.ts", exact: true })).toHaveAttribute("aria-selected", "true");
  for (const dark of [false, true]) {
    await page.evaluate(dark => document.documentElement.classList.toggle("dark", dark), dark);
    await contained(page);
    for (const selector of [".workbench-sidebar", ".editor-group", ".terminal-pane"]) {
      const host = page.locator(selector);
      await expect(host).toHaveCSS("border-radius", "8px");
      const rect = (await host.boundingBox())!;
      expect(rect.x).toBeGreaterThanOrEqual(0); expect(rect.x + rect.width).toBeLessThanOrEqual(1440);
    }
    await page.screenshot({ path: info.outputPath(`desktop-${dark ? "dark" : "light"}.png`) });
  }
  expect(h.errors).toEqual([]); expect(h.mutations).toEqual([]);
});

test.describe("触屏单内容视图", () => {
  test.use({ hasTouch: true, isMobile: true, viewport: { width: 390, height: 844 } });
  test("横竖屏均保留输入视图与可达导航，不出现双重边框或横向溢出", async ({ page }, info) => {
    const h = await fixture(page);
    await page.getByRole("treeitem", { name: "file-0.ts", exact: true }).click();
    for (const viewport of [{ width: 390, height: 844 }, { width: 844, height: 390 }]) {
      await page.setViewportSize(viewport);
      for (const dark of [false, true]) {
        await page.evaluate(dark => document.documentElement.classList.toggle("dark", dark), dark);
        await contained(page);
        await expect(page.getByRole("textbox", { name: "file-0.ts 内容", exact: true })).toHaveValue("const answer = 42;\n");
        await expect(page.locator(".monaco-editor")).toHaveCount(0);
        await expect(page.locator(".mobile-workbench-content")).toHaveCSS("border-radius", "8px");
        await expect(page.locator(".editor-group")).toHaveCSS("border-width", "0px");
        await page.screenshot({ path: info.outputPath(`touch-${viewport.width}-${dark ? "dark" : "light"}.png`) });
      }
    }
    await page.getByRole("button", { name: "文件", exact: true }).click();
    await expect(page.getByRole("treeitem", { name: "file-0.ts", exact: true })).toHaveAttribute("aria-selected", "true");
    expect(h.errors).toEqual([]); expect(h.mutations).toEqual([]);
  });
});
