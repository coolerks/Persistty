import { afterEach, expect, it, vi } from "vitest";
import { EditorScope, type EditorAPI } from "./editor-session";
import { ApiError } from "@/lib/api/client";
import type { ElevationAPI } from "@/lib/api/elevation-client";
import type { ElevationResult } from "@/lib/api/elevation-decoder";
import type { FileDraft, DraftStorage } from "./editor-drafts";
const id = "a".repeat(64), version = { identity: "1:2", etag: `sha256:${"b".repeat(64)}`, size: 3, mtime: "2026-10-02T00:00:00Z" };
const prepared = () => ({ id, state: "prepared" as const, target_id: "example", target_path: "/fixture/file.txt", content_hash: `sha256:${"c".repeat(64)}`, expires_at: new Date(Date.now() + 60000).toISOString() });
const applied: ElevationResult = { id, state: "applied", code: null, version: { ...version, identity: "1:3", etag: `sha256:${"d".repeat(64)}` } };
async function fixture() {
  const rows = new Map<string, FileDraft>();
  const storage: DraftStorage = { list: vi.fn(async () => [...rows.values()]), put: vi.fn(async row => { rows.set(row.id, row); }), remove: vi.fn(async (id, generation) => { const row = rows.get(id); if (row && row.generation <= generation) rows.delete(id); }) };
  const client: EditorAPI = { content: vi.fn(async () => ({ content: "old", kind: "text" as const, version })), metadata: vi.fn(), saveContent: vi.fn(async () => { throw new ApiError(403, "permission_denied", "权限不足", "request", null); }) };
  const elevation: ElevationAPI = { prepare: vi.fn(async () => prepared()), execute: vi.fn(async () => applied), cancel: vi.fn(async () => ({ id, state: "cancelled" as const, code: "cancelled", version: null })), status: vi.fn(async () => applied) };
  const scope = new EditorScope({ id: "p", name: "项目", version: 1, main_folder_id: "f", folders: [{ id: "f", path: "/fixture" }] }, storage, client, elevation); scope.csrf = "csrf";
  const buffer = scope.open({ folderId: "f", path: "file.txt" }); await vi.waitFor(() => expect(buffer.state.status).toBe("ready")); buffer.change("submitted"); await buffer.save();
  return { scope, buffer, rows, client, elevation };
}
afterEach(() => vi.useRealTimers());
it("权限入口只接受 permission_denied，关闭和过期零执行", async () => {
  const h = await fixture(); expect(h.buffer.canElevate).toBe(true); await h.buffer.beginElevation(); h.buffer.dismissElevation(); expect(h.elevation.cancel).toHaveBeenCalledOnce(); expect(h.elevation.execute).not.toHaveBeenCalled();
  await h.buffer.beginElevation(); h.scope.elevation!.request!.expires_at = new Date(0).toISOString(); await h.buffer.executeElevation("synthetic"); expect(h.scope.elevation?.result?.state).toBe("expired"); expect(h.elevation.execute).not.toHaveBeenCalled(); h.scope.dispose();
});
it("双击只提交冻结快照，期间新输入和草稿保留暂停", async () => {
  const h = await fixture(); let finish!: (value: ElevationResult) => void; vi.mocked(h.elevation.execute).mockReturnValueOnce(new Promise(resolve => { finish = resolve; }));
  await h.buffer.beginElevation(); const pending = h.buffer.executeElevation("synthetic-system-password"); await h.buffer.executeElevation("second"); h.buffer.change("new input"); finish(applied); await pending;
  expect(h.elevation.execute).toHaveBeenCalledOnce(); expect(h.elevation.execute).toHaveBeenCalledWith(id, "submitted", "synthetic-system-password", "csrf", expect.any(AbortSignal)); expect(h.buffer.state.base?.content).toBe("submitted"); expect(h.buffer.state.content).toBe("new input"); expect(h.buffer.state.saveState).toBe("paused"); expect(h.rows.get(h.buffer.draftId)?.content).toBe("new input");
  expect(Object.keys(h.scope.elevation!)).not.toContain("password"); expect(JSON.stringify([...h.rows.values()])).not.toContain("synthetic-system-password"); h.scope.dispose();
});
it("断线只查询状态，不重发密码或正文", async () => {
  const h = await fixture(); vi.mocked(h.elevation.execute).mockRejectedValueOnce(new Error("connection lost")); await h.buffer.beginElevation(); await h.buffer.executeElevation("synthetic"); expect(h.scope.elevation?.phase).toBe("unknown"); await h.buffer.executeElevation("retry"); await h.buffer.queryElevation(); expect(h.elevation.execute).toHaveBeenCalledOnce(); expect(h.elevation.status).toHaveBeenCalledOnce(); expect(h.buffer.dirty).toBe(false); h.scope.dispose();
});
it("配置更新丢弃迟到响应，取消尚未接受的执行", async () => {
  const h = await fixture(); let finish!: (value: ElevationResult) => void; vi.mocked(h.elevation.execute).mockReturnValueOnce(new Promise(resolve => { finish = resolve; })); await h.buffer.beginElevation(); const pending = h.buffer.executeElevation("synthetic"); await h.scope.configure({ ...h.scope.project, version: 2 }, "csrf"); finish(applied); await pending; expect(h.buffer.state.base?.content).toBe("old"); expect(h.buffer.state.content).toBe("submitted"); expect(h.scope.elevation).toBeNull(); expect(h.elevation.cancel).toHaveBeenCalledOnce(); h.scope.dispose();
});
it("提权冲突保留输入，比较前不可普通保存绕过", async () => {
  const h = await fixture(); vi.mocked(h.elevation.execute).mockResolvedValueOnce({ id, state: "rejected", code: "conflict", version: null }); await h.buffer.beginElevation(); await h.buffer.executeElevation("synthetic"); h.buffer.dismissElevation(); await h.buffer.save(true); expect(h.buffer.state.saveState).toBe("conflict"); expect(h.buffer.state.content).toBe("submitted"); expect(h.client.saveContent).toHaveBeenCalledOnce(); expect(h.rows.size).toBe(1); h.scope.dispose();
});
