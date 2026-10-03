import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, expect, test, vi } from "vitest";
import { FileQuickOpen } from "./FileQuickOpen";
import { searchGitAPI } from "@/lib/api/search-git-client";
import type { FileNames } from "@/lib/api/search-git-decoder";
const project = { id: "p", name: "项目", version: 1, main_folder_id: "f", folders: [{ id: "f", path: "/fixture" }, { id: "other", path: "/other" }] };
afterEach(() => { cleanup(); vi.useRealTimers(); vi.restoreAllMocks(); });
test("同版本项目轮询不取消文件名搜索，配置版本变化仍取消", async () => {
  let signal: AbortSignal | undefined;
  let finish: (data: FileNames) => void = () => {};
  const load = vi.spyOn(searchGitAPI, "fileNames").mockImplementation((_p, _v, _q, pending) => {
    signal = pending; return new Promise(resolve => { finish = resolve; });
  });
  const onOpen = vi.fn();
  const view = render(<FileQuickOpen project={project} onOpen={onOpen} />);
  await userEvent.click(screen.getByRole("button", { name: "按名称搜索文件" }));
  vi.useFakeTimers();
  fireEvent.change(screen.getByRole("textbox", { name: "文件名关键词" }), { target: { value: "project" } });
  await act(() => vi.advanceTimersByTimeAsync(250));
  view.rerender(<FileQuickOpen project={{ ...project, folders: project.folders.map(folder => ({ ...folder })) }} onOpen={onOpen} />);
  await act(() => vi.advanceTimersByTimeAsync(16000));
  expect(signal?.aborted).toBe(false); expect(load).toHaveBeenCalledTimes(1);
  await act(async () => finish({ project_version: 1, items: [{ folder_id: "f", path: "project.go" }], truncated: false }));
  expect(screen.getByRole("button", { name: "打开 project.go" })).toBeInTheDocument();
  view.rerender(<FileQuickOpen project={{ ...project, version: 2 }} onOpen={onOpen} />);
  expect(signal?.aborted).toBe(true);
});
test("防抖取消旧请求，迟到结果不覆盖新关键词；键盘打开第二根文件", async () => {
  let finish: (data: FileNames) => void = () => {}; let first: AbortSignal | undefined;
  const api = vi.spyOn(searchGitAPI, "fileNames").mockImplementation((_p, _v, query, signal) => {
    if (query === "old") { first = signal; return new Promise(resolve => { finish = resolve; }); }
    return Promise.resolve({ project_version: 1, items: [{ folder_id: "other", path: "新文件.ts" }], truncated: false });
  });
  const onOpen = vi.fn(); render(<FileQuickOpen project={project} onOpen={onOpen} />);
  await userEvent.click(screen.getByRole("button", { name: "按名称搜索文件" }));
  const input = screen.getByRole("textbox", { name: "文件名关键词" });
  vi.useFakeTimers(); fireEvent.change(input, { target: { value: "old" } });
  await act(() => vi.advanceTimersByTimeAsync(249)); expect(api).not.toHaveBeenCalled();
  await act(() => vi.advanceTimersByTimeAsync(1)); expect(api).toHaveBeenCalledTimes(1);
  fireEvent.change(input, { target: { value: "新" } }); expect(first?.aborted).toBe(true);
  await act(() => vi.advanceTimersByTimeAsync(250));
  await act(async () => finish({ project_version: 1, items: [{ folder_id: "f", path: "obsolete.ts" }], truncated: false }));
  expect(screen.queryByText("obsolete.ts")).not.toBeInTheDocument();
  expect(screen.getByRole("button", { name: "打开 新文件.ts" })).toBeInTheDocument();
  fireEvent.keyDown(input, { key: "Enter" }); expect(onOpen).toHaveBeenCalledWith({ folderId: "other", path: "新文件.ts" });
});
test("快捷键、超长关键词与项目版本漂移可见，关闭取消请求", async () => {
  const load = vi.spyOn(searchGitAPI, "fileNames").mockResolvedValue({ project_version: 2, items: [], truncated: false });
  render(<FileQuickOpen project={project} onOpen={vi.fn()} />);
  fireEvent.keyDown(window, { key: "p", ctrlKey: true });
  const input = await screen.findByRole("textbox", { name: "文件名关键词" });
  vi.useFakeTimers(); fireEvent.change(input, { target: { value: "中".repeat(90) } });
  await act(() => vi.advanceTimersByTimeAsync(300)); expect(load).not.toHaveBeenCalled(); expect(screen.getByRole("alert")).toHaveTextContent("关键词过长");
  fireEvent.change(input, { target: { value: "new" } }); await act(() => vi.advanceTimersByTimeAsync(250));
  expect(screen.getByRole("alert")).toHaveTextContent("项目配置已变化");
  fireEvent.change(input, { target: { value: "cancelled" } });
  fireEvent.keyDown(input, { key: "Escape" }); await act(() => vi.advanceTimersByTimeAsync(500)); expect(load).toHaveBeenCalledTimes(1);
});
