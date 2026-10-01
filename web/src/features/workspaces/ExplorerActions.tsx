import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { api, errorMessage } from "@/lib/api/client";
import type { DeletePreview, FileEntry, Project } from "@/lib/api/decoder";
import { useAuth } from "@/features/auth/auth-context";

export type EntryRef = { folderId: string; path: string; kind: FileEntry["kind"]; identity: string };
export type ClipboardEntry = { source: EntryRef; cut: boolean };
import { editorScopes } from "./editor-session";

type Naming = { kind: "create_file" | "create_directory" | "rename"; folderId: string; parent: string; source?: EntryRef; name: string };

function join(parent: string, name: string): string { return parent ? `${parent}/${name}` : name; }
function parentOf(path: string): string { return path.split("/").slice(0, -1).join("/"); }
function basename(path: string): string { return path.split("/").at(-1) ?? path; }
function validName(name: string): boolean { return name.length > 0 && name !== "." && name !== ".." && !name.includes("/") && !name.includes("\0") && new TextEncoder().encode(name).length <= 255; }

async function copyText(value: string): Promise<void> {
  if (navigator.clipboard?.writeText) { await navigator.clipboard.writeText(value); return; }
  const field = document.createElement("textarea");
  field.value = value; field.style.position = "fixed"; field.style.opacity = "0";
  document.body.append(field); field.select();
  const copied = document.execCommand("copy"); field.remove();
  if (!copied) throw new Error("浏览器无法复制路径。");
}

export function useExplorerActions(project: Project, onChanged: (source?: EntryRef, target?: EntryRef, kind?: "rename" | "copy" | "move" | "delete") => void) {
  const auth = useAuth();
  const [clipboard, setClipboard] = useState<ClipboardEntry | null>(null);
  const [naming, setNaming] = useState<Naming | null>(null);
  const [deleting, setDeleting] = useState<{ source: EntryRef; preview: DeletePreview } | null>(null);
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const csrf = auth.state.status === "authenticated" ? auth.state.session.csrf_token : "";

  async function operate(source: EntryRef, folderId: string, targetPath: string, kind: "rename" | "copy" | "move"): Promise<boolean> {
    if (pending || !csrf) return false;
    setPending(true); setError(null);
    try {
      const scope = editorScopes.get(project.id);
      if (kind !== "copy" && scope && !await scope.protect(source)) throw new Error("草稿保护失败，请先导出未保存输入。");
      const expected_version = source.kind === "file" ? (await api.metadata(project.id, source.folderId, project.version, source.path, new AbortController().signal)).version : undefined;
      const result = await api.fileOperation(project.id, { kind, project_version: project.version, source_folder_id: source.folderId, source_path: source.path, target_folder_id: folderId, target_path: targetPath, expected_identity: source.identity, ...(expected_version ? { expected_version } : {}) }, csrf, new AbortController().signal);
      if (result.state === "partial") {
        onChanged();
        setError("操作仅部分完成，请刷新源和目标目录后检查磁盘状态。");
        return false;
      }
      const target: EntryRef = { ...source, folderId, path: targetPath };
      if (kind !== "copy") await scope?.relocate(source, target);
      onChanged(source, target, kind);
      if (kind === "move") setClipboard(null);
      return true;
    } catch (cause: unknown) { setError(errorMessage(cause)); return false; }
    finally { setPending(false); }
  }
  async function paste(folderId: string, path: string) {
    if (!clipboard) return;
    await operate(clipboard.source, folderId, join(path, basename(clipboard.source.path)), clipboard.cut ? "move" : "copy");
  }
  async function submitNaming() {
    if (!naming || !validName(naming.name) || pending || !csrf) return;
    const target = join(naming.parent, naming.name);
    if (naming.kind === "rename" && naming.source) {
      if (await operate(naming.source, naming.folderId, target, "rename")) setNaming(null);
      return;
    }
    setPending(true); setError(null);
    try {
      const result = await api.fileOperation(project.id, { kind: naming.kind, project_version: project.version, target_folder_id: naming.folderId, target_path: target }, csrf, new AbortController().signal);
      onChanged(); setNaming(null);
      if (result.state === "partial") setError("新建操作仅部分完成，请刷新目录检查磁盘状态。");
    } catch (cause: unknown) { setError(errorMessage(cause)); }
    finally { setPending(false); }
  }
  async function previewDelete(source: EntryRef) {
    if (!csrf || pending) return;
    setPending(true); setError(null);
    try {
      const preview = await api.deletePreview(project.id, source.folderId, project.version, source.path, csrf, new AbortController().signal);
      setDeleting({ source, preview });
    } catch (cause: unknown) { setError(errorMessage(cause)); }
    finally { setPending(false); }
  }
  async function confirmDelete() {
    if (!deleting || !csrf || pending) return;
    setPending(true); setError(null);
    try {
      const scope = editorScopes.get(project.id);
      if (scope && !await scope.protect(deleting.source)) throw new Error("草稿保护失败，未执行删除，请先导出输入。");
      const result = await api.fileOperation(project.id, { kind: "delete", project_version: project.version, source_folder_id: deleting.source.folderId, source_path: deleting.source.path, delete_token: deleting.preview.token }, csrf, new AbortController().signal);
      onChanged(deleting.source, undefined, "delete");
      setDeleting(null);
      if (result.state === "partial") setError("部分文件已删除，请刷新目录检查剩余内容。");
    } catch (cause: unknown) { setError(errorMessage(cause)); }
    finally { setPending(false); }
  }
  async function copyPath(source: EntryRef, absolute: boolean) {
    const root = project.folders.find(folder => folder.id === source.folderId)?.path ?? "";
    try { await copyText(absolute ? `${root.replace(/\/$/, "")}/${source.path}` : source.path); setError(null); }
    catch (cause: unknown) { setError(errorMessage(cause)); }
  }
  const dialogs = <>
    {error && !naming && !deleting && <p role="alert" className="px-3 py-2 text-sm text-destructive">{error}</p>}
    <Dialog open={naming !== null} onOpenChange={open => { if (!open && !pending) setNaming(null); }}><DialogContent><DialogHeader><DialogTitle>{naming?.kind === "rename" ? "重命名" : naming?.kind === "create_directory" ? "新建文件夹" : "新建文件"}</DialogTitle><DialogDescription>目标已存在时不会覆盖。</DialogDescription></DialogHeader>
      <Input aria-label="名称" value={naming?.name ?? ""} onChange={event => setNaming(current => current ? { ...current, name: event.target.value } : null)} onKeyDown={event => { if (event.key === "Enter") void submitNaming(); }} />
      {naming && !validName(naming.name) && <p role="alert" className="text-sm text-destructive">请输入有效的单个文件名。</p>}
      {error && <p role="alert" className="text-sm text-destructive">{error}</p>}
      <DialogFooter><Button variant="outline" disabled={pending} onClick={() => setNaming(null)}>取消</Button><Button disabled={!naming || !validName(naming.name) || pending} onClick={() => void submitNaming()}>确定</Button></DialogFooter>
    </DialogContent></Dialog>
    <Dialog open={deleting !== null} onOpenChange={open => { if (!open && !pending) setDeleting(null); }}><DialogContent><DialogHeader><DialogTitle>永久删除</DialogTitle><DialogDescription>将永久删除 {deleting?.preview.path}，共 {deleting?.preview.count} 项、{deleting?.preview.bytes} 字节。此操作无法撤销。已打开文件的未保存输入会先保留为本地草稿；保存失败时不会删除。</DialogDescription></DialogHeader>
      {error && <p role="alert" className="text-sm text-destructive">{error}</p>}
      <DialogFooter><Button variant="outline" disabled={pending} onClick={() => setDeleting(null)}>取消</Button><Button variant="destructive" disabled={pending} onClick={() => void confirmDelete()}>永久删除</Button></DialogFooter>
    </DialogContent></Dialog>
  </>;
  return {
    clipboard, pending, dialogs, paste, operate,
    createFile: (folderId: string, parent: string) => { setError(null); setNaming({ kind: "create_file", folderId, parent, name: "" }); },
    createDirectory: (folderId: string, parent: string) => { setError(null); setNaming({ kind: "create_directory", folderId, parent, name: "" }); },
    rename: (source: EntryRef) => { setError(null); setNaming({ kind: "rename", folderId: source.folderId, parent: parentOf(source.path), source, name: basename(source.path) }); },
    copy: (source: EntryRef) => setClipboard({ source, cut: false }),
    cut: (source: EntryRef) => setClipboard({ source, cut: true }),
    previewDelete, copyPath,
  };
}
