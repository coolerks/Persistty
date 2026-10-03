import { afterEach, expect, it, vi } from "vitest";
import { randomUUID } from "./random-uuid";

const browserCrypto = globalThis.crypto;
afterEach(() => { vi.unstubAllGlobals(); });

it("安全上下文复用浏览器的 UUID 方法", () => {
  const native = vi.spyOn(browserCrypto, "randomUUID");
  expect(randomUUID()).toMatch(/^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/);
  expect(native).toHaveBeenCalledOnce();
});

it.each([
  [0, "00000000-0000-4000-8000-000000000000"],
  [255, "ffffffff-ffff-4fff-bfff-ffffffffffff"],
])("HTTP 下生成 UUID v4，保留随机位并设置版本和变体（%i）", (byte, expected) => {
  const getRandomValues = vi.fn((values: Uint8Array) => values.fill(byte));
  vi.stubGlobal("crypto", { getRandomValues });
  expect(randomUUID()).toBe(expected);
  expect(getRandomValues).toHaveBeenCalledWith(expect.any(Uint8Array));
});

it("HTTP 下多次生成 UUID，保留独立视图和定位事件身份", () => {
  vi.stubGlobal("crypto", { getRandomValues: browserCrypto.getRandomValues.bind(browserCrypto) });
  const ids = Array.from({ length: 1000 }, () => randomUUID());
  expect(new Set(ids).size).toBe(ids.length);
  expect(ids.every(id => /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/.test(id))).toBe(true);
});
