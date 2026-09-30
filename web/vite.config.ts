import { fileURLToPath, URL } from "node:url";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import { defineConfig } from "vitest/config";

const apiTarget = process.env.PERSISTTY_DEV_API_TARGET ?? "http://127.0.0.1:8080";
if (!/^http:\/\/127\.0\.0\.1:\d{1,5}$/.test(apiTarget)) throw new Error("开发 API 代理只允许 127.0.0.1 端口");

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: { alias: { "@": fileURLToPath(new URL("./src", import.meta.url)) } },
  server: {
    port: 5173,
    strictPort: true,
    proxy: { "/api": { target: apiTarget, changeOrigin: false, ws: true } },
  },
  test: { include: ["src/**/*.test.{ts,tsx}"], environment: "jsdom", setupFiles: ["./src/test/setup.ts"], restoreMocks: true },
});
