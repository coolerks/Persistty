import { expect, test, type Page } from "@playwright/test";
import { controlled, menuAction } from "./terminal-actions";

type Terminal = { id: string; display_name: string; state: string };

async function project(page: Page, multi = false) {
  test.setTimeout(100000);
  await page.goto("/projects");
  await page.getByLabel("访问密码").fill(process.env.PERSISTTY_E2E_PASSWORD!);
  const login = page.getByRole("button", { name: "登录", exact: true });
  let [response] = await Promise.all([page.waitForResponse(item => item.url().endsWith("/api/v1/auth/login")), login.click()]);
  if (response.status() === 429) {
    // Respect the real login cooldown; never disable authentication limits in the harness.
    await expect(login).toBeEnabled({ timeout: 70000 });
    [response] = await Promise.all([page.waitForResponse(item => item.url().endsWith("/api/v1/auth/login")), login.click()]);
  }
  expect(response.status()).toBe(200);
  await expect(page.getByRole("button", { name: `打开 ${process.env.PERSISTTY_E2E_PROJECT}` })).toBeVisible();
  const session = (await (await page.request.get("/api/v1/auth/session")).json()).data;
  const source = (await (await page.request.get("/api/v1/projects")).json()).data.items.find((item: { name: string }) => item.name === process.env.PERSISTTY_E2E_PROJECT);
  const path = source.folders[0].path;
  const createdResponse = await page.request.post("/api/v1/projects", { headers: { "Origin": process.env.PERSISTTY_E2E_BASE_URL!, "X-CSRF-Token": session.csrf_token },
    data: { name: `UI 验收 ${Date.now()}`, folder_paths: multi ? [path, `${process.env.PERSISTTY_E2E_REMOTE_ROOT}/extra`] : [path], main_index: 0 } });
  expect(createdResponse.ok()).toBe(true);
  const created = (await createdResponse.json()).data;
  await page.goto(`/projects/${created.id}`);
  await expect(page.getByRole("heading", { name: created.name })).toBeVisible();
  return created;
}

async function create(page: Page): Promise<Terminal> {
  const [response] = await Promise.all([
    page.waitForResponse(item => item.url().endsWith("/api/v1/terminals") && item.request().method() === "POST"),
    page.getByRole("button", { name: "新建终端" }).click(),
  ]);
  const created = (await response.json()).data as Terminal;
  await controlled(page.locator(`.terminal-runtime[data-terminal-id="${created.id}"]`));
  return created;
}

test("观察端输入确认丢弃字符粘贴及 IME，图标接管直接生效", async ({ page }) => {
  await project(page);
  const item = await create(page);
  const observer = await page.context().newPage();
  const sent: Buffer[] = [];
  observer.on("websocket", socket => socket.on("framesent", frame => { if (Buffer.isBuffer(frame.payload)) sent.push(frame.payload); }));
  await observer.goto(`/terminals/${item.id}`);
  const runtime = observer.locator(".terminal-runtime");
  await expect(runtime).toHaveAttribute("data-connection", "connected");
  await expect(runtime).toHaveAttribute("data-role", "observer");
  const input = runtime.getByRole("textbox", { name: "Terminal input" });
  await input.click(); await observer.keyboard.type("x");
  await expect(observer.getByRole("dialog", { name: "接管终端？" })).toBeVisible();
  expect(sent).toHaveLength(0);
  await observer.getByRole("button", { name: "接管", exact: true }).click();
  await controlled(runtime);
  expect(sent).toHaveLength(0);
  await input.click(); await observer.keyboard.type("echo UI_INPUT_ONCE"); await observer.keyboard.press("Enter");
  await expect(runtime.locator(".xterm-rows")).toContainText("UI_INPUT_ONCE");
  expect(Buffer.concat(sent.map(bytes => bytes.subarray(8))).toString()).toBe("echo UI_INPUT_ONCE\r");
  await page.getByRole("button", { name: `接管 ${item.display_name}`, exact: true }).click();
  await controlled(page.locator(".terminal-runtime:visible"));
  await expect(page.getByRole("dialog", { name: "接管终端？" })).toHaveCount(0);
  await expect(runtime).toHaveAttribute("data-role", "observer"); sent.length = 0;
  for (const type of ["paste", "compositionstart", "beforeinput"]) {
    await input.evaluate((element, type) => element.dispatchEvent(new Event(type, { bubbles: true, cancelable: true })), type);
    await expect(observer.getByRole("dialog", { name: "接管终端？" })).toBeVisible();
    await observer.getByRole("button", { name: "取消", exact: true }).click();
  }
  expect(sent).toHaveLength(0);
  await observer.close();
});

test("默认编号空号复用、重命名跨端同步且实例不变", async ({ page }) => {
  await project(page);
  const first = await create(page); const second = await create(page);
  expect([first.display_name, second.display_name]).toEqual(["终端", "终端1"]);
  const runtime = page.locator(`.terminal-runtime[data-terminal-id="${second.id}"]`);
  await runtime.evaluate(element => { element.setAttribute("data-test-identity", "stable-runtime"); });
  let connections = 0; page.on("websocket", () => connections++);
  const observer = await page.context().newPage(); await observer.goto(`/terminals/${second.id}`);
  await expect(observer.locator(".terminal-runtime")).toHaveAttribute("data-connection", "connected");
  await page.locator(`.terminal-tab[data-terminal-id="${second.id}"]`).getByRole("tab").dblclick();
  await page.getByLabel("终端名称").fill("构建任务"); await page.getByRole("button", { name: "保存", exact: true }).click();
  await expect(observer.locator(`.terminal-tab[data-terminal-id="${second.id}"]`).getByRole("tab")).toHaveText(/构建任务/);
  await expect(runtime).toHaveAttribute("data-test-identity", "stable-runtime");
  await menuAction(page, runtime, "刷新终端");
  await expect(runtime).toHaveAttribute("data-test-identity", "stable-runtime");
  expect(connections).toBe(0);
  const reused = await create(page); expect(reused.display_name).toBe("终端1"); expect(reused.id).not.toBe(second.id);
  await observer.close();
});

test("批量区域冻结与 v2 观察端取消整批，上方不受影响", async ({ page }) => {
  const configured = await project(page);
  const upper = await create(page); await menuAction(page, page.locator(`.terminal-runtime[data-terminal-id="${upper.id}"]`), "移到上方标签");
  const lower1 = await create(page); const lower2 = await create(page);
  const lower = page.locator(`.terminal-pane .terminal-runtime[data-terminal-id="${lower2.id}"]`);
  await page.evaluate(id => new Promise<void>(resolve => {
    const socket = new WebSocket(`ws://${location.host}/api/v1/terminals/${id}/stream`);
    const state = window as unknown as { legacySocket: WebSocket; legacyPending: Record<string, unknown> | null };
    state.legacySocket = socket; state.legacyPending = null;
    socket.onmessage = event => { if (typeof event.data !== "string") return; const message = JSON.parse(event.data); if (message.type === "ready") resolve(); if (message.type === "termination_pending") state.legacyPending = message; };
  }), lower1.id);
  await menuAction(page, lower, "全部关闭");
  const confirm = page.getByRole("dialog", { name: "终止终端？" });
  await expect(confirm.locator("li")).toHaveText([lower1.display_name, lower2.display_name]);
  await confirm.getByRole("button", { name: "接管并关闭" }).click();
  const pending = page.getByRole("dialog", { name: "终端终止状态" });
  await expect(pending.locator("li")).toHaveCount(2);
  await expect.poll(() => page.evaluate(() => (window as unknown as { legacyPending: unknown }).legacyPending)).not.toBeNull();
  const legacy = await page.evaluate(() => (window as unknown as { legacyPending: Record<string, unknown> }).legacyPending);
  expect(Object.keys(legacy).sort()).toEqual(["deadline", "request_id", "type"]);
  await page.evaluate(() => {
    const state = window as unknown as { legacySocket: WebSocket; legacyPending: { request_id: string } };
    state.legacySocket.send(JSON.stringify({ type: "cancel_termination", request_id: state.legacyPending.request_id }));
  });
  await expect(pending).toHaveCount(0);
  const items = (await (await page.request.get("/api/v1/terminals")).json()).data.items.filter((item: { project_id: string }) => item.project_id === configured.id);
  expect(items.map((item: Terminal) => item.state)).toEqual(["running", "running", "running"]);
  await page.evaluate(() => (window as unknown as { legacySocket: WebSocket }).legacySocket.close());
});

test("长多根文件树、三主题与桌面手机均无 document 溢出", async ({ page }) => {
  test.skip(!process.env.PERSISTTY_E2E_REMOTE_ROOT, "需要自有多根文件树 fixture");
  await project(page, true);
  await expect(page.locator(".app-header")).toHaveCount(0);
  await expect(page.locator(".workbench-title")).toHaveCount(1);
  await page.getByRole("button", { name: "加载更多" }).click();
  await page.getByRole("treeitem", { name: "ui-fixture-000.txt", exact: true }).click();
  await expect(page.locator(".monaco-editor")).toBeVisible();
  await page.getByRole("button", { name: "新建终端" }).click(); await page.getByRole("menuitem").first().click();
  await controlled(page.locator(".terminal-runtime:visible"));
  for (const [width, height] of [[1440, 656], [1024, 540], [390, 844], [844, 390]]) {
    await page.setViewportSize({ width: width!, height: height! });
    for (const theme of ["浅色", "深色", "跟随系统"]) {
      await page.getByRole("combobox", { name: "主题" }).click(); await page.getByRole("option", { name: theme, exact: true }).click();
      await page.keyboard.press("Escape");
      await expect.poll(() => page.evaluate(() => ({ height: document.documentElement.scrollHeight, client: document.documentElement.clientHeight, width: document.documentElement.scrollWidth, viewport: innerWidth }))).toEqual({ height, client: height, width, viewport: width });
    }
    const scroll = page.locator(".explorer-scroll");
    if (width! > 760) {
      await expect.poll(() => scroll.evaluate(element => element.scrollHeight > element.clientHeight)).toBe(true);
      await scroll.evaluate(element => { element.scrollTop = element.scrollHeight; });
    }
    await page.screenshot({ path: `test-results/ui-${width}x${height}.png` });
  }
  await page.setViewportSize({ width: 390, height: 844 });
  await page.getByRole("navigation", { name: "工作区视图" }).getByRole("button", { name: "终端" }).click();
  await page.getByRole("button", { name: "新建终端" }).click();
  await expect(page.getByRole("menuitem")).toHaveCount(2);
  await page.getByRole("menuitem").first().click();
  await controlled(page.locator(".terminal-runtime:visible"));
  await expect(page.locator(".terminal-heading, .terminal-session-bar, .terminal-runtime-toolbar")).toHaveCount(0);
  await expect(page.locator(".terminal-live")).toBeVisible();
  await page.screenshot({ path: "test-results/ui-mobile-terminal.png" });
});

test("真实 ABC 接管撤销不影响 DE 多成员终止，登出撤销整批", async ({ page }) => {
  const configured = await project(page);
  const session = (await (await page.request.get("/api/v1/auth/session")).json()).data;
  const headers = { Origin: process.env.PERSISTTY_E2E_BASE_URL!, "X-CSRF-Token": session.csrf_token };
  const ids: string[] = [];
  for (let i = 0; i < 5; i++) {
    const response = await page.request.post("/api/v1/terminals", { headers, data: { project_id: configured.id, project_version: configured.version, folder_id: configured.main_folder_id } });
    expect(response.status()).toBe(200); ids.push((await response.json()).data.id);
  }
  const targets = await page.evaluate(async ids => {
    const state = window as unknown as { batchSockets: WebSocket[]; batchTargets: Array<{ terminal_id: string; viewer_id: string; generation: number; role: string }> };
    state.batchSockets = []; state.batchTargets = [];
    return Promise.all(ids.map((id, index) => new Promise<{ terminal_id: string; viewer_id: string; generation: number }>(resolve => {
      const socket = new WebSocket(`ws://${location.host}/api/v1/terminals/${id}/stream?protocol=3`);
      state.batchSockets[index] = socket;
      socket.onmessage = event => {
        if (typeof event.data !== "string") return;
        const message = JSON.parse(event.data);
        if (message.type === "ready") {
          state.batchTargets[index] = { terminal_id: id, viewer_id: message.viewer_id, generation: message.generation, role: message.role };
          resolve({ terminal_id: id, viewer_id: message.viewer_id, generation: message.generation });
        } else if (message.type === "control") Object.assign(state.batchTargets[index]!, { generation: message.generation, role: message.role });
      };
    })));
  }, ids);
  const start = async (members: typeof targets) => {
    const response = await page.request.post("/api/v1/terminals/termination-batches", { headers, data: { members } });
    expect(response.status()).toBe(200); return (await response.json()).data;
  };
  const abc = await start(targets.slice(0, 3)); const de = await start(targets.slice(3));
  await page.evaluate(id => new Promise<void>(resolve => {
    const socket = new WebSocket(`ws://${location.host}/api/v1/terminals/${id}/stream`);
    const state = window as unknown as { takeoverSocket: WebSocket };
    state.takeoverSocket = socket;
    socket.onmessage = event => {
      if (typeof event.data !== "string") return;
      const message = JSON.parse(event.data);
      if (message.type === "ready") socket.send(JSON.stringify({ type: "takeover", generation: message.generation }));
      if (message.type === "control" && message.role === "controller") resolve();
    };
  }), ids[1]!);
  const read = async (id: string) => (await (await page.request.get(`/api/v1/terminals/termination-batches/${id}`)).json()).data;
  expect((await read(abc.request_id)).state).toBe("cancelled");
  expect((await read(de.request_id)).state).toBe("pending");
  await expect.poll(async () => (await read(de.request_id)).state, { timeout: 15000 }).toBe("completed");
  expect((await read(de.request_id)).results).toEqual(targets.slice(3).map(target => ({ terminal_id: target.terminal_id, state: "terminated" })).sort((a, b) => a.terminal_id.localeCompare(b.terminal_id)));
  const list = (await (await page.request.get("/api/v1/terminals")).json()).data.items;
  expect(list.filter((item: Terminal) => ids.slice(0, 3).includes(item.id)).every((item: Terminal) => item.state === "running")).toBe(true);
  await page.evaluate(() => {
    const state = window as unknown as { batchSockets: WebSocket[]; batchTargets: Array<{ generation: number }> };
    state.batchSockets[1]!.send(JSON.stringify({ type: "takeover", generation: state.batchTargets[1]!.generation }));
  });
  await expect.poll(() => page.evaluate(() => (window as unknown as { batchTargets: Array<{ role: string }> }).batchTargets[1]!.role)).toBe("controller");
  const current = await page.evaluate(() => (window as unknown as { batchTargets: Array<{ terminal_id: string; viewer_id: string; generation: number }> }).batchTargets.slice(0, 3).map(({ terminal_id, viewer_id, generation }) => ({ terminal_id, viewer_id, generation })));
  const revoked = await start(current);
  expect((await page.request.post("/api/v1/auth/logout", { headers })).status()).toBe(204);
  const login = await page.request.post("/api/v1/auth/login", { headers: { Origin: process.env.PERSISTTY_E2E_BASE_URL! }, data: { password: process.env.PERSISTTY_E2E_PASSWORD! } });
  expect(login.status()).toBe(200);
  await expect.poll(async () => (await read(revoked.request_id)).state, { timeout: 15000 }).toBe("cancelled");
  const remaining = (await (await page.request.get("/api/v1/terminals")).json()).data.items;
  expect(remaining.filter((item: Terminal) => ids.slice(0, 3).includes(item.id)).every((item: Terminal) => item.state === "running")).toBe(true);
  await page.evaluate(() => {
    const state = window as unknown as { batchSockets: WebSocket[]; takeoverSocket: WebSocket };
    state.batchSockets.forEach(socket => socket.close()); state.takeoverSocket.close();
  });
});
