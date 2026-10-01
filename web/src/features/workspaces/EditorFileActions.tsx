import { Download, RefreshCw, Save } from "lucide-react";
import { Button, buttonVariants } from "@/components/ui/button";
import { fileDownloadURL } from "@/lib/api/client";
import type { Project } from "@/lib/api/decoder";
import { useEditorScope } from "./editor-context";
import { fileKey, type OpenFile } from "./workspace-view";

export function EditorFileActions({ project, file }: { project: Project; file: OpenFile }) {
  const scope = useEditorScope();
  const buffer = scope.buffers.get(fileKey(file));
  const ready = buffer?.state.status === "ready";
  return <>
    <Button size="icon" variant="ghost" aria-label="保存文件" title="保存文件" disabled={!ready || !buffer?.dirty || buffer.state.saveState === "saving" || buffer.state.saveState === "conflict"} onClick={() => void buffer?.save(true)}><Save /></Button>
    <Button size="icon" variant="ghost" aria-label="刷新文件" title="刷新文件" onClick={() => void (ready ? buffer?.refresh() : buffer?.load())}><RefreshCw /></Button>
    <a className={buttonVariants({ variant: "ghost", size: "icon" })} aria-label="下载文件" title="下载文件" href={fileDownloadURL(project.id, file.folderId, project.version, file.path)}><Download /></a>
  </>;
}
