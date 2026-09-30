import { menuAction, controlled } from "./terminal-actions";
import { execFileSync } from "node:child_process";
import { expect, test } from "@playwright/test";

test("Web 重启后浏览器重连且不重放输入", async ({ page }) => {
  const root = process.env.PERSISTTY_E2E_REMOTE_ROOT;
  test.skip(!root, "仅在隔离 Debian 浏览器实例运行");
  await page.setViewportSize({ width: 1280, height: 800 });
  await page.goto("/projects");
  await page.getByRole("textbox", { name: "访问密码" }).fill(process.env.PERSISTTY_E2E_PASSWORD!);
  await page.getByRole("button", { name: "登录", exact: true }).click();
  await page.getByRole("button", { name: `打开 ${process.env.PERSISTTY_E2E_PROJECT}` }).click();
  await page.getByRole("button", { name: "当前标签页" }).click();
  const [createdResponse] = await Promise.all([
    page.waitForResponse(response => response.url().endsWith("/api/v1/terminals") && response.request().method() === "POST"),
    page.getByRole("button", { name: "新建终端" }).click(),
  ]);
  const created = await createdResponse.json() as { data: { id: string } };
  const file = `w03-restart-${created.data.id}.txt`;
  const runtime = page.locator(`.terminal-runtime[data-terminal-id="${created.data.id}"]`);
  await controlled(runtime);
  await runtime.getByRole("textbox", { name: "Terminal input" }).click();
  await page.keyboard.type(`printf 'once\\n' >> ${file}; wc -l ${file}`);
  await page.keyboard.press("Enter");
  await expect(runtime.locator(".xterm-rows")).toContainText(`1 ${file}`);
  await menuAction(page, runtime, "关闭当前");
  await page.getByRole("button", { name: "接管并关闭" }).click();
  await expect(page.getByRole("dialog", { name: "终端终止状态" })).toBeVisible();

  const result = execFileSync("python3", ["-B", "../tests/integration/debian/run_remote.py", "browser-restart", "--root", root!], {
    cwd: process.cwd(), encoding: "utf8", timeout: 30000,
  });
  const restart = JSON.parse(result) as { checks: Record<string, boolean> };
  expect(Object.values(restart.checks).every(Boolean)).toBe(true);
  await controlled(runtime, 20000);
  await expect(page.getByRole("dialog", { name: "终端终止状态" })).toHaveCount(0);
  await runtime.getByRole("textbox", { name: "Terminal input" }).click();
  await page.keyboard.type(`printf 'resumed\\n'; wc -l ${file}`);
  await page.keyboard.press("Enter");
  await expect(runtime.locator(".xterm-rows")).toContainText("resumed");
  await expect(runtime.locator(".xterm-rows")).toContainText(`1 ${file}`);
});
