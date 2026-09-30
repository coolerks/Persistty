import { menuAction, controlled } from "./terminal-actions";
import { expect, test, type Page } from "@playwright/test";

async function openProject(page: Page) {
  await page.goto("/projects");
  await page.getByRole("textbox", { name: "访问密码" }).fill(process.env.PERSISTTY_E2E_PASSWORD!);
  await page.getByRole("button", { name: "登录", exact: true }).click();
  await page.getByRole("button", { name: `打开 ${process.env.PERSISTTY_E2E_PROJECT}` }).click();
  await page.getByRole("button", { name: "当前标签页" }).click();
  await expect(page.getByRole("heading", { name: process.env.PERSISTTY_E2E_PROJECT! })).toBeVisible();
}

test("桌面终端与历史快照", async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 800 });
  await openProject(page);
  const [createdResponse] = await Promise.all([
    page.waitForResponse(response => response.url().endsWith("/api/v1/terminals") && response.request().method() === "POST"),
    page.getByRole("button", { name: "新建终端" }).click(),
  ]);
  const created = await createdResponse.json() as { data: { id: string } };
  const runtime = page.locator(`.terminal-runtime[data-terminal-id="${created.data.id}"]`);
  await controlled(runtime);
  const input = runtime.getByRole("textbox", { name: "Terminal input" });
  await input.click();
  await page.keyboard.type("pwd");
  await page.keyboard.press("Enter");
  await expect(runtime.locator(".terminal-live .xterm-screen")).toBeVisible();
  await expect(runtime.locator(".terminal-live .xterm-rows")).toContainText("pwd");
  await page.screenshot({ path: "test-results/w03-desktop.png" });
  await menuAction(page, runtime, "查看终端历史");
  await expect(page.getByText("普通历史快照")).toBeVisible();
  await expect(page.getByRole("alert")).toHaveCount(0);
  await menuAction(page, runtime, "返回实时终端");
});

test("手机终端快捷键和窄屏布局", async ({ page }) => {
  const inputs: Buffer[] = [];
  let id = "";
  page.on("websocket", socket => socket.on("framesent", frame => {
    if (id && socket.url().includes(`/${id}/stream`) && Buffer.isBuffer(frame.payload)) inputs.push(frame.payload.subarray(8));
  }));
  await page.setViewportSize({ width: 390, height: 844 });
  await openProject(page);
  await page.getByRole("navigation", { name: "工作区视图" }).getByRole("button", { name: "终端" }).click();
  const [response] = await Promise.all([
    page.waitForResponse(item => item.url().endsWith("/api/v1/terminals") && item.request().method() === "POST"),
    page.getByRole("button", { name: "新建终端" }).click(),
  ]);
  id = ((await response.json()) as { data: { id: string } }).data.id;
  await controlled(page.locator(".terminal-runtime:visible"));
  const shortcuts = page.getByLabel("手机终端快捷键");
  await expect(shortcuts).toBeVisible();
  for (const name of ["Ctrl", "Alt", "Esc", "Tab", "上箭头", "下箭头", "左箭头", "右箭头", "^C", "^L", "^S", "^Z", "/"]) {
    await expect(shortcuts.getByRole("button", { name, exact: true })).toHaveCount(1);
  }
  for (const [name, bytes] of [["Esc", "\x1b"], ["Tab", "\t"], ["上箭头", "\x1b[A"], ["下箭头", "\x1b[B"], ["左箭头", "\x1b[D"], ["右箭头", "\x1b[C"], ["^C", "\x03"], ["^L", "\x0c"], ["^S", "\x13"], ["^Z", "\x1a"], ["/", "/"]]) {
    inputs.length = 0;
    await shortcuts.getByRole("button", { name, exact: true }).click();
    await expect.poll(() => Buffer.concat(inputs).toString()).toBe(bytes);
  }
  inputs.length = 0;
  await shortcuts.getByRole("button", { name: "Ctrl", exact: true }).click();
  await page.keyboard.type("c");
  await expect.poll(() => Buffer.concat(inputs).toString()).toBe("\x03");
  inputs.length = 0;
  await shortcuts.getByRole("button", { name: "Alt", exact: true }).click();
  await page.keyboard.type("x");
  await expect.poll(() => Buffer.concat(inputs).toString()).toBe("\x1bx");
  await page.keyboard.press("Control+q");
  await shortcuts.getByRole("button", { name: "^C", exact: true }).click();
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBeTruthy();
  await page.screenshot({ path: "test-results/w03-mobile.png" });
});

test("终端粘贴和 Ctrl 点击原样打开链接", async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 800 });
  await openProject(page);
  const [response] = await Promise.all([
    page.waitForResponse(item => item.url().endsWith("/api/v1/terminals") && item.request().method() === "POST"),
    page.getByRole("button", { name: "新建终端" }).click(),
  ]);
  const id = ((await response.json()) as { data: { id: string } }).data.id;
  const runtime = page.locator(`.terminal-runtime[data-terminal-id="${id}"]`);
  await controlled(runtime);
  const input = runtime.getByRole("textbox", { name: "Terminal input" });
  await input.click();
  await input.evaluate(element => {
    const clipboardData = new DataTransfer();
    clipboardData.setData("text/plain", "printf 'PASTE_CONFIRMED\\n'");
    element.dispatchEvent(new ClipboardEvent("paste", { clipboardData, bubbles: true }));
  });
  await page.keyboard.press("Enter");
  await expect(runtime.locator(".xterm-rows")).toContainText("PASTE_CONFIRMED");
  await page.context().route("http://localhost:43111/w03-link", route => route.fulfill({ contentType: "text/plain", body: "W03_LINK" }));
  await page.keyboard.type("printf '\\nhttp://localhost:43111/w03-link\\n'");
  await page.keyboard.press("Enter");
  const link = runtime.locator(".xterm-rows > div").filter({ hasText: /^http:\/\/localhost:43111\/w03-link$/ });
  await expect(link).toBeVisible();
  const bounds = (await link.boundingBox())!;
  const popupPromise = page.context().waitForEvent("page");
  await page.mouse.move(bounds.x + 25, bounds.y + 7);
  await page.keyboard.down("Control");
  await page.mouse.click(bounds.x + 25, bounds.y + 7);
  await page.keyboard.up("Control");
  const popup = await popupPromise;
  await expect.poll(() => popup.url()).toBe("http://localhost:43111/w03-link");
  await popup.close();
});

test("双端接管、取消终止并在闭页后恢复", async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 800 });
  await openProject(page);
  const [createdResponse] = await Promise.all([
    page.waitForResponse(response => response.url().endsWith("/api/v1/terminals") && response.request().method() === "POST"),
    page.getByRole("button", { name: "新建终端" }).click(),
  ]);
  const created = await createdResponse.json() as { data: { id: string } };
  const runtime = page.locator(`.terminal-runtime[data-terminal-id="${created.data.id}"]`);
  await controlled(runtime);
  const url = `/terminals/${created.data.id}`;
  const second = await page.context().newPage();
  await second.goto(url);
  await expect(second.locator(".terminal-runtime")).toHaveAttribute("data-role", "observer");
  await menuAction(second, second.locator(".terminal-runtime"), "接管");
  await controlled(second.locator(".terminal-runtime:visible"));
  await expect(runtime).toHaveAttribute("data-role", "observer");

  await menuAction(second, second.locator(".terminal-runtime"), "关闭当前");
  await second.getByRole("button", { name: "接管并关闭" }).click();
  await expect(page.getByRole("dialog", { name: "终端终止状态" })).toBeVisible();
  const third = await page.context().newPage();
  await third.goto(url);
  await expect(third.getByRole("dialog", { name: "终端终止状态" })).toBeVisible();
  await page.getByRole("button", { name: "取消终止" }).click();
  await expect(page.getByRole("dialog", { name: "终端终止状态" })).toHaveCount(0);
  await expect(second.getByRole("dialog", { name: "终端终止状态" })).toHaveCount(0);
  await expect(third.getByRole("dialog", { name: "终端终止状态" })).toHaveCount(0);
  await third.close();
  await expect(runtime).toHaveAttribute("data-connection", "connected");

  await second.close();
  await page.close();
  const reopened = await page.context().newPage();
  await reopened.goto(url);
  await controlled(reopened.locator(".terminal-runtime"));
});

test("终端在上下宿主移动并保留控制权", async ({ page }) => {
  let connections = 0;
  page.on("websocket", () => connections++);
  await page.setViewportSize({ width: 1280, height: 800 });
  await openProject(page);
  const [createdResponse] = await Promise.all([
    page.waitForResponse(response => response.url().endsWith("/api/v1/terminals") && response.request().method() === "POST"),
    page.getByRole("button", { name: "新建终端" }).click(),
  ]);
  const created = await createdResponse.json() as { data: { id: string } };
  const lower = page.locator(`.terminal-pane .terminal-runtime[data-terminal-id="${created.data.id}"]`);
  const upper = page.locator(`.editor-upper-terminal .terminal-runtime[data-terminal-id="${created.data.id}"]`);
  await controlled(lower);
  const connected = connections;
  await menuAction(page, lower, "移到上方标签");
  await controlled(upper);

  await menuAction(page, upper, "关闭当前");
  await expect(page.getByRole("dialog", { name: "终止终端？" })).toBeVisible();
  await page.getByRole("button", { name: "接管并关闭" }).click();
  await expect(page.getByRole("dialog", { name: "终端终止状态" })).toBeVisible();
  await page.getByRole("button", { name: "取消终止" }).click();
  await expect(page.getByRole("dialog", { name: "终端终止状态" })).toHaveCount(0);
  await menuAction(page, upper, "移回下方终端面板");
  await controlled(lower);
  await page.getByRole("button", { name: "刷新终端", exact: true }).click();
  await controlled(lower);
  await page.getByRole("button", { name: "收起终端面板" }).click();
  await page.getByRole("button", { name: "终端面板", exact: true }).click();
  await controlled(lower);
  await page.locator(`.terminal-tab[data-terminal-id="${created.data.id}"]`).dragTo(page.getByRole("region", { name: "左侧编辑器" }));
  await controlled(upper);
  await page.locator(`.terminal-tab[data-terminal-id="${created.data.id}"]`).dragTo(page.getByRole("region", { name: "终端面板" }));
  await controlled(lower);
  const response = await page.request.get("/api/v1/terminals");
  const list = await response.json() as { data: { items: Array<{ id: string; state: string }> } };
  expect(list.data.items.filter(item => item.id === created.data.id)).toHaveLength(1);
  expect(list.data.items).toContainEqual(expect.objectContaining({ id: created.data.id, state: "running" }));
  expect(connections).toBe(connected);
});

test("倒计时到期后显示已结束且其他会话继续", async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 800 });
  await openProject(page);
  const [response] = await Promise.all([
    page.waitForResponse(item => item.url().endsWith("/api/v1/terminals") && item.request().method() === "POST"),
    page.getByRole("button", { name: "新建终端" }).click(),
  ]);
  const id = ((await response.json()) as { data: { id: string } }).data.id;
  const runtime = page.locator(`.terminal-runtime[data-terminal-id="${id}"]`);
  await controlled(runtime);
  const before = ((await (await page.request.get("/api/v1/terminals")).json()) as { data: { items: Array<{ id: string; state: string }> } }).data.items;
  await menuAction(page, runtime, "关闭当前");
  await page.getByRole("button", { name: "接管并关闭" }).click();
  await expect(page.getByRole("dialog", { name: "终端终止状态" })).toBeVisible();
  await expect(page.getByText("会话已结束；不会自动重新执行命令。", { exact: true })).toBeVisible({ timeout: 15000 });
  await page.getByRole("button", { name: "关闭结果" }).click();
  await expect(page.getByRole("dialog", { name: "终端终止状态" })).toHaveCount(0);
  const after = ((await (await page.request.get("/api/v1/terminals")).json()) as { data: { items: Array<{ id: string; state: string }> } }).data.items;
  expect(after.find(item => item.id === id)?.state).toBe("terminated");
  for (const item of before.filter(item => item.id !== id && item.state === "running")) expect(after.find(other => other.id === item.id)?.state).toBe("running");
});

test("手机外链确认和深色终端", async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await openProject(page);
  await page.getByRole("navigation", { name: "工作区视图" }).getByRole("button", { name: "终端" }).click();
  const [response] = await Promise.all([
    page.waitForResponse(item => item.url().endsWith("/api/v1/terminals") && item.request().method() === "POST"),
    page.getByRole("button", { name: "新建终端" }).click(),
  ]);
  const id = ((await response.json()) as { data: { id: string } }).data.id;
  const runtime = page.locator(`.terminal-runtime[data-terminal-id="${id}"]`);
  await controlled(runtime);
  await page.getByRole("combobox", { name: "主题" }).click();
  await page.getByRole("option", { name: "深色", exact: true }).click();
  await expect(page.locator("html")).toHaveClass(/dark/);
  const url = "http://localhost:43111/w03-mobile";
  await page.context().route(url, route => route.fulfill({ contentType: "text/plain", body: "MOBILE_LINK" }));
  await runtime.getByRole("textbox", { name: "Terminal input" }).click();
  await page.keyboard.type(`printf '\\n${url}\\n'`);
  await page.keyboard.press("Enter");
  const line = runtime.locator(".xterm-rows > div").filter({ hasText: /^http:\/\/localhost:43111\/w03-mobile$/ });
  await expect(line).toBeVisible();
  const bounds = (await line.boundingBox())!;
  await page.mouse.click(bounds.x + 25, bounds.y + 7);
  await expect(page.getByRole("dialog", { name: "打开外部链接？" })).toBeVisible();
  await page.getByRole("button", { name: "取消", exact: true }).click();
  await expect(page.getByRole("dialog", { name: "打开外部链接？" })).toHaveCount(0);
  await page.mouse.move(bounds.x + 25, bounds.y + 30);
  await page.mouse.move(bounds.x + 25, bounds.y + 7);
  await page.mouse.click(bounds.x + 25, bounds.y + 7);
  await expect(page.getByRole("dialog", { name: "打开外部链接？" })).toBeVisible();
  const popupPromise = page.context().waitForEvent("page");
  await page.getByRole("button", { name: "打开", exact: true }).click();
  const popup = await popupPromise;
  await expect.poll(() => popup.url()).toBe(url);
  await popup.close();
  await page.screenshot({ path: "test-results/w03-mobile-dark.png" });
});
