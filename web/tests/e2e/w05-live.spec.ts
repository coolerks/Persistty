import { expect, test } from "@playwright/test";
import { createHash } from "node:crypto";
// Explicit isolated runtime probe only; no arbitrary user's project is writable.
test("W05 Debian 产品 HTTP 原子落盘、冲突、预览与布局恢复", async ({ page, browser }, testInfo) => {
  test.skip(process.env.PERSISTTY_E2E_W05 !== "1", "需显式指定本轮 W05 Debian 隔离实例");
  expect(process.env.PERSISTTY_E2E_PROJECT).toBe("W03 隔离终端"); expect(process.env.PERSISTTY_E2E_REMOTE_ROOT).toMatch(/^\/tmp\/persistty-browser-[A-Za-z0-9]{8}$/);
  const suffix = Date.now(); const note = `w05-note-${suffix}.txt`, safe = `w05-safe-${suffix}.svg`, unsafeName = `w05-unsafe-${suffix}.svg`, imageName = `w05-image-${suffix}.txt`;
  const errors: string[] = []; page.on("pageerror", error => errors.push(error.message)); page.on("dialog", dialog => void dialog.accept());
  await page.setViewportSize({ width: 1800, height: 1000 }); await page.goto("/projects"); await page.getByLabel("访问密码").fill(process.env.PERSISTTY_E2E_PASSWORD!); await page.getByRole("button", { name: "登录", exact: true }).click(); await expect(page.getByRole("button", { name: "新建项目", exact: true })).toBeVisible();
  const session = (await (await page.request.get("/api/v1/auth/session")).json()).data;
  const projects = (await (await page.request.get("/api/v1/projects")).json()).data.items; const project = projects.find((item: { name: string }) => item.name === process.env.PERSISTTY_E2E_PROJECT); expect(project).toBeTruthy();
  expect(project.folders[0].path).toBe(`${process.env.PERSISTTY_E2E_REMOTE_ROOT}/project`);
  const headers = { Origin: process.env.PERSISTTY_E2E_BASE_URL!, "X-CSRF-Token": session.csrf_token }; const base = `/api/v1/projects/${project.id}/folders/${project.main_folder_id}`;
  const contentURL = (path: string) => `${base}/content?project_version=1&path=${encodeURIComponent(path)}`;
  const create = async (path: string, content: string) => {
    const result = await page.request.post(`/api/v1/projects/${project.id}/file-operations`, { headers, data: { kind: "create_file", project_version: 1, target_folder_id: project.main_folder_id, target_path: path } }); expect(result.ok()).toBe(true);
    const empty = (await (await page.request.get(contentURL(path))).json()).data;
    const saved = await page.request.put(`${base}/content`, { headers, data: { project_version: 1, path, expected_version: empty.version, content } }); expect(saved.ok()).toBe(true);
  };
  await create(note, "\uFEFFone\r\ntwo\nthree"); await create(safe, '<svg xmlns="http://www.w3.org/2000/svg" width="40" height="30"><rect width="40" height="30" fill="red"/></svg>'); await create(unsafeName, '<svg onload="window.pwned=true"><script>alert(1)</script></svg>');
  const png = Buffer.from("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+a/QsAAAAASUVORK5CYII=", "base64"); const hash = createHash("sha256").update(png).digest("hex");
  const uploaded = await page.request.post("/api/v1/uploads", { headers, data: { project_id: project.id, folder_id: project.main_folder_id, project_version: 1, path: imageName, batch_id: "w05-preview", size: png.length, sha256: hash } }); expect(uploaded.ok()).toBe(true); const upload = (await uploaded.json()).data;
  expect((await page.request.put(`/api/v1/uploads/${upload.id}/chunks/0`, { headers: { ...headers, "Content-Type": "application/octet-stream", "X-Chunk-SHA256": hash }, data: png })).ok()).toBe(true);
  expect((await page.request.post(`/api/v1/uploads/${upload.id}/complete`, { headers })).ok()).toBe(true);
  await page.goto(`/projects/${project.id}`); await page.getByRole("treeitem", { name: note, exact: true }).click(); await expect(page.locator(".monaco-editor")).toBeVisible();
  const text = page.locator(".monaco-editor .view-lines"); await text.click({ position: { x: 20, y: 8 } }); for (let i = 0; i < 20; i++) await page.keyboard.press("ArrowLeft"); await page.keyboard.type("start ");
  await expect(page.getByText("已保存", { exact: true })).toBeVisible(); await expect.poll(async () => (await (await page.request.get(contentURL(note))).json()).data.content).toBe("\uFEFFstart one\r\ntwo\nthree");
  await page.getByRole("button", { name: "向右拆分编辑器", exact: true }).last().click(); await page.getByRole("button", { name: "向右拆分编辑器", exact: true }).last().click(); await expect(page.getByRole("region", { name: "第 3 组编辑器", exact: true })).toBeVisible(); await page.reload(); await expect(page.getByRole("tab", { name: note, exact: true })).toHaveCount(3);
  await page.getByRole("button", { name: "合并编辑器", exact: true }).first().click();
  const old = (await (await page.request.get(contentURL(note))).json()).data;
  await page.locator(".monaco-editor .view-lines").click({ position: { x: 20, y: 8 } }); for (let i = 0; i < 20; i++) await page.keyboard.press("ArrowLeft"); await page.keyboard.type("mine "); expect((await page.request.put(`${base}/content`, { headers, data: { project_version: 1, path: note, expected_version: old.version, content: "external" } })).ok()).toBe(true); await expect(page.getByText("冲突", { exact: true })).toBeVisible();
  await page.getByRole("button", { name: "查看并处理", exact: true }).click(); await expect(page.getByRole("dialog").locator(".monaco-diff-editor")).toBeVisible(); await page.getByRole("button", { name: "确认保存本地内容", exact: true }).click(); await expect(page.getByText("已保存", { exact: true })).toBeVisible();
  expect((await (await page.request.get(contentURL(note))).json()).data.content).toBe("\uFEFFmine start one\r\ntwo\nthree");
  for (const name of [imageName, safe]) { await page.getByRole("treeitem", { name, exact: true }).click(); const image = page.locator(".file-preview img"); await expect(image).toBeVisible(); await expect.poll(() => image.evaluate(node => node instanceof HTMLImageElement && node.complete && node.naturalWidth > 0)).toBe(true); }
  const unauthorized = await browser.newContext(); const anon = await unauthorized.request.get(`${process.env.PERSISTTY_E2E_BASE_URL}${base}/preview?project_version=1&path=${encodeURIComponent(safe)}`); expect(anon.status()).toBe(401); await unauthorized.close();
  const unsafe = await page.request.get(`${base}/preview?project_version=1&path=${encodeURIComponent(unsafeName)}`); expect(unsafe.status()).toBe(415);
  await page.getByRole("treeitem", { name: unsafeName, exact: true }).click(); await expect(page.getByText(/不支持安全预览/)).toBeVisible(); expect(await page.evaluate(() => Object.hasOwn(window, "pwned"))).toBe(false);
  await page.screenshot({ path: testInfo.outputPath("w05-live-preview.png") }); expect(errors).toEqual([]);
});
