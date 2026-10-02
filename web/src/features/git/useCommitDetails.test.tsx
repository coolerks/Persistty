import { act, cleanup, renderHook, waitFor } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";
import fixture from "../../../../tests/fixtures/search-git.json";
import { decodeDetail } from "@/lib/api/search-git-decoder";
import { searchGitAPI } from "@/lib/api/search-git-client";
import type { Project } from "@/lib/api/decoder";
import { useCommitDetails } from "./useCommitDetails";
const detail = decodeDetail(fixture.detail);
const project: Project = { id: "p", name: "Fixture", version: 1, main_folder_id: "f", folders: [{ id: "f", path: "/fixture" }] };
afterEach(() => { cleanup(); vi.restoreAllMocks(); });

test("悬停与展开复用有界内存缓存，刷新使缓存失效", async () => {
  const load = vi.spyOn(searchGitAPI, "detail").mockResolvedValue(detail);
  const view = renderHook(({ revision }) => useCommitDetails(project, "r", true, revision, false), { initialProps: { revision: 0 } });
  act(() => view.result.current.show(detail.commit.id, "", true));
  await waitFor(() => expect(view.result.current.preview?.detail).toEqual(detail));
  act(() => view.result.current.show(detail.commit.id, "", false));
  act(() => view.result.current.show(detail.commit.id, "", true));
  await waitFor(() => expect(view.result.current.preview?.detail).toEqual(detail));
  await act(async () => { expect(await view.result.current.read(detail.commit.id, "", new AbortController().signal)).toEqual(detail); });
  expect(load).toHaveBeenCalledTimes(1);
  view.rerender({ revision: 1 });
  act(() => view.result.current.show(detail.commit.id, "", true));
  await waitFor(() => expect(view.result.current.preview?.detail).toEqual(detail));
  expect(load).toHaveBeenCalledTimes(2);
});

test("旧悬停和旧项目的迟到响应不回填，新请求期间只有一个悬停读取", async () => {
  let finish: (value: typeof detail) => void = () => {}; let oldSignal: AbortSignal | undefined;
  const load = vi.spyOn(searchGitAPI, "detail").mockImplementation((_p, version, _repo, _commit, _parent, signal) => {
    if (version === 1) { oldSignal = signal; return new Promise(resolve => { finish = resolve; }); }
    return Promise.resolve({ ...detail, message: "新项目详情" });
  });
  const view = renderHook(({ version }) => useCommitDetails({ ...project, version }, "r", true, 0, false), { initialProps: { version: 1 } });
  act(() => view.result.current.show(detail.commit.id, "", true));
  await waitFor(() => expect(load).toHaveBeenCalledTimes(1));
  view.rerender({ version: 2 }); expect(oldSignal?.aborted).toBe(true);
  act(() => view.result.current.show(detail.commit.id, "", true));
  await waitFor(() => expect(view.result.current.preview?.detail?.message).toBe("新项目详情"));
  await act(async () => { finish(detail); });
  expect(view.result.current.preview?.detail?.message).toBe("新项目详情"); expect(load).toHaveBeenCalledTimes(2);
});

test("前台繁忙时等待，失败不自动重试，关闭后重开可重试", async () => {
  const load = vi.spyOn(searchGitAPI, "detail").mockRejectedValueOnce(new Error("失败")).mockResolvedValue(detail);
  const view = renderHook(({ busy }) => useCommitDetails(project, "r", true, 0, busy), { initialProps: { busy: true } });
  act(() => view.result.current.show(detail.commit.id, "", true)); expect(load).not.toHaveBeenCalled();
  view.rerender({ busy: false }); await waitFor(() => expect(view.result.current.preview?.error).toBeTruthy());
  view.rerender({ busy: true }); view.rerender({ busy: false }); expect(load).toHaveBeenCalledTimes(1);
  act(() => view.result.current.show(detail.commit.id, "", false)); act(() => view.result.current.show(detail.commit.id, "", true));
  await waitFor(() => expect(view.result.current.preview?.detail).toEqual(detail)); expect(load).toHaveBeenCalledTimes(2);
});

test("缓存最多八个条目，隐藏及卸载中止悬停请求", async () => {
  const load = vi.spyOn(searchGitAPI, "detail").mockResolvedValue(detail);
  const view = renderHook(({ visible }) => useCommitDetails(project, "r", visible, 0, false), { initialProps: { visible: true } });
  for (let i = 0; i < 9; i++) await act(async () => { await view.result.current.read(String(i), "", new AbortController().signal); });
  await act(async () => { await view.result.current.read("0", "", new AbortController().signal); }); expect(load).toHaveBeenCalledTimes(10);
  let signal: AbortSignal | undefined;
  load.mockImplementation((_p, _v, _r, _c, _parent, current) => { signal = current; return new Promise(() => {}); });
  act(() => view.result.current.show("uncached", "", true)); await waitFor(() => expect(signal).toBeDefined());
  view.rerender({ visible: false }); expect(signal?.aborted).toBe(true); expect(view.result.current.preview).toBeNull();
  view.rerender({ visible: true }); act(() => view.result.current.show("again", "", true));
  await waitFor(() => expect(signal?.aborted).toBe(false)); view.unmount(); expect(signal?.aborted).toBe(true);
});

test("超出缓存字节预算的合法详情仍可显示，但不驻留内存缓存", async () => {
  const files = Array.from({ length: 220 }, (_, i) => `${`${"a".repeat(100)}/`.repeat(30)}file-${i}.ts`);
  const large = decodeDetail({ ...detail, files, stats: files.map(path => ({ ...detail.stats[0], path })) });
  const load = vi.spyOn(searchGitAPI, "detail").mockResolvedValue(large);
  const view = renderHook(() => useCommitDetails(project, "r", true, 0, false));
  act(() => view.result.current.show("large", "", true));
  await waitFor(() => expect(view.result.current.preview?.detail?.files).toHaveLength(220));
  await act(async () => { expect((await view.result.current.read("large", "", new AbortController().signal)).files).toHaveLength(220); });
  expect(load).toHaveBeenCalledTimes(2);
});
