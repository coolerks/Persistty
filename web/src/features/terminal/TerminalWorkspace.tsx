import { useEffect, useState } from "react";
import { Check, Plus, RefreshCw, TerminalSquare, PanelRight, PanelRightClose, Maximize2, Minimize2, X } from "lucide-react";
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
import { groupIndexes, useWorkspaceView, type GroupIndex } from "@/features/workspaces/workspace-view";
import { decodeTerminal } from "@/lib/api/decoder";
import { terminalVisible, useTerminalView } from "./terminal-view";

export function TerminalWorkspace({ project, initialId, onHide, onToggleMaximize, maximized = false, onMoveToTop, onMoveToBottom, upperIds = [], focusRequest, group = 0, mobile = false }: {
  group?: GroupIndex; mobile?: boolean; project?: Project; initialId?: string | undefined; onHide?(): void; onMoveToTop?(terminal: Terminal): void;
  onToggleMaximize?(): void; maximized?: boolean;
  onMoveToBottom?(id: string): void; upperIds?: string[]; focusRequest?: { id: string } | null;
}) {
  const auth = useAuth();
  const view = useWorkspaceView(state => project ? state.projects[project.id] : undefined);
  const dismissed = useTerminalView(state => state.dismissed);
  const { resource, refresh } = useResource(api.terminals);
  const [selectedId, setSelectedId] = useState<string | null>(initialId ?? null);
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState<string | null>(null);
  useTerminalStateChanges(refresh);
  useEffect(() => { if (focusRequest) setSelectedId(focusRequest.id); }, [focusRequest]);
  const items = resource.status === "ready" ? resource.data.filter(item => (!project || item.project_id === project.id) && (mobile || (!upperIds.includes(item.id) && (view?.terminals[item.id]?.region ?? "bottom") === "bottom" && (view?.terminals[item.id]?.group ?? 0) === group)) && terminalVisible(item, dismissed)).sort((a, b) => { const order = view?.terminalOrder ?? []; const left = order.indexOf(a.id); const right = order.indexOf(b.id); return (left < 0 ? 1000 : left) - (right < 0 ? 1000 : right); }) : [];
  const activate = (id: string) => { setSelectedId(id); if (project) useWorkspaceView.getState().selectTerminal(project.id, id, mobile ? "mobile" : { region: "bottom", group }); };
  const persistedId = project ? mobile ? view?.mobileTerminal : view?.lowerActive[group] : null;
  const missingInitial = Boolean(initialId && selectedId === initialId && !items.some(item => item.id === initialId));
  const selected = missingInitial ? undefined : items.find(item => item.id === (persistedId ?? selectedId)) ?? items.find(item => item.state === "running") ?? items[0];
  const dropTarget = useDropTarget(onMoveToBottom ? ["application/x-persistty-upper-terminal", "application/x-persistty-lower-terminal"] : [], event => {
    if (!project || !event.dataTransfer) return;
    try {
      const payload: unknown = JSON.parse(event.dataTransfer.getData("application/x-persistty-upper-terminal") || event.dataTransfer.getData("application/x-persistty-lower-terminal"));
      const terminal = decodeTerminal(payload);
      if (terminal.project_id !== project.id || resource.status !== "ready" || !resource.data.some(item => item.id === terminal.id)) return;
      useWorkspaceView.getState().placeTerminal(project.id, terminal.id, { region: "bottom", group });
      const before = event.target instanceof Element ? event.target.closest("[data-terminal-id]")?.getAttribute("data-terminal-id") : null;
      if (before && before !== terminal.id) useWorkspaceView.getState().orderTerminal(project.id, terminal.id, before, items.map(item => item.id));
      activate(terminal.id);
    } catch { /* Unrelated drag data. */ }
  });
  async function create(folderId: string) {
    if (!project || auth.state.status !== "authenticated" || creating) return;
    setCreating(true); setCreateError(null);
    try {
      const result = await api.createTerminal({ project_id: project.id, project_version: project.version, folder_id: folderId }, auth.state.session.csrf_token, new AbortController().signal);
      if (!mobile) useWorkspaceView.getState().placeTerminal(project.id, result.id, { region: "bottom", group });
      activate(result.id); refresh();
    } catch (error) { setCreateError(errorMessage(error)); }
    finally { setCreating(false); }
  }
  const add = <Button size="icon-sm" variant="ghost" aria-label="新建终端" title="新建终端" disabled={creating}><Plus /></Button>;
  return <section ref={dropTarget} className="terminal-pane" aria-label={project ? "终端面板" : "全部终端"}>
    <Tabs value={selected?.id ?? "empty"} onValueChange={value => activate(String(value))} className="terminal-tab-layout">
      <div className="terminal-tab-row"><TabsList variant="line" className="terminal-tabs" aria-label="终端会话">
        {items.map(item => <TerminalTab key={item.id} terminal={item} region={items} active={selected?.id === item.id} onActivate={() => activate(item.id)} onRefresh={refresh} onMove={mobile ? undefined : onMoveToTop} allowBatch={Boolean(project)} moves={!mobile && project ? [
          ...groupIndexes.filter(i => i !== group).map(i => ({ label: `移到下方第 ${i + 1} 组`, run: (terminal: Terminal) => useWorkspaceView.getState().placeTerminal(project.id, terminal.id, { region: "bottom", group: i }) })),
          ...groupIndexes.map(i => ({ label: `移到上方第 ${i + 1} 组`, run: (terminal: Terminal) => { useWorkspaceView.getState().placeTerminal(project.id, terminal.id, { region: "top", group: i }); refresh(); } })),
          ...(items.indexOf(item) > 0 ? [{ label: "标签向左排序", run: (terminal: Terminal) => { const before = items[items.indexOf(item) - 1]; if (before) useWorkspaceView.getState().orderTerminal(project.id, terminal.id, before.id, items.map(item => item.id)); } }] : []),
        ] : []} />)}
      </TabsList>
      {project && (project.folders.length > 1 ? <DropdownMenu><DropdownMenuTrigger render={add} /><DropdownMenuContent className="terminal-menu"><DropdownMenuGroup><DropdownMenuLabel>选择目录</DropdownMenuLabel>{project.folders.map(folder => <DropdownMenuItem key={folder.id} onClick={() => void create(folder.id)}><span className="truncate" title={folder.path}>{folder.path}</span>{folder.id === project.main_folder_id && <Check className="ml-auto shrink-0" aria-label="主目录" />}</DropdownMenuItem>)}</DropdownMenuGroup></DropdownMenuContent></DropdownMenu> : <Button size="icon-sm" variant="ghost" aria-label="新建终端" title="新建终端" disabled={creating} onClick={() => void create(project.main_folder_id)}><Plus /></Button>)}
      {!mobile && project && <Button size="icon-sm" variant="ghost" aria-label="拆分终端面板" title="拆分终端面板" disabled={(view?.lowerCount ?? 1) >= 4} onClick={() => useWorkspaceView.getState().splitTerminal(project.id)}><PanelRight /></Button>}
      {!mobile && project && (view?.lowerCount ?? 1) > 1 && <Button size="icon-sm" variant="ghost" aria-label="合并此终端分组" title="合并此终端分组（保留终端会话）" onClick={() => useWorkspaceView.getState().mergeTerminalGroup(project.id, group)}><PanelRightClose /></Button>}
      <span className="terminal-heading-spacer" /><Button size="icon-sm" variant="ghost" aria-label="刷新终端" title="刷新终端" onClick={refresh}><RefreshCw /></Button>
      {onToggleMaximize && <Button size="icon-sm" variant="ghost" aria-label={maximized ? "恢复终端面板" : "全屏终端面板"} title={maximized ? "恢复终端面板" : "全屏终端面板"} aria-pressed={maximized} onClick={onToggleMaximize}>{maximized ? <Minimize2 /> : <Maximize2 />}</Button>}
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
