import { expect, test } from "@playwright/test";

test("隔离真实工作台的文件树/标签图标、语言选择、分组及窄屏", async ({ page }, testInfo) => {
  test.skip(!process.env.PERSISTTY_E2E_EDITOR_ROOT, "需显式提供本轮专属合成文件 fixture 根目录");
  const errors: string[] = []; const writes: string[] = []; const connections: string[] = []; const external: string[] = [];
  page.on("pageerror", error => errors.push(error.message));
  page.on("request", request => {
    if (request.method() !== "GET" && request.url().includes("/content")) writes.push(request.url());
    if (!request.url().startsWith(process.env.PERSISTTY_E2E_BASE_URL!) && !request.url().startsWith("data:")) external.push(request.url());
  });
  page.on("websocket", socket => { if (socket.url().includes("/terminals/")) connections.push(socket.url()); });
  await page.goto("/projects");
  await page.getByLabel("访问密码").fill(process.env.PERSISTTY_E2E_PASSWORD!);
  await page.getByRole("button", { name: "登录", exact: true }).click();
  await expect(page.getByRole("button", { name: "新建项目", exact: true })).toBeVisible();
  const session = (await (await page.request.get("/api/v1/auth/session")).json()).data;
  const response = await page.request.post("/api/v1/projects", {
    headers: { Origin: process.env.PERSISTTY_E2E_BASE_URL!, "X-CSRF-Token": session.csrf_token },
    data: { name: "高亮图标隔离验收", folder_paths: [process.env.PERSISTTY_E2E_EDITOR_ROOT], main_index: 0 },
  });
  expect(response.ok()).toBe(true); const project = (await response.json()).data;
  await page.goto(`/projects/${project.id}`);
  const treeIcon = (name: string) => page.getByRole("treeitem", { name, exact: true }).locator("img");
  await expect(treeIcon("config.yaml")).toHaveAttribute("data-file-icon", "yaml");
  await expect(treeIcon("config.yml")).toHaveAttribute("data-file-icon", "yaml");
  await expect(treeIcon("a.d.ts")).toHaveAttribute("data-file-icon", "typescript-def");
  await expect(treeIcon("a.ts")).toHaveAttribute("data-file-icon", "typescript");
  await expect(treeIcon("package.json")).toHaveAttribute("data-file-icon", "nodejs");
  await expect(treeIcon("go.mod")).toHaveAttribute("data-file-icon", "go-mod");
  await expect(treeIcon(".env")).toHaveAttribute("data-file-icon", "tune");
  await expect(treeIcon("Dockerfile")).toHaveAttribute("data-file-icon", "docker");
  await expect(treeIcon("src")).toHaveAttribute("data-file-icon", "folder-src");
  await page.getByRole("treeitem", { name: "src", exact: true }).click();
  await expect(treeIcon("src")).toHaveAttribute("data-file-icon", "folder-src-open");
  await page.getByRole("treeitem", { name: "query.sql", exact: true }).click();
  await expect(page.locator(".monaco-editor")).toBeVisible();
  await page.getByRole("combobox", { name: "语言模式" }).click(); await page.getByRole("option", { name: "PostgreSQL", exact: true }).click();
  await expect(page.getByRole("combobox", { name: "语言模式" })).toHaveText(/PostgreSQL/);
  await page.getByRole("treeitem", { name: "config.yaml", exact: true }).click();
  await expect(page.getByRole("tab", { name: "config.yaml", exact: true }).locator("img")).toHaveAttribute("data-file-icon", "yaml");
  await expect(page.getByRole("combobox", { name: "语言模式" })).toHaveText(/YAML/);
  await page.getByRole("tab", { name: "query.sql", exact: true }).click();
  await expect(page.getByRole("combobox", { name: "语言模式" })).toHaveText(/PostgreSQL/);
  await page.getByRole("button", { name: "向右拆分编辑器" }).click();
  await expect(page.getByRole("combobox", { name: "语言模式" })).toHaveCount(1);
  const left = page.getByRole("region", { name: "左侧编辑器" });
  const right = page.getByRole("region", { name: "右侧编辑器" });
  await left.getByRole("tab", { name: "query.sql", exact: true }).click(); await page.getByRole("combobox", { name: "语言模式" }).click(); await page.getByRole("option", { name: "MySQL", exact: true }).click();
  await right.getByRole("tab", { name: "query.sql", exact: true }).click(); await expect(page.getByRole("combobox", { name: "语言模式" })).toHaveText(/MySQL/);
  await page.getByRole("button", { name: "合并编辑器" }).click();
  await page.getByRole("treeitem", { name: "config.toml", exact: true }).click();
  await page.getByRole("combobox", { name: "主题", exact: true }).click(); await page.getByRole("option", { name: "浅色", exact: true }).click();
  await expect(treeIcon("config.toml")).toHaveAttribute("data-file-icon", "toml_light");
  await expect(page.getByRole("tab", { name: "config.toml", exact: true }).locator("img")).toHaveAttribute("data-file-icon", "toml_light");
  await page.getByRole("combobox", { name: "主题", exact: true }).click(); await page.getByRole("option", { name: "深色", exact: true }).click();
  await expect(treeIcon("config.toml")).toHaveAttribute("data-file-icon", "toml");
  await expect(page.getByRole("option", { name: "深色", exact: true })).toBeHidden();
  await page.emulateMedia({ colorScheme: "dark" });
  await page.getByRole("combobox", { name: "主题", exact: true }).click(); await page.getByRole("option", { name: "跟随系统", exact: true }).click();
  await expect(treeIcon("config.toml")).toHaveAttribute("data-file-icon", "toml");
  await page.emulateMedia({ colorScheme: "light" });
  await expect(treeIcon("config.toml")).toHaveAttribute("data-file-icon", "toml_light");
  await expect(page.getByRole("option", { name: "跟随系统", exact: true })).toBeHidden();
  await expect.poll(() => page.locator("img").evaluateAll(images => images.every(image => image instanceof HTMLImageElement && image.complete && image.naturalWidth > 0))).toBe(true);
  for (const [width, height] of [[1440, 656], [1024, 540], [390, 844], [844, 390]]) {
    await page.setViewportSize({ width: width!, height: height! });
    if (width! <= 760) {
      await page.getByRole("button", { name: "文件", exact: true }).click();
      await page.getByRole("treeitem", { name: "config.toml", exact: true }).click();
      await page.getByRole("button", { name: "编辑器", exact: true }).click();
      await expect(page.getByRole("textbox", { name: "config.toml 内容", exact: true })).toBeVisible();
      await expect(page.getByRole("combobox", { name: "语言模式" })).toHaveCount(0);
    }
    await expect.poll(() => page.evaluate(() => ({ width: document.documentElement.scrollWidth - innerWidth, height: document.documentElement.scrollHeight - innerHeight })) ).toEqual({ width: 0, height: 0 });
    await page.screenshot({ path: testInfo.outputPath(`editor-${width}x${height}.png`) });
  }
  expect(writes).toEqual([]); expect(connections).toEqual([]); expect(external).toEqual([]); expect(errors).toEqual([]);
});
