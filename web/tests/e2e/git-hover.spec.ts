import { expect, test } from "@playwright/test";
import { mkdir, readFile, rm, writeFile } from "node:fs/promises";
import { execFileSync } from "node:child_process";
import { join } from "node:path";
import { createHash } from "node:crypto";

test("提交与文件悬浮卡片展示完整消息、真实统计并复用详情，GitHub 链接可打开", async ({ page, context, browserName }, info) => {
  test.setTimeout(60000);
  const root = process.env.PERSISTTY_E2E_W06_ROOT;
  test.skip(!root?.startsWith("/private/tmp/persistty-w06-browser-"), "仅操作专属隔离项目");
  const repo = join(root!, "..", `hover-fixture-${browserName}`);
  const git = (...args: string[]) => execFileSync("git", ["-c", "user.name=悬浮测试作者", "-c", "user.email=fixture@example.invalid", ...args], { cwd: repo, env: { ...process.env, GIT_CONFIG_GLOBAL: "/dev/null", GIT_CONFIG_NOSYSTEM: "1" }, timeout: 5000 });
  await mkdir(join(repo, "src"), { recursive: true });
  try {
    git("init", "--quiet", "--initial-branch=main");
    await writeFile(join(repo, "src/modify.ts"), "before\n");
    await writeFile(join(repo, "src/old.ts"), "unchanged\n");
    await writeFile(join(repo, "delete.txt"), "removed\n");
    git("add", "--", "."); git("commit", "--quiet", "-m", "root hover fixture");
    git("mv", "--", "src/old.ts", "src/new.ts");
    await writeFile(join(repo, "src/modify.ts"), "after\nextra\n");
    await writeFile(join(repo, "added.md"), "added\n");
    await writeFile(join(repo, "binary.dat"), Buffer.from([0, 1, 2]));
    await rm(join(repo, "delete.txt")); git("add", "--", ".");
    const subject = `悬浮 fixture ${"长提交消息".repeat(30)}`, body = "完整消息第二段\n保留换行和特殊文本 <script>正文</script>";
    git("commit", "--quiet", "-m", subject, "-m", body);
    git("remote", "add", "origin", "git@github.com:fixture/hover-cards.git");
    const id = git("rev-parse", "HEAD").toString().trim(), url = `https://github.com/fixture/hover-cards/commit/${id}`;
    const hashes = async () => Promise.all(["HEAD", "index", "config"].map(async name => createHash("sha256").update(await readFile(join(repo, ".git", name))).digest("hex")));
    const before = await hashes(), errors: string[] = [];
    page.on("pageerror", error => errors.push(error.message));
    page.on("console", message => { if (message.type() === "error" && message.text().includes("Base UI")) errors.push(message.text()); });
    await page.goto("/projects"); await page.getByLabel("访问密码").fill(process.env.PERSISTTY_E2E_PASSWORD!); await page.getByRole("button", { name: "登录", exact: true }).click();
    await expect(page.getByRole("button", { name: "新建项目", exact: true })).toBeVisible();
    const session = (await (await page.request.get("/api/v1/auth/session")).json()).data;
    const headers = { Origin: process.env.PERSISTTY_E2E_BASE_URL!, "X-CSRF-Token": session.csrf_token };
    const created = await page.request.post("/api/v1/projects", { headers, data: { name: "W06 悬浮卡片临时检查", folder_paths: [repo], main_index: 0 } });
    expect(created.status()).toBe(201); const project = (await created.json()).data;
    try {
      const requests: string[] = [];
      page.on("request", request => { if (new URL(request.url()).pathname.endsWith(`/commits/${id}`)) requests.push(request.url()); });
      // 外部页面使用本地响应；只验证用户点击后的浏览器跳转，不访问 GitHub 网络。
      await context.route(url, route => route.fulfill({ contentType: "text/html", body: "<title>GitHub fixture</title>" }));
      await page.goto(`/projects/${project.id}`); await page.getByRole("button", { name: "只读 Git", exact: true }).click();
      const panel = page.getByRole("region", { name: "只读 Git" }), commit = panel.locator(".git-commit").first(), trigger = commit.locator(".git-commit-trigger");
      await expect(trigger).toContainText(subject);
      const truncated = await trigger.locator(".truncate").evaluate(el => el.scrollWidth > el.clientWidth); expect(truncated).toBe(true);
      await trigger.hover();
      const card = page.getByRole("region", { name: "提交信息", exact: true });
      await expect(card).toContainText("已更改 5 个文件"); await expect(card).toContainText("3 行插入 (+)"); await expect(card).toContainText("2 行删除 (-)");
      await expect(card).toContainText("另有 1 个二进制文件"); await expect(card).toContainText("悬浮测试作者"); await expect(card.locator("time")).toHaveAttribute("datetime", /\d{4}-\d{2}-\d{2}T/);
      await expect(card.locator(".git-hover-message")).toHaveText(`${subject}\n\n${body}\n`); await expect(card.locator("code")).toHaveText(id);
      await card.hover(); const link = card.getByRole("link", { name: "在 GitHub 上打开", exact: true });
      await expect(link).toHaveAttribute("href", url); await expect(link).toHaveAttribute("target", "_blank"); await expect(link).toHaveAttribute("rel", "noopener noreferrer");
      await page.screenshot({ path: info.outputPath("commit-hover.png") });
      const popupPromise = context.waitForEvent("page"); await link.click(); const popup = await popupPromise; await popup.waitForURL(url); await expect(popup).toHaveTitle("GitHub fixture"); await popup.close();
      await panel.getByRole("heading").hover(); await expect(card).toBeHidden();
      await trigger.hover(); await expect(card).toContainText("已更改 5 个文件"); expect(requests).toHaveLength(1);
      await trigger.click(); await expect(commit.getByRole("button", { name: "src/modify.ts", exact: true })).toBeVisible(); expect(requests).toHaveLength(1);
      const fileCard = page.getByRole("region", { name: "文件变更信息", exact: true });
      await commit.getByRole("button", { name: "src/modify.ts", exact: true }).hover(); await expect(fileCard).toContainText("2 行插入 (+)"); await expect(fileCard).toContainText("1 行删除 (-)");
      await commit.getByRole("button", { name: "binary.dat", exact: true }).hover(); await expect(fileCard).toContainText("二进制文件，不统计文本行数");
      await commit.getByRole("button", { name: "src/new.ts", exact: true }).hover(); await expect(fileCard).toContainText("原路径：src/old.ts"); await expect(fileCard).toContainText("重命名");
      await panel.getByRole("button", { name: "历史文件展示：文件树", exact: true }).click();
      await commit.getByRole("button", { name: "src/modify.ts", exact: true }).hover(); await expect(fileCard).toContainText("2 行插入 (+)");
      expect(requests).toHaveLength(1); await page.screenshot({ path: info.outputPath("file-hover-tree.png") });
      expect(await hashes()).toEqual(before); expect(errors).toEqual([]);
    } finally {
      expect((await page.request.delete(`/api/v1/projects/${project.id}`, { headers, data: { expected_version: project.version } })).status()).toBe(204);
    }
  } finally { await rm(repo, { recursive: true, force: true }); }
});
