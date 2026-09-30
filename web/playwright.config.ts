import { defineConfig } from "@playwright/test";

if (!process.env.PERSISTTY_E2E_BASE_URL || !process.env.PERSISTTY_E2E_PASSWORD || !process.env.PERSISTTY_E2E_PROJECT) {
  throw new Error("E2E 只允许显式指定隔离实例地址、测试密码和项目名");
}

export default defineConfig({
  testDir: "./tests/e2e",
  workers: 1,
  reporter: "list",
  use: {
    baseURL: process.env.PERSISTTY_E2E_BASE_URL,
    screenshot: "only-on-failure",
    trace: "retain-on-failure",
  },
});
