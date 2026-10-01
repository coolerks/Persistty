import { expect, test } from "@playwright/test";
import { readFile } from "node:fs/promises";
import { createHash } from "node:crypto";
import { decodeEnvelope, decodeList, decodeProject, decodeSession } from "../../src/lib/api/decoder";

test("W05 六种真实图片解码、隔离响应、三主题字体与横竖视图", async ({ page, browser }, testInfo) => {
  test.skip(process.env.PERSISTTY_E2E_W05_MATRIX !== "1", "仅在显式隔离验收实例运行");
  test.setTimeout(60000);
  const errors: string[] = []; page.on("pageerror", error => errors.push(error.message));
  await page.setViewportSize({ width: 1440, height: 900 });
  await page.goto("/projects"); await page.getByLabel("访问密码").fill(process.env.PERSISTTY_E2E_PASSWORD!); await page.getByRole("button", { name: "登录", exact: true }).click();
  await expect(page.getByRole("button", { name: "新建项目", exact: true })).toBeVisible();
  const session = decodeEnvelope(await (await page.request.get("/api/v1/auth/session")).json(), decodeSession);
  const projects = decodeEnvelope(await (await page.request.get("/api/v1/projects")).json(), decodeList(decodeProject));
  const project = projects.find(item => item.name === process.env.PERSISTTY_E2E_PROJECT);
  if (!project || !(project.folders[0]?.path.startsWith("/private/tmp/persistty-w06-browser-") || (process.env.PERSISTTY_E2E_REMOTE_ROOT?.match(/^\/tmp\/persistty-browser-[A-Za-z0-9]{8}$/) && project.folders[0]?.path === `${process.env.PERSISTTY_E2E_REMOTE_ROOT}/project`))) throw new Error("需专属隔离项目");
  const base = `/api/v1/projects/${project.id}/folders/${project.main_folder_id}`, headers = { Origin: process.env.PERSISTTY_E2E_BASE_URL!, "X-CSRF-Token": session.csrf_token };
  const names: string[] = [], suffix = `${testInfo.project.name}-${Date.now()}`;
  for (const format of ["png", "jpeg", "gif", "webp", "avif", "svg"]) {
    const content = await readFile(new URL(`../../../tests/fixtures/images/w05-valid.${format}`, import.meta.url));
    const hash = createHash("sha256").update(content).digest("hex"), path = `matrix-${suffix}-${format}.txt`;
    const init = await page.request.post("/api/v1/uploads", { headers, data: { project_id: project.id, folder_id: project.main_folder_id, project_version: project.version, path, batch_id: `matrix-${suffix}`, size: content.length, sha256: hash } });
    expect(init.ok()).toBe(true); const upload = (await init.json()).data;
    expect((await page.request.put(`/api/v1/uploads/${upload.id}/chunks/0`, { headers: { ...headers, "Content-Type": "application/octet-stream", "X-Chunk-SHA256": hash }, data: content })).ok()).toBe(true);
    expect((await page.request.post(`/api/v1/uploads/${upload.id}/complete`, { headers })).ok()).toBe(true);
    const inspection = (await (await page.request.get(`${base}/inspect?project_version=${project.version}&path=${encodeURIComponent(path)}`)).json()).data;
    expect(inspection).toMatchObject({ kind: "image", previewable: true, width: 3, height: 2 });
    const response = await page.request.get(`${base}/preview?project_version=${project.version}&path=${encodeURIComponent(path)}`);
    expect(response.ok()).toBe(true); expect(response.headers()["x-content-type-options"]).toBe("nosniff"); expect(response.headers()["cache-control"]).toBe("no-store"); expect(response.headers()["content-security-policy"]).toContain("sandbox");
    expect(createHash("sha256").update(await response.body()).digest("hex")).toBe(hash); names.push(path);
  }
  const anonymous = await browser.newContext();
  expect((await anonymous.request.get(`${process.env.PERSISTTY_E2E_BASE_URL}${base}/preview?project_version=${project.version}&path=${encodeURIComponent(names[0]!)}`)).status()).toBe(401); await anonymous.close();
  await page.goto(`/projects/${project.id}`);
  for (const theme of ["浅色", "深色", "跟随系统"]) {
    await page.getByRole("combobox", { name: "主题" }).click(); await page.getByRole("option", { name: theme, exact: true }).click();
    for (const name of names) {
      await page.getByRole("treeitem", { name, exact: true }).click(); const image = page.locator(".file-preview img");
      await expect(image).toBeVisible(); await expect.poll(() => image.evaluate(node => node instanceof HTMLImageElement && node.complete && node.naturalWidth === 3 && node.naturalHeight === 2)).toBe(true);
    }
    const font = await page.evaluate(async () => {
      await document.fonts.load('13px "Persistty Nerd Mono"');
      const canvas = document.createElement("canvas"); canvas.width = 64; canvas.height = 32; const context = canvas.getContext("2d");
      if (!context) throw new Error("Canvas不可用"); context.font = '20px "Persistty Nerd Mono"';
      const widths = ["i", "W", "\ue0b0", "\ue0b1", "\uf017"].map(value => context.measureText(value).width);
      const pixels = ["\ufffd", "\ue0b0", "\ue0b1", "\uf017"].map(value => { context.clearRect(0, 0, 64, 32); context.fillText(value, 0, 24); return Array.from(context.getImageData(0, 0, 64, 32).data).join(","); });
      return { loaded: document.fonts.check('13px "Persistty Nerd Mono"'), widths, uniqueGlyphs: new Set(pixels).size };
    });
    expect(font.loaded).toBe(true); expect(font.uniqueGlyphs).toBe(4); expect(new Set(font.widths).size).toBe(1);
    await page.screenshot({ path: testInfo.outputPath(`w05-${theme}.png`) });
  }
  for (const viewport of [{ width: 390, height: 844 }, { width: 844, height: 390 }]) {
    await page.setViewportSize(viewport);
    if (viewport.width <= 760) await page.getByRole("button", { name: "文件", exact: true }).click();
    else await expect(page.getByRole("navigation", { name: "活动栏" })).toBeVisible();
    await page.getByRole("treeitem", { name: names[0]!, exact: true }).click();
    await expect(page.locator(".file-preview img")).toBeVisible();
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
  }
  expect(errors).toEqual([]);
});

test("W05 触屏横屏保持基础编辑器、单内容导航和自动保存", async ({ browser }, testInfo) => {
  test.skip(process.env.PERSISTTY_E2E_W05_MATRIX !== "1", "仅在显式隔离验收实例运行");
  const context = await browser.newContext({ hasTouch: true, viewport: { width: 390, height: 844 } });
  const page = await context.newPage();
  try {
    await page.goto(`${process.env.PERSISTTY_E2E_BASE_URL}/projects`); await page.getByLabel("访问密码").fill(process.env.PERSISTTY_E2E_PASSWORD!); await page.getByRole("button", { name: "登录", exact: true }).click();
    await expect(page.getByRole("button", { name: "新建项目", exact: true })).toBeVisible();
    const session = decodeEnvelope(await (await page.request.get("/api/v1/auth/session")).json(), decodeSession);
    const project = decodeEnvelope(await (await page.request.get("/api/v1/projects")).json(), decodeList(decodeProject)).find(item => item.name === process.env.PERSISTTY_E2E_PROJECT);
    if (!project || !project.folders[0]?.path.startsWith("/private/tmp/persistty-w06-browser-")) throw new Error("需本轮本地隔离项目");
    const path = `touch-${Date.now()}.txt`, headers = { Origin: process.env.PERSISTTY_E2E_BASE_URL!, "X-CSRF-Token": session.csrf_token };
    expect((await page.request.post(`/api/v1/projects/${project.id}/file-operations`, { headers, data: { kind: "create_file", project_version: project.version, target_folder_id: project.main_folder_id, target_path: path } })).ok()).toBe(true);
    await page.goto(`${process.env.PERSISTTY_E2E_BASE_URL}/projects/${project.id}`); await page.getByRole("treeitem", { name: path, exact: true }).click();
    const editor = page.getByRole("textbox", { name: `${path} 内容`, exact: true }); await expect(editor).toBeVisible(); await editor.fill("中文😀 portrait");
    const url = `/api/v1/projects/${project.id}/folders/${project.main_folder_id}/content?project_version=${project.version}&path=${path}`;
    await expect.poll(async () => (await (await page.request.get(url)).json()).data.content).toBe("中文😀 portrait");
    await editor.fill("中文😀 未保存旋转");
    await page.setViewportSize({ width: 844, height: 390 });
    await expect(page.getByRole("navigation", { name: "工作区视图" })).toBeVisible(); await expect(editor).toHaveValue("中文😀 未保存旋转"); await expect(page.locator(".monaco-editor")).toHaveCount(0);
    await expect.poll(async () => (await (await page.request.get(url)).json()).data.content).toBe("中文😀 未保存旋转");
    await editor.fill("中文😀 landscape"); await expect.poll(async () => (await (await page.request.get(url)).json()).data.content).toBe("中文😀 landscape");
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true); await page.screenshot({ path: testInfo.outputPath("w05-touch-landscape.png") });
  } finally { await context.close(); }
});
