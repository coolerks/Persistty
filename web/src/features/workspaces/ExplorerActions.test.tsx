import type { ReactNode } from "react";
import { act, renderHook } from "@testing-library/react";
import { afterEach, expect, it, vi } from "vitest";
import { AuthContext } from "@/features/auth/auth-context";
import { api } from "@/lib/api/client";
import type { Project } from "@/lib/api/decoder";
import { useExplorerActions } from "./ExplorerActions";

const project: Project = { id: "project1", name: "项目", version: 1, main_folder_id: "folder1", folders: [{ id: "folder1", path: "/home/test/project" }, { id: "folder2", path: "/home/test/other" }] };
const source = { folderId: "folder1", path: "note.txt", kind: "file" as const, identity: "1:1" };

afterEach(() => vi.restoreAllMocks());

it("跨根移动部分成功只刷新，不把已打开文件重定位到目标", async () => {
  vi.spyOn(api, "metadata").mockResolvedValue({ kind: "file", version: { mtime: "2026-09-28T00:00:00Z", size: 4, etag: "old", identity: source.identity } });
  vi.spyOn(api, "fileOperation").mockResolvedValue({ state: "partial", source_removed: false, target_created: true, failure_code: "source_changed" });
  const onChanged = vi.fn();
  const wrapper = ({ children }: { children: ReactNode }) => <AuthContext.Provider value={{ state: { status: "authenticated", session: { authenticated: true, expires_at: "2026-10-01T00:00:00Z", csrf_token: "csrf" } }, login: async () => undefined, logout: async () => undefined, expire: () => undefined, retry: () => undefined }}>{children}</AuthContext.Provider>;
  const { result } = renderHook(() => useExplorerActions(project, onChanged), { wrapper });
  let succeeded = true;
  await act(async () => { succeeded = await result.current.operate(source, "folder2", "note.txt", "move"); });
  expect(succeeded).toBe(false);
  expect(onChanged).toHaveBeenCalledOnce();
  expect(onChanged).toHaveBeenCalledWith();
});
