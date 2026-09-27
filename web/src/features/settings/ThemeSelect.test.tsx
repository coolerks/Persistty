import { act, fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { expect, it, vi } from "vitest";
import { ThemeSelect } from "./ThemeSelect";
import { applyTheme, readTheme, themeKey } from "./theme";

it("无效/不可读的偏好回退system", () => {
  localStorage.setItem(themeKey, "invalid");
  expect(readTheme()).toBe("system");
  vi.spyOn(Storage.prototype, "getItem").mockImplementation(() => { throw new Error("blocked"); });
  expect(readTheme()).toBe("system");
});
it("跟随系统监听，显式主题不被系统覆盖且移除监听", async () => {
  const listeners = new Set<() => void>();
  const media = { matches: false, addEventListener: vi.fn((_name: string, callback: () => void) => listeners.add(callback)), removeEventListener: vi.fn((_name: string, callback: () => void) => listeners.delete(callback)) };
  vi.stubGlobal("matchMedia", vi.fn(() => media));
  const { unmount } = render(<ThemeSelect />);
  act(() => { media.matches = true; listeners.forEach(listener => listener()); });
  expect(document.documentElement).toHaveClass("dark");
  const user = userEvent.setup();
  await user.click(screen.getByRole("combobox", { name: "主题" }));
  await user.click(screen.getByRole("option", { name: "浅色" }));
  expect(localStorage.getItem(themeKey)).toBe("light");
  act(() => listeners.forEach(listener => listener()));
  expect(document.documentElement).not.toHaveClass("dark");
  unmount(); expect(listeners.size).toBe(0); vi.unstubAllGlobals();
});
it("首帧有效主题及写入失败可见", async () => {
  applyTheme("dark"); expect(document.documentElement).toHaveClass("dark");
  vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => { throw new Error("quota"); });
  render(<ThemeSelect />);
  fireEvent.keyDown(screen.getByRole("combobox"), { key: "ArrowDown" });
  await userEvent.click(await screen.findByRole("option", { name: "深色" }));
  expect(screen.getByRole("status")).toHaveTextContent("主题偏好未保存");
});
