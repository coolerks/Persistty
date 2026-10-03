import { expect, test, type Page } from "@playwright/test";

async function fixture(page: Page) {
  const errors: string[] = []; const mutations: string[] = [];
  page.on("pageerror", error => errors.push(error.message));
  await page.routeWebSocket(/\/api\/v1\/events/, () => {});
  await page.route("**/api/v1/**", async route => {
    const request = route.request(), url = new URL(request.url());
    if (request.method() !== "GET") mutations.push(url.pathname);
    const data = url.pathname === "/api/v1/terminals" ? { items: [] } :
      url.pathname.endsWith("/file-names") ? { project_version: 1, items: [{ folder_id: "root", path: "file-0.ts" }, { folder_id: "other", path: "file-1.ts" }], truncated: true } :
      url.pathname.endsWith("/repositories") ? { items: [{ id: "repo", folder_id: "root", path: "", name: "repo", state: "available", reason: "" }], truncated: false } :
      url.pathname.endsWith("/log") ? { head: "a".repeat(40), items: [], next_offset: -1 } :
      url.pathname.endsWith("/status") ? { repo_id: "repo", head: "a".repeat(40), branch: "main", changes: [], total_paths: [] } :
      url.pathname.endsWith("/entries") ? { project_version: 1, next_cursor: "", items: url.pathname.includes("/other/") ? [] : Array.from({ length: 30 }, (_, i) => ({ name: `file-${i}.ts`, kind: "file", identity: `file-${i}`, size: 19, mtime: "2026-10-02T00:00:00Z" })) } :
      url.pathname.endsWith("/content") ? { kind: "text", content: "const answer = 42;\n", version: { mtime: "2026-10-02T00:00:00Z", size: 19, etag: "fixture", identity: url.searchParams.get("path") } } :
      url.pathname.endsWith("/git-baseline") ? { state: "tracked", repo_id: "repo", head: "a".repeat(40), content: "const answer = 42;\n", version: { mtime: "2026-10-02T00:00:00Z", size: 19, etag: "fixture", identity: url.searchParams.get("path") } } : null;
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
  await expect(page.locator(".editor-breadcrumb")).toContainText("HEAD");
  await expect(page.locator(".editor-breadcrumb")).not.toContainText("aaaaaaaa");
  await expect(page.getByRole("button", { name: "资源管理器", exact: true })).toHaveAttribute("aria-pressed", "true");
  for (const label of ["资源管理器", "搜索与替换", "只读 Git"]) {
    const button = page.getByRole("button", { name: label, exact: true });
    if (label !== "资源管理器") await button.click();
    await expect(button).toHaveAttribute("aria-pressed", "true");
    await button.click(); await expect(page.getByTestId("sidebar")).toHaveCSS("width", "0px"); await expect(button).toHaveAttribute("aria-pressed", "false");
    await button.click(); await expect(button).toHaveAttribute("aria-pressed", "true");
  }
  await expect(page.getByRole("button", { name: "变更", exact: true })).toHaveCSS("background-color", "rgba(0, 0, 0, 0)");
  await expect(page.getByRole("tab", { name: "变更文件展示：列表", exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "刷新仓库", exact: true })).toBeVisible();
  const baseline = page.getByRole("tab", { name: "HEAD", exact: true });
  await expect(baseline).toHaveCSS("border-radius", "4px"); await expect(baseline).toHaveCSS("box-shadow", "none");
  await expect(baseline).toHaveCSS("height", "22px");
  await page.getByRole("button", { name: "资源管理器", exact: true }).click();
  const tab = page.getByRole("tab", { name: "file-0.ts", exact: true });
  expect(Math.abs((await tab.boundingBox())!.y - (await page.locator(".editor-tab-row").boundingBox())!.y)).toBeLessThanOrEqual(1);
  const sidebar = page.locator(".workbench-sidebar"), separator = page.locator(".workbench-body > [data-group] > .workbench-separator");
  const before = (await sidebar.boundingBox())!.width;
  await separator.focus(); await page.keyboard.press("ArrowRight");
  const line = await separator.evaluate(element => ({ width: getComputedStyle(element, "::before").width, height: getComputedStyle(element, "::before").height, own: element.getBoundingClientRect().height }));
  await expect.poll(() => separator.evaluate(element => getComputedStyle(element, "::before").borderRadius)).toBe("2px");
  expect(line.width).toBe("4px"); expect(parseFloat(line.height)).toBeCloseTo(line.own, 2);
  await expect.poll(async () => (await sidebar.boundingBox())!.width).toBeGreaterThan(before);
  const bounds = (await separator.boundingBox())!;
  const resized = (await sidebar.boundingBox())!.width;
  await page.mouse.move(bounds.x + bounds.width / 2, bounds.y + 100); await page.mouse.down(); await page.mouse.move(bounds.x + 24, bounds.y + 100); await page.mouse.up();
  await expect.poll(async () => (await sidebar.boundingBox())!.width).toBeGreaterThan(resized);
  await page.getByRole("button", { name: "切换终端面板", exact: true }).click();
  await expect(page.getByTestId("terminal")).toHaveCSS("height", "0px");
  await expect.poll(async () => {
    const left = (await sidebar.boundingBox())!, right = (await page.locator(".editor-group").boundingBox())!;
    return Math.abs(left.y + left.height - right.y - right.height);
  }).toBeLessThanOrEqual(1);
  const centering = await page.locator(".workbench-status").evaluate(element => {
    const footer = element.getBoundingClientRect();
    return Array.from(element.querySelectorAll(":scope > span")).map(span => {
      const range = document.createRange(); range.selectNodeContents(span); const text = range.getBoundingClientRect();
      return Math.abs((text.top + text.bottom - footer.top - footer.bottom) / 2);
    });
  });
  expect(centering.every(offset => offset <= 2)).toBe(true);
  await expect(page.getByRole("button", { name: "终端面板", exact: true })).toHaveAttribute("aria-pressed", "false");
  await page.getByRole("button", { name: "切换终端面板", exact: true }).click();
  await expect(page.locator(".terminal-pane")).toBeVisible();
  const horizontal = page.locator(".workbench-separator.horizontal").first();
  await expect(horizontal).toHaveAttribute("data-collapsed", "false");
  await horizontal.focus();
  const crossLine = await horizontal.evaluate(element => ({ height: getComputedStyle(element, "::before").height, width: getComputedStyle(element, "::before").width, own: element.getBoundingClientRect().width }));
  await expect.poll(() => horizontal.evaluate(element => getComputedStyle(element, "::before").borderRadius)).toBe("2px");
  expect(crossLine.height).toBe("4px"); expect(parseFloat(crossLine.width)).toBeCloseTo(crossLine.own, 2);
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

test("Git 变更数量随基线变化，慢状态加载不移动标签，干净状态留空", async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 845 });
  const h = await fixture(page);
  let release: () => void = () => {}; let pending = false;
  const gate = new Promise<void>(resolve => { release = resolve; });
  await page.route("**/repositories/repo/status?*", async route => {
    pending = true; await gate;
    await route.fulfill({ json: { data: { repo_id: "repo", head: "a".repeat(40), branch: "main", changes: [
      { path: "file-0.ts", old_path: "", index: "M", worktree: " " }, { path: "file-1.ts", old_path: "", index: " ", worktree: "M" },
    ], total_paths: ["file-0.ts", "file-1.ts"] }, request_id: "fixture" } });
  });
  await page.getByRole("button", { name: "只读 Git", exact: true }).click();
  await expect.poll(() => pending).toBe(true);
  const tabs = page.getByRole("tablist", { name: "比较基线" });
  const before = (await tabs.boundingBox())!.y;
  release();
  await expect(page.getByLabel("2 个变更文件", { exact: true })).toBeVisible();
  expect(Math.abs((await tabs.boundingBox())!.y - before)).toBeLessThanOrEqual(1);
  await expect(page.locator(".git-branch-summary")).toHaveText("main");
  await page.getByRole("tab", { name: "暂存", exact: true }).click();
  await expect(page.getByLabel("1 个变更文件", { exact: true })).toBeVisible();
  await page.route("**/repositories/repo/status?*", route => route.fulfill({ json: { data: { repo_id: "repo", head: "a".repeat(40), branch: "main", changes: [], total_paths: [] }, request_id: "fixture" } }));
  await page.getByRole("button", { name: "刷新变更", exact: true }).click();
  await expect(page.getByLabel("0 个变更文件", { exact: true })).toBeVisible();
  await expect(page.getByText("工作区没有变更。", { exact: true })).toHaveCount(0);
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

test("同版本项目轮询不取消正在等待的文件名搜索", async ({ page }) => {
  const h = await fixture(page);
  let projectReads = 0, searches = 0;
  const cancelled: string[] = [];
  let release = () => {};
  page.on("requestfailed", request => { if (request.url().includes("/file-names?")) cancelled.push(request.failure()?.errorText ?? "failed"); });
  await page.route("**/api/v1/auth/session", route => route.fulfill({ json: { data: { authenticated: true, csrf_token: "fixture", expires_at: "2099-01-01T00:00:00Z" }, request_id: "fixture" } }));
  await page.route("**/api/v1/projects/interaction", route => {
    projectReads++;
    return route.fulfill({ json: { data: { id: "interaction", name: "交互验收", version: 1, main_folder_id: "root", folders: [{ id: "root", path: "/fixture" }, { id: "other", path: "/other" }] }, request_id: "fixture" } });
  });
  await page.route("**/file-names?*", async route => {
    searches++;
    await new Promise<void>(resolve => { release = resolve; });
    await route.fulfill({ json: { data: { project_version: 1, items: [{ folder_id: "root", path: "project.ts" }], truncated: false }, request_id: "fixture" } });
  });
  await page.clock.install();
  await page.goto("/projects/interaction");
  await page.getByRole("button", { name: "按名称搜索文件" }).click();
  await page.getByRole("textbox", { name: "文件名关键词" }).fill("project");
  await page.clock.runFor(300);
  await expect.poll(() => searches).toBe(1);
  await page.clock.runFor(16000);
  await expect.poll(() => projectReads).toBeGreaterThanOrEqual(2);
  expect(searches).toBe(1); expect(cancelled).toEqual([]);
  release();
  await expect(page.getByRole("button", { name: "打开 project.ts" })).toBeVisible();
  await page.getByRole("button", { name: "关闭搜索" }).click();
  expect(h.errors).toEqual([]); expect(h.mutations).toEqual([]);
});

test("顶栏按名称快速打开保留多根文件身份、键盘导航与取消", async ({ page }) => {
  const h = await fixture(page);
  await page.getByRole("button", { name: "项目菜单", exact: true }).click();
  await expect(page.getByRole("menuitem", { name: "项目设置", exact: true })).toBeVisible();
  await page.keyboard.press("Escape");
  await page.keyboard.press("Control+p");
  const dialog = page.getByRole("dialog", { name: "按名称搜索文件", exact: true });
  const rect = (await dialog.boundingBox())!;
  expect(Math.abs(rect.x + rect.width / 2 - page.viewportSize()!.width / 2)).toBeLessThanOrEqual(1);
  expect(rect.y).toBeGreaterThan(40);
  const input = page.getByRole("textbox", { name: "文件名关键词" });
  await expect(input).toBeFocused(); await input.fill("file");
  await expect(page.getByRole("button", { name: "打开 file-1.ts", exact: true })).toBeVisible();
  await expect(dialog).not.toContainText("结果已截断");
  await input.press("ArrowDown"); await input.press("Enter");
  await expect(page.getByRole("dialog")).toHaveCount(0);
  await expect(page.getByRole("tab", { name: "file-1.ts", exact: true })).toBeVisible();
  await expect(page.locator(".editor-breadcrumb")).toContainText("/other/file-1.ts");
  await page.getByRole("button", { name: "按名称搜索文件", exact: true }).click();
  const close = page.getByRole("button", { name: "关闭搜索", exact: true });
  const inputRect = (await input.boundingBox())!, closeRect = (await close.boundingBox())!;
  expect(Math.abs(inputRect.y + inputRect.height / 2 - closeRect.y - closeRect.height / 2)).toBeLessThan(1);
  expect(closeRect.x + closeRect.width).toBeLessThan(inputRect.x + inputRect.width);
  await close.click(); await expect(page.getByRole("dialog")).toHaveCount(0);
  await page.getByRole("button", { name: "按名称搜索文件", exact: true }).click();
  await input.fill("file"); await input.press("Escape");
  await expect(page.getByRole("dialog")).toHaveCount(0);
  expect(h.errors).toEqual([]); expect(h.mutations).toEqual([]);
});
