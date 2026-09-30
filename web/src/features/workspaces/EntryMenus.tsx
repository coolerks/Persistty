import { Copy, Download, FolderInput, Pencil, Scissors, Trash2 } from "lucide-react";
import { ContextMenu, ContextMenuContent, ContextMenuItem, ContextMenuSeparator, ContextMenuTrigger } from "@/components/ui/context-menu";
import { fileDownloadURL } from "@/lib/api/client";
import type { Project } from "@/lib/api/decoder";
import type { EntryRef, useExplorerActions } from "./ExplorerActions";

type Actions = ReturnType<typeof useExplorerActions>;
export function EntryMenus({ project, source, children, actions, onOpen, onArchive }: { project: Project; source: EntryRef; children: React.ReactElement; actions: Actions; onOpen(): void; onArchive(): void }) {
  const isDirectory = source.kind === "directory";
  return <ContextMenu><ContextMenuTrigger render={children} /><ContextMenuContent>
    <ContextMenuItem onClick={onOpen}>打开</ContextMenuItem>
    <ContextMenuItem onClick={() => void actions.copyPath(source, true)}>复制路径</ContextMenuItem>
    <ContextMenuItem onClick={() => void actions.copyPath(source, false)}>复制相对路径</ContextMenuItem>
    <ContextMenuSeparator />
    <ContextMenuItem onClick={() => actions.rename(source)}><Pencil />重命名</ContextMenuItem>
    <ContextMenuItem onClick={() => actions.copy(source)}><Copy />复制</ContextMenuItem>
    <ContextMenuItem onClick={() => actions.cut(source)}><Scissors />剪切</ContextMenuItem>
    {isDirectory && <ContextMenuItem disabled={!actions.clipboard} onClick={() => void actions.paste(source.folderId, source.path)}><FolderInput />粘贴到此处</ContextMenuItem>}
    <ContextMenuSeparator />
    {isDirectory ? <ContextMenuItem onClick={onArchive}><Download />下载 ZIP</ContextMenuItem> : <ContextMenuItem onClick={() => window.location.assign(fileDownloadURL(project.id, source.folderId, project.version, source.path))}><Download />下载文件</ContextMenuItem>}
    <ContextMenuItem variant="destructive" onClick={() => void actions.previewDelete(source)}><Trash2 />永久删除</ContextMenuItem>
  </ContextMenuContent></ContextMenu>;
}
