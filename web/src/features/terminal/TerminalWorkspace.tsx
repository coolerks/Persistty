import { useEffect, useState } from "react";
import { Link } from "react-router";
import { History, PanelTop, Plus, RefreshCw, TerminalSquare, Trash2, X } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button, buttonVariants } from "@/components/ui/button";
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Failure, Loading } from "@/components/Feedback";
import { useAuth } from "@/features/auth/auth-context";
import { api, errorMessage } from "@/lib/api/client";
import { useResource } from "@/lib/api/use-resource";
import type { Project, Terminal } from "@/lib/api/decoder";
import { TerminalSession } from "./TerminalRuntime";
import { useTerminalStateChanges } from "./runtime-context";

export function TerminalWorkspace({ project, initialId, onHide, onMoveToTop, onMoveToBottom, upperIds = [], focusRequest }: {
  project?: Project; initialId?: string | undefined; onHide?(): void; onMoveToTop?(terminal: Terminal): void;
  onMoveToBottom?(id: string): void; upperIds?: string[]; focusRequest?: { id: string } | null;
}) {
  const auth = useAuth();
  const { resource, refresh } = useResource(api.terminals);
  const [selectedId, setSelectedId] = useState<string | null>(initialId ?? null);
  const [folderId, setFolderId] = useState(project?.main_folder_id ?? "");
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState<string | null>(null);
  const [closeRequested, setCloseRequested] = useState(false);
  useTerminalStateChanges(refresh);
  useEffect(() => { if (focusRequest) setSelectedId(focusRequest.id); }, [focusRequest]);
  const items = resource.status === "ready" ? resource.data.filter(item => (!project || item.project_id === project.id) && !upperIds.includes(item.id)) : [];
  const selected = selectedId ? items.find(item => item.id === selectedId) : items.find(item => item.state === "running") ?? items[0];

  async function create() {
    if (!project || auth.state.status !== "authenticated" || creating) return;
    setCreating(true);
    setCreateError(null);
    try {
      const result = await api.createTerminal({ project_id: project.id, project_version: project.version,
        folder_id: folderId || project.main_folder_id }, auth.state.session.csrf_token, new AbortController().signal);
      setSelectedId(result.id);
      refresh();
    } catch (error) {
      setCreateError(errorMessage(error));
    } finally {
      setCreating(false);
    }
  }

  return <section className="terminal-pane" aria-label={project ? "终端面板" : "全部终端"} onDragOver={event => {
    if (onMoveToBottom && event.dataTransfer.types.includes("application/x-persistty-upper-terminal")) event.preventDefault();
  }} onDrop={event => {
    if (!onMoveToBottom || !event.dataTransfer.types.includes("application/x-persistty-upper-terminal")) return;
    event.preventDefault();
    try {
      const payload: unknown = JSON.parse(event.dataTransfer.getData("application/x-persistty-upper-terminal"));
      if (payload && typeof payload === "object" && "id" in payload && typeof payload.id === "string" && upperIds.includes(payload.id)) onMoveToBottom(payload.id);
    } catch { /* Unrelated drag data. */ }
  }}>
    <div className="terminal-heading">
      <TerminalSquare className="size-4" /><strong>终端</strong>
      {project && project.folders.length > 1 && <Select value={folderId} onValueChange={value => setFolderId(value ?? project.main_folder_id)}>
        <SelectTrigger size="sm" aria-label="新终端文件夹" title="新终端文件夹"><SelectValue /></SelectTrigger>
        <SelectContent><SelectGroup>{project.folders.map(folder => <SelectItem key={folder.id} value={folder.id}>{folder.path}</SelectItem>)}</SelectGroup></SelectContent>
      </Select>}
      <span className="terminal-heading-spacer" />
      {project && <Button size="icon-sm" variant="ghost" aria-label="新建终端" title="新建终端" disabled={creating} onClick={() => void create()}><Plus /></Button>}
      <Button size="icon-sm" variant="ghost" aria-label="刷新终端" title="刷新终端" onClick={refresh}><RefreshCw /></Button>
      {onHide && <Button size="icon-sm" variant="ghost" aria-label="收起终端面板" title="收起终端面板" onClick={onHide}><X /></Button>}
    </div>
    {createError && <Alert variant="destructive" className="shrink-0"><AlertDescription>{createError}</AlertDescription></Alert>}
    {resource.status === "loading" && <div className="terminal-feedback"><Loading /></div>}
    {resource.status === "error" && <div className="terminal-feedback"><Failure error={resource.error} retry={refresh} /></div>}
    {resource.status === "ready" && !selected && <Empty className="terminal-empty"><EmptyHeader><EmptyMedia variant="icon"><TerminalSquare /></EmptyMedia><EmptyTitle>{selectedId && !upperIds.includes(selectedId) ? "终端不存在" : project ? "此项目暂无下方面板终端" : "暂无终端"}</EmptyTitle></EmptyHeader></Empty>}
    {resource.status === "ready" && items.length > 0 && <>
      <nav className="terminal-tabs" aria-label="终端会话">
        {items.map(item => <Button key={item.id} size="sm" variant={selected?.id === item.id ? "secondary" : "ghost"} draggable={Boolean(onMoveToTop)}
          onDragStart={event => { if (onMoveToTop) { event.dataTransfer.setData("application/x-persistty-lower-terminal", JSON.stringify(item)); event.dataTransfer.effectAllowed = "move"; } }}
          className="terminal-tab" aria-current={selected?.id === item.id ? "page" : undefined} onClick={() => { setSelectedId(item.id); setCloseRequested(false); }}>
          <TerminalSquare data-icon="inline-start" /><span className="truncate">{item.display_name}</span>
          <span className={`terminal-state-dot ${item.state}`} aria-label={stateLabel(item.state)} />
        </Button>)}
      </nav>
      {selected && <div className="terminal-session-wrap">
        <div className="terminal-session-bar"><span className="truncate">{selected.working_directory}</span><Badge variant="secondary">{stateLabel(selected.state)}</Badge>
          {onMoveToTop && <Button size="icon-sm" variant="ghost" aria-label="移到上方标签" title="移到上方标签" onClick={() => { onMoveToTop(selected); setSelectedId(null); }}><PanelTop /></Button>}
          {selected.state === "running" && <Button size="icon-sm" variant="ghost" aria-label="关闭当前终端" title="关闭当前终端" onClick={() => setCloseRequested(true)}><Trash2 /></Button>}
          {!project && <Link className={buttonVariants({ variant: "ghost", size: "sm" })} to="/projects">项目</Link>}
        </div>
        {selected.state === "running" ? <TerminalSession key={selected.id} terminal={selected} closeRequested={closeRequested}
          onCloseRequestHandled={() => setCloseRequested(false)} /> :
          <Empty><EmptyHeader><EmptyMedia variant="icon"><History /></EmptyMedia><EmptyDescription>{selected.state === "terminated" ? "会话已结束；不会自动重新执行命令。" : "终端服务暂时不可用。"}</EmptyDescription></EmptyHeader></Empty>}
      </div>}
    </>}
  </section>;
}

function stateLabel(state: Terminal["state"]): string {
  switch (state) {
    case "running": return "运行中";
    case "terminated": return "已结束";
    case "unavailable": return "不可用";
  }
}
