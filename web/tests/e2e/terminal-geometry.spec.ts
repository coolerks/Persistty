import { expect, test } from "@playwright/test";

// Real xterm/FitAddon in the workbench, with isolated transport and no shell.
test("终端实时与历史网格完整容纳最后一行和光标", async ({ page }) => {
  const inputs: string[] = []; let connections = 0;
  await page.route("**/api/v1/**", async route => {
    const path = new URL(route.request().url()).pathname;
    const content = Array.from({ length: 80 }, (_, index) => `history ${index}`).join("\n");
    const data = path === "/api/v1/terminals" ? { items: [{ id: "geometry", project_id: "interaction", display_name: "尺寸验收", working_directory: "/fixture", state: "running" }] } :
      path.endsWith("/entries") ? { items: [], next_cursor: "", project_version: 1 } :
      path.endsWith("/history") ? { content_base64: Buffer.from(content).toString("base64"), history_size: 80, returned_lines: 80, alternate_on: false, cols: 80, rows: 24, truncated: false } : null;
    if (!data) throw new Error(`未预期的请求 ${path}`);
    await route.fulfill({ json: { data, request_id: "fixture" } });
  });
  await page.routeWebSocket(/\/api\/v1\/events/, () => {});
  await page.routeWebSocket(/\/api\/v1\/terminals\/.*\/stream/, socket => {
    connections++;
    socket.onMessage(message => {
      if (typeof message !== "string") { inputs.push(Buffer.from(message).toString("base64")); return; }
      const command = JSON.parse(message) as { type: string; rows: number };
      if (command.type === "resize") socket.send(Buffer.from(`\x1b[2J\x1b[H\x1b[${command.rows};1Hlast row> \x1b[4munderline\x1b[0m `));
    });
    socket.send(JSON.stringify({ type: "ready", protocol: 3, terminal_id: "geometry", viewer_id: "fixture-viewer", role: "controller", generation: 1, cols: 80, rows: 24, pending_termination: null }));
    socket.send(Buffer.from("\x1b[?1049h"));
  });
  await page.goto("/tests/fixtures/workbench-interactions.html");
  const live = page.getByLabel("实时终端", { exact: true });
  const history = page.getByLabel("终端历史", { exact: true });
  const overflow = (selector: string) => page.locator(selector).evaluate(host => {
    const screen = host.querySelector(".xterm-screen")!.getBoundingClientRect();
    const bounds = host.getBoundingClientRect();
    const padding = getComputedStyle(host.querySelector(".xterm")!);
    return { bottom: screen.bottom - (bounds.bottom - Number.parseFloat(padding.paddingBottom)),
      right: screen.right - (bounds.right - Number.parseFloat(padding.paddingRight)) };
  });
  const backgroundsMatch = (selector: string) => page.locator(selector).evaluate(host =>
    getComputedStyle(host.querySelector(".xterm-viewport")!).backgroundColor ===
    getComputedStyle(host.querySelector(".xterm-scrollable-element")!).backgroundColor);
  for (const viewport of [{ width: 1440, height: 656 }, { width: 1001, height: 701 }, { width: 1280, height: 800 }]) {
    await page.setViewportSize(viewport);
    await page.evaluate(dark => document.documentElement.classList.toggle("dark", dark), viewport.width === 1001);
    await expect(live.locator(".xterm-rows > div").last()).toContainText("last row> underline");
    await live.click();
    await expect(live.locator(".xterm-cursor")).toBeVisible();
    await expect.poll(async () => Math.max(...Object.values(await overflow(".terminal-live")))).toBeLessThanOrEqual(0);
    await expect.poll(() => backgroundsMatch(".terminal-live")).toBe(true);
    await expect(live.locator(".xterm-underline-1")).toHaveText("underline");
    await page.locator(".terminal-tab").click({ button: "right" });
    await page.getByRole("menuitem", { name: "查看终端历史", exact: true }).click();
    await expect(history.locator(".xterm-rows")).toContainText("history 79");
    await expect.poll(async () => Math.max(...Object.values(await overflow(".terminal-history-surface")))).toBeLessThanOrEqual(0);
    await expect.poll(() => backgroundsMatch(".terminal-history-surface")).toBe(true);
    await page.locator(".terminal-tab").click({ button: "right" });
    await page.getByRole("menuitem", { name: "返回实时终端", exact: true }).click();
  }
  await live.click();
  await live.screenshot({ path: "test-results/terminal-geometry.png" });
  expect(inputs).toEqual([]); expect(connections).toBe(1);
});
