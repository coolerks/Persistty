import { editorScopes } from "./editor-session";
import { useEffect, useRef, useState } from "react";
import { ChevronDown, ChevronRight, ClipboardPaste, Search, FilePlus, FolderPlus, RefreshCw } from "lucide-react";
import { Button } from "@/components/ui/button";
import { api, ApiError, errorMessage } from "@/lib/api/client";
import { useResource } from "@/lib/api/use-resource";
import type { FileEntry, Project } from "@/lib/api/decoder";
import { Failure, Loading } from "@/components/Feedback";
import { useExplorerActions } from "./ExplorerActions";
import type { EntryRef } from "./ExplorerActions";
import { EntryMenus } from "./EntryMenus";
import { useArchiveAction } from "./ArchiveAction";
import { UploadControls } from "./UploadControls";
import { useFileEvents } from "./use-file-events";
import { fileKey, useWorkspaceView, type OpenFile } from "./workspace-view";
import { FileTypeIcon } from "./FileTypeIcon";

type Actions = ReturnType<typeof useExplorerActions>;

function draggedEntry(event: React.DragEvent): EntryRef | null {
  if (!event.dataTransfer.types.includes("application/x-persistty-entry")) return null;
  try {
    const value: unknown = JSON.parse(event.dataTransfer.getData("application/x-persistty-entry"));
    if (!value || typeof value !== "object" || !("folderId" in value) || !("path" in value) || !("identity" in value) || !("kind" in value)) return null;
    if (typeof value.folderId !== "string" || typeof value.path !== "string" || typeof value.identity !== "string" || (value.kind !== "file" && value.kind !== "directory")) return null;
    return { folderId: value.folderId, path: value.path, identity: value.identity, kind: value.kind };
  } catch { return null; }
}

export function Explorer({ project, onFile, reveal, onSearch }: { project: Project; onFile(folderId: string, path: string): void; reveal?: { file: OpenFile | null }; onSearch?(folderId: string, path: string, replace: boolean): void }) {
  const [epoch, setEpoch] = useState(0);
  const [selectedFile, setSelectedFile] = useState<OpenFile | null>(reveal?.file ?? null);
  useEffect(() => { if (reveal) setSelectedFile(reveal.file); }, [reveal]);
  const relocate = useWorkspaceView(state => state.relocate);
  const remove = useWorkspaceView(state => state.remove);
  const actions = useExplorerActions(project, (source, target, kind) => {
    setEpoch(value => value + 1);
    if (!source || !kind) return;
    if (kind === "delete") remove(project.id, source);
    else if (target && kind !== "copy") relocate(project.id, source, target);
  });
  const archive = useArchiveAction(project);
  return <aside className="explorer" aria-label="资源管理器">
    <div className="explorer-heading"><strong>资源管理器</strong><Button size="icon" variant="ghost" aria-label="刷新文件树" title="刷新文件树" onClick={() => setEpoch(value => value + 1)}><RefreshCw /></Button></div>
    {actions.dialogs}{archive.element}
    <div className="explorer-scroll">{project.folders.map(folder => <FolderTree key={`${folder.id}:${project.version}`} project={project} folderId={folder.id} rootPath={folder.path} onSearch={onSearch ? (path, replace) => onSearch(folder.id, path, replace) : undefined} epoch={epoch} actions={actions} selectedFile={selectedFile} reveal={reveal} onArchive={path => void archive.begin(folder.id, path)} onFile={path => { setSelectedFile({ folderId: folder.id, path }); onFile(folder.id, path); }} />)}</div>
  </aside>;
}

type Common = { onSearch?: ((path: string, replace: boolean) => void) | undefined; project: Project; folderId: string; actions: Actions; onArchive(path: string): void; onFile(path: string): void; epoch: number; selectedFile: OpenFile | null; reveal?: { file: OpenFile | null } | undefined };

function FolderTree({ project, folderId, rootPath, epoch, actions, onArchive, onFile, onSearch, selectedFile, reveal }: Common & { rootPath: string }) {
  const [selected, setSelected] = useState("");
  const [localEpoch, setLocalEpoch] = useState(0);
  useEffect(() => {
    if (reveal?.file?.folderId === folderId) setSelected(reveal.file.path.split("/").slice(0, -1).join("/"));
  }, [folderId, reveal]);
  const containerRef = useRef<HTMLDivElement>(null);
  const refresh = () => setLocalEpoch(value => value + 1);
  function dropRoot(event: React.DragEvent) {
    const source = draggedEntry(event);
    if (!source) return;
    event.preventDefault();
    const name = source.path.split("/").at(-1);
    if (name && (source.folderId !== folderId || source.path !== name)) void actions.operate(source, folderId, name, "move");
  }
  return <section className="tree-root" ref={containerRef} aria-label={rootPath}>
    <div className="tree-root-title" onDragOver={event => { if (event.dataTransfer.types.includes("application/x-persistty-entry")) event.preventDefault(); }} onDrop={dropRoot}><FileTypeIcon path={rootPath} kind="root" expanded /><span className="truncate" title={rootPath}>{rootPath.split("/").at(-1) || rootPath}</span>{folderId === project.main_folder_id && <span className="tree-root-main">主</span>}</div>
    <div className="tree-toolbar" aria-label="文件操作"><span className="tree-target" title={selected || rootPath}>{selected ? `/${selected}` : "/"}</span>
      <Button size="icon" variant="ghost" title="新建文件" aria-label="新建文件" onClick={() => actions.createFile(folderId, selected)}><FilePlus /></Button>
      <Button size="icon" variant="ghost" title="新建文件夹" aria-label="新建文件夹" onClick={() => actions.createDirectory(folderId, selected)}><FolderPlus /></Button>
      <Button size="icon" variant="ghost" title="粘贴" aria-label="粘贴" disabled={!actions.clipboard || actions.pending} onClick={() => void actions.paste(folderId, selected)}><ClipboardPaste /></Button>
      {onSearch && <Button size="icon" variant="ghost" aria-label="在目录中搜索" title="在目录中搜索" onClick={() => onSearch(selected, false)}><Search /></Button>}
      <UploadControls project={project} folderId={folderId} path={selected} containerRef={containerRef} onChanged={refresh} />
      <Button size="icon" variant="ghost" title="刷新目录" aria-label="刷新目录" onClick={refresh}><RefreshCw /></Button>
    </div>
    <Directory key={`${epoch}:${localEpoch}`} {...{ project, folderId, actions, onArchive, onFile, onSearch, epoch, selectedFile, reveal }} path="" depth={0} selected={selected} onSelect={setSelected} />
  </section>;
}

type DirectoryProps = Common & { path: string; depth: number; selected: string; onSelect(path: string): void };

function Directory({ project, folderId, path, depth, actions, onArchive, onFile, onSearch, epoch, selected, onSelect, selectedFile, reveal }: DirectoryProps) {
  const [load] = useState(() => (signal: AbortSignal) => api.entries(project.id, folderId, project.version, path, "", signal));
  const { resource, refreshQuietly } = useResource(load);
  const [extra, setExtra] = useState<FileEntry[]>([]);
  const [cursor, setCursor] = useState<string | null | undefined>(undefined);
  const [nextError, setNextError] = useState<string | null>(null);
  const [pending, setPending] = useState(false);
  const request = useRef<AbortController | null>(null);
  useEffect(() => () => request.current?.abort(), []);
  function reload() { request.current?.abort(); request.current = null; setPending(false); setExtra([]); setCursor(undefined); setNextError(null); refreshQuietly(); if (depth === 0) void editorScopes.get(project.id)?.refresh(); }
  const eventMode = useFileEvents(project.id, folderId, project.version, path, reload);
  async function next(nextCursor: string) {
    if (pending) return;
    const controller = new AbortController(); request.current = controller;
    setPending(true); setNextError(null);
    try {
      const page = await api.entries(project.id, folderId, project.version, path, nextCursor, controller.signal);
      if (!controller.signal.aborted) { setExtra(current => [...current, ...page.items]); setCursor(page.next_cursor); }
    } catch (cause: unknown) {
      if (!controller.signal.aborted) {
        if (cause instanceof ApiError && cause.status === 409) reload();
        else setNextError(errorMessage(cause));
      }
    } finally { if (request.current === controller) { request.current = null; setPending(false); } }
  }
  function dragOver(event: React.DragEvent) { if (event.dataTransfer.types.includes("application/x-persistty-entry")) { event.preventDefault(); event.dataTransfer.dropEffect = "move"; } }
  function drop(event: React.DragEvent, destination: string) {
    const source = draggedEntry(event);
    if (!source) return;
    event.preventDefault();
    const name = source.path.split("/").at(-1);
    if (!name) return;
    const target = destination ? `${destination}/${name}` : name;
    if (source.folderId !== folderId || source.path !== target) void actions.operate(source, folderId, target, "move");
  }
  const items = resource.status === "ready" ? [...resource.data.items, ...extra] : [];
  const nextCursor = cursor === undefined ? (resource.status === "ready" ? resource.data.next_cursor : null) : cursor;
  const nextRef = useRef(next); nextRef.current = next;
  const revealPath = reveal?.file?.folderId === folderId ? reveal.file.path : null;
  const childName = revealPath && (!path || revealPath.startsWith(`${path}/`)) ? revealPath.slice(path ? path.length + 1 : 0).split("/")[0] : null;
  const hasChild = items.some(item => item.name === childName);
  useEffect(() => {
    if (childName && !hasChild && nextCursor && !pending && !nextError) void nextRef.current(nextCursor);
  }, [childName, hasChild, nextCursor, pending, nextError]);
  return <div role="group" aria-label={path || "文件夹根目录"}>
    {resource.status === "loading" && <div className="tree-message"><Loading /></div>}
    {resource.status === "error" && <div className="tree-message"><Failure error={resource.error} retry={reload} /></div>}
    {items.map(item => <TreeEntry key={item.name} {...{ item, project, folderId, actions, onArchive, onFile, onSearch, epoch, selected, onSelect, selectedFile, reveal, dragOver, drop }} fullPath={path ? `${path}/${item.name}` : item.name} depth={depth} />)}
    {resource.status === "ready" && items.length === 0 && depth === 0 && <p className="tree-message">空文件夹</p>}
    {nextCursor && <Button variant="ghost" size="sm" disabled={pending} onClick={() => void next(nextCursor)}>{pending ? "加载中" : "加载更多"}</Button>}
    {nextError && <p role="alert" className="tree-message text-destructive">{nextError}</p>}
    {depth === 0 && <span className="sr-only">{eventMode === "watching" ? "文件变更监听中" : "定时刷新"}</span>}
  </div>;
}

function TreeEntry({ item, fullPath, depth, project, folderId, actions, onArchive, onFile, onSearch, epoch, selected, onSelect, selectedFile, reveal, dragOver, drop }: Omit<DirectoryProps, "path"> & { item: FileEntry; fullPath: string; dragOver(event: React.DragEvent): void; drop(event: React.DragEvent, destination: string): void }) {
  const directory = item.kind === "directory";
  const ancestor = directory && reveal?.file?.folderId === folderId && reveal.file.path.startsWith(`${fullPath}/`);
  const [expanded, setExpanded] = useState(() => ancestor || !!useWorkspaceView.getState().projects[project.id]?.expanded?.[fileKey({ folderId, path: fullPath })]);
  const rowRef = useRef<HTMLButtonElement>(null);
  const selectedEntry = !directory && selectedFile !== null && fileKey(selectedFile) === fileKey({ folderId, path: fullPath });
  useEffect(() => { if (ancestor) { setExpanded(true); useWorkspaceView.getState().expand(project.id, { folderId, path: fullPath }, true); } }, [ancestor, reveal, project.id, folderId, fullPath]);
  useEffect(() => {
    if (reveal?.file?.folderId === folderId && reveal.file.path === fullPath) rowRef.current?.scrollIntoView({ block: "nearest", inline: "nearest" });
  }, [folderId, fullPath, reveal]);
  const source: EntryRef = { folderId, path: fullPath, kind: item.kind, identity: item.identity };
  const open = () => { if (directory) { setExpanded(!expanded); useWorkspaceView.getState().expand(project.id, { folderId, path: fullPath }, !expanded); onSelect(fullPath); } else { onSelect(fullPath.split("/").slice(0, -1).join("/")); onFile(fullPath); } };
  const row = <Button ref={rowRef} variant="ghost" role="treeitem" aria-expanded={directory ? expanded : undefined} aria-selected={directory ? undefined : selectedEntry} className={`tree-entry ${selectedEntry ? "tree-entry-selected" : ""}`} style={{ paddingLeft: 10 + depth * 16 }} title={item.name} onClick={open}>
    {directory ? expanded ? <ChevronDown className="size-3 shrink-0" /> : <ChevronRight className="size-3 shrink-0" /> : <span className="size-3 shrink-0" />}
    <FileTypeIcon path={fullPath} kind={directory ? "directory" : "file"} expanded={expanded} />
    <span className="truncate">{item.name}</span>
  </Button>;
  return <div role="group">
    <div className="tree-entry-wrap" draggable={item.kind !== "unsupported"} onDragStart={event => { event.dataTransfer.setData("application/x-persistty-entry", JSON.stringify(source)); event.dataTransfer.effectAllowed = "move"; }} onDragOver={directory ? dragOver : undefined} onDrop={directory ? event => drop(event, fullPath) : undefined}>
      {item.kind === "unsupported" ? row : <EntryMenus project={project} source={source} actions={actions} onOpen={open} onArchive={() => onArchive(fullPath)} onSearch={onSearch ? replace => onSearch(fullPath, replace) : undefined}>{row}</EntryMenus>}
    </div>
    {directory && expanded && <Directory {...{ project, folderId, actions, onArchive, onFile, onSearch, epoch, selected, onSelect, selectedFile, reveal }} path={fullPath} depth={depth + 1} />}
  </div>;
}
