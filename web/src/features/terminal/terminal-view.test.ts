import { beforeEach, expect, it } from "vitest";
import { terminalVisible, useTerminalView } from "./terminal-view";
import type { Terminal } from "@/lib/api/decoder";

const terminal: Terminal = { id: "ended", project_id: "p", display_name: "已结束", working_directory: "/test", state: "terminated" };
beforeEach(() => useTerminalView.setState({ dismissed: {} }));
it("只关闭已结束的浏览器标签并持久化，running/unavailable 不受旧隐藏状态影响", async () => {
  useTerminalView.getState().dismiss(terminal);
  expect(terminalVisible(terminal, useTerminalView.getState().dismissed)).toBe(false);
  const saved = localStorage.getItem("persistty.terminal-view.v1");
  expect(saved).not.toBeNull();
  useTerminalView.setState({ dismissed: {} });
  // Rehydrate from the browser preference saved by dismiss, without another write.
  if (saved) localStorage.setItem("persistty.terminal-view.v1", saved);
  await useTerminalView.persist.rehydrate();
  expect(terminalVisible(terminal, useTerminalView.getState().dismissed)).toBe(false);
  for (const state of ["running", "unavailable"] as const) {
    const item = { ...terminal, id: state, state }; useTerminalView.getState().dismiss(item);
    expect(useTerminalView.getState().dismissed[state]).toBeUndefined();
    expect(terminalVisible({ ...item, id: "ended" }, useTerminalView.getState().dismissed)).toBe(true);
  }
});
