import { expect, test, type WebSocketRoute } from "@playwright/test";

test.use({ launchOptions: { ignoreDefaultArgs: ["--hide-scrollbars"] } });

for (const role of ["controller", "observer"] as const) test(`${role} 历史 LF 排版、滚轮滚动零 shell 输入、live DOM 与 TUI 鼠标`, async ({ page }, testInfo) => {
  const errors: string[] = []; const inputs: number[][] = []; let connects = 0; let histories = 0; let stream: WebSocketRoute | undefined;
  let releaseHistory: () => void = () => {};
  const pendingHistory = new Promise<void>(resolve => { releaseHistory = resolve; });
  page.on("pageerror", error => errors.push(error.message));
  const terminal = { id: "scroll-terminal", project_id: "interaction", display_name: "滚动验收", working_directory: "/fixture", state: "running" };
  const lines = Array.from({ length: 90 }, (_, index) => `\x1b[32m行${String(index).padStart(3, "0")}\x1b[0m 中文输出 ${index}`);
  const content = lines.join("\n") + "\nCRLF一\r\nCRLF二\r\n";
  await page.route("**/api/v1/**", async route => {
    const url = new URL(route.request().url()); let data: unknown;
    if (url.pathname === "/api/v1/terminals") data = { items: [terminal] };
    else if (url.pathname.endsWith("/entries")) data = { items: [], next_cursor: "", project_version: 1 };
    else if (url.pathname.endsWith("/history")) { histories++; if (histories === 2) await pendingHistory; data = { content_base64: Buffer.from(content).toString("base64"), history_size: 92, returned_lines: 92, alternate_on: false, cols: 80, rows: 24, truncated: false }; }
    else throw new Error(`未预期的请求 ${url.pathname}`);
    await route.fulfill({ json: { data, request_id: "fixture" } });
  });
  await page.routeWebSocket(/\/api\/v1\/events/, () => {});
  await page.routeWebSocket(/\/api\/v1\/terminals\/.*\/stream/, socket => {
    connects++; stream = socket;
    socket.onMessage(message => { if (typeof message !== "string") inputs.push([...message]); });
    socket.send(JSON.stringify({ type: "ready", protocol: 3, terminal_id: terminal.id, viewer_id: "fixture-viewer", role, generation: 1, cols: 80, rows: 24, pending_termination: null }));
    // Reproduce tmux's outer alternate screen even when the pane is an ordinary shell.
    socket.send(Buffer.from("\x1b[?1049h\x1b[2J\x1b[Hlive prompt> "));
  });
  await page.goto("/tests/fixtures/workbench-interactions.html");
  const live = page.getByLabel("实时终端", { exact: true });
  const history = page.getByLabel("终端历史", { exact: true });
  const rows = () => history.locator(".xterm-rows > div").evaluateAll(elements => elements.map(element => (element.textContent ?? "").replaceAll("\u00a0", " ").trimEnd()).filter(Boolean));
  await expect(page.locator(".terminal-runtime")).toHaveAttribute("data-connection", "connected");
  const liveHeight = (await live.boundingBox())!.height;
  const original = await live.locator(".xterm").elementHandle(); if (!original) throw new Error("live xterm 未创建");
  await page.locator(".terminal-tab").click({ button: "right" });
  await page.getByRole("menuitem", { name: "查看终端历史", exact: true }).click();
  await expect.poll(rows).toContain("CRLF二");
  await expect.poll(async () => (await live.boundingBox())!.height).toBe(liveHeight);
  const visible = await rows();
  expect(visible.filter(line => line.includes("中文输出")).every(line => /^行\d{3} 中文输出 \d+$/.test(line))).toBe(true);
  expect(visible).toContain("CRLF一"); expect(visible).toContain("CRLF二");
  // Trackpad momentum can stay latched to the original live target after hiding it.
  await original.dispatchEvent("wheel", { deltaX: -180, deltaY: -0.25, bubbles: true, cancelable: true });
  await expect.poll(rows).toEqual(visible);
  const historyElement = await history.locator(".xterm").elementHandle();
  if (!historyElement) throw new Error("history xterm 未创建");
  const preciseWheel = (count: number, deltaY: number, deltaX = 0) => historyElement.evaluate((element, args) => {
    for (let index = 0; index < args.count; index++) {
      const event = new WheelEvent("wheel", { deltaY: args.deltaY, deltaX: args.deltaX, bubbles: true, cancelable: true });
      // Emulate modern fractional trackpad events, without synthetic Chromium's
      // zero-valued deprecated wheelDelta properties masking deltaY in xterm.
      Object.defineProperties(event, { wheelDeltaY: { value: undefined }, wheelDeltaX: { value: undefined }, wheelDelta: { value: undefined } });
      element.querySelector(".xterm-scrollable-element")?.dispatchEvent(event);
    }
  }, { count, deltaY, deltaX });
  await preciseWheel(1, -0.25);
  await expect.poll(rows).toEqual(visible);
  await preciseWheel(40, -0.5);
  const firstIndex = async () => Number((await rows())[0]?.match(/^行(\d+)/)?.[1]);
  const initialIndex = Number(visible[0]?.match(/^行(\d+)/)?.[1]);
  await expect.poll(async () => initialIndex - await firstIndex()).toBeGreaterThanOrEqual(1);
  expect(initialIndex - await firstIndex()).toBeLessThanOrEqual(4);
  const preciseRows = await rows();
  await preciseWheel(20, -0.25, -180);
  await history.hover(); await page.mouse.wheel(180, 0);
  await expect.poll(rows).toEqual(preciseRows);
  await original.dispatchEvent("wheel", { deltaY: 64, bubbles: true, cancelable: true });
  await expect.poll(rows).toContain("CRLF二");
  await preciseWheel(1, 0.25);
  await expect(history).toBeVisible();
  expect(histories).toBe(1);
  expect(await historyElement.evaluate(element => element.isConnected)).toBe(true);
  await history.hover(); await page.mouse.wheel(0, -160);
  await expect.poll(rows).not.toEqual(visible);
  expect(inputs).toEqual([]);
  await page.screenshot({ path: testInfo.outputPath(`${role}-history.png`) });
  await page.locator(".terminal-tab").click({ button: "right" });
  await page.getByRole("menuitem", { name: "返回实时终端", exact: true }).click();
  await expect(live).toBeVisible();
  expect(await original.evaluate(element => element.isConnected)).toBe(true);
  await live.hover();
  await live.dispatchEvent("wheel", { deltaY: -120, ctrlKey: true, bubbles: true, cancelable: true });
  await expect(history).toHaveCount(0); expect(inputs).toEqual([]); expect(histories).toBe(1);
  await live.hover(); await page.mouse.wheel(0, 120);
  await expect(history).toHaveCount(0); expect(inputs).toEqual([]);
  await page.mouse.wheel(0, -120);
  await expect(page.getByRole("status")).toContainText("读取中");
  await expect(live).toBeVisible(); await expect(live.locator(".xterm-rows")).toContainText("live prompt>");
  for (let index = 0; index < 20; index++) await original.dispatchEvent("wheel", { deltaY: -0.5, bubbles: true, cancelable: true });
  await live.locator("textarea").evaluate(element => element.focus());
  await page.keyboard.type("readonly"); expect(inputs).toEqual([]); expect(histories).toBe(2);
  releaseHistory();
  await expect(history).toBeVisible(); await expect.poll(() => histories).toBe(2);
  await expect.poll(rows).not.toEqual(visible);
  await history.hover();
  await original.dispatchEvent("wheel", { deltaY: 300, bubbles: true, cancelable: true });
  await expect.poll(rows).toContain("CRLF二"); await page.mouse.wheel(0, 120);
  await expect(live).toBeVisible(); await expect(history).toHaveCount(0);
  expect(inputs).toEqual([]); expect(connects).toBe(1);
  // Explicit application mouse tracking preserves TUI wheel for controller only.
  if (!stream) throw new Error("WS 未连接");
  stream.send(Buffer.from("\x1b[?1000h\x1b[?1006h"));
  await live.hover(); await page.mouse.wheel(0, -120);
  if (role === "controller") {
    await expect.poll(() => inputs.length).toBeGreaterThan(0);
    expect(inputs.every(frame => new TextDecoder().decode(Uint8Array.from(frame.slice(8))).startsWith("\x1b[<64;"))).toBe(true);
    const count = inputs.length;
    await page.keyboard.down("Shift"); await page.mouse.wheel(0, -120); await page.keyboard.up("Shift");
    await expect(history).toBeVisible(); expect(inputs).toHaveLength(count);
  } else { await expect(history).toBeVisible(); expect(inputs).toEqual([]); }
  expect(connects).toBe(1); expect(errors).toEqual([]);
});

for (const snapshot of [
  { history_size: 0, returned_lines: 1, content: "\n" },
  { history_size: 0, returned_lines: 0, content: "" },
  { history_size: 5, returned_lines: 1, content: "   \n" },
]) test(`空历史 ${snapshot.history_size}/${snapshot.returned_lines} 行时保留实时画面，不显示空白历史`, async ({ page }) => {
  const inputs: string[] = []; let connections = 0;
  const terminal = { id: "empty-history", project_id: "interaction", display_name: "空历史验收", working_directory: "/fixture", state: "running" };
  await page.route("**/api/v1/**", async route => {
    const path = new URL(route.request().url()).pathname;
    const data = path === "/api/v1/terminals" ? { items: [terminal] } : path.endsWith("/entries") ? { items: [], next_cursor: "", project_version: 1 } : path.endsWith("/history") ? { history_size: snapshot.history_size, returned_lines: snapshot.returned_lines, content_base64: Buffer.from(snapshot.content).toString("base64"), cols: 80, rows: 24, alternate_on: false, truncated: false } : null;
    if (!data) throw new Error(`unexpected ${path}`);
    await route.fulfill({ json: { data, request_id: "fixture" } });
  });
  await page.routeWebSocket(/\/api\/v1\/events/, () => {});
  await page.routeWebSocket(/\/api\/v1\/terminals\/.*\/stream/, socket => {
    connections++; socket.onMessage(message => { if (typeof message !== "string") inputs.push(Buffer.from(message).toString("base64")); });
    socket.send(JSON.stringify({ type: "ready", protocol: 3, terminal_id: terminal.id, viewer_id: "fixture", role: "controller", generation: 1, cols: 80, rows: 24, pending_termination: null }));
    socket.send(Buffer.from("\x1b[?1049h\x1b[2J\x1b[Hvisible prompt> "));
  });
  await page.goto("/tests/fixtures/workbench-interactions.html");
  const live = page.getByLabel("实时终端", { exact: true });
  await expect(live.locator(".xterm-rows")).toContainText("visible prompt>");
  await live.hover(); await page.mouse.wheel(0, -120);
  await expect(page.getByText("暂无历史输出", { exact: true })).toBeVisible();
  await expect(live).toBeVisible(); await expect(live.locator(".xterm-rows")).toContainText("visible prompt>");
  await expect(page.getByLabel("终端历史", { exact: true })).toHaveCount(0);
  expect(connections).toBe(1); expect(inputs).toEqual([]);
});

test("短历史重复上下切换仍有内容、保持单连接且零 shell 输入", async ({ page }) => {
  const inputs: string[] = []; let connections = 0;
  const terminal = { id: "short-history", project_id: "interaction", display_name: "短历史验收", working_directory: "/fixture", state: "running" };
  await page.route("**/api/v1/**", async route => {
    const path = new URL(route.request().url()).pathname;
    const data = path === "/api/v1/terminals" ? { items: [terminal] } : path.endsWith("/entries") ? { items: [], next_cursor: "", project_version: 1 } : path.endsWith("/history") ? { history_size: 2, returned_lines: 2, content_base64: Buffer.from("earlier line\nrecent line\n").toString("base64"), cols: 80, rows: 24, alternate_on: false, truncated: false } : null;
    if (!data) throw new Error(`unexpected ${path}`);
    await route.fulfill({ json: { data, request_id: "fixture" } });
  });
  await page.routeWebSocket(/\/api\/v1\/events/, () => {});
  await page.routeWebSocket(/\/api\/v1\/terminals\/.*\/stream/, socket => {
    connections++; socket.onMessage(message => { if (typeof message !== "string") inputs.push(Buffer.from(message).toString("base64")); });
    socket.send(JSON.stringify({ type: "ready", protocol: 3, terminal_id: terminal.id, viewer_id: "fixture", role: "controller", generation: 1, cols: 80, rows: 24, pending_termination: null }));
    socket.send(Buffer.from("\x1b[?1049h\x1b[2J\x1b[Hcurrent prompt> "));
  });
  await page.goto("/tests/fixtures/workbench-interactions.html");
  const live = page.getByLabel("实时终端", { exact: true }), history = page.getByLabel("终端历史", { exact: true });
  await expect(live.locator(".xterm-rows")).toContainText("current prompt>");
  for (let i = 0; i < 4; i++) {
    await live.hover(); await page.mouse.wheel(0, -120);
    await expect(history.locator(".xterm-rows")).toContainText("recent line");
    await history.hover(); await page.mouse.wheel(0, 240);
    await expect(live).toBeVisible(); await expect(live.locator(".xterm-rows")).toContainText("current prompt>");
  }
  expect(connections).toBe(1); expect(inputs).toEqual([]);
});


test("历史绘制、刷新和返回实时始终保留非空画面与尺寸", async ({ page }, info) => {
  let requests = 0, connections = 0; const inputs: string[] = [], errors: string[] = [];
  page.on("pageerror", error => errors.push(error.message));
  await page.route("**/api/v1/**", async route => {
    const path = new URL(route.request().url()).pathname;
    let data: unknown;
    if (path === "/api/v1/terminals") data = { items: [{ id: "paint", project_id: "interaction", display_name: "绘制验收", working_directory: "/fixture", state: "running" }] };
    else if (path.endsWith("/entries")) data = { items: [], next_cursor: "", project_version: 1 };
    else if (path.endsWith("/history")) {
      requests++;
      data = { content_base64: Buffer.from(Array.from({ length: 200 }, (_, index) => `snapshot ${requests} row ${index}`).join("\n")).toString("base64"), history_size: 200, returned_lines: 200, cols: 80, rows: 24, alternate_on: false, truncated: false };
    } else throw new Error(`unexpected ${path}`);
    await route.fulfill({ json: { data, request_id: "fixture" } });
  });
  await page.routeWebSocket(/\/api\/v1\/events/, () => {});
  await page.routeWebSocket(/\/api\/v1\/terminals\/.*\/stream/, socket => {
    connections++; socket.onMessage(message => { if (typeof message !== "string") inputs.push(Buffer.from(message).toString("base64")); });
    socket.send(JSON.stringify({ type: "ready", protocol: 3, terminal_id: "paint", viewer_id: "fixture", role: "controller", generation: 1, cols: 80, rows: 24, pending_termination: null }));
    socket.send(Buffer.from("retained live prompt> "));
  });
  await page.goto("/tests/fixtures/workbench-interactions.html");
  const live = page.getByLabel("实时终端", { exact: true }), history = page.getByLabel("终端历史", { exact: true });
  await expect(live.locator(".xterm-rows")).toContainText("retained live prompt>");
  const original = await live.locator(".xterm").elementHandle(); if (!original) throw new Error("missing live");
  await page.locator(".terminal-runtime").evaluate(runtime => {
    const state = window as typeof window & { blankPaints: number; stopPaintSampling(): void };
    state.blankPaints = 0; let active = true;
    const sample = () => {
      const surfaces = runtime.querySelectorAll<HTMLElement>(".terminal-live, .terminal-history-surface");
      const visibleContent = Array.from(surfaces).some(surface => surface.getBoundingClientRect().height > 0 && getComputedStyle(surface).visibility !== "hidden" && (surface.querySelector(".xterm-rows")?.textContent ?? "").trim().length > 0);
      if (!visibleContent) state.blankPaints++;
      if (active) requestAnimationFrame(sample);
    };
    state.stopPaintSampling = () => { active = false; }; requestAnimationFrame(sample);
  });
  const retainedSizes: boolean[] = [];
  for (let index = 0; index < 3; index++) {
    const liveHeight = (await live.boundingBox())!.height;
    await live.hover(); await page.mouse.wheel(0, -120);
    await expect(history).toBeVisible(); await expect(history.locator(".xterm-rows")).toContainText(`snapshot ${index * 2 + 1}`);
    retainedSizes.push((await live.boundingBox())?.height === liveHeight);
    await page.locator(".terminal-tab").click({ button: "right" });
    await page.getByRole("menuitem", { name: "刷新终端历史", exact: true }).click();
    await expect(history.locator(".xterm-rows")).toContainText(`snapshot ${index * 2 + 2}`); await expect(history).toBeVisible();
    await page.setViewportSize({ width: 1440 - index * 160, height: 845 - index * 60 });
    await expect(history.locator(".xterm-rows")).toContainText(`snapshot ${index * 2 + 2}`);
    await page.locator(".terminal-tab").click({ button: "right" });
    await page.getByRole("menuitem", { name: "返回实时终端", exact: true }).click();
    await expect(live).toBeVisible(); await expect(live.locator(".xterm-rows")).toContainText("retained live prompt>");
    expect(await original.evaluate(element => element.isConnected)).toBe(true);
  }
  await page.screenshot({ path: info.outputPath("history-return-live.png") });
  const blankPaints = await page.evaluate(() => {
    const state = window as typeof window & { blankPaints: number; stopPaintSampling(): void };
    state.stopPaintSampling(); return state.blankPaints;
  });
  expect(blankPaints).toBe(0); expect(retainedSizes).toEqual([true, true, true]); expect(requests).toBe(6); expect(connections).toBe(1); expect(inputs).toEqual([]); expect(errors).toEqual([]);
});
