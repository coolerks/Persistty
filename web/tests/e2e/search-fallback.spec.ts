import { expect, test } from "@playwright/test";
import { readFile } from "node:fs/promises";

const project = process.env.PERSISTTY_E2E_W06_PROJECT, root = process.env.PERSISTTY_E2E_W06_ROOT;
test("无 rg 的真实全文搜索仍可过滤、定位、预览和明确应用", async ({ page }, info) => {
  test.skip(!project || !root?.startsWith("/private/tmp/persistty-w06-browser-ui-"), "只操作显式隔离且缺 rg 的 fixture");
  const errors: string[] = []; page.on("pageerror", error => errors.push(error.message));
  await page.clock.install();
  await page.goto(`/projects/${project}`);
  await page.getByLabel("访问密码").fill(process.env.PERSISTTY_E2E_PASSWORD!);
  await page.getByRole("button", { name: "登录", exact: true }).click();
  await expect(page.getByRole("main", { name: "Persistty UI 预览 工作台" })).toBeVisible();
  await page.getByRole("button", { name: "终端面板", exact: true }).click();
  await page.getByRole("button", { name: "搜索与替换", exact: true }).click();
  await page.getByLabel("搜索内容", { exact: true }).fill("hit");
  await expect(page.getByLabel("包含文件", { exact: true })).toHaveCount(0);
  await page.getByRole("button", { name: "搜索范围与文件过滤", exact: true }).click();
  await page.getByLabel("包含文件", { exact: true }).fill("sample.txt;ignored.txt");
  await page.getByRole("button", { name: "搜索", exact: true }).click();
  await expect(page.getByRole("status").filter({ hasText: "1 个文件 · 2 处匹配" })).toBeVisible();
  await page.clock.fastForward(16000);
  await expect(page.getByLabel("搜索内容", { exact: true })).toHaveValue("hit");
  await expect(page.getByRole("status").filter({ hasText: "1 个文件 · 2 处匹配" })).toBeVisible();
  await expect(page.getByRole("button", { name: /^ignored.txt/ })).toHaveCount(0);
  await page.getByRole("button", { name: "1:3 😀hit", exact: true }).click();
  await expect(page.locator(".monaco-editor")).toBeVisible();
  await expect(page.locator(".editor-breadcrumb")).toContainText("sample.txt");
  await expect(page.locator(".editor-breadcrumb")).not.toContainText(/HEAD [a-f0-9]/);
  await page.getByLabel("替换为", { exact: true }).fill("done");
  await page.getByRole("button", { name: /预览选中项替换/ }).click();
  const dialog = page.getByRole("dialog", { name: "替换预览", exact: true });
  await expect(dialog).toBeVisible();
  await expect(dialog.getByRole("button", { name: "关闭预览", exact: true })).toBeFocused();
  expect(await readFile(`${root}/sample.txt`, "utf8")).toBe("😀hit\nsecond hit\n");
  await dialog.getByRole("button", { name: "确认应用到 1 个文件", exact: true }).click();
  await expect(dialog.getByText("已替换", { exact: true })).toBeVisible();
  expect(await readFile(`${root}/sample.txt`, "utf8")).toBe("😀done\nsecond done\n");
  await dialog.getByRole("button", { name: "关闭预览", exact: true }).click();
  for (const dark of [false, true]) {
    await page.evaluate(dark => document.documentElement.classList.toggle("dark", dark), dark);
    await page.screenshot({ path: info.outputPath(`search-${dark ? "dark" : "light"}.png`) });
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
  }
  expect(errors).toEqual([]);
});
