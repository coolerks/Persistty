import { createRef } from "react";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, expect, it, vi } from "vitest";
import { AuthContext } from "@/features/auth/auth-context";
import { api, ApiError } from "@/lib/api/client";
import type { FileVersion, Project } from "@/lib/api/decoder";
import { UploadControls } from "./UploadControls";

const project: Project = { id: "project1", name: "项目", version: 1, main_folder_id: "folder1", folders: [{ id: "folder1", path: "/home/test/project" }] };
const version: FileVersion = { mtime: "2026-09-28T00:00:00Z", size: 0, etag: "old-hash", identity: "1:1" };
const nextVersion: FileVersion = { ...version, etag: "newer-hash" };
const state = { id: "upload1", status: "pending" as const, size: 0, chunk_bytes: 4194304, received: [], expires_at: "2026-09-29T00:00:00Z", result: null };

afterEach(() => { vi.restoreAllMocks(); localStorage.clear(); });

function mount(containerRef = createRef<HTMLDivElement>()) {
  const onChanged = vi.fn();
  render(<AuthContext.Provider value={{ state: { status: "authenticated", session: { authenticated: true, expires_at: "2026-10-01T00:00:00Z", csrf_token: "csrf" } }, login: async () => undefined, logout: async () => undefined, expire: () => undefined, retry: () => undefined }}>
    <div ref={containerRef}><UploadControls project={project} folderId="folder1" path="" containerRef={containerRef} onChanged={onChanged} /></div>
  </AuthContext.Provider>);
  return onChanged;
}

it("替换沿用弹窗出现时的版本，变更后拒绝旧确认并显示错误", async () => {
  const create = vi.spyOn(api, "createUpload").mockResolvedValue(state);
  vi.spyOn(api, "completeUpload").mockRejectedValue(new ApiError(409, "conflict", "文件已变化", "test", null));
  const cancel = vi.spyOn(api, "cancelUpload").mockResolvedValue(undefined);
  const metadata = vi.spyOn(api, "metadata").mockResolvedValueOnce({ kind: "file", version }).mockResolvedValue({ kind: "file", version: nextVersion });
  mount();
  const user = userEvent.setup();
  await user.upload(screen.getByLabelText("选择上传文件"), new File([], "note.txt"));
  expect(await screen.findByRole("dialog", { name: "文件已存在" })).toHaveTextContent("note.txt");
  await user.click(screen.getByRole("button", { name: "替换" }));
  await waitFor(() => expect(create).toHaveBeenCalledTimes(2));
  expect(create.mock.calls[1]?.[0].expected_version).toEqual(version);
  await waitFor(() => expect(metadata).toHaveBeenCalledTimes(2));
  expect(await screen.findByRole("alert")).toHaveTextContent("服务器文件再次变化");
  expect(screen.getByRole("dialog", { name: "文件已存在" })).toBeInTheDocument();
  expect(cancel).toHaveBeenCalledTimes(2);
  await user.click(screen.getByRole("button", { name: "替换" }));
  await waitFor(() => expect(create).toHaveBeenCalledTimes(3));
  expect(create.mock.calls[2]?.[0].expected_version).toEqual(nextVersion);
});

it("拖入空目录仍创建目录", async () => {
  const create = vi.spyOn(api, "fileOperation").mockResolvedValue({ state: "applied", source_removed: false, target_created: true, failure_code: "" });
  const ref = createRef<HTMLDivElement>();
  mount(ref);
  const entry = {
    name: "empty", isDirectory: true, isFile: false,
    createReader: () => ({ readEntries: (resolve: (entries: FileSystemEntry[]) => void) => resolve([]) }),
  } as unknown as FileSystemEntry;
  fireEvent.drop(ref.current!, { dataTransfer: { types: ["Files"], items: [{ webkitGetAsEntry: () => entry }], files: [] } });
  await waitFor(() => expect(create).toHaveBeenCalledWith("project1", { kind: "create_directory", project_version: 1, target_folder_id: "folder1", target_path: "empty" }, "csrf", expect.any(AbortSignal)));
});

it("已存在的上传目录即使位于后续目录页也继续上传", async () => {
  vi.spyOn(api, "fileOperation").mockRejectedValue(new ApiError(409, "conflict", "已存在", "test", null));
  const entries = vi.spyOn(api, "entries")
    .mockResolvedValueOnce({ items: [], next_cursor: "next", project_version: 1 })
    .mockResolvedValueOnce({ items: [{ name: "a", kind: "directory", size: 0, mtime: "2026-09-28T00:00:00Z", identity: "1:2" }], next_cursor: "", project_version: 1 });
  const create = vi.spyOn(api, "createUpload").mockResolvedValue(state);
  vi.spyOn(api, "completeUpload").mockResolvedValue({ state: "uploaded", version });
  const onChanged = mount();
  const file = new File([], "note.txt");
  Object.defineProperty(file, "webkitRelativePath", { value: "a/note.txt" });
  fireEvent.change(screen.getByLabelText("选择上传文件夹"), { target: { files: [file] } });
  await waitFor(() => expect(create).toHaveBeenCalledTimes(1));
  expect(entries.mock.calls.map(call => call[4])).toEqual(["", "next"]);
  expect(create.mock.calls[0]?.[0].path).toBe("a/note.txt");
  await waitFor(() => expect(onChanged).toHaveBeenCalled());
});
