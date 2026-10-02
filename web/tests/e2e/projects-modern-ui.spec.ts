import { expect, test } from "@playwright/test";

test("项目页深浅色及窄屏保留完整操作，打开与移除确认不产生写入", async ({ page }, info) => {
  const errors: string[] = [], writes: string[] = [];
  page.on("pageerror", error => errors.push(error.message));
  const projects = Array.from({ length: 3 }, (_, i) => ({ id: `project-${i}`, name: `项目 ${i}`, version: 1, main_folder_id: `folder-${i}`, folders: [{ id: `folder-${i}`, path: `/fixture/很长的项目目录/${"嵌套目录/".repeat(12)}root-${i}` }] }));
  await page.route("**/api/v1/**", async route => {
    const request = route.request(), url = new URL(request.url());
    if (request.method() !== "GET") writes.push(url.pathname);
    const data = url.pathname.endsWith("/auth/session") ? { authenticated: true, csrf_token: "fixture", expires_at: "2099-01-01T00:00:00Z" } : url.pathname === "/api/v1/projects" ? { items: projects } : null;
    if (!data) throw new Error(`未预期的请求 ${url.pathname}`);
    await route.fulfill({ json: { data, request_id: "fixture" } });
  });
  await page.goto("/projects");
  for (const width of [1440, 390]) {
    await page.setViewportSize({ width, height: 845 });
    for (const dark of [false, true]) {
      await page.evaluate(dark => document.documentElement.classList.toggle("dark", dark), dark);
      await expect(page.locator(".projects-panel")).toHaveCSS("border-radius", "8px");
      await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth)).toBe(width);
      for (const project of projects) for (const action of ["打开", "编辑", "移除"]) {
        const button = page.getByRole("button", { name: `${action} ${project.name}`, exact: true });
        await expect(button).toBeVisible();
        const rect = (await button.boundingBox())!, panel = (await page.locator(".projects-panel").boundingBox())!;
        expect(rect.x).toBeGreaterThanOrEqual(panel.x); expect(rect.x + rect.width).toBeLessThanOrEqual(panel.x + panel.width);
      }
      await page.screenshot({ path: info.outputPath(`projects-${width}-${dark ? "dark" : "light"}.png`) });
    }
  }
  await page.getByRole("button", { name: "打开 项目 0", exact: true }).click();
  const opening = page.getByRole("dialog", { name: "打开项目", exact: true });
  await expect(opening.getByRole("button", { name: "当前标签页", exact: true })).toBeVisible();
  await expect(opening.getByRole("link", { name: "新标签页", exact: true })).toHaveAttribute("rel", "noopener noreferrer");
  await page.keyboard.press("Escape");
  await page.getByRole("button", { name: "编辑 项目 0", exact: true }).click();
  await expect(page.getByRole("dialog", { name: "编辑项目", exact: true })).toBeVisible();
  await page.getByRole("button", { name: "取消", exact: true }).click();
  await page.getByRole("button", { name: "移除 项目 0", exact: true }).click();
  await expect(page.getByRole("dialog", { name: "移除项目", exact: true })).toContainText("不删除磁盘文件或终止已有终端");
  await page.getByRole("button", { name: "取消", exact: true }).click();
  expect(errors).toEqual([]); expect(writes).toEqual([]);
});
