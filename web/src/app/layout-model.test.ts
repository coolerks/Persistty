import { expect, it, expectTypeOf } from "vitest";
import { acceptsTab, layoutStorageKey, tabCloseCommand, type DesktopLayout, type WorkbenchCommand } from "./layout-model";
import { safeReturnPath } from "@/features/auth/return-path";

it("区分终端终止、文件关闭与面板折叠，不允许文件下移", () => {
  expect(tabCloseCommand({ kind: "terminal", id: "t" })).toEqual({ type: "request_terminal_termination", terminalId: "t" });
  expect(tabCloseCommand({ kind: "file", id: "f" })).toEqual({ type: "close_file_view", fileId: "f" });
  expect(acceptsTab("terminal", { kind: "file", id: "f" })).toBe(false);
  expect(acceptsTab("editor", { kind: "terminal", id: "t" })).toBe(true);
  expectTypeOf<DesktopLayout["orientation"]>().toEqualTypeOf<"horizontal">();
  expectTypeOf<{ type: "collapse_panel"; panel: "terminal" }>().toExtend<WorkbenchCommand>();
  expect(layoutStorageKey("p", "mobile")).not.toBe(layoutStorageKey("p", "desktop"));
});
it("登录返回仅允许明确内部资源地址", () => {
  expect(safeReturnPath("/projects/project-test")).toBe("/projects/project-test");
  expect(safeReturnPath("/terminals/terminal-test")).toBe("/terminals/terminal-test");
  for (const value of [null, "https://evil.test", "//evil.test", "/projects/../login", "/projects/%2f%2fevil", "/projects/p?return=https://evil.test", "/projects/p#test"]) expect(safeReturnPath(value)).toBe("/projects");
});
