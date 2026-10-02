import { expect, test } from "@playwright/test";
import { mkdir, readFile, rm, writeFile } from "node:fs/promises";
import { execFileSync } from "node:child_process";
import { join } from "node:path";
import { createHash } from "node:crypto";
test("Git 双区域默认平分，可拖动、折叠并切换带 Material 图标的变更树", async ({ page }, info) => {
  const project = process.env.PERSISTTY_E2E_W06_PROJECT, root = process.env.PERSISTTY_E2E_W06_ROOT;
  test.skip(!project || !root?.startsWith("/private/tmp/persistty-w06-browser-"), "仅操作专属隔离项目");
  await mkdir(`${root}/src/nested`, { recursive: true });
  await writeFile(`${root}/src/nested/example.ts`, "export const changed = true;\n");
  const errors: string[] = []; page.on("pageerror", error => errors.push(error.message));
  await page.goto(`/projects/${project}`); await page.getByLabel("访问密码").fill(process.env.PERSISTTY_E2E_PASSWORD!); await page.getByRole("button", { name: "登录", exact: true }).click();
  await page.getByRole("button", { name: "只读 Git", exact: true }).click();
  const panel = page.getByRole("region", { name: "只读 Git" });
  const changes = panel.getByRole("region", { name: "变更内容" }), history = panel.getByRole("region", { name: "历史内容" });
  await expect(changes.getByRole("button", { name: "src/nested/example.ts", exact: true })).toBeVisible();
  const sizes = async () => [await changes.evaluate(el => el.parentElement!.getBoundingClientRect().height), await history.evaluate(el => el.parentElement!.getBoundingClientRect().height)];
  const initial = await sizes(); expect(Math.abs(initial[0]! - initial[1]!)).toBeLessThan(3);
  const separator = panel.getByRole("separator", { name: "调整变更与历史高度" });
  const box = (await separator.boundingBox())!;
  await page.mouse.move(box.x + box.width / 2, box.y + .5); await page.mouse.down(); await page.mouse.move(box.x + box.width / 2, box.y + 65, { steps: 10 }); await page.mouse.up();
  await expect.poll(async () => (await sizes())[0]!).toBeGreaterThan(initial[0]! + 30);
  const resized = await sizes();
  await panel.getByRole("button", { name: "变更", exact: true }).click(); await expect(changes).toBeHidden(); await expect(history).toBeVisible();
  await panel.getByRole("button", { name: "历史", exact: true }).click(); await expect(history).toBeHidden();
  await panel.getByRole("button", { name: "变更", exact: true }).click(); await expect(changes).toBeVisible();
  await panel.getByRole("button", { name: "历史", exact: true }).click(); await expect(history).toBeVisible();
  await expect.poll(async () => (await sizes())[0]!).toBeCloseTo(resized[0]!, 0);
  await panel.getByRole("button", { name: "变更文件展示：文件树", exact: true }).click();
  await expect(changes.locator('.git-files[data-view="tree"]')).toBeVisible();
  const src = changes.getByRole("button", { name: "src", exact: true });
  await expect(src.locator("img[data-file-icon]")).toHaveCount(1);
  await expect(changes.getByRole("button", { name: "src/nested/example.ts", exact: true }).locator("img[data-file-icon]")).toHaveAttribute("data-file-icon", "typescript");
  await src.click(); await expect(changes.getByRole("button", { name: "src/nested/example.ts", exact: true })).toBeHidden(); await src.click();
  await history.getByRole("button", { name: /^initial/ }).click();
  await expect(history.locator(".git-commit-detail").getByRole("button", { name: "sample.txt", exact: true })).toBeVisible();
  await panel.getByRole("button", { name: "历史文件展示：文件树", exact: true }).click();
  await expect(history.locator('.git-files[data-view="tree"]')).toBeVisible();
  await expect(history.locator("svg circle")).toHaveCount(1);
  await panel.screenshot({ path: info.outputPath("git-split-tree.png") });
  expect(errors).toEqual([]);
});


test("真实合并历史展示两条父关系，详情内切父节点并以文件树比较", async ({ page, browserName }, info) => {
  const root = process.env.PERSISTTY_E2E_W06_ROOT;
  test.skip(!root?.startsWith("/private/tmp/persistty-w06-browser-"), "仅操作专属隔离项目");
  const graphRoot = join(root!, "..", `graph-fixture-${browserName}`);
  await rm(graphRoot, { recursive: true, force: true });
  await mkdir(`${graphRoot}/src`, { recursive: true }); await mkdir(`${graphRoot}/ui`, { recursive: true });
  const git = (...args: string[]) => execFileSync("git", ["-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", ...args], { cwd: graphRoot, env: { ...process.env, GIT_CONFIG_GLOBAL: "/dev/null", GIT_CONFIG_NOSYSTEM: "1" }, timeout: 5000 });
  git("init", "--quiet", "--initial-branch=main"); await writeFile(`${graphRoot}/src/main.ts`, "export const initial = true;\n"); git("add", "--", "src/main.ts"); git("commit", "--quiet", "-m", "root fixture");
  git("checkout", "--quiet", "-b", "feature"); await writeFile(`${graphRoot}/ui/view.ts`, "export const view = true;\n"); git("add", "--", "ui/view.ts"); git("commit", "--quiet", "-m", "feature fixture");
  git("checkout", "--quiet", "main"); await writeFile(`${graphRoot}/src/main.ts`, "export const backend = true;\n"); git("add", "--", "src/main.ts"); git("commit", "--quiet", "-m", "backend fixture"); git("merge", "--quiet", "--no-ff", "-m", "Merge fixture", "feature");
  const hash = async (name: string) => createHash("sha256").update(await readFile(`${graphRoot}/.git/${name}`)).digest("hex");
  const before = await Promise.all([hash("HEAD"), hash("index"), hash("config")]);
  await page.goto("/projects"); await page.getByLabel("访问密码").fill(process.env.PERSISTTY_E2E_PASSWORD!); await page.getByRole("button", { name: "登录", exact: true }).click();
  await expect(page.getByRole("button", { name: "新建项目", exact: true })).toBeVisible();
  const session = (await (await page.request.get("/api/v1/auth/session")).json()).data;
  const headers = { Origin: process.env.PERSISTTY_E2E_BASE_URL!, "X-CSRF-Token": session.csrf_token };
  const created = await page.request.post("/api/v1/projects", { headers, data: { name: "W06 合并图临时检查", folder_paths: [graphRoot], main_index: 0 } });
  expect(created.status()).toBe(201); const project = (await created.json()).data;
  try {
    await page.goto(`/projects/${project.id}`); await page.getByRole("button", { name: "只读 Git", exact: true }).click();
    const panel = page.getByRole("region", { name: "只读 Git" });
    await expect(panel.locator(".git-commit")).toHaveCount(4);
    const merge = panel.locator(".git-commit").first();
    await expect(merge.locator("path[data-parent]")).toHaveCount(2);
    await merge.getByRole("button", { name: /^Merge fixture/ }).click();
    await expect(merge.getByRole("button", { name: "ui/view.ts", exact: true })).toBeVisible();
    await panel.getByRole("button", { name: "历史文件展示：文件树", exact: true }).click();
    await expect(merge.getByRole("button", { name: "ui", exact: true })).toBeVisible();
    const parents = git("rev-list", "--parents", "-n", "1", "HEAD").toString().trim().split(" ").slice(1);
    await expect(merge.getByRole("combobox", { name: "比较父提交" })).toHaveCount(0);
    await merge.getByRole("button", { name: /^Merge fixture/ }).click({ button: "right" });
    await expect(page.getByRole("menuitemradio", { name: `父提交 1 · ${parents[0]!.slice(0, 12)}`, exact: true })).toHaveAttribute("aria-checked", "true");
    await page.getByRole("menuitemradio", { name: `父提交 2 · ${parents[1]!.slice(0, 12)}`, exact: true }).click();
    await expect(page.getByRole("menu")).toBeHidden();
    await expect(merge.getByRole("button", { name: "src/main.ts", exact: true })).toBeVisible(); await expect(merge.getByRole("button", { name: "ui/view.ts", exact: true })).toHaveCount(0);
    const boundary = await merge.locator(".git-commit-files").evaluate(el => ({ margin: getComputedStyle(el).marginLeft, border: getComputedStyle(el).borderLeftWidth, height: el.getBoundingClientRect().height }));
    expect(boundary.margin).toBe("3px"); expect(boundary.border).toBe("1px"); expect(boundary.height).toBeGreaterThan(28);
    await merge.getByRole("button", { name: /^Merge fixture/ }).click();
    await expect(merge.getByRole("button", { name: "src/main.ts", exact: true })).toBeHidden();
    await merge.getByRole("button", { name: /^Merge fixture/ }).click({ button: "right" });
    await page.getByRole("menuitemradio", { name: `父提交 1 · ${parents[0]!.slice(0, 12)}`, exact: true }).click();
    await expect(merge.getByRole("button", { name: "ui/view.ts", exact: true })).toBeVisible();
    await merge.getByRole("button", { name: /^Merge fixture/ }).click({ button: "right" });
    await page.getByRole("menuitemradio", { name: `父提交 2 · ${parents[1]!.slice(0, 12)}`, exact: true }).click();
    await expect(merge.getByRole("button", { name: "src/main.ts", exact: true })).toBeVisible();
    await panel.screenshot({ path: info.outputPath("git-merge-graph.png") });
    await merge.getByRole("button", { name: "src/main.ts", exact: true }).click(); await expect(page.getByRole("dialog", { name: /只读比较/ })).toBeVisible();
    await page.getByRole("button", { name: "关闭比较", exact: true }).click();
    expect(await Promise.all([hash("HEAD"), hash("index"), hash("config")])).toEqual(before);
  } finally {
    expect((await page.request.delete(`/api/v1/projects/${project.id}`, { headers, data: { expected_version: project.version } })).status()).toBe(204);
    await rm(graphRoot, { recursive: true, force: true });
  }
});


test.afterEach(async ({ browserName }) => {
  const root = process.env.PERSISTTY_E2E_W06_ROOT;
  if (root?.startsWith("/private/tmp/persistty-w06-browser-")) await rm(join(root, "..", `graph-fixture-${browserName}`), { recursive: true, force: true });
});
