import { expect, test, type Page } from "@playwright/test";
import type { FileVersion } from "../../src/lib/api/decoder";
const url = "/tests/fixtures/editor-recovery.html";
async function fixture(page: Page) {
  const files = new Map<string, { content: string; revision: number }>([["one.txt", { content: "\uFEFFone\r\ntwo\nthree", revision: 1 }], ["two.txt", { content: "second", revision: 1 }], ["three.txt", { content: "third", revision: 1 }]]);
  const puts: { content: string; expected_version: FileVersion }[] = []; const mutations: string[] = []; const errors: string[] = []; let reject = false;
  const version = (path: string) => { const row = files.get(path)!; return { mtime: "2026-09-30T00:00:00Z", size: Buffer.byteLength(row.content), identity: `${path}:${row.revision}`, etag: `${path}:${row.revision}` }; };
  page.on("pageerror", error => errors.push(error.message)); page.on("dialog", dialog => void dialog.accept());
  await page.routeWebSocket(/\/api\/v1\/events/, () => {});
  await page.route("**/api/v1/**", async route => {
    const request = route.request(), path = new URL(request.url()).pathname, query = new URL(request.url()).searchParams;
    if (request.method() !== "GET") mutations.push(path);
    let data: unknown;
    if (path === "/api/v1/terminals") data = { items: [{ id: "ended", display_name: "结束会话", project_id: "interaction", state: "terminated", working_directory: "/fixture" }] };
    else if (path.endsWith("/entries")) data = { project_version: 1, next_cursor: "", items: path.includes("/other/") ? [] : [...files].map(([name, row]) => ({ name, kind: "file", identity: name, size: Buffer.byteLength(row.content), mtime: "2026-09-30T00:00:00Z" })) };
    else if (path.endsWith("/content") && request.method() === "PUT") {
      const input = request.postDataJSON(); puts.push(input); const row = files.get(input.path)!;
      if (reject || input.expected_version.etag !== version(input.path).etag) { await route.fulfill({ status: 409, json: { error: { code: "file_conflict", message: "服务器内容已变化" }, request_id: "fixture" } }); return; }
      expect(request.headers()["x-csrf-token"]).toBe("fixture"); row.content = input.content; row.revision++; data = { version: version(input.path) };
    } else if (path.endsWith("/content")) { const path = query.get("path")!; data = { content: files.get(path)!.content, version: version(path), kind: "text" }; }
    else throw new Error(`未预期请求 ${request.method()} ${path}`);
    await route.fulfill({ json: { data, request_id: "fixture" } });
  });
  return { files, puts, mutations, errors, reject: (value: boolean) => { reject = value; } };
}
async function edit(page: Page, text: string, path = "one.txt") {
  await expect(page.locator(".monaco-editor").first()).toBeVisible();
  await page.evaluate(({ text, path }) => { const model = window.editorRecovery.monaco.editor.getModels().find(model => model.uri.path.endsWith(path)); if (!model) throw new Error("model missing"); model.pushEditOperations([], [{ range: model.getFullModelRange(), text }], () => []); }, { text, path });
}
test("自动保存保留 BOM/换行，四组与设备记录恢复、移动保持 model/undo", async ({ page }) => {
  await page.setViewportSize({ width: 1800, height: 1000 }); const h = await fixture(page); await page.goto(url);
  await page.getByRole("treeitem", { name: "one.txt", exact: true }).click(); await edit(page, "one\nTWO\nthree");
  await expect.poll(() => h.puts.length).toBe(1); expect(h.puts[0]?.content).toBe("\uFEFFone\r\nTWO\nthree"); await expect(page.getByText("已保存", { exact: true })).toBeVisible();
  const id = await page.evaluate(() => window.editorRecovery.monaco.editor.getModels()[0]?.id);
  await page.getByRole("button", { name: "向右拆分编辑器", exact: true }).last().click();
  await page.getByRole("button", { name: "向右拆分编辑器", exact: true }).last().click();
  await expect(page.getByRole("region", { name: "第 3 组编辑器", exact: true })).toBeVisible();
  expect(await page.evaluate(() => window.editorRecovery.monaco.editor.getModels()[0]?.id)).toBe(id);
  await page.evaluate(() => window.editorRecovery.monaco.editor.getModels()[0]?.undo()); await expect.poll(() => h.files.get("one.txt")?.content).toBe("\uFEFFone\r\ntwo\nthree");
  await page.setViewportSize({ width: 390, height: 844 }); await page.getByRole("button", { name: "文件", exact: true }).click(); await page.getByRole("treeitem", { name: "two.txt", exact: true }).click();
  await expect(page.getByRole("textbox", { name: "two.txt 内容" })).toBeVisible(); await page.getByRole("textbox", { name: "two.txt 内容" }).fill("phone edit"); await expect.poll(() => h.files.get("two.txt")?.content).toBe("phone edit");
  await page.setViewportSize({ width: 1800, height: 1000 }); await expect(page.getByRole("tab", { name: "one.txt", exact: true })).toHaveCount(3); await expect(page.getByRole("tab", { name: "two.txt", exact: true })).toHaveCount(0);
  await page.reload(); await expect(page.getByRole("tab", { name: "one.txt", exact: true })).toHaveCount(3); await expect.poll(() => page.evaluate(() => document.fonts.check('13px "Persistty Nerd Mono"'))).toBe(true);
  expect(h.errors).toEqual([]); expect(h.mutations.every(path => path.endsWith("/content"))).toBe(true);
});
test("409 与真实 IndexedDB 草稿刷新后零自动 PUT，desktop diff 后明确保存", async ({ page }) => {
  const h = await fixture(page); await page.goto(url); await page.getByRole("treeitem", { name: "one.txt", exact: true }).click(); h.reject(true); await edit(page, "mine");
  await expect(page.getByText("冲突", { exact: true })).toBeVisible(); expect(h.puts).toHaveLength(1);
  h.files.set("one.txt", { content: "external", revision: 5 }); await page.reload(); await expect(page.getByText(/此文件有 1 份本地草稿/)).toBeVisible(); expect(h.puts).toHaveLength(1);
  await page.getByRole("button", { name: "查看草稿 1", exact: true }).click(); await expect(page.getByRole("dialog")).toBeVisible(); await expect(page.getByRole("dialog").locator(".monaco-diff-editor")).toBeVisible();
  await page.getByRole("button", { name: "恢复到编辑器", exact: true }).click(); await expect(page.getByText("已暂停自动保存", { exact: true })).toBeVisible(); await page.waitForTimeout(1200); expect(h.puts).toHaveLength(1);
  h.reject(false); await page.getByRole("button", { name: "保存恢复内容", exact: true }).click(); await expect(page.getByText("已保存", { exact: true })).toBeVisible(); expect(h.puts).toHaveLength(2); expect(h.puts[1]?.expected_version.etag).toBe("one.txt:5"); expect(h.files.get("one.txt")?.content).toBe("\uFEFFmine"); expect(h.errors).toEqual([]);
});
test("终端多组与上下位置恢复不创建、终止或写终端输入", async ({ page }) => {
  await page.setViewportSize({ width: 1800, height: 1000 }); const h = await fixture(page); await page.goto(url); await page.locator('[data-terminal-id="ended"]').click({ button: "right" }); await page.getByRole("menuitem", { name: "移到上方第 3 组", exact: true }).click();
  await expect(page.getByRole("region", { name: "第 3 组编辑器", exact: true }).getByRole("tab", { name: /结束会话/ })).toBeVisible();
  await page.reload(); await expect(page.getByRole("region", { name: "第 3 组编辑器", exact: true }).getByRole("tab", { name: /结束会话/ })).toBeVisible();
  await page.locator('[data-terminal-id="ended"]').click({ button: "right" }); await page.getByRole("menuitem", { name: "移到下方第 2 组", exact: true }).click(); await expect(page.locator(".terminal-pane")).toHaveCount(2);
  await page.reload(); await expect(page.locator(".terminal-pane")).toHaveCount(2); await expect(page.getByRole("tab", { name: /结束会话/ })).toHaveCount(1); expect(h.mutations).toEqual([]); expect(h.errors).toEqual([]);
});

test("本地草稿不可写时最后标签关闭保留输入且不触发保存", async ({ page }) => {
  const h = await fixture(page);
  await page.addInitScript(() => { Object.defineProperty(IDBFactory.prototype, "open", { configurable: true, value() { throw new DOMException("quota", "QuotaExceededError"); } }); });
  await page.goto(url); await page.getByRole("treeitem", { name: "one.txt", exact: true }).click(); await edit(page, "important");
  await page.getByRole("button", { name: "关闭 one.txt", exact: true }).click();
  await expect(page.getByText("草稿保存失败，文件仍保持打开，请导出内容。", { exact: true })).toBeVisible();
  await expect(page.getByRole("tab", { name: "one.txt", exact: true })).toBeVisible();
  expect(await page.evaluate(() => window.editorRecovery.monaco.editor.getModels()[0]?.getValue())).toBe("important");
  await page.waitForTimeout(1200); expect(h.puts).toEqual([]); expect(h.errors).toEqual([]);
});

test("手机基线变化的草稿只能保留或导出，不恢复或自动写入", async ({ page }) => {
  const h = await fixture(page); await page.goto(url); await page.getByRole("treeitem", { name: "one.txt", exact: true }).click(); h.reject(true); await edit(page, "mine");
  await expect(page.getByText("冲突", { exact: true })).toBeVisible();
  h.files.set("one.txt", { content: "external", revision: 5 });
  await page.setViewportSize({ width: 390, height: 844 }); await page.reload();
  await page.getByRole("button", { name: "文件", exact: true }).click(); await page.getByRole("treeitem", { name: "one.txt", exact: true }).click();
  await page.getByRole("button", { name: "查看草稿 1", exact: true }).click();
  await expect(page.getByText("请导出草稿，在电脑上比较和处理。", { exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "恢复到编辑器", exact: true })).toHaveCount(0);
  await expect(page.locator(".monaco-diff-editor")).toHaveCount(0); await page.waitForTimeout(1200);
  expect(h.puts).toHaveLength(1); expect(h.errors).toEqual([]);
});
