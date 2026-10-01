import { StrictMode } from "react";
import { act, cleanup, renderHook } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";
import { searchGitAPI } from "@/lib/api/search-git-client";
import type { Project } from "@/lib/api/decoder";
import { refreshGitBaselines } from "./baseline-requests";
import { useGitBaseline } from "./use-git-baseline";

const project: Project = { id: "project", name: "Fixture", version: 1, main_folder_id: "folder", folders: [{ id: "folder", path: "/fixture" }] };
const file = { folderId: "folder", path: "file.txt" };
const baseline = { state: "tracked" as const, repo_id: "repo", head: "a".repeat(40), content: "initial", version: null };
afterEach(() => { cleanup(); vi.restoreAllMocks(); vi.useRealTimers(); });

test("多组与 StrictMode 共享基线；静置无轮询，焦点有冷却，刷新可更新 HEAD", async () => {
  vi.useFakeTimers();
  const load = vi.spyOn(searchGitAPI, "baseline").mockResolvedValue(baseline);
  const first = renderHook(() => useGitBaseline(project, file, "etag", true), { wrapper: StrictMode });
  const second = renderHook(() => useGitBaseline(project, file, "etag", true));
  await act(async () => {});
  expect(load).toHaveBeenCalledTimes(1); expect(first.result.current?.content).toBe("initial"); expect(second.result.current?.content).toBe("initial");
  await act(async () => { window.dispatchEvent(new Event("focus")); window.dispatchEvent(new Event("online")); });
  expect(load).toHaveBeenCalledTimes(1);
  await act(async () => { await vi.advanceTimersByTimeAsync(120000); });
  expect(load).toHaveBeenCalledTimes(1);
  load.mockResolvedValue({ ...baseline, head: "b".repeat(40), content: "new HEAD" });
  await act(async () => { refreshGitBaselines(project.id); });
  expect(load).toHaveBeenCalledTimes(2); expect(first.result.current?.content).toBe("new HEAD"); expect(second.result.current?.content).toBe("new HEAD");
});

test("不同文件基线串行；卸载取消请求且旧响应不覆盖新身份", async () => {
  let finish: (value: typeof baseline) => void = () => {};
  let firstSignal: AbortSignal | undefined;
  const load = vi.spyOn(searchGitAPI, "baseline").mockImplementation((_p, _v, _f, _path, signal) => { firstSignal = signal; return new Promise(resolve => { finish = resolve; }); });
  const first = renderHook(() => useGitBaseline(project, file, "old", true));
  const second = renderHook(() => useGitBaseline(project, { ...file, path: "other.txt" }, "new", true));
  await act(async () => {}); expect(load).toHaveBeenCalledTimes(1);
  first.unmount(); expect(firstSignal?.aborted).toBe(true);
  load.mockResolvedValue({ ...baseline, content: "second" });
  await act(async () => { finish(baseline); });
  expect(load).toHaveBeenCalledTimes(2); expect(second.result.current?.content).toBe("second");
});

test("失败不自动重试或风暴式重试；文件保存版本变化触发一次新基线", async () => {
  vi.useFakeTimers();
  const load = vi.spyOn(searchGitAPI, "baseline").mockRejectedValue(new Error("capacity"));
  const view = renderHook(({ identity }) => useGitBaseline(project, file, identity, true), { initialProps: { identity: "one" } });
  await act(async () => {}); expect(view.result.current?.state).toBe("unavailable");
  await act(async () => { for (let i = 0; i < 5; i++) window.dispatchEvent(new Event("focus")); await vi.advanceTimersByTimeAsync(60000); });
  expect(load).toHaveBeenCalledTimes(1);
  load.mockResolvedValue(baseline); view.rerender({ identity: "two" });
  await act(async () => {}); expect(load).toHaveBeenCalledTimes(2); expect(view.result.current?.state).toBe("tracked");
});
