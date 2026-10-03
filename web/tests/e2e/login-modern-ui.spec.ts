import { expect, test, type Page } from "@playwright/test";

async function anonymous(page: Page) {
  const errors: string[] = [];
  page.on("pageerror", error => errors.push(error.message));
  await page.route("**/api/v1/auth/session", route => route.fulfill({ status: 401, json: { error: { code: "unauthenticated", message: "请先登录。" }, request_id: "fixture" } }));
  return errors;
}

async function geometry(page: Page) {
  const size = await page.evaluate(() => ({ scroll: document.documentElement.scrollWidth, viewport: innerWidth }));
  expect(size.scroll).toBe(size.viewport);
  const input = (await page.getByLabel("访问密码", { exact: true }).boundingBox())!, frame = (await page.locator(".login-frame").boundingBox())!;
  expect(input.x).toBeGreaterThan(frame.x); expect(input.x + input.width).toBeLessThan(frame.x + frame.width);
  await expect(page.getByRole("button", { name: "登录", exact: true })).toBeVisible();
}

test("登录页桌面深浅主题沿用工作台细框线", async ({ page }, info) => {
  await page.setViewportSize({ width: 1440, height: 845 }); const errors = await anonymous(page);
  await page.goto("/login"); await expect(page.getByRole("heading", { name: "登录工作台" })).toBeVisible();
  await expect(page.getByLabel("访问密码", { exact: true })).toBeFocused();
  for (const dark of [false, true]) {
    await page.evaluate(dark => document.documentElement.classList.toggle("dark", dark), dark);
    await geometry(page); await expect(page.locator(".login-frame")).toHaveCSS("border-radius", "8px");
    await page.screenshot({ path: info.outputPath(`login-desktop-${dark ? "dark" : "light"}.png`) });
  }
  expect(errors).toEqual([]);
});

test.describe("触屏登录", () => {
  test.use({ hasTouch: true, isMobile: true });
  test("横竖屏单列布局与密码控件可达", async ({ page }, info) => {
    const errors = await anonymous(page); await page.goto("/login");
    for (const viewport of [{ width: 390, height: 844 }, { width: 844, height: 390 }]) {
      await page.setViewportSize(viewport);
      for (const dark of [false, true]) {
        await page.evaluate(dark => document.documentElement.classList.toggle("dark", dark), dark);
        await geometry(page);
        const intro = (await page.locator(".login-intro").boundingBox())!, form = (await page.locator(".login-form").boundingBox())!;
        expect(form.y).toBeGreaterThanOrEqual(intro.y + intro.height - 1);
        await page.getByLabel("访问密码", { exact: true }).fill("test-only-password");
        await expect(page.getByRole("button", { name: "登录", exact: true })).toBeEnabled();
        await page.screenshot({ path: info.outputPath(`login-touch-${viewport.width}-${dark ? "dark" : "light"}.png`) });
      }
    }
    expect(errors).toEqual([]);
  });
});

test("密码提交去重、错误清空、429冷却和成功返回路径", async ({ page }) => {
  const errors = await anonymous(page); let requests = 0; let release = () => {};
  await page.clock.install();
  await page.route("**/api/v1/auth/login", async route => {
    requests++;
    expect(route.request().method()).toBe("POST");
    expect(route.request().postDataJSON()).toEqual({ password: "test-only-password" });
    if (requests === 1) {
      await new Promise<void>(resolve => { release = resolve; });
      await route.fulfill({ status: 401, json: { error: { code: "invalid_credentials", message: "访问密码不正确。" }, request_id: "fixture" } });
    } else if (requests === 2) await route.fulfill({ status: 429, headers: { "Retry-After": "2" }, json: { error: { code: "rate_limited", message: "请稍后再试。" }, request_id: "fixture" } });
    else await route.fulfill({ json: { data: { authenticated: true, csrf_token: "fixture", expires_at: new Date(Date.now() + 600000).toISOString() }, request_id: "fixture" } });
  });
  await page.route("**/api/v1/projects", route => route.fulfill({ json: { data: { items: [] }, request_id: "fixture" } }));
  await page.goto("/login?return=%2Fprojects");
  const input = page.getByLabel("访问密码", { exact: true }), submit = page.getByRole("button", { name: "登录", exact: true });
  await input.fill("test-only-password"); await input.press("Enter");
  await expect.poll(() => requests).toBe(1); await expect(input).toHaveValue(""); await expect(input).toBeDisabled();
  await expect(page.getByRole("button", { name: "正在登录" })).toBeDisabled();
  await page.keyboard.press("Enter"); expect(requests).toBe(1); release();
  await expect(page.getByText("访问密码不正确。", { exact: true })).toBeVisible(); await expect(input).toHaveAttribute("aria-invalid", "true");
  await input.fill("test-only-password"); await submit.click();
  await expect(page.getByText("请求过于频繁。请在 2 秒后重试。", { exact: true })).toBeVisible(); await expect(input).toBeDisabled();
  await page.clock.runFor(2001); await expect(input).toBeEnabled(); expect(requests).toBe(2);
  await input.fill("test-only-password"); await input.press("Enter");
  await expect(page).toHaveURL(/\/projects$/); expect(requests).toBe(3); expect(errors).toEqual([]);
});
