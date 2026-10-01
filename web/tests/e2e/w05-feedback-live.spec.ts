import { expect, test } from "@playwright/test";
import { decodeEnvelope, decodeList, decodeProject, decodeSession, decodeTerminal, decodeTerminalHistory } from "../../src/lib/api/decoder";
import { controlled } from "./terminal-actions";

test("W05 Debian 空历史保留 live，长历史上下切换单 WS 且零额外输入", async ({ page }, testInfo) => {
  test.skip(process.env.PERSISTTY_E2E_W05 !== "1", "只在显式 W05 私有 Debian 实例运行");
  const root = process.env.PERSISTTY_E2E_REMOTE_ROOT;
  expect(root).toMatch(/^\/tmp\/persistty-browser-[A-Za-z0-9]{8}$/); expect(process.env.PERSISTTY_E2E_PROJECT).toBe("W03 隔离终端");
  await page.setViewportSize({ width: 1440, height: 900 }); await page.goto("/projects");
  await page.getByLabel("访问密码").fill(process.env.PERSISTTY_E2E_PASSWORD!); await page.getByRole("button", { name: "登录", exact: true }).click();
  await expect(page.getByRole("button", { name: "新建项目", exact: true })).toBeVisible();
  const session = decodeEnvelope(await (await page.request.get("/api/v1/auth/session")).json(), decodeSession);
  const projects = decodeEnvelope(await (await page.request.get("/api/v1/projects")).json(), decodeList(decodeProject));
  const project = projects.find(item => item.name === "W03 隔离终端"); if (!project) throw new Error("专属测试项目缺失");
  expect(project.folders[0]?.path).toBe(`${root}/project`);
  const response = await page.request.post("/api/v1/terminals", { headers: { Origin: process.env.PERSISTTY_E2E_BASE_URL!, "X-CSRF-Token": session.csrf_token }, data: { project_id: project.id, project_version: project.version, folder_id: project.main_folder_id } });
  expect(response.ok()).toBe(true); const terminal = decodeEnvelope(await response.json(), decodeTerminal);
  let connects = 0, inputs = 0; const errors: string[] = [];
  page.on("pageerror", error => errors.push(error.message));
  page.on("websocket", socket => { if (!socket.url().includes(`/${terminal.id}/stream`)) return; connects++; socket.on("framesent", frame => { if (Buffer.isBuffer(frame.payload)) inputs++; }); });
  await page.goto(`/projects/${project.id}`);
  await page.locator(`.terminal-tab[data-terminal-id="${terminal.id}"]`).click();
  const runtime = page.locator(`.terminal-runtime[data-terminal-id="${terminal.id}"]`); await controlled(runtime);
  const live = runtime.getByLabel("实时终端", { exact: true }), history = runtime.getByLabel("终端历史", { exact: true });
  await expect.poll(() => live.locator(".xterm-rows").innerText()).not.toBe("");
  const empty = decodeEnvelope(await (await page.request.get(`/api/v1/terminals/${terminal.id}/history`)).json(), decodeTerminalHistory); expect(empty.history_size).toBe(0);
  const beforeEmptyScroll = inputs;
  await live.hover(); await page.mouse.wheel(0, -160); await expect(runtime.getByText("暂无历史输出", { exact: true })).toBeVisible();
  await expect(live).toBeVisible(); await expect(history).toHaveCount(0); expect(inputs).toBe(beforeEmptyScroll);
  await live.getByRole("textbox", { name: "Terminal input" }).click();
  await page.keyboard.type("for i in $(seq 1 120); do printf 'SCROLL-ROW-%03d\\n' \"$i\"; done"); await page.keyboard.press("Enter");
  await expect(live.locator(".xterm-rows")).toContainText("SCROLL-ROW-120"); const sent = inputs;
  for (let i = 0; i < 3; i++) {
    const historyResponse = page.waitForResponse(response => new URL(response.url()).pathname === `/api/v1/terminals/${terminal.id}/history`);
    await live.hover(); await page.mouse.wheel(0, -160); await expect(history.locator(".xterm-rows")).toContainText("SCROLL-ROW-");
    const captured = decodeEnvelope(await (await historyResponse).json(), decodeTerminalHistory);
    const lastHistoryLine = Buffer.from(captured.content_base64, "base64").toString("utf8").trimEnd().split("\n").at(-1)!;
    expect(lastHistoryLine).toMatch(/^SCROLL-ROW-\d+$/);
    await history.hover(); await page.mouse.wheel(0, -120); await expect(history.locator(".xterm-rows")).toContainText("SCROLL-ROW-");
    await expect.poll(async () => {
      if (await live.isVisible()) return true;
      await page.mouse.wheel(0, 500);
      if (await live.isVisible()) return true;
      return (await history.locator(".xterm-rows").innerText()).includes(lastHistoryLine);
    }).toBe(true);
    if (!await live.isVisible()) await page.mouse.wheel(0, 160);
    await expect(live).toBeVisible(); await expect(live.locator(".xterm-rows")).toContainText("SCROLL-ROW-120");
  }
  expect(inputs).toBe(sent); expect(connects).toBe(1); expect(errors).toEqual([]);
  await page.screenshot({ path: testInfo.outputPath("w05-live-scroll-feedback.png") });
});
