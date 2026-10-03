import { afterEach, expect, it, vi } from "vitest";
import { EditorScope, type EditorAPI } from "./editor-session";
import type { DraftStorage, FileDraft } from "./editor-drafts";
import type { FileContent, FileVersion, Project } from "@/lib/api/decoder";
import { ApiError } from "@/lib/api/client";
import { applyEditorChanges, applyEditorText, editorText } from "./editor-text";

const project: Project = { id: "p", name: "项目", version: 1, main_folder_id: "a", folders: [{ id: "a", path: "/a" }, { id: "b", path: "/b" }] };
const file = { folderId: "a", path: "file.txt" };
const version: FileVersion = { identity: "1:2", etag: "sha256:old", mtime: "2026-09-30T00:00:00Z", size: 3 };
const snapshot: FileContent = { content: "old", kind: "text", version };
function harness() {
  const rows = new Map<string, FileDraft>();
  const storage: DraftStorage = { list: vi.fn(async () => [...rows.values()]), put: vi.fn(async draft => { rows.set(draft.id, draft); }), remove: vi.fn(async (id, generation) => { const item = rows.get(id); if (item && item.generation <= generation) rows.delete(id); }) };
  const client: EditorAPI = { content: vi.fn(async () => snapshot), metadata: vi.fn(async () => ({ kind: "file" as const, version })), saveContent: vi.fn(async () => ({ version: { ...version, identity: "1:3", etag: "sha256:new" } })) };
  const scope = new EditorScope(project, storage, client); scope.csrf = "csrf";
  return { scope, client, storage, rows };
}
afterEach(() => { vi.useRealTimers(); vi.unstubAllGlobals(); });
it("局域网 HTTP 缺少 randomUUID 时初始化视图和文件，草稿保持独立身份", async () => {
  vi.stubGlobal("crypto", { getRandomValues: crypto.getRandomValues.bind(crypto) });
  const first = harness(), second = harness();
  try {
    const a = first.scope.open(file), b = second.scope.open(file);
    await vi.waitFor(() => expect(a.state.status).toBe("ready"));
    await vi.waitFor(() => expect(b.state.status).toBe("ready"));
    expect(first.scope.viewId).not.toBe(second.scope.viewId);
    expect(a.id).not.toBe(b.id);
    a.change("HTTP draft");
    expect(await first.scope.protect(file)).toBe(true);
    expect([...first.rows.values()]).toEqual([expect.objectContaining({ viewId: first.scope.viewId, content: "HTTP draft" })]);
    expect(second.rows.size).toBe(0);
  } finally { first.scope.dispose(); second.scope.dispose(); }
});
it("视图编辑保留 BOM、未改混合换行和末尾换行状态", () => {
  expect(editorText("\uFEFFa\r\nb\nc")).toBe("a\nb\nc");
  expect(applyEditorText("\uFEFFa\r\nb\nc", "a\nB\nc")).toBe("\uFEFFa\r\nB\nc");
  expect(applyEditorText("a\r\nb\r\n", "a\nb\nc\n")).toBe("a\r\nb\r\nc\r\n");
  expect(applyEditorText("a\rb", "A\nb")).toBe("A\rb");
});
it("默认一秒自动保存附基线与 CSRF，新输入不会被旧响应标为已保存", async () => {
  vi.useFakeTimers(); const h = harness();
  const buffer = h.scope.open(file); await vi.waitFor(() => expect(buffer.state.status).toBe("ready"));
  let finish: (value: { version: FileVersion }) => void = () => {};
  vi.mocked(h.client.saveContent).mockReturnValueOnce(new Promise(resolve => { finish = resolve; }));
  buffer.change("first"); await vi.advanceTimersByTimeAsync(1000);
  expect(h.client.saveContent).toHaveBeenCalledWith("p", "a", expect.objectContaining({ expected_version: version, content: "first" }), "csrf", expect.any(AbortSignal));
  buffer.change("second"); finish({ version: { ...version, etag: "sha256:first" } });
  await vi.waitFor(() => expect(buffer.state.base?.content).toBe("first"));
  expect(buffer.state.content).toBe("second"); expect(buffer.dirty).toBe(true); expect(buffer.state.saveState).toBe("pending");
  await vi.advanceTimersByTimeAsync(1000); expect(h.client.saveContent).toHaveBeenCalledTimes(2); h.scope.dispose();
});
it("409 保留两边内容且禁止去抖/普通保存重试绕过版本", async () => {
  const h = harness(); const b = h.scope.open(file); await vi.waitFor(() => expect(b.state.status).toBe("ready"));
  vi.mocked(h.client.saveContent).mockRejectedValue(new ApiError(409, "conflict", "冲突", "r", null));
  b.change("local"); await b.save(); expect(b.state.saveState).toBe("conflict"); expect(b.state.content).toBe("local");
  await b.save(true); expect(h.client.saveContent).toHaveBeenCalledOnce(); expect(h.rows.size).toBe(1); h.scope.dispose();
});
it("草稿恢复不自动保存且采用比较过的服务器版本", async () => {
  const h = harness(); const b = h.scope.open(file); await vi.waitFor(() => expect(b.state.status).toBe("ready"));
  const draft: FileDraft = { schema: 1, id: "other", projectId: "p", file, sourceRoot: "/a", viewId: "other", generation: 9, base: snapshot, content: "recovered", updatedAt: 0 };
  h.rows.set(draft.id, draft); b.restore(draft); await b.save(); expect(h.client.saveContent).not.toHaveBeenCalled();
  await b.save(true); expect(h.rows.has("other")).toBe(true); expect(b.state.saveState).toBe("saved"); h.scope.dispose();
});
it("草稿存储失败阻止保护成功，仍保留内存输入", async () => {
  const h = harness(); const b = h.scope.open(file); await vi.waitFor(() => expect(b.state.status).toBe("ready"));
  vi.mocked(h.storage.put).mockRejectedValue(new Error("quota")); b.change("important");
  expect(await h.scope.protect(file)).toBe(false); expect(b.state.content).toBe("important"); expect(b.state.draftError).toBe("quota"); h.scope.dispose();
});
it("重叠根相同真实 identity 共用保存调度", async () => {
  const h = harness(); const first = h.scope.open(file); await vi.waitFor(() => expect(first.state.status).toBe("ready"));
  const alias = { folderId: "b", path: "nested/file.txt" }; h.scope.open(alias);
  await vi.waitFor(() => expect(h.scope.open(alias)).toBe(first));
  first.change("shared"); await first.save(); expect(h.client.saveContent).toHaveBeenCalledOnce(); h.scope.dispose();
});
it("干净外部修改自动刷新，脏外部修改只标冲突并保留输入", async () => {
  const h = harness(); const b = h.scope.open(file); await vi.waitFor(() => expect(b.state.status).toBe("ready"));
  vi.mocked(h.client.content).mockResolvedValue({ ...snapshot, content: "disk", version: { ...version, etag: "new" } });
  await b.refresh(); expect(b.state.content).toBe("disk"); b.change("mine");
  vi.mocked(h.client.content).mockResolvedValue({ ...snapshot, content: "new disk", version: { ...version, etag: "newer" } });
  await b.refresh(); expect(b.state.content).toBe("mine"); expect(b.state.saveState).toBe("conflict"); expect(h.client.saveContent).not.toHaveBeenCalled(); h.scope.dispose();
});
it("scope 退出只保留草稿，不发写入或销毁真实资源", async () => {
  const h = harness(); const b = h.scope.open(file); await vi.waitFor(() => expect(b.state.status).toBe("ready"));
  b.change("offline"); h.scope.dispose(); await b.persist();
  expect(h.rows.size).toBe(1); expect(h.client.saveContent).not.toHaveBeenCalled();
});
it("重命名保护输入并迁移草稿，不恢复无关冲突的保存", async () => {
  const h = harness(); const b = h.scope.open(file); await vi.waitFor(() => expect(b.state.status).toBe("ready"));
  b.change("mine"); await h.scope.protect(file); await h.scope.relocate(file, { ...file, path: "renamed.txt" });
  expect(b.file.path).toBe("renamed.txt"); expect([...h.rows.values()][0]?.file.path).toBe("renamed.txt"); expect(b.state.content).toBe("mine");
  await b.save(); expect(h.client.saveContent).toHaveBeenCalledWith("p", "a", expect.objectContaining({ path: "renamed.txt", content: "mine" }), "csrf", expect.any(AbortSignal)); h.scope.dispose();
});
it("配置移除后由剩余覆盖根复验完整版本再重绑，输入暂停等待明确保存", async () => {
  const h = harness(); const b = h.scope.open(file); await vi.waitFor(() => expect(b.state.status).toBe("ready")); b.change("mine");
  await h.scope.configure({ ...project, version: 2, main_folder_id: "b", folders: [{ id: "b", path: "/" }] }, "csrf");
  expect(h.client.metadata).toHaveBeenCalledWith("p", "b", 2, "a/file.txt", expect.any(AbortSignal)); expect(b.file).toEqual({ folderId: "b", path: "a/file.txt" }); expect(b.state.saveState).toBe("paused"); await b.save(); expect(h.client.saveContent).not.toHaveBeenCalled(); h.scope.dispose();
});
it("旧 compare 响应在退出后不可推进版本或输入", async () => {
  const h = harness(); const b = h.scope.open(file); await vi.waitFor(() => expect(b.state.status).toBe("ready"));
  let finish: (value: FileContent) => void = () => {}; vi.mocked(h.client.content).mockReturnValueOnce(new Promise(resolve => { finish = resolve; })); const pending = b.compare(); h.scope.dispose(); finish({ ...snapshot, content: "late" }); await pending; expect(b.state.comparison).toBeNull();
});

it("Monaco 多光标同时修改保留跨编辑之间未修改的混合换行", () => {
  expect(applyEditorChanges("\uFEFFa\r\nb\nc\rd", [{ rangeOffset: 0, rangeLength: 1, text: "A" }, { rangeOffset: 6, rangeLength: 1, text: "D" }])).toBe("\uFEFFA\r\nb\nc\rD");
});

it("保存完成后迟到的外部读取不得倒退强版本或正文", async () => {
  const h = harness(); const b = h.scope.open(file); await vi.waitFor(() => expect(b.state.status).toBe("ready"));
  let finish: (value: FileContent) => void = () => {};
  vi.mocked(h.client.content).mockReturnValueOnce(new Promise(resolve => { finish = resolve; }));
  const refresh = b.refresh(); b.change("saved"); await b.save();
  finish(snapshot); await refresh;
  expect(b.state.content).toBe("saved"); expect(b.state.base?.version.etag).toBe("sha256:new"); expect(b.state.saveState).toBe("saved"); h.scope.dispose();
});

it("干净标签关闭释放缓存，重开重新读取并正常自动保存", async () => {
  const h = harness(); const first = h.scope.open(file); await vi.waitFor(() => expect(first.state.status).toBe("ready"));
  h.scope.closeInactive(); expect(h.scope.buffers.size).toBe(0);
  const reopened = h.scope.open(file); await vi.waitFor(() => expect(reopened.state.status).toBe("ready"));
  expect(reopened).not.toBe(first); reopened.change("reopened"); await reopened.save();
  expect(h.client.content).toHaveBeenCalledTimes(2); expect(h.client.saveContent).toHaveBeenCalledOnce(); h.scope.dispose();
});
it("替换仅保护目标干净缓冲区，新输入保留且不自动覆盖新磁盘基线", async () => {
  vi.useFakeTimers(); const h = harness(); const buffer = h.scope.open(file); await vi.waitFor(() => expect(buffer.state.status).toBe("ready"));
  const protection = h.scope.prepareReplacement([{ id: "target", folder_id: file.folderId, path: file.path, version }]);
  expect(protection.protectedIDs).toEqual([]); buffer.change("during replacement"); await buffer.save(true);
  expect(h.client.saveContent).not.toHaveBeenCalled();
  vi.mocked(h.client.content).mockResolvedValue({ ...snapshot, content: "replacement", version: { ...version, identity: "1:9", etag: "new" } });
  await protection.release(); await vi.advanceTimersByTimeAsync(2000);
  expect(buffer.state.content).toBe("during replacement"); expect(buffer.dirty).toBe(true); expect(buffer.state.saveState).toBe("paused"); expect(h.client.saveContent).not.toHaveBeenCalled(); expect(h.rows.size).toBe(1); h.scope.dispose();
});
it("替换跳过脏、恢复暂停及版本不同的缓冲区，干净文件刷新", async () => {
  const h = harness(), buffer = h.scope.open(file); await vi.waitFor(() => expect(buffer.state.status).toBe("ready"));
  const target = { id: "target", folder_id: file.folderId, path: file.path, version };
  buffer.change("dirty"); expect(h.scope.prepareReplacement([target]).protectedIDs).toEqual(["target"]);
  buffer.suspend(); buffer.update({ content: snapshot.content, saveState: "paused" }); expect(h.scope.prepareReplacement([target]).protectedIDs).toEqual(["target"]); buffer.resume();
  expect(h.scope.prepareReplacement([{ ...target, version: { ...version, etag: "other" } }]).protectedIDs).toEqual(["target"]);
  const protection = h.scope.prepareReplacement([target]); vi.mocked(h.client.content).mockResolvedValue({ ...snapshot, content: "replacement", version: { ...version, etag: "new" } }); await protection.release(); expect(buffer.state.content).toBe("replacement"); expect(buffer.state.saveState).toBe("saved"); h.scope.dispose();
});
