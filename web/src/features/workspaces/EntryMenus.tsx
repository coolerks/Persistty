import { Copy, Download, FolderInput, MoreHorizontal, Pencil, Scissors, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { ContextMenu, ContextMenuContent, ContextMenuItem, ContextMenuSeparator, ContextMenuTrigger } from "@/components/ui/context-menu";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { fileDownloadURL } from "@/lib/api/client";
import type { Project } from "@/lib/api/decoder";
import type { EntryRef, useExplorerActions } from "./ExplorerActions";

type Actions = ReturnType<typeof useExplorerActions>;
export function EntryMenus({ project, source, children, actions, onOpen, onArchive }: { project: Project; source: EntryRef; children: React.ReactElement; actions: Actions; onOpen(): void; onArchive(): void }) {
  const isDirectory = source.kind === "directory";
  const items = (Item: typeof ContextMenuItem | typeof DropdownMenuItem, Separator: typeof ContextMenuSeparator | typeof DropdownMenuSeparator) => <>
    <Item onClick={onOpen}>打开</Item>
    <Item onClick={() => void actions.copyPath(source, true)}>复制路径</Item>
    <Item onClick={() => void actions.copyPath(source, false)}>复制相对路径</Item>
    <Separator />
    <Item onClick={() => actions.rename(source)}><Pencil />重命名</Item>
    <Item onClick={() => actions.copy(source)}><Copy />复制</Item>
    <Item onClick={() => actions.cut(source)}><Scissors />剪切</Item>
    {isDirectory && <Item disabled={!actions.clipboard} onClick={() => void actions.paste(source.folderId, source.path)}><FolderInput />粘贴到此处</Item>}
    <Separator />
    {isDirectory ? <Item onClick={onArchive}><Download />下载 ZIP</Item> : <Item onClick={() => window.location.assign(fileDownloadURL(project.id, source.folderId, project.version, source.path))}><Download />下载文件</Item>}
    <Item variant="destructive" onClick={() => void actions.previewDelete(source)}><Trash2 />永久删除</Item>
  </>;
  return <ContextMenu><ContextMenuTrigger render={children} /><ContextMenuContent>{items(ContextMenuItem, ContextMenuSeparator)}</ContextMenuContent>
    <DropdownMenu><DropdownMenuTrigger render={<Button size="icon" variant="ghost" className="size-8 shrink-0" aria-label={`操作 ${source.path.split("/").at(-1)}`} title="更多操作" />}><MoreHorizontal /></DropdownMenuTrigger><DropdownMenuContent align="end">{items(DropdownMenuItem, DropdownMenuSeparator)}</DropdownMenuContent></DropdownMenu>
  </ContextMenu>;
}
