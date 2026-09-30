import { expect, test } from "@playwright/test";
import type {} from "../fixtures/editor-assets";

test("全部 91 个模式实际加载语法，包含无后缀变体", async ({ page }) => {
  test.setTimeout(90000);
  const errors: string[] = []; page.on("pageerror", error => errors.push(error.message));
  await page.goto("/tests/fixtures/editor-assets.html");
  await expect(page.locator(".monaco-editor")).toBeVisible();
  const results = await page.evaluate(async () => {
    const { monaco, samples, ids } = window.editorAssets;
    const registered = new Set(monaco.languages.getLanguages().map(item => item.id));
    const failed: string[] = [];
    for (const id of ids) {
      if (!registered.has(id) || !Object.hasOwn(samples, id)) { failed.push(`${id}: 缺注册或样例`); continue; }
      const sample = samples[id]!;
      const model = monaco.editor.createModel(sample, id);
      if (id !== "plaintext") {
        let highlighted = false;
        for (let retry = 0; retry < 100; retry++) {
          highlighted = monaco.editor.tokenize(sample, id).flat().some(token => token.type !== "" && !token.type.startsWith("white"));
          if (highlighted) break;
          await new Promise(resolve => setTimeout(resolve, 20));
        }
        if (!highlighted) failed.push(`${id}: 无语法 token`);
      }
      model.dispose();
    }
    return { count: ids.length, failed };
  });
  expect(results.count).toBe(91); expect(results.failed).toEqual([]); expect(errors).toEqual([]);
});

test("真实 model 切模式/标签/主题保持选择和滚动，内置图标使用本地资源", async ({ page }) => {
  const external: string[] = []; const errors: string[] = [];
  page.on("request", request => { if (!request.url().startsWith("http://127.0.0.1:") && !request.url().startsWith("data:")) external.push(request.url()); });
  page.on("pageerror", error => errors.push(error.message));
  await page.goto("/tests/fixtures/editor-assets.html"); await expect(page.locator(".monaco-editor")).toBeVisible();
  const before = await page.evaluate(() => {
    const { monaco, uri } = window.editorAssets;
    const model = monaco.editor.getModel(monaco.Uri.parse(uri("fixture", { folderId: "root", path: "query.sql" })))!;
    const editor = monaco.editor.getEditors().find(item => item.getModel() === model)!;
    editor.setPosition({ lineNumber: 40, column: 4 }); editor.revealLineInCenter(40);
    return { id: model.id, position: editor.getPosition(), scroll: editor.getScrollTop(), content: model.getValue() };
  });
  await page.getByRole("combobox", { name: "语言模式" }).click();
  await page.getByRole("option", { name: "PostgreSQL", exact: true }).click();
  await expect.poll(() => page.evaluate(() => window.editorAssets.monaco.editor.getModels().find(model => model.uri.path.endsWith("query.sql"))?.getLanguageId())).toBe("pgsql");
  await page.getByRole("button", { name: "view.ftl", exact: true }).click();
  await page.getByRole("combobox", { name: "语言模式" }).click();
  await page.getByRole("option", { name: "FreeMarker2 (Bracket/Bracket)", exact: true }).click();
  await page.getByRole("button", { name: "query.sql", exact: true }).click();
  await expect(page.getByRole("combobox", { name: "语言模式" })).toHaveText(/PostgreSQL/);
  await page.getByRole("button", { name: "切换主题" }).click();
  await expect(page.locator('img[data-file-icon="toml"]')).toHaveCount(1);
  const after = await page.evaluate(() => {
    const { monaco, uri } = window.editorAssets;
    const model = monaco.editor.getModel(monaco.Uri.parse(uri("fixture", { folderId: "root", path: "query.sql" })))!;
    const editor = monaco.editor.getEditors().find(item => item.getModel() === model)!;
    return { id: model.id, position: editor.getPosition(), scroll: editor.getScrollTop(), content: model.getValue() };
  });
  expect(after).toEqual(before);
  await page.getByRole("combobox", { name: "语言模式" }).click(); await page.getByRole("option", { name: "自动识别", exact: true }).click();
  await expect.poll(() => page.evaluate(() => window.editorAssets.monaco.editor.getModels().find(model => model.uri.path.endsWith("query.sql"))?.getLanguageId())).toBe("sql");
  await expect.poll(() => page.locator("img").evaluateAll(images => images.every(image => image instanceof HTMLImageElement && image.complete && image.naturalWidth > 0))).toBe(true);
  expect(await page.locator('img[data-file-icon="yaml"]').count()).toBe(2);
  await expect(page.locator('img[data-file-icon="typescript-def"]')).toHaveCount(1); await expect(page.locator('img[data-file-icon="typescript"]')).toHaveCount(1);
  expect(external).toEqual([]); expect(errors).toEqual([]);
});
