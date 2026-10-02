import { expect, test, type Page } from "@playwright/test";
import { createHash } from "node:crypto";
const id = "a".repeat(64), url = "/tests/fixtures/editor-recovery.html";
const hash = (text: string) => `sha256:${createHash("sha256").update(text).digest("hex")}`;
async function fixture(page: Page, mode: "success" | "unknown" | "conflict" | "unavailable" | "expired" = "success") {
  let disk = "old", frozen = "", generation = 1, executes = 0, statuses = 0, cancels = 0;
  const errors: string[] = []; let release!: () => void; const gate = new Promise<void>(resolve => { release = resolve; });
  page.on("pageerror", error => errors.push(error.message)); page.on("dialog", dialog => void dialog.accept());
  await page.routeWebSocket(/\/api\/v1\/events/, () => {});
  const version = () => ({ identity: `1:${generation}`, mtime: "2026-10-02T00:00:00Z", size: Buffer.byteLength(disk), etag: hash(disk) });
  await page.route("**/api/v1/**", async route => {
    const request = route.request(), path = new URL(request.url()).pathname;
    const failure = async (status: number, code: string) => route.fulfill({ status, json: { error: { code, message: "测试错误，输入已保留" }, request_id: "fixture" } });
    let data: unknown;
    if (path === "/api/v1/terminals") data = { items: [] };
    else if (path.endsWith("/entries")) data = { project_version: 1, next_cursor: "", items: path.includes("/other/") ? [] : [{ name: "one.txt", kind: "file", identity: "1:1", size: 3, mtime: version().mtime }] };
    else if (path.endsWith("/content") && request.method() === "PUT") { await failure(403, "permission_denied"); return; }
    else if (path.endsWith("/content")) data = { content: disk, kind: "text", version: version() };
    else if (path.endsWith("/git-baseline")) data = { state: "no_repository", repo_id: "", head: "", content: "", version: null };
    else if (path.endsWith("/elevation-requests")) {
      expect(request.headers()["x-csrf-token"]).toBe("fixture"); frozen = request.postDataJSON().content;
      if (mode === "unavailable") { await failure(503, "elevation_unavailable"); return; }
      data = { id, state: "prepared", target_id: "example", target_path: "/fixture/one.txt", content_hash: hash(frozen), expires_at: new Date(Date.now() + (mode === "expired" ? 300 : 60000)).toISOString() };
    } else if (path.endsWith("/execute")) {
      executes++; const input = request.postDataJSON(); expect(input.content).toBe(frozen); expect(input.password).toBe("synthetic-system-password"); expect(request.headers()["x-csrf-token"]).toBe("fixture"); await gate;
      if (mode === "unknown") { disk = frozen; generation++; await route.abort("failed"); return; }
      if (mode === "conflict") data = { id, state: "rejected", code: "conflict", version: null };
      else { disk = frozen; generation++; data = { id, state: "applied", code: null, version: version() }; }
    } else if (path.endsWith(id) && request.method() === "DELETE") { cancels++; data = { id, state: "cancelled", code: "cancelled", version: null }; }
    else if (path.endsWith(id)) { statuses++; data = { id, state: "applied", code: null, version: version() }; }
    else throw new Error(`未预期请求 ${request.method()} ${path}`);
    await route.fulfill({ json: { data, request_id: "fixture" } });
  });
  return { get executes() { return executes; }, get statuses() { return statuses; }, get cancels() { return cancels; }, get disk() { return disk; }, release, errors };
}
async function edit(page: Page, content: string, mobile = false) {
  if (mobile) { await page.getByRole("textbox", { name: "one.txt 内容" }).fill(content); return; }
  await expect(page.locator(".monaco-editor").first()).toBeVisible();
  await page.evaluate(content => { const model = window.editorRecovery.monaco.editor.getModels().find(model => model.uri.path.endsWith("one.txt")); if (!model) throw new Error("model missing"); model.pushStackElement(); model.pushEditOperations([], [{ range: model.getFullModelRange(), text: content }], () => []); model.pushStackElement(); }, content);
}
async function openAttempt(page: Page, mobile = false) {
  await page.goto(url, { waitUntil: "domcontentloaded" }); if (mobile) await page.getByRole("button", { name: "文件", exact: true }).click();
  await page.getByRole("treeitem", { name: "one.txt", exact: true }).click();
  if (mobile) await edit(page, "submitted", true);
  else { await page.locator(".monaco-editor .view-lines").first().click({ position: { x: 20, y: 10 } }); await expect(page.getByRole("textbox", { name: "Editor content", exact: true })).toBeFocused(); await page.keyboard.press("ControlOrMeta+A"); await page.keyboard.insertText("submitted"); }
  await page.getByRole("button", { name: "提权保存", exact: true }).click(); await expect(page.getByRole("dialog", { name: "单文件提权保存" })).toBeVisible();
}
test("默认取消焦点、快照可见，Escape 零执行", async ({ page }) => {
  const h = await fixture(page); await openAttempt(page); await expect(page.getByRole("button", { name: "取消", exact: true })).toBeFocused();
  await page.getByRole("button", { name: "查看本次保存内容" }).click(); await expect(page.getByLabel("本次保存内容")).toHaveText("submitted");
  await page.getByLabel("系统用户密码").fill("synthetic-system-password"); await page.keyboard.press("Escape"); await expect(page.getByRole("dialog")).toHaveCount(0); await expect.poll(() => h.cancels).toBe(1); expect(h.executes).toBe(0); expect(h.disk).toBe("old"); expect(h.errors).toEqual([]);
});
for (const mobile of [false, true]) test(`${mobile ? "手机" : "桌面"}提交只写冻结内容，密码字段消失并保留新输入/undo`, async ({ page }, info) => {
  if (mobile) await page.setViewportSize({ width: 390, height: 844 }); const h = await fixture(page); await openAttempt(page, mobile); const modelID = mobile ? null : await page.evaluate(() => window.editorRecovery.monaco.editor.getModels()[0]?.id);
  await page.getByLabel("系统用户密码").fill("synthetic-system-password"); await page.getByRole("button", { name: "确认一次保存" }).click(); await expect.poll(() => h.executes).toBe(1); await expect(page.getByLabel("系统用户密码")).toHaveCount(0);
  if (mobile) await page.evaluate(() => { const input = document.querySelector<HTMLTextAreaElement>('textarea[aria-label="one.txt 内容"]'); if (!input) throw new Error("input missing"); const setter = Object.getOwnPropertyDescriptor(HTMLTextAreaElement.prototype, "value")!.set!; setter.call(input, "new input"); input.dispatchEvent(new Event("input", { bubbles: true })); });
  else await edit(page, "new input");
  h.release(); await expect(page.getByText(/本次内容已保存/)).toBeVisible(); expect(h.disk).toBe("submitted"); await page.screenshot({ path: info.outputPath(`w07-${mobile ? "mobile" : "desktop"}.png`) }); await page.getByRole("button", { name: "关闭", exact: true }).click(); await expect(page.getByText("已暂停自动保存", { exact: true })).toBeVisible();
  if (mobile) await expect(page.getByRole("textbox", { name: "one.txt 内容" })).toHaveValue("new input"); else { expect(await page.evaluate(() => window.editorRecovery.monaco.editor.getModels()[0]?.id)).toBe(modelID); expect(await page.evaluate(() => window.editorRecovery.monaco.editor.getModels()[0]?.getValue())).toBe("new input"); await page.evaluate(() => window.editorRecovery.monaco.editor.getModels()[0]?.undo()); expect(await page.evaluate(() => window.editorRecovery.monaco.editor.getModels()[0]?.getValue())).toBe("submitted"); }
  expect(h.executes).toBe(1); expect(h.errors).toEqual([]);
});
test("未知结果只查询，不重发保存", async ({ page }) => {
  const h = await fixture(page, "unknown"); await openAttempt(page); await page.getByLabel("系统用户密码").fill("synthetic-system-password"); await page.getByRole("button", { name: "确认一次保存" }).click(); h.release(); await expect(page.getByRole("button", { name: "查询本次结果" })).toBeVisible(); await page.getByRole("button", { name: "查询本次结果" }).click(); await expect(page.getByText(/本次内容已保存/)).toBeVisible(); expect(h.statuses).toBe(1); expect(h.executes).toBe(1); expect(h.errors).toEqual([]);
});
test("服务不可用和过期不发送口令", async ({ page }) => {
  const h = await fixture(page, "unavailable"); await openAttempt(page); await expect(page.getByRole("dialog").getByRole("alert")).toBeVisible(); await expect(page.getByLabel("系统用户密码")).toHaveCount(0); expect(h.executes).toBe(0); expect(h.errors).toEqual([]);
});
test("授权过期只能重新申请，不接受确认", async ({ page }) => {
  const h = await fixture(page, "expired"); await openAttempt(page); await expect(page.getByText("本次授权已过期，请重新申请。", { exact: true })).toBeVisible(); await expect(page.getByRole("button", { name: "确认一次保存" })).toHaveCount(0); expect(h.executes).toBe(0); expect(h.errors).toEqual([]);
});

test("提权冲突保留输入并要求比较", async ({ page }) => {
 const h = await fixture(page, "conflict"); await openAttempt(page); await page.getByLabel("系统用户密码").fill("synthetic-system-password"); await page.getByRole("button", { name: "确认一次保存" }).click(); h.release(); await expect(page.getByText("文件已变化，请返回编辑器比较后重新申请。", { exact: true })).toBeVisible(); await page.getByRole("button", { name: "关闭", exact: true }).click(); expect(h.disk).toBe("old"); expect(await page.evaluate(() => window.editorRecovery.monaco.editor.getModels()[0]?.getValue())).toBe("submitted"); expect(h.executes).toBe(1); expect(h.errors).toEqual([]);
});
