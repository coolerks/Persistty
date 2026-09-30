import { useEffect, useState } from "react";
import { Check, Plus, RefreshCw, TerminalSquare, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Tabs, TabsContent, TabsList } from "@/components/ui/tabs";
import { DropdownMenu, DropdownMenuContent, DropdownMenuGroup, DropdownMenuItem, DropdownMenuLabel, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { Empty, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Failure, Loading } from "@/components/Feedback";
import { useAuth } from "@/features/auth/auth-context";
import { api, errorMessage } from "@/lib/api/client";
import { useResource } from "@/lib/api/use-resource";
import { useDropTarget } from "@/lib/use-drop-target";
import type { Project, Terminal } from "@/lib/api/decoder";
import { TerminalSession } from "./TerminalRuntime";
import { TerminalTab } from "./TerminalTab";
import { useTerminalStateChanges } from "./runtime-context";

export function TerminalWorkspace({ project, initialId, onHide, onMoveToTop, onMoveToBottom, upperIds = [], focusRequest }: {
  project?: Project; initialId?: string | undefined; onHide?(): void; onMoveToTop?(terminal: Terminal): void;
  onMoveToBottom?(id: string): void; upperIds?: string[]; focusRequest?: { id: string } | null;
}) {
  const auth = useAuth();
  const { resource, refresh } = useResource(api.terminals);
  const [selectedId, setSelectedId] = useState<string | null>(initialId ?? null);
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState<string | null>(null);
  useTerminalStateChanges(refresh);
  useEffect(() => { if (focusRequest) setSelectedId(focusRequest.id); }, [focusRequest]);
  const items = resource.status === "ready" ? resource.data.filter(item => (!project || item.project_id === project.id) && !upperIds.includes(item.id)) : [];
  const missingInitial = Boolean(initialId && selectedId === initialId && !items.some(item => item.id === initialId));
  const selected = missingInitial ? undefined : items.find(item => item.id === selectedId) ?? items.find(item => item.state === "running") ?? items[0];
  const dropTarget = useDropTarget(onMoveToBottom ? ["application/x-persistty-upper-terminal"] : [], event => {
    if (!onMoveToBottom || !event.dataTransfer) return;
    try {
      const payload: unknown = JSON.parse(event.dataTransfer.getData("application/x-persistty-upper-terminal"));
      if (payload && typeof payload === "object" && "id" in payload && typeof payload.id === "string" && upperIds.includes(payload.id)) onMoveToBottom(payload.id);
    } catch { /* Unrelated drag data. */ }
  });
  async function create(folderId: string) {
    if (!project || auth.state.status !== "authenticated" || creating) return;
    setCreating(true); setCreateError(null);
    try {
      const result = await api.createTerminal({ project_id: project.id, project_version: project.version, folder_id: folderId }, auth.state.session.csrf_token, new AbortController().signal);
      setSelectedId(result.id); refresh();
    } catch (error) { setCreateError(errorMessage(error)); }
    finally { setCreating(false); }
  }
  const add = <Button size="icon-sm" variant="ghost" aria-label="新建终端" title="新建终端" disabled={creating}><Plus /></Button>;
  return <section ref={dropTarget} className="terminal-pane" aria-label={project ? "终端面板" : "全部终端"}>
    <Tabs value={selected?.id ?? "empty"} onValueChange={value => setSelectedId(String(value))} className="terminal-tab-layout">
      <div className="terminal-tab-row"><TabsList variant="line" className="terminal-tabs" aria-label="终端会话">
        {items.map(item => <TerminalTab key={item.id} terminal={item} region={items} active={selected?.id === item.id} onActivate={() => setSelectedId(item.id)} onRefresh={refresh} onMove={onMoveToTop} allowBatch={Boolean(project)} />)}
      </TabsList>
      {project && (project.folders.length > 1 ? <DropdownMenu><DropdownMenuTrigger render={add} /><DropdownMenuContent className="terminal-menu"><DropdownMenuGroup><DropdownMenuLabel>选择目录</DropdownMenuLabel>{project.folders.map(folder => <DropdownMenuItem key={folder.id} onClick={() => void create(folder.id)}><span className="truncate" title={folder.path}>{folder.path}</span>{folder.id === project.main_folder_id && <Check className="ml-auto shrink-0" aria-label="主目录" />}</DropdownMenuItem>)}</DropdownMenuGroup></DropdownMenuContent></DropdownMenu> : <Button size="icon-sm" variant="ghost" aria-label="新建终端" title="新建终端" disabled={creating} onClick={() => void create(project.main_folder_id)}><Plus /></Button>)}
      <span className="terminal-heading-spacer" /><Button size="icon-sm" variant="ghost" aria-label="刷新终端" title="刷新终端" onClick={refresh}><RefreshCw /></Button>
      {onHide && <Button size="icon-sm" variant="ghost" aria-label="收起终端面板" title="收起终端面板" onClick={onHide}><X /></Button>}
      </div>
      {createError && <Alert variant="destructive" className="shrink-0"><AlertDescription>{createError}</AlertDescription></Alert>}
      {resource.status === "loading" && <div className="terminal-feedback"><Loading /></div>}
      {resource.status === "error" && <div className="terminal-feedback"><Failure error={resource.error} retry={refresh} /></div>}
      {resource.status === "ready" && !selected && <Empty className="terminal-empty"><EmptyHeader><EmptyMedia variant="icon"><TerminalSquare /></EmptyMedia><EmptyTitle>{missingInitial ? "终端不存在" : project ? "此项目暂无下方面板终端" : "暂无终端"}</EmptyTitle></EmptyHeader></Empty>}
      {selected && <TabsContent value={selected.id} className="terminal-session-wrap"><TerminalSession terminal={selected} /></TabsContent>}
    </Tabs>
  </section>;
}
