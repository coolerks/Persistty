import { menuAction, controlled } from "./terminal-actions";
import { execFileSync } from "node:child_process";
import { expect, test, type Page } from "@playwright/test";

async function createTerminal(page: Page) {
  const [response] = await Promise.all([
    page.waitForResponse(item => item.url().endsWith("/api/v1/terminals") && item.request().method() === "POST"),
    page.getByRole("button", { name: "新建终端" }).click(),
  ]);
  const created = await response.json() as { data: { id: string } };
  const runtime = page.locator(`.terminal-runtime[data-terminal-id="${created.data.id}"]`);
  await controlled(runtime);
  return runtime;
}

async function send(page: Page, runtime: ReturnType<Page["locator"]>, command: string) {
  await runtime.getByRole("textbox", { name: "Terminal input" }).click();
  await page.keyboard.type(command);
  await page.keyboard.press("Enter");
}

test("三个产品终端跨 Web SIGKILL 保持计数、HTTP 与 TUI", async ({ page }) => {
  test.setTimeout(120000);
  const root = process.env.PERSISTTY_E2E_REMOTE_ROOT;
  test.skip(!root || process.env.PERSISTTY_E2E_FORCE !== "1", "仅在全新隔离 Debian 实例运行");
  await page.setViewportSize({ width: 1280, height: 800 });
  await page.goto("/projects");
  await page.getByRole("textbox", { name: "访问密码" }).fill(process.env.PERSISTTY_E2E_PASSWORD!);
  await page.getByRole("button", { name: "登录", exact: true }).click();
  await page.getByRole("button", { name: `打开 ${process.env.PERSISTTY_E2E_PROJECT}` }).click();
  await page.getByRole("button", { name: "当前标签页" }).click();

  const counter = await createTerminal(page);
  await send(page, counter, "i=0; while :; do i=$((i+1)); printf '%s\\n' \"$i\" >> w03-counter.log; sleep 0.2; done");
  const http = await createTerminal(page);
  await send(page, http, "python3 -c 'import http.server,socketserver,pathlib; s=socketserver.TCPServer((\"127.0.0.1\",0),http.server.SimpleHTTPRequestHandler); pathlib.Path(\"w03-http-port\").write_text(str(s.server_address[1])); s.serve_forever()'");
  const tui = await createTerminal(page);
  await send(page, tui, "python3 ../tui.py");
  await expect(tui.locator(".xterm-rows")).toContainText("W03_DETERMINISTIC_TUI");
  await expect(tui.locator(".xterm-rows")).toContainText("雪");

  const output = execFileSync("python3", ["-B", "../tests/integration/debian/run_remote.py", "browser-force-restart", "--root", root!], {
    cwd: process.cwd(), encoding: "utf8", timeout: 45000,
  });
  const result = JSON.parse(output) as { checks: Record<string, boolean>; pane_count: number };
  expect(result.pane_count).toBe(3);
  expect(Object.values(result.checks).every(Boolean)).toBe(true);
  await controlled(tui, 20000);
  await expect(tui.locator(".xterm-rows")).toContainText("W03_DETERMINISTIC_TUI");
  await tui.getByRole("textbox", { name: "Terminal input" }).click();
  await page.keyboard.type("k");
  await expect(tui.locator(".xterm-rows")).toContainText("KEY: 6b");
  await page.keyboard.press("ArrowLeft");
  await expect(tui.locator(".xterm-rows")).toContainText("LEFT_RECEIVED");
  await tui.locator(".xterm-screen").click({ position: { x: 100, y: 70 } });
  await expect(tui.locator(".xterm-rows")).toContainText("MOUSE_RECEIVED");
  await page.setViewportSize({ width: 1100, height: 720 });
  await expect(tui.locator(".xterm-rows")).toContainText("W03_DETERMINISTIC_TUI");

  const id = await tui.getAttribute("data-terminal-id");
  await page.context().setOffline(true);
  // CDP offline blocks requests but may retain loopback WS and omit this event.
  await page.evaluate(() => window.dispatchEvent(new Event("offline")));
  await expect(tui).toHaveAttribute("data-role", "observer");
  await page.context().setOffline(false);
  await expect(tui).toHaveAttribute("data-connection", "connected", { timeout: 20000 });
  if (await tui.getAttribute("data-role") !== "controller") await menuAction(page, tui, "接管");
  await controlled(tui);
  await page.getByRole("button", { name: "退出登录" }).click();
  await page.getByRole("textbox", { name: "访问密码" }).fill(process.env.PERSISTTY_E2E_PASSWORD!);
  await page.getByRole("button", { name: "登录", exact: true }).click();
  await expect(page.getByRole("button", { name: "退出登录" })).toBeVisible();
  await page.goto(`/terminals/${id}`);
  await controlled(tui, 20000);
  await page.close();
  const reopened = await page.context().newPage();
  await reopened.goto(`/terminals/${id}`);
  await expect(reopened.locator(".xterm-rows")).toContainText("W03_DETERMINISTIC_TUI");
  const normal = JSON.parse(execFileSync("python3", ["-B", "../tests/integration/debian/run_remote.py", "browser-restart", "--root", root!], {
    cwd: process.cwd(), encoding: "utf8", timeout: 45000,
  })) as { checks: Record<string, boolean>; pane_count: number };
  expect(normal.pane_count).toBe(3);
  expect(Object.values(normal.checks).every(Boolean)).toBe(true);
  await controlled(reopened.locator(".terminal-runtime:visible"), 20000);
  await reopened.getByRole("textbox", { name: "Terminal input" }).click();
  await reopened.keyboard.type("z");
  await expect(reopened.locator(".xterm-rows")).toContainText("KEY: 7a");
});
