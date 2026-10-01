import { decodeFileContent, type FileContent } from "@/lib/api/decoder";
import type { OpenFile } from "./workspace-view";

export type FileDraft = { schema: 1; id: string; projectId: string; file: OpenFile; sourceRoot: string; viewId: string; generation: number; base: FileContent; content: string; updatedAt: number };
export interface DraftStorage { list(projectId: string): Promise<FileDraft[]>; put(draft: FileDraft): Promise<void>; remove(id: string, generation: number): Promise<void> }
const MAX_BYTES = 64 * 1024 * 1024;
const MAX_ITEMS = 100;
function valid(value: unknown): value is FileDraft {
  if (!value || typeof value !== "object") return false;
  // Stored data is checked before presenting it; database entries are not protocol DTOs.
  const item = value as Partial<FileDraft>;
  try { decodeFileContent(item.base); } catch { return false; }
  return item.schema === 1 && typeof item.id === "string" && typeof item.projectId === "string" && typeof item.viewId === "string" && typeof item.sourceRoot === "string" && typeof item.content === "string" && typeof item.updatedAt === "number" && Number.isFinite(item.updatedAt) && item.updatedAt >= 0 && typeof item.generation === "number" && item.generation >= 0 && Number.isSafeInteger(item.generation) && !!item.file && typeof item.file.folderId === "string" && typeof item.file.path === "string" && !!item.base && typeof item.base.content === "string" && item.base.kind === "text" && !!item.base.version && typeof item.base.version.etag === "string" && typeof item.base.version.identity === "string" && typeof item.base.version.mtime === "string" && typeof item.base.version.size === "number";
}
export class IndexedDraftStorage implements DraftStorage {
  private db: Promise<IDBDatabase> | undefined;
  private database(): Promise<IDBDatabase> {
    this.db ??= new Promise((resolve, reject) => {
      if (!globalThis.indexedDB) { reject(new Error("此浏览器无法使用本地草稿存储。")); return; }
      const request = indexedDB.open("persistty.editor-drafts", 1);
      request.onupgradeneeded = () => { request.result.createObjectStore("drafts", { keyPath: "id" }); };
      request.onerror = () => { this.db = undefined; reject(new Error("无法打开本地草稿存储。")); };
      request.onblocked = () => { this.db = undefined; reject(new Error("本地草稿存储升级被其他标签页阻止。")); };
      request.onsuccess = () => { request.result.onversionchange = () => { request.result.close(); this.db = undefined; }; resolve(request.result); };
    });
    return this.db;
  }
  async list(projectId: string): Promise<FileDraft[]> {
    const db = await this.database();
    return new Promise((resolve, reject) => {
      const tx = db.transaction("drafts", "readonly"); const request = tx.objectStore("drafts").getAll();
      request.onsuccess = () => {
        const values: unknown[] = request.result;
        if (!values.every(valid)) { reject(new Error("本地草稿格式无法读取，未删除原有记录。")); return; }
        resolve(values.filter(item => item.projectId === projectId).sort((a, b) => b.updatedAt - a.updatedAt));
      };
      request.onerror = () => reject(new Error("无法读取本地草稿。"));
    });
  }
  async put(draft: FileDraft): Promise<void> {
    if (!valid(draft) || new TextEncoder().encode(draft.content).length > (8 << 20)) throw new Error("草稿超过单文件 8 MiB 上限，请导出内容。");
    const db = await this.database();
    return new Promise((resolve, reject) => {
      const tx = db.transaction("drafts", "readwrite"); const store = tx.objectStore("drafts"); const request = store.getAll();
      let quota = false;
      request.onsuccess = () => {
        const values: unknown[] = request.result;
        if (!values.every(valid)) { tx.abort(); return; }
        const entries = [...values.filter(item => item.id !== draft.id), draft];
        const bytes = entries.reduce((sum, item) => sum + new TextEncoder().encode(item.content).length + new TextEncoder().encode(item.base.content).length, 0);
        if (entries.length > MAX_ITEMS || bytes > MAX_BYTES) { quota = true; tx.abort(); return; }
        store.put(draft);
      };
      tx.oncomplete = () => resolve();
      tx.onabort = tx.onerror = () => reject(new Error(quota ? "本地草稿已达 64 MiB/100 条上限，请导出或清理草稿。" : "草稿未保存在本地，请保留页面或导出内容。"));
    });
  }
  async remove(id: string, generation: number): Promise<void> {
    const db = await this.database();
    return new Promise((resolve, reject) => {
      const tx = db.transaction("drafts", "readwrite"); const store = tx.objectStore("drafts"); const request = store.get(id);
      request.onsuccess = () => { const value: unknown = request.result; if (valid(value) && value.generation <= generation) store.delete(id); };
      tx.oncomplete = () => resolve(); tx.onabort = tx.onerror = () => reject(new Error("无法清理本地草稿，已保留记录。"));
    });
  }
}
export const draftStorage = new IndexedDraftStorage();
export function downloadDraft(content: string, path: string): void {
  const url = URL.createObjectURL(new Blob([content], { type: "text/plain;charset=utf-8" }));
  const link = document.createElement("a"); link.href = url; link.download = path.split("/").at(-1) ?? "draft.txt"; link.click(); setTimeout(() => URL.revokeObjectURL(url), 0);
}
