import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, expect, it, vi } from "vitest";
import { AuthContext } from "@/features/auth/auth-context";
import { api, ApiError } from "@/lib/api/client";
import type { FileEntry, Project } from "@/lib/api/decoder";
import { Explorer } from "./Explorer";

const project: Project = { id: "project1", name: "项目", version: 1, main_folder_id: "folder1", folders: [{ id: "folder1", path: "/home/test/project" }] };
const entry: FileEntry = { name: "note.txt", kind: "file", size: 4, mtime: "2026-09-28T00:00:00Z", identity: "1:1" };
const envelope = (data: unknown) => Response.json({ data, request_id: "test-request" });

afterEach(() => { vi.restoreAllMocks(); vi.unstubAllGlobals(); });
function mount() {
  return render(<AuthContext.Provider value={{ state: { status: "authenticated", session: { authenticated: true, expires_at: "2026-10-01T00:00:00Z", csrf_token: "csrf" } }, login: async () => undefined, logout: async () => undefined, expire: () => undefined, retry: () => undefined }}><Explorer project={project} onFile={() => undefined} /></AuthContext.Provider>);
}

it("新建文件调用受保护操作；删除预览取消不删除", async () => {
  vi.stubGlobal("WebSocket", class { close() { /* test socket */ } });
  const fetch = vi.fn((url: string, options: RequestInit) => {
    if (url.includes("/entries?")) return Promise.resolve(envelope({ items: [entry], next_cursor: "", project_version: 1 }));
    if (url.endsWith("/delete-preview")) return Promise.resolve(envelope({ path: "note.txt", kind: "file", count: 1, bytes: 4, token: "delete-token" }));
    if (url.endsWith("/file-operations")) return Promise.resolve(envelope({ state: "applied", source_removed: false, target_created: true, failure_code: "" }));
    return Promise.reject(new Error(`unexpected ${url} ${options.method}`));
  });
  vi.stubGlobal("fetch", fetch);
  mount();
  const user = userEvent.setup();
  await screen.findByRole("treeitem", { name: "note.txt" });
  await user.click(screen.getByRole("button", { name: "新建文件" }));
  await user.type(screen.getByRole("textbox", { name: "名称" }), "new.txt");
  await user.click(screen.getByRole("button", { name: "确定" }));
  await waitFor(() => expect(fetch.mock.calls.some(([url, options]) => url.endsWith("/file-operations") && JSON.parse(String(options.body)).target_path === "new.txt")).toBe(true));
  await user.click(await screen.findByRole("button", { name: "操作 note.txt" }));
  await user.click(await screen.findByRole("menuitem", { name: "永久删除" }));
  expect(await screen.findByRole("dialog", { name: "永久删除" })).toHaveTextContent("note.txt");
  await user.click(screen.getByRole("button", { name: "取消" }));
  expect(fetch.mock.calls.filter(([url, options]) => url.endsWith("/file-operations") && JSON.parse(String(options.body)).kind === "delete")).toHaveLength(0);
});

it("分页请求被重新扫描取消后仍可继续加载", async () => {
  class Socket {
    static current: Socket;
    onmessage: ((event: { data: string }) => void) | null = null;
    constructor() { Socket.current = this; }
    close() { /* test socket */ }
  }
  vi.stubGlobal("WebSocket", Socket);
  const page = { items: [entry], next_cursor: "next", project_version: 1 };
  const entries = vi.spyOn(api, "entries")
    .mockResolvedValueOnce(page)
    .mockImplementationOnce((_project, _folder, _version, _path, _cursor, signal) => new Promise((_resolve, reject) => {
      signal.addEventListener("abort", () => reject(new DOMException("已取消", "AbortError")));
    }))
    .mockResolvedValueOnce(page);
  mount();
  const user = userEvent.setup();
  await screen.findByRole("treeitem", { name: "note.txt" });
  await user.click(screen.getByRole("button", { name: "加载更多" }));
  expect(screen.getByRole("button", { name: "加载中" })).toBeDisabled();
  Socket.current.onmessage?.({ data: JSON.stringify({ project_id: project.id, folder_id: "folder1", rescan: true, mode: "watching" }) });
  await waitFor(() => expect(screen.getByRole("button", { name: "加载更多" })).toBeEnabled());
  expect(entries).toHaveBeenCalledTimes(3);
});

it("重命名冲突保留输入弹窗和错误，不误报成功", async () => {
  vi.stubGlobal("WebSocket", class { close() { /* test socket */ } });
  vi.spyOn(api, "entries").mockResolvedValue({ items: [entry], next_cursor: "", project_version: 1 });
  vi.spyOn(api, "metadata").mockResolvedValue({ kind: "file", version: { mtime: entry.mtime, size: 4, etag: "old", identity: entry.identity } });
  vi.spyOn(api, "fileOperation").mockRejectedValue(new ApiError(409, "conflict", "目标已存在", "test", null));
  mount();
  const user = userEvent.setup();
  await screen.findByRole("treeitem", { name: "note.txt" });
  await user.click(screen.getByRole("button", { name: "操作 note.txt" }));
  await user.click(await screen.findByRole("menuitem", { name: "重命名" }));
  await user.clear(screen.getByRole("textbox", { name: "名称" }));
  await user.type(screen.getByRole("textbox", { name: "名称" }), "other.txt");
  await user.click(screen.getByRole("button", { name: "确定" }));
  expect(await screen.findByRole("alert")).toHaveTextContent("目标已存在");
  expect(screen.getByRole("dialog", { name: "重命名" })).toBeInTheDocument();
  expect(screen.getByRole("textbox", { name: "名称" })).toHaveValue("other.txt");
});

it("目录分页游标失效时自动重新列举", async () => {
  vi.stubGlobal("WebSocket", class { close() { /* test socket */ } });
  const entries = vi.spyOn(api, "entries")
    .mockResolvedValueOnce({ items: [entry], next_cursor: "next", project_version: 1 })
    .mockRejectedValueOnce(new ApiError(409, "rescan_required", "目录已变化", "test", null))
    .mockResolvedValueOnce({ items: [entry], next_cursor: "", project_version: 1 });
  mount();
  await screen.findByRole("treeitem", { name: "note.txt" });
  await userEvent.setup().click(screen.getByRole("button", { name: "加载更多" }));
  await waitFor(() => expect(entries).toHaveBeenCalledTimes(3));
  expect(screen.queryByRole("button", { name: "加载更多" })).not.toBeInTheDocument();
  expect(screen.getByRole("treeitem", { name: "note.txt" })).toBeInTheDocument();
});

it("文件可拖入项目根目录移动", async () => {
  vi.stubGlobal("WebSocket", class { close() { /* test socket */ } });
  vi.spyOn(api, "entries").mockResolvedValue({ items: [entry], next_cursor: "", project_version: 1 });
  vi.spyOn(api, "metadata").mockResolvedValue({ kind: "file", version: { mtime: entry.mtime, size: 4, etag: "old", identity: entry.identity } });
  const operate = vi.spyOn(api, "fileOperation").mockResolvedValue({ state: "applied", source_removed: true, target_created: true, failure_code: "" });
  const { container } = mount();
  await screen.findByRole("treeitem", { name: "note.txt" });
  const title = container.querySelector(".tree-root-title");
  if (!title) throw new Error("项目根目录未渲染");
  fireEvent.drop(title, { dataTransfer: { types: ["application/x-persistty-entry"], getData: () => JSON.stringify({ folderId: "folder1", path: "docs/note.txt", kind: "file", identity: entry.identity }) } });
  await waitFor(() => expect(operate).toHaveBeenCalled());
  expect(operate.mock.calls[0]?.[1]).toMatchObject({ kind: "move", source_path: "docs/note.txt", target_path: "note.txt" });
});
