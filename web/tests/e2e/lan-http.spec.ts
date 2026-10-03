import { expect, test } from "@playwright/test";

test("非安全 HTTP 产物可初始化工作台、打开文件并重复定位搜索结果", async ({ page, context, baseURL }) => {
  if (!baseURL || !/^http:\/\/127\.0\.0\.1:\d+$/.test(baseURL)) throw new Error("静态产物仅从显式本机隔离实例读取");
  // 浏览器保持普通私有 HTTP origin；静态资源转取本机产物，API/WS使用明确fixture。
  const origin = "http://10.42.0.10";
  const errors: string[] = [], unexpected: string[] = [];
  const project = { id: "http-fixture", name: "局域网HTTP", version: 1, main_folder_id: "root", folders: [{ id: "root", path: "/fixture" }] };
  const version = { mtime: "2026-10-03T00:00:00Z", size: 19, etag: "fixture", identity: "sample" };
  page.on("pageerror", error => errors.push(error.message));
  await context.routeWebSocket(`${origin.replace("http:", "ws:")}/api/v1/events**`, () => {});
  await context.route(`${origin}/**`, async route => {
    const request = route.request(), url = new URL(request.url());
    if (!url.pathname.startsWith("/api/")) {
      const response = await route.fetch({ url: new URL(url.pathname + url.search, baseURL).href });
      await route.fulfill({ response });
      return;
    }
    const data = url.pathname.endsWith("/auth/session") ? { authenticated: true, csrf_token: "fixture", expires_at: "2099-01-01T00:00:00Z" } :
      url.pathname === `/api/v1/projects/${project.id}` ? project :
      url.pathname === "/api/v1/terminals" ? { items: [] } :
      url.pathname.endsWith("/entries") ? { project_version: 1, next_cursor: "", items: [{ name: "sample.ts", kind: "file", identity: "sample", size: 19, mtime: version.mtime }] } :
      url.pathname.endsWith("/content") ? { kind: "text", content: "const answer = 42;\n", version } :
      url.pathname.endsWith("/metadata") ? { kind: "file", version } :
      url.pathname.endsWith("/git-baseline") ? { state: "no_repository", repo_id: "", head: "", content: "", version: null } :
      url.pathname.endsWith("/repositories") ? { items: [], truncated: false } :
      url.pathname.endsWith("/searches") && request.method() === "POST" ? { id: "search", project_version: 1, files: [{ id: "file", folder_id: "root", path: "sample.ts", version, matches: [{ id: "match", line: 1, column: 7, end_column: 13, preview: "const answer = 42;" }] }], truncated: false, skipped: [], expires_at: "2099-01-01T00:00:00Z" } : null;
    if (data === null) {
      unexpected.push(`${request.method()} ${url.pathname}`);
      await route.fulfill({ status: 500, json: {} });
      return;
    }
    await route.fulfill({ json: { data, request_id: "fixture" } });
  });
  await page.goto(`${origin}/projects/${project.id}`);
  expect(await page.evaluate(() => ({ secure: isSecureContext, uuid: typeof crypto.randomUUID, random: typeof crypto.getRandomValues }))).toEqual({ secure: false, uuid: "undefined", random: "function" });
  await expect(page.getByRole("main", { name: "局域网HTTP 工作台", exact: true })).toBeVisible();
  await page.getByRole("treeitem", { name: "sample.ts", exact: true }).click();
  await expect(page.locator(".monaco-editor").first()).toBeVisible();
  await page.getByRole("button", { name: "搜索与替换", exact: true }).click();
  await page.getByLabel("搜索内容", { exact: true }).fill("answer");
  await page.getByRole("button", { name: "搜索", exact: true }).click();
  await expect(page.getByText("1 个文件 · 1 处匹配 · 0 项跳过", { exact: true })).toBeVisible();
  for (let attempt = 0; attempt < 2; attempt++) {
    await page.getByRole("button", { name: /^1:7/ }).click();
    await expect(page.locator(".monaco-editor .selected-text").first()).toBeVisible();
  }
  expect(errors).toEqual([]);
  expect(unexpected).toEqual([]);
});
