import { expect, test } from "@playwright/test";
import { mkdir, rm } from "node:fs/promises";
import { execFileSync } from "node:child_process";
import { join } from "node:path";

test("历史滚动到底自动追加三页，慢请求不重复且末页停止", async ({ page, browserName }) => {
  const root = process.env.PERSISTTY_E2E_W06_ROOT;
  test.skip(!root?.startsWith("/private/tmp/persistty-w06-browser-"), "仅操作专属隔离项目");
  const repo = join(root!, "..", `pagination-fixture-${browserName}`);
  await mkdir(repo, { recursive: true });
  const git = (...args: string[]) => execFileSync("git", ["-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", ...args], { cwd: repo, env: { ...process.env, GIT_CONFIG_GLOBAL: "/dev/null", GIT_CONFIG_NOSYSTEM: "1" }, timeout: 5000 });
  let release: () => void = () => {};
  const gate = new Promise<void>(resolve => { release = resolve; });
  try {
    git("init", "--quiet", "--initial-branch=main");
    for (let i = 0; i < 105; i++) git("commit", "--quiet", "--allow-empty", "-m", `Pagination fixture ${i}`);
    const head = git("rev-parse", "HEAD").toString().trim();
    await page.goto("/projects");
    await page.getByLabel("访问密码").fill(process.env.PERSISTTY_E2E_PASSWORD!);
    await page.getByRole("button", { name: "登录", exact: true }).click();
    await expect(page.getByRole("button", { name: "新建项目", exact: true })).toBeVisible();
    const session = (await (await page.request.get("/api/v1/auth/session")).json()).data;
    const headers = { Origin: process.env.PERSISTTY_E2E_BASE_URL!, "X-CSRF-Token": session.csrf_token };
    const created = await page.request.post("/api/v1/projects", { headers, data: { name: "W06 自动分页临时检查", folder_paths: [repo], main_index: 0 } });
    expect(created.status()).toBe(201);
    const project = (await created.json()).data;
    try {
      const pages: { offset: string | null; head: string | null }[] = [];
      await page.route(`**/api/v1/projects/${project.id}/repositories/*/log?*`, async route => {
        const params = new URL(route.request().url()).searchParams;
        pages.push({ offset: params.get("offset"), head: params.get("head") });
        if (params.get("offset") === "50") await gate;
        await route.continue();
      });
      await page.goto(`/projects/${project.id}`);
      await page.getByRole("button", { name: "只读 Git", exact: true }).click();
      const panel = page.getByRole("region", { name: "只读 Git" }), history = panel.getByRole("region", { name: "历史内容" });
      await expect(history.locator(".git-commit")).toHaveCount(50);
      await expect(panel.getByRole("button", { name: "加载更多提交" })).toHaveCount(0);
      expect(pages).toEqual([{ offset: "0", head: "" }]);
      await history.hover(); await page.mouse.wheel(0, 10000);
      await expect.poll(() => pages.length).toBe(2);
      await page.mouse.wheel(0, -60); await page.mouse.wheel(0, 10000); await page.mouse.wheel(0, 10000);
      expect(pages).toHaveLength(2);
      release();
      await expect(history.locator(".git-commit")).toHaveCount(100);
      expect(pages).toHaveLength(2);
      await history.hover(); await page.mouse.wheel(0, 10000);
      await expect(history.locator(".git-commit")).toHaveCount(105);
      await page.mouse.wheel(0, -60); await page.mouse.wheel(0, 10000); await page.mouse.wheel(0, 10000);
      await expect(history.locator(".git-commit").last()).toBeVisible();
      expect(pages).toEqual([{ offset: "0", head: "" }, { offset: "50", head }, { offset: "100", head }]);
      expect(await history.locator(".git-commit-trigger .truncate").allTextContents()).toEqual(Array.from({ length: 105 }, (_, i) => `Pagination fixture ${104 - i}`));
      expect(git("rev-parse", "HEAD").toString().trim()).toBe(head);
    } finally {
      release();
      expect((await page.request.delete(`/api/v1/projects/${project.id}`, { headers, data: { expected_version: project.version } })).status()).toBe(204);
    }
  } finally { release(); await rm(repo, { recursive: true, force: true }); }
});
