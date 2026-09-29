import { useEffect, useRef, useState } from "react";
import { sha256 } from "@noble/hashes/sha2.js";
import { FolderUp, Upload, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { api, ApiError, errorMessage } from "@/lib/api/client";
import type { FileVersion, Project } from "@/lib/api/decoder";
import { useAuth } from "@/features/auth/auth-context";

type UploadFile = { file: File; path: string; hash?: string };
type UploadConflict = UploadFile & { version: FileVersion };
type Selection = { files: UploadFile[]; directories: string[] };
function join(parent: string, name: string): string { return parent ? `${parent}/${name}` : name; }
function hex(bytes: Uint8Array): string { return Array.from(bytes, byte => byte.toString(16).padStart(2, "0")).join(""); }
function batchID(): string { const bytes = new Uint8Array(16); crypto.getRandomValues(bytes); return hex(bytes); }
function stored(key: string): string | null { try { return localStorage.getItem(key); } catch { return null; } }
function remember(key: string, value: string | null) { try { if (value === null) localStorage.removeItem(key); else localStorage.setItem(key, value); } catch { /* Storage can be unavailable in private mode. */ } }
async function digestFile(file: File, signal: AbortSignal): Promise<string> {
  const hash = sha256.create();
  for (let offset = 0; offset < file.size; offset += 4 << 20) {
    if (signal.aborted) throw new DOMException("已取消", "AbortError");
    hash.update(new Uint8Array(await file.slice(offset, offset + (4 << 20)).arrayBuffer()));
  }
  return hex(hash.digest());
}

async function walk(entry: FileSystemEntry, prefix: string, selection: Selection): Promise<void> {
  const path = join(prefix, entry.name);
  if (entry.isFile) {
    const file = await new Promise<File>((resolve, reject) => (entry as FileSystemFileEntry).file(resolve, reject));
    selection.files.push({ file, path });
    return;
  }
  if (!entry.isDirectory) return;
  selection.directories.push(path);
  const reader = (entry as FileSystemDirectoryEntry).createReader();
  for (;;) {
    const entries = await new Promise<FileSystemEntry[]>((resolve, reject) => reader.readEntries(resolve, reject));
    if (entries.length === 0) break;
    for (const child of entries) await walk(child, path, selection);
  }
}

export function UploadControls({ project, folderId, path, containerRef, onChanged }: { project: Project; folderId: string; path: string; containerRef: React.RefObject<HTMLDivElement | null>; onChanged(): void }) {
  const auth = useAuth();
  const fileInput = useRef<HTMLInputElement>(null);
  const folderInput = useRef<HTMLInputElement>(null);
  const controller = useRef<AbortController | null>(null);
  const active = useRef<{ id: string; key: string } | null>(null);
  const [busy, setBusy] = useState(false);
  const [progress, setProgress] = useState(0);
  const [label, setLabel] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [conflicts, setConflicts] = useState<UploadConflict[]>([]);
  const csrf = auth.state.status === "authenticated" ? auth.state.session.csrf_token : "";
  useEffect(() => () => controller.current?.abort(), []);

  async function ensureDirectory(relative: string, signal: AbortSignal) {
    try {
      await api.fileOperation(project.id, { kind: "create_directory", project_version: project.version, target_folder_id: folderId, target_path: relative }, csrf, signal);
      onChanged();
    } catch (cause: unknown) {
      if (!(cause instanceof ApiError) || cause.status !== 409) throw cause;
      const parent = relative.split("/").slice(0, -1).join("/");
      const name = relative.split("/").at(-1);
      let cursor = "";
      for (;;) {
        const listing = await api.entries(project.id, folderId, project.version, parent, cursor, signal);
        if (listing.items.some(item => item.name === name && item.kind === "directory")) return;
        if (!listing.next_cursor) throw cause;
        cursor = listing.next_cursor;
      }
    }
  }

  async function sendFile(item: UploadFile, batchId: string, signal: AbortSignal, expected?: FileVersion) {
    const hash = item.hash ?? await digestFile(item.file, signal);
    const key = `persistty:upload:${project.id}:${folderId}:${project.version}:${item.path}:${item.file.size}:${hash}:${expected?.etag ?? "new"}`;
    let id = stored(key);
    let state = id ? await api.uploadStatus(id, signal).catch(() => null) : null;
    if (!state) {
      remember(key, null);
      state = await api.createUpload({ project_id: project.id, folder_id: folderId, project_version: project.version, path: item.path, batch_id: batchId, size: item.file.size, sha256: hash, ...(expected ? { expected_version: expected } : {}) }, csrf, signal);
      id = state.id; remember(key, id);
    }
    active.current = { id: state.id, key };
    const received = new Set(state.received);
    const total = Math.max(1, Math.ceil(item.file.size / state.chunk_bytes));
    for (let index = 0; index < Math.ceil(item.file.size / state.chunk_bytes); index++) {
      if (signal.aborted) throw new DOMException("已取消", "AbortError");
      if (!received.has(index)) {
        const bytes = await item.file.slice(index * state.chunk_bytes, (index + 1) * state.chunk_bytes).arrayBuffer();
        await api.uploadChunk(state.id, index, bytes, hex(sha256(new Uint8Array(bytes))), csrf, signal);
      }
      setProgress(Math.round(((index + 1) / total) * 100));
    }
    try {
      const result = await api.completeUpload(state.id, csrf, signal);
      remember(key, null); active.current = null;
      setLabel(`${item.path}：${result.state === "skipped" ? "内容相同，已跳过" : "上传完成"}`);
      onChanged();
    } catch (cause: unknown) {
      if (cause instanceof ApiError && cause.status === 409) {
        await api.cancelUpload(state.id, csrf, new AbortController().signal).catch(() => undefined);
        remember(key, null); active.current = null;
        if (!expected) {
          const current = await api.metadata(project.id, folderId, project.version, item.path, signal);
          setConflicts(existing => [...existing, { ...item, hash, version: current.version }]);
          return;
        }
      }
      throw cause;
    }
  }

  async function upload(selection: Selection) {
    if (busy || !csrf) return;
    const signal = new AbortController(); controller.current = signal;
    setBusy(true); setError(null); setProgress(0); setConflicts([]);
    const batchId = batchID();
    try {
      const directories = new Set(selection.directories);
      for (const item of selection.files) {
        const parts = item.path.split("/");
        for (let index = 1; index < parts.length; index++) directories.add(parts.slice(0, index).join("/"));
      }
      for (const directory of [...directories].sort((a, b) => a.split("/").length - b.split("/").length)) {
        if (signal.signal.aborted) throw new DOMException("已取消", "AbortError");
        setLabel(`建立文件夹 ${directory}`);
        await ensureDirectory(directory, signal.signal);
      }
      for (const item of selection.files) {
        if (signal.signal.aborted) throw new DOMException("已取消", "AbortError");
        setLabel(`上传 ${item.path}`); setProgress(0);
        await sendFile(item, batchId, signal.signal);
      }
    } catch (cause: unknown) { if (!signal.signal.aborted) setError(errorMessage(cause)); }
    finally { controller.current = null; setBusy(false); }
  }

  function fromInput(list: FileList | null, directory: boolean) {
    if (!list) return;
    const files = Array.from(list, file => ({ file, path: join(path, directory && file.webkitRelativePath ? file.webkitRelativePath : file.name) }));
    void upload({ files, directories: [] });
  }
  async function fromDrop(event: DragEvent) {
    const transfer = event.dataTransfer;
    if (!transfer || !transfer.types.includes("Files") || transfer.types.includes("application/x-persistty-entry")) return;
    event.preventDefault();
    const selection: Selection = { files: [], directories: [] };
    const entries = Array.from(transfer.items, item => item.webkitGetAsEntry()).filter((entry): entry is FileSystemEntry => entry !== null);
    if (entries.length) {
      for (const entry of entries) await walk(entry, path, selection);
    } else selection.files = Array.from(transfer.files, file => ({ file, path: join(path, file.name) }));
    void upload(selection);
  }
  useEffect(() => {
    const element = containerRef.current;
    if (!element) return;
    function dragOver(event: DragEvent) {
      if (event.dataTransfer?.types.includes("Files") && !event.dataTransfer.types.includes("application/x-persistty-entry")) event.preventDefault();
    }
    function drop(event: DragEvent) { void fromDrop(event).catch(cause => setError(errorMessage(cause))); }
    element.addEventListener("dragover", dragOver);
    element.addEventListener("drop", drop);
    return () => { element.removeEventListener("dragover", dragOver); element.removeEventListener("drop", drop); };
  });
  async function cancel() {
    controller.current?.abort();
    if (active.current) {
      const current = active.current; active.current = null;
      try { await api.cancelUpload(current.id, csrf, new AbortController().signal); remember(current.key, null); }
      catch (cause: unknown) { setError(errorMessage(cause)); }
    }
    setBusy(false); setLabel("上传已取消");
  }
  async function replace() {
    const item = conflicts[0];
    if (!item || busy) return;
    const signal = new AbortController(); controller.current = signal;
    setBusy(true); setError(null); setLabel(`替换 ${item.path}`); setProgress(0);
    try {
      await sendFile(item, batchID(), signal.signal, item.version);
      setConflicts(existing => existing.slice(1));
    } catch (cause: unknown) {
      if (!signal.signal.aborted) {
        if (cause instanceof ApiError && cause.status === 409) {
          try {
            const current = await api.metadata(project.id, folderId, project.version, item.path, signal.signal);
            setConflicts(existing => existing.map((conflict, index) => index === 0 ? { ...conflict, version: current.version } : conflict));
            setError("服务器文件再次变化，请检查后重新确认替换。");
          } catch (metadataError: unknown) { setError(errorMessage(metadataError)); }
        } else setError(errorMessage(cause));
      }
    }
    finally { controller.current = null; setBusy(false); }
  }
  return <div className="flex flex-wrap items-center gap-1">
    <input ref={fileInput} className="sr-only" type="file" multiple aria-label="选择上传文件" onChange={event => { fromInput(event.target.files, false); event.target.value = ""; }} />
    <input ref={folderInput} className="sr-only" type="file" multiple {...{ webkitdirectory: "" }} aria-label="选择上传文件夹" onChange={event => { fromInput(event.target.files, true); event.target.value = ""; }} />
    <Button size="icon" variant="ghost" title="上传文件" aria-label="上传文件" disabled={busy} onClick={() => fileInput.current?.click()}><Upload /></Button>
    <Button size="icon" variant="ghost" className="hidden md:inline-flex" title="上传文件夹" aria-label="上传文件夹" disabled={busy} onClick={() => folderInput.current?.click()}><FolderUp /></Button>
    {busy && <Button size="icon" variant="ghost" title="取消上传" aria-label="取消上传" onClick={() => void cancel()}><X /></Button>}
    {(busy || label) && <span className="max-w-56 truncate text-xs text-muted-foreground" role="status" title={label}>{label}{busy ? ` ${progress}%` : ""}</span>}
    {error && conflicts.length === 0 && <p role="alert" className="w-full text-xs text-destructive">{error}</p>}
    <Dialog open={conflicts.length > 0} onOpenChange={open => { if (!open && !busy) { setError(null); setConflicts(current => current.slice(1)); } }}><DialogContent><DialogHeader><DialogTitle>文件已存在</DialogTitle><DialogDescription>{conflicts[0]?.path} 与服务器上的文件内容不同。是否替换？仅这一个文件有效。</DialogDescription></DialogHeader>{error && <p role="alert" className="text-sm text-destructive">{error}</p>}<DialogFooter><Button variant="outline" disabled={busy} onClick={() => { setError(null); setConflicts(current => current.slice(1)); }}>跳过</Button><Button disabled={busy} onClick={() => void replace()}>替换</Button></DialogFooter></DialogContent></Dialog>
  </div>;
}
