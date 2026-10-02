import { expect, test, type Page } from "@playwright/test";
import { decodeEnvelope, decodeList, decodeProject } from "../../src/lib/api/decoder";

async function login(page: Page) {
  await page.goto("/projects");
  await page.getByLabel("访问密码").fill(process.env.PERSISTTY_E2E_PASSWORD!);
  await page.getByRole("button", { name: "登录", exact: true }).click();
  await expect(page.getByRole("button", { name: "新建项目", exact: true })).toBeVisible();
}

test("长服务器路径与目录名称保持在对话框内，桌面和窄屏均可取消", async ({ page }, info) => {
  const path = "/fixture/" + "long-project-name/".repeat(14);
  const name = "很长的目录名称".repeat(20);
  const mutations: string[] = [];
  page.on("request", request => { if (request.method() !== "GET" && !request.url().includes("/auth/")) mutations.push(request.url()); });
  await page.route("**/api/v1/directories?*", route => route.fulfill({ json: { data: { path, parent: "/fixture", items: [name, "images"] }, request_id: "fixture" } }));
  await login(page);
  for (const width of [1440, 390]) {
    await page.setViewportSize({ width, height: 844 });
    await page.getByRole("button", { name: "新建项目", exact: true }).click();
    await page.getByRole("button", { name: "添加", exact: true }).click();
    const dialog = page.getByRole("dialog", { name: "选择服务器文件夹" });
    await expect(dialog.getByRole("textbox", { name: "服务器绝对路径" })).toHaveValue(path);
    const layout = await dialog.evaluate(element => {
      const bounds = element.getBoundingClientRect();
      return { left: bounds.left, right: bounds.right, width: innerWidth, overflow: element.scrollWidth > element.clientWidth,
        controlsContained: [...element.querySelectorAll('input,button,ul')].every(child => { const r = child.getBoundingClientRect(); return r.left >= bounds.left - 1 && r.right <= bounds.right + 1; }) };
    });
    expect(layout.left).toBeGreaterThanOrEqual(0); expect(layout.right).toBeLessThanOrEqual(layout.width);
    expect(layout.overflow).toBe(false); expect(layout.controlsContained).toBe(true);
    await dialog.screenshot({ path: info.outputPath(`directory-${width}.png`) });
    await dialog.getByRole("button", { name: "取消", exact: true }).click();
    await expect(dialog).toHaveCount(0);
    await page.getByRole("dialog").getByRole("button", { name: "取消", exact: true }).click();
  }
  expect(mutations).toEqual([]);
});

test("多文件夹无仓库禁用选择器，新仓库无提交可查看并全屏切换差异", async ({ page }, info) => {
  test.skip(!process.env.PERSISTTY_E2E_W06_ROOT?.startsWith("/private/tmp/persistty-w06-browser-"), "仅使用专属合成项目");
  const errors: string[] = []; const writes: string[] = [];
  page.on("pageerror", error => errors.push(error.message));
  page.on("request", request => { if (request.method() !== "GET" && !request.url().includes("/auth/") && !request.url().includes("/comparisons")) writes.push(request.url()); });
  await login(page);
  const projects = decodeEnvelope(await (await page.request.get("/api/v1/projects")).json(), decodeList(decodeProject));
  const empty = projects.find(project => project.name === "W06 无仓库验收");
  const unborn = projects.find(project => project.name === "W06 新仓库验收");
  if (!empty || !unborn) throw new Error("缺专属合成项目");
  await page.goto(`/projects/${empty.id}`); await page.getByRole("button", { name: "只读 Git", exact: true }).click();
  await expect(page.getByRole("combobox", { name: "选择仓库" })).toBeDisabled();
  await expect(page.getByText("未发现 Git 仓库", { exact: true })).toHaveCount(2);
  await page.getByRole("button", { name: "刷新", exact: true }).click();
  await expect(page.getByRole("combobox", { name: "选择仓库" })).toBeDisabled();
  await page.goto(`/projects/${unborn.id}`); await page.getByRole("button", { name: "只读 Git", exact: true }).click();
  await expect(page.getByText("main · 尚无提交", { exact: true })).toBeVisible();
  await expect(page.getByRole("combobox", { name: "选择仓库" })).toBeEnabled();
  const refreshed = page.waitForResponse(response => /\/repositories\/[^/]+\/status\?/.test(response.url()) && response.ok());
  await page.getByRole("button", { name: "刷新", exact: true }).click(); await refreshed;
  await expect(page.getByText("main · 尚无提交", { exact: true })).toBeVisible();
  await expect(page.getByText("尚无提交历史。", { exact: true })).toBeVisible();
  await page.getByRole("button", { name: "本地引用", exact: true }).click(); await expect(page.getByRole("combobox", { name: "本地分支或标签" })).toBeDisabled();
  await page.getByRole("button", { name: "关闭引用", exact: true }).click(); await page.getByRole("button", { name: /^new.txt/ }).click();
  const dialog = page.getByRole("dialog"); const diff = dialog.locator(".monaco-diff-editor");
  await expect(diff).toHaveClass(/side-by-side/);
  await dialog.getByRole("tab", { name: "行内", exact: true }).click(); await expect(diff).not.toHaveClass(/side-by-side/);
  await dialog.getByRole("button", { name: "全屏比较", exact: true }).click();
  await expect.poll(() => dialog.evaluate(element => { const r = element.getBoundingClientRect(); return [Math.round(r.left), Math.round(r.top), Math.round(r.width), Math.round(r.height)]; })).toEqual([0, 0, 1280, 720]);
  await dialog.screenshot({ path: info.outputPath("git-inline-fullscreen.png") });
  await dialog.getByRole("tab", { name: "并排", exact: true }).click(); await expect(diff).toHaveClass(/side-by-side/);
  await dialog.getByRole("button", { name: "退出全屏比较", exact: true }).click(); await expect(dialog).toHaveAttribute("data-fullscreen", "false");
  await dialog.getByRole("button", { name: "关闭比较", exact: true }).click(); await expect(dialog).toHaveCount(0);
  await page.getByRole("button", { name: /^new.txt/ }).click(); await expect(diff).toHaveClass(/side-by-side/); await expect(dialog).toHaveAttribute("data-fullscreen", "false");
  await dialog.getByRole("button", { name: "关闭比较", exact: true }).click();
  expect(writes).toEqual([]); expect(errors).toEqual([]);
});
