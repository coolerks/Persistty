import { expect, test, type WebSocketRoute } from "@playwright/test";
import fixture from "../../../tests/contracts/terminal-runtime.json" with { type: "json" };
import grid from "../fixtures/terminal-grid-recording.json" with { type: "json" };

for (const role of ["controller", "observer"] as const) test(`${role} 设备查询只回答自身 attach，不生成 shell 数字输入`, async ({ page }) => {
  const commands: unknown[] = []; const inputs: string[] = []; const errors: string[] = [];
  let stream: WebSocketRoute | undefined; let connects = 0;
  page.on("pageerror", error => errors.push(error.message));
  await page.route("**/api/v1/**", async route => {
    const path = new URL(route.request().url()).pathname;
    const data = path === "/api/v1/terminals" ? { items: [{ id: "device-terminal", project_id: "interaction", display_name: "设备查询验收", working_directory: "/fixture", state: "running" }] } :
      path.endsWith("/entries") ? { items: [], next_cursor: "", project_version: 1 } : null;
    if (!data) throw new Error(`未预期的请求 ${path}`);
    await route.fulfill({ json: { data, request_id: "fixture" } });
  });
  await page.routeWebSocket(/\/api\/v1\/events/, () => {});
  await page.routeWebSocket(/\/api\/v1\/terminals\/.*\/stream/, socket => {
    stream = socket; connects++;
    socket.onMessage(message => {
      if (typeof message === "string") { const value: unknown = JSON.parse(message); commands.push(value); }
      else inputs.push(new TextDecoder().decode(message.subarray(8)));
    });
    socket.send(JSON.stringify({ type: "ready", protocol: 3, terminal_id: "device-terminal", viewer_id: "fixture-viewer", role, generation: 1, cols: 80, rows: 24, pending_termination: null }));
    socket.send(Buffer.from("\x1b[?1049h\x1b[2J\x1b[Hprompt> "));
  });
  await page.goto("/tests/fixtures/workbench-interactions.html");
  const live = page.getByLabel("实时终端", { exact: true });
  await expect(live.locator(".xterm-rows")).toContainText("prompt>");
  if (!stream) throw new Error("终端 WS 未连接");
  // Omitted/zero parameters and fragmented/repeated query packets.
  stream.send(Buffer.from("\x1b[c\x1b[0c\x1b[>c\x1b["));
  stream.send(Buffer.from(">0c\x1b[1c\x1b[>1c\r\nquery done"));
  const replies = () => commands.filter(value => typeof value === "object" && value !== null && "type" in value && value.type === "device_attributes");
  await expect.poll(replies).toEqual([fixture.device_attributes_primary_v3, fixture.device_attributes_primary_v3, fixture.device_attributes_secondary_v3, fixture.device_attributes_secondary_v3]);
  await expect(live.locator(".xterm-rows")).toContainText("query done");
  expect(inputs).toEqual([]);
  await expect(page.locator(".terminal-runtime")).toHaveAttribute("data-role", role);
  if (role === "controller") {
    await live.click(); await page.keyboard.type("1;2c0;276;0c"); await page.keyboard.press("ArrowUp");
    await expect.poll(() => inputs.join("")).toBe("1;2c0;276;0c\x1b[A");
    const beforePaste = inputs.length;
    await live.locator("textarea").evaluate(element => {
      const clipboardData = new DataTransfer(); clipboardData.setData("text/plain", "\x1b[?1;2c\x1b[>0;276;0c");
      element.dispatchEvent(new ClipboardEvent("paste", { clipboardData, bubbles: true, cancelable: true }));
    });
    await expect.poll(() => inputs.slice(beforePaste).join("")).toBe("\x1b[?1;2c\x1b[>0;276;0c");
  }
  const count = inputs.length;
  stream.send(Buffer.from("\x1b[>c\x1b[c\r\nlate query done"));
  await expect.poll(() => replies().length).toBe(6);
  await expect(live.locator(".xterm-rows")).toContainText("late query done");
  if (role === "observer") {
    await expect(live.locator(".xterm-rows > div")).toHaveCount(24);
    // Real isolated tmux recording: an 80x24 output attach sees a 140x12 pane.
    // The border and middots are terminal bytes, not a CSS scrollbar.
    stream.send(Buffer.from(grid.initial, "base64"));
    stream.send(Buffer.from(grid.before, "base64"));
    await expect(live.locator(".xterm-rows")).toContainText("─".repeat(20));
    await expect(live.locator(".xterm-rows")).toContainText("·".repeat(20));
    stream.send(JSON.stringify({ type: "resized", cols: 140, rows: 12 }));
    await expect(live.locator(".xterm-rows > div")).toHaveCount(12);
    stream.send(Buffer.from(grid.after, "base64"));
    await expect(live.locator(".xterm-rows")).not.toContainText("─");
    await expect(live.locator(".xterm-rows")).not.toContainText("·");
    await page.setViewportSize({ width: 900, height: 600 });
    await expect(live.locator(".xterm-rows > div")).toHaveCount(12);
    expect(commands.some(value => typeof value === "object" && value !== null && "type" in value && value.type === "resize")).toBe(false);
  }
  expect(inputs).toHaveLength(count); expect(connects).toBe(1); expect(errors).toEqual([]);
});
