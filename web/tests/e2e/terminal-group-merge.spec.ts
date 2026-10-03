import { expect, test, type Page } from "@playwright/test";

async function fixture(page: Page, baseURL: string | undefined, running = false) {
  if (!baseURL || !/^http:\/\/127\.0\.0\.1:\d+$/.test(baseURL)) throw new Error("静态产物仅从显式本机隔离实例读取");
  const origin = "http://10.42.0.10", errors: string[] = [], mutations: string[] = [];
  const project = { id: "merge-fixture", name: "分组合并", version: 1, main_folder_id: "root", folders: [{ id: "root", path: "/fixture" }] };
  const terminals = running ? ["alpha", "beta"].map(id => ({ id, project_id: project.id, display_name: id, working_directory: "/fixture", state: "running" })) : [];
  const connections: string[] = [], messages: unknown[] = [], inputs: string[] = [];
  page.on("pageerror", error => errors.push(error.message));
  await page.routeWebSocket(/\/api\/v1\/events/, () => {});
  await page.routeWebSocket(/\/api\/v1\/terminals\/.*\/stream/, socket => {
    const id = new URL(socket.url()).pathname.split("/").at(-2);
    if (!id) throw new Error("缺少终端ID");
    connections.push(id);
    socket.onMessage(message => {
      if (typeof message === "string") { const value: unknown = JSON.parse(message); messages.push(value); }
      else inputs.push(Buffer.from(message).toString("base64"));
    });
    socket.send(JSON.stringify({ type: "ready", protocol: 3, terminal_id: id, viewer_id: `viewer-${id}`, role: "controller", generation: 1, cols: 80, rows: 24, pending_termination: null }));
    socket.send(Buffer.from(`\x1b[?1049h\x1b[2J\x1b[H${id} prompt> `));
  });
  await page.route(`${origin}/**`, async route => {
    const request = route.request(), url = new URL(request.url());
    if (!url.pathname.startsWith("/api/")) { await route.fulfill({ response: await route.fetch({ url: new URL(url.pathname + url.search, baseURL).href }) }); return; }
    if (request.method() !== "GET") mutations.push(`${request.method()} ${url.pathname}`);
    const data = url.pathname.endsWith("/auth/session") ? { authenticated: true, csrf_token: "fixture", expires_at: "2099-01-01T00:00:00Z" } :
      url.pathname === `/api/v1/projects/${project.id}` ? project :
      url.pathname === "/api/v1/terminals" ? { items: terminals } :
      url.pathname.endsWith("/entries") ? { project_version: 1, next_cursor: "", items: [] } : null;
    if (!data) throw new Error(`未预期的请求 ${request.method()} ${url.pathname}`);
    await route.fulfill({ json: { data, request_id: "fixture" } });
  });
  if (running) await page.addInitScript(() => localStorage.setItem("persistty.workspace-view.v1", JSON.stringify({ version: 2, state: { projects: { "merge-fixture": {
    groups: [[], [], [], []], lowerCount: 2, terminals: { alpha: { region: "bottom", group: 0 }, beta: { region: "bottom", group: 1 } }, terminalOrder: ["alpha", "beta"], lowerActive: ["alpha", "beta", null, null],
  } } } })));
  await page.goto(`${origin}/projects/${project.id}`);
  await expect(page.getByRole("main", { name: "分组合并 工作台", exact: true })).toBeVisible();
  return { errors, mutations, connections, messages, inputs };
}

test("四个空终端分组可选择合并、刷新保持、恢复一组并重新拆分", async ({ page, baseURL }, info) => {
  const h = await fixture(page, baseURL);
  for (let i = 0; i < 3; i++) await page.getByRole("button", { name: "拆分终端面板", exact: true }).first().click();
  await expect(page.locator(".terminal-pane")).toHaveCount(4);
  await expect(page.getByRole("button", { name: "合并此终端分组", exact: true })).toHaveCount(4);
  await page.getByRole("button", { name: "合并此终端分组", exact: true }).nth(1).click();
  await expect(page.locator(".terminal-pane")).toHaveCount(3);
  await page.reload(); await expect(page.locator(".terminal-pane")).toHaveCount(3);
  for (let i = 0; i < 2; i++) await page.getByRole("button", { name: "合并此终端分组", exact: true }).first().click();
  await expect(page.locator(".terminal-pane")).toHaveCount(1);
  await expect(page.getByRole("button", { name: "合并此终端分组", exact: true })).toHaveCount(0);
  await page.reload(); await expect(page.locator(".terminal-pane")).toHaveCount(1);
  await page.getByRole("button", { name: "拆分终端面板", exact: true }).click();
  await expect(page.locator(".terminal-pane")).toHaveCount(2);
  await page.screenshot({ path: info.outputPath("merge-button.png") });
  await page.getByRole("button", { name: "合并此终端分组", exact: true }).last().click();
  await page.getByRole("button", { name: "收起终端面板", exact: true }).click();
  await expect(page.getByTestId("terminal")).toHaveCSS("height", "0px");
  await page.getByRole("button", { name: "终端面板", exact: true }).click();
  await expect(page.getByTestId("terminal")).not.toHaveCSS("height", "0px");
  expect(h.errors).toEqual([]); expect(h.mutations).toEqual([]);
});

test("运行中分组合并保留两条会话、稳定DOM和WS，零终止接管及输入", async ({ page, baseURL }) => {
  const h = await fixture(page, baseURL, true);
  const beta = page.locator('.terminal-runtime[data-terminal-id="beta"]');
  await expect(beta.locator(".xterm-rows")).toContainText("beta prompt>");
  const original = await beta.elementHandle();
  await page.getByRole("button", { name: "合并此终端分组", exact: true }).last().click();
  await expect(page.locator(".terminal-pane")).toHaveCount(1);
  await expect(page.locator(".terminal-tab")).toHaveCount(2);
  await page.locator('.terminal-tab[data-terminal-id="beta"]').click();
  await expect(beta.locator(".xterm-rows")).toContainText("beta prompt>");
  expect(await beta.evaluate((current, previous) => current === previous, original)).toBe(true);
  expect(h.connections.sort()).toEqual(["alpha", "beta"]);
  expect(h.messages.every(message => typeof message === "object" && message !== null && "type" in message && message.type === "resize")).toBe(true);
  expect(h.inputs).toEqual([]); expect(h.mutations).toEqual([]); expect(h.errors).toEqual([]);
});
