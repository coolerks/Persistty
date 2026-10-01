import { Fragment, useCallback, useEffect, useRef, useState, type ReactNode } from "react";
import { Link, useOutletContext } from "react-router";
import { Group, Panel, Separator, useDefaultLayout, usePanelRef } from "react-resizable-panels";
import { Files, FolderKanban, PanelBottom, PanelLeft, PanelRight, TerminalSquare, X, FileText } from "lucide-react";
import { Button, buttonVariants } from "@/components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useDropTarget } from "@/lib/use-drop-target";
import { api } from "@/lib/api/client";
import { decodeTerminal, type Project, type Terminal } from "@/lib/api/decoder";
import { Explorer } from "./Explorer";
import { watchEditorModels } from "./editor-model-lifecycle";
import { allOpenFiles, fileKey, groupIndexes, useWorkspaceView } from "./workspace-view";
import type { GroupIndex, OpenFile } from "./workspace-view";
import { TerminalWorkspace } from "@/features/terminal/TerminalWorkspace";
import { TerminalSession, TerminalRuntimeProvider } from "@/features/terminal/TerminalRuntime";
import { TerminalTab } from "@/features/terminal/TerminalTab";
import { useTerminalRuntime } from "@/features/terminal/runtime-context";
import { FileTypeIcon } from "./FileTypeIcon";
import { terminalVisible, useTerminalView } from "@/features/terminal/terminal-view";

import { Alert, AlertDescription } from "@/components/ui/alert";
import { useResource } from "@/lib/api/use-resource";
import { panelStorage, useViewStorageWarning } from "./view-storage";
import { FileEditor } from "./FileEditor";
import { EditorFileActions } from "./EditorFileActions";
import { EditorLanguageStatus } from "./EditorLanguageStatus";
import { EditorScopeProvider } from "./EditorScopeProvider";
import { useEditorScope } from "./editor-context";

function useMobile(): boolean {
  const [mobile, setMobile] = useState(() => window.matchMedia("(max-width: 760px)").matches);
  useEffect(() => {
    const media = window.matchMedia("(max-width: 760px)");
    const update = () => setMobile(media.matches);
    media.addEventListener("change", update);
    return () => media.removeEventListener("change", update);
  }, []);
  return mobile;
}

export function ProjectWorkbench({ project, onEdit }: { project: Project; onEdit(): void }) {
  const shell = useOutletContext<{ headerActions: ReactNode } | undefined>();
  const mobile = useMobile();
  const storageWarning = useViewStorageWarning();
  const recoveryError = useWorkspaceView(state => state.recoveryError);
  useEffect(() => watchEditorModels(project.id), [project.id]);
  const view = useWorkspaceView(state => state.projects[project.id]);
  const open = useWorkspaceView(state => state.open);
  const mobileView = view?.mobileView ?? "files";
  const setMobileView = (value: "files" | "editor" | "terminal") => useWorkspaceView.getState().mobileView(project.id, value);
  const { resource: terminals, refreshQuietly: refreshTerminals } = useResource(api.terminals);
  const refreshRef = useRef(refreshTerminals); refreshRef.current = refreshTerminals;
  useEffect(() => { const id = setInterval(() => refreshRef.current(), 15000); return () => clearInterval(id); }, []);
  const upperTerminals = terminals.status === "ready" ? terminals.data.filter(item => item.project_id === project.id && view?.terminals[item.id]?.region === "top") : [];
  const activeUpperId = view?.upperActive[view.focused] ?? null;
  const setActiveUpperId = (id: string | null, group: GroupIndex = view?.focused ?? 0) => useWorkspaceView.getState().selectTerminal(project.id, id, { region: "top", group });
  const dismissed = useTerminalView(state => state.dismissed);
  const [closeUpperId, setCloseUpperId] = useState<string | null>(null);
  const [bottomFocusRequest, setBottomFocusRequest] = useState<{ id: string } | null>(null);
  const sidebarRef = usePanelRef();
  const terminalRef = usePanelRef();
  const sidebarLayout = useDefaultLayout({ id: `persistty-sidebar-v1-${project.id}`, storage: panelStorage });
  const verticalLayout = useDefaultLayout({ id: `persistty-vertical-v1-${project.id}`, storage: panelStorage });
  const splitLayout = useDefaultLayout({ id: `persistty-editor-split-v1-${project.id}`, storage: panelStorage });
  const bottomLayout = useDefaultLayout({ id: `persistty-terminal-split-v2-${project.id}`, storage: panelStorage });
  const focused = view?.focused ?? 0;
  const activeKey = mobile ? view?.mobileActive : view?.active[focused];
  const activeFile = (mobile ? view?.mobileFiles : view?.groups[focused])?.find(file => fileKey(file) === activeKey);
  const activeFileRef = useRef(activeFile); activeFileRef.current = activeUpperId && !dismissed[activeUpperId] ? undefined : activeFile;
  const [reveal, setReveal] = useState<{ file: OpenFile | null }>(() => ({ file: activeFile ?? null }));
  const revealCurrentFile = useCallback(() => setReveal({ file: activeFileRef.current ?? null }), []);
  function showFiles() { if (mobileView !== "files") revealCurrentFile(); setMobileView("files"); }
  function moveToTop(terminal: Terminal, group: GroupIndex = focused) { useWorkspaceView.getState().placeTerminal(project.id, terminal.id, { region: "top", group }); refreshRef.current(); }
  function moveToBottom(id: string, group: GroupIndex = 0) { useWorkspaceView.getState().placeTerminal(project.id, id, { region: "bottom", group }); setBottomFocusRequest({ id }); terminalRef.current?.expand(); }
  const explorer = <Explorer project={project} reveal={reveal} onFile={(folderId, path) => { if (mobile) useWorkspaceView.getState().openMobile(project.id, { folderId, path }); else { open(project.id, { folderId, path }); setActiveUpperId(null); } }} />;
  const editorAreaRef = useRef<HTMLDivElement>(null);
  const [editorWidth, setEditorWidth] = useState(window.innerWidth - 300);
  useEffect(() => { const element = editorAreaRef.current; if (!element) return; const observer = new ResizeObserver(entries => { const entry = entries[0]; if (entry) setEditorWidth(entry.contentRect.width); }); observer.observe(element); return () => observer.disconnect(); }, [mobile]);
  const visibleGroups = groupIndexes.filter(i => i === 0 || !!view?.groups[i].length || upperTerminals.some(item => view?.terminals[item.id]?.group === i));
  const narrow = editorWidth < visibleGroups.length * 240;
  const renderedGroups = narrow ? [visibleGroups.includes(focused) ? focused : 0 as const] : visibleGroups;
  const editorGroup = (group: GroupIndex) => <EditorGroup project={project} group={group} upperTerminals={upperTerminals.filter(item => view?.terminals[item.id]?.group === group)} activeUpperId={view?.upperActive[group] ?? null} closeUpperId={closeUpperId} onActivateUpper={id => setActiveUpperId(id, group)} onCloseUpper={setCloseUpperId} onMoveToTop={terminal => moveToTop(terminal, group)} onMoveToBottom={moveToBottom} />;
  const editors = <div className="editor-area" ref={editorAreaRef}>
    {mobile ? <EditorGroup project={project} group={0} mobile /> : <>
      {narrow && visibleGroups.length > 1 && <nav className="group-selector" aria-label="编辑器分组">{visibleGroups.map(group => <Button key={group} size="sm" variant={focused === group ? "secondary" : "ghost"} onClick={() => useWorkspaceView.setState(state => ({ projects: { ...state.projects, [project.id]: { ...state.projects[project.id]!, focused: group } } }))}>第 {group + 1} 组</Button>)}</nav>}
      {renderedGroups.length > 1 ? <Group orientation="horizontal" defaultLayout={splitLayout.defaultLayout} onLayoutChanged={splitLayout.onLayoutChanged}>{renderedGroups.map((group, index) => <Fragment key={group}>{index > 0 && <Separator className="workbench-separator vertical" />}<Panel id={group === 0 ? "editor-left" : group === 1 ? "editor-right" : `editor-${group}`} minSize={240}>{editorGroup(group)}</Panel></Fragment>)}</Group> : editorGroup(renderedGroups[0] ?? 0)}
    </>}
  </div>;
  return <EditorScopeProvider key={project.id} project={project}><TerminalRuntimeProvider key={project.id}><main className="workbench" aria-label={`${project.name} 工作台`}>
    <div className="workbench-title"><h1 className="workbench-title-name">{project.name}</h1><Button size="icon-sm" variant="ghost" aria-label="编辑项目" title="编辑项目" onClick={onEdit}><FolderKanban /></Button><span className="terminal-heading-spacer" /><div className="workbench-title-actions">{shell?.headerActions}</div></div>
    {(recoveryError || storageWarning) && <Alert><AlertDescription>{recoveryError ?? storageWarning}</AlertDescription></Alert>}
    {mobile ? <>
      <nav className="mobile-workbench-nav" aria-label="工作区视图">
        <Button variant={mobileView === "files" ? "secondary" : "ghost"} onClick={showFiles}><Files />文件</Button>
        <Button variant={mobileView === "editor" ? "secondary" : "ghost"} onClick={() => setMobileView("editor")}><FileText />编辑器</Button>
        <Button variant={mobileView === "terminal" ? "secondary" : "ghost"} onClick={() => setMobileView("terminal")}><TerminalSquare />终端</Button>
      </nav>
      <div className="mobile-workbench-content">{mobileView === "files" ? explorer : mobileView === "editor" ? editors : <TerminalWorkspace project={project} mobile />}</div>
    </> : <div className="workbench-body">
      <nav className="activity-bar" aria-label="活动栏">
        <Button variant="ghost" size="icon" aria-label="资源管理器" title="资源管理器" onClick={() => sidebarRef.current?.isCollapsed() ? sidebarRef.current?.expand() : sidebarRef.current?.collapse()}><Files /></Button>
        <Link className={buttonVariants({ variant: "ghost", size: "icon" })} to="/projects" aria-label="项目面板" title="项目面板"><FolderKanban /></Link>
        <Button variant="ghost" size="icon" aria-label="终端面板" title="终端面板" onClick={() => terminalRef.current?.isCollapsed() ? terminalRef.current?.expand() : terminalRef.current?.collapse()}><TerminalSquare /></Button>
      </nav>
      <Group orientation="horizontal" defaultLayout={sidebarLayout.defaultLayout} onLayoutChanged={sidebarLayout.onLayoutChanged}>
        <Panel id="sidebar" panelRef={sidebarRef} defaultSize="24%" minSize={200} maxSize="45%" collapsible collapsedSize={0} className="workbench-sidebar" onResize={(size, _id, previous) => { if (size.inPixels > 0 && (!previous || previous.inPixels === 0)) revealCurrentFile(); }}>{explorer}</Panel>
        <Separator className="workbench-separator vertical" />
        <Panel id="main" minSize="40%"><Group orientation="vertical" defaultLayout={verticalLayout.defaultLayout} onLayoutChanged={verticalLayout.onLayoutChanged}>
          <Panel id="editors" defaultSize="62%" minSize={160}>{editors}</Panel>
          <Separator className="workbench-separator horizontal" />
          <Panel id="terminal" panelRef={terminalRef} defaultSize="38%" minSize={110} collapsible collapsedSize={0}><Group orientation="horizontal" defaultLayout={bottomLayout.defaultLayout} onLayoutChanged={bottomLayout.onLayoutChanged}>{Array.from({ length: view?.lowerCount ?? 1 }, (_, index) => <Fragment key={index}>{index > 0 && <Separator className="workbench-separator vertical" />}<Panel id={`terminal-group-${index}`} minSize="15%"><TerminalWorkspace project={project} group={index as GroupIndex} upperIds={upperTerminals.map(item => item.id)} onMoveToTop={moveToTop} onMoveToBottom={id => moveToBottom(id, index as GroupIndex)} focusRequest={bottomFocusRequest} onHide={() => terminalRef.current?.collapse()} /></Panel></Fragment>)}</Group></Panel>
        </Group></Panel>
      </Group>
    </div>}
    <footer className="workbench-status"><span>{project.folders.find(folder => folder.id === project.main_folder_id)?.path ?? project.name}</span><span>{activeFile ? `${activeFile.path} · UTF-8` : `${project.folders.length} 个文件夹`}</span>{!mobile && <EditorLanguageStatus project={project} file={activeUpperId && !dismissed[activeUpperId] ? undefined : activeFile} />}<div className="status-controls"><Button size="icon" variant="ghost" aria-label="切换资源管理器" title="切换资源管理器" onClick={() => mobile ? showFiles() : sidebarRef.current?.isCollapsed() ? sidebarRef.current?.expand() : sidebarRef.current?.collapse()}><PanelLeft /></Button><Button size="icon" variant="ghost" aria-label="切换终端面板" title="切换终端面板" onClick={() => mobile ? setMobileView("terminal") : terminalRef.current?.isCollapsed() ? terminalRef.current?.expand() : terminalRef.current?.collapse()}><PanelBottom /></Button></div></footer>
  </main></TerminalRuntimeProvider></EditorScopeProvider>;
}

function EditorGroup({ project, group, mobile = false, upperTerminals = [], activeUpperId = null, closeUpperId = null, onActivateUpper, onCloseUpper, onMoveToTop, onMoveToBottom }: {
  project: Project; group: GroupIndex; mobile?: boolean; upperTerminals?: Terminal[]; activeUpperId?: string | null; closeUpperId?: string | null;
  onActivateUpper?(id: string | null): void; onCloseUpper?(id: string | null): void;
  onMoveToTop?(terminal: Terminal): void; onMoveToBottom?(id: string): void;
}) {
  const { scope } = useTerminalRuntime();
  const editors = useEditorScope();
  const [closeError, setCloseError] = useState<string | null>(null);
  const dismissed = useTerminalView(state => state.dismissed);
  const [refreshError, setRefreshError] = useState<string | null>(null);
  const view = useWorkspaceView(state => state.projects[project.id]);
  const activate = useWorkspaceView(state => state.activate);
  const close = useWorkspaceView(state => state.close);
  const split = useWorkspaceView(state => state.split);
  const move = useWorkspaceView(state => state.move);
  const unsplit = useWorkspaceView(state => state.unsplit);
  const activateFile = (file: OpenFile) => { if (mobile) useWorkspaceView.getState().openMobile(project.id, file); else activate(project.id, file, group); };
  const closeFile = (file: OpenFile) => { if (mobile) useWorkspaceView.getState().closeMobile(project.id, file); else close(project.id, file, group); };
  const files = (mobile ? view?.mobileFiles : view?.groups[group]) ?? [];
  const active = files.find(file => fileKey(file) === (mobile ? view?.mobileActive : view?.active[group]));
  const visibleUpper = !mobile ? upperTerminals.map(item => scope.entry(item.id)?.terminal ?? item).filter(item => terminalVisible(item, dismissed)).sort((a, b) => { const order = view?.terminalOrder ?? []; const left = order.indexOf(a.id), right = order.indexOf(b.id); return (left < 0 ? 1000 : left) - (right < 0 ? 1000 : right); }) : [];
  const activeTerminal = visibleUpper.find(item => item.id === activeUpperId);
  const activeValue = activeTerminal?.id ?? (active ? fileKey(active) : "empty");
  async function refreshTerminals() {
    setRefreshError(null);
    try { const items = await api.terminals(new AbortController().signal); for (const item of items) if (scope.entry(item.id)) scope.ensure(item); }
    catch (reason) { setRefreshError(reason instanceof Error ? reason.message : "无法读取终端状态。"); }
  }
  function dragStart(event: React.DragEvent, file: OpenFile) { event.dataTransfer.setData("application/x-persistty-tab", JSON.stringify({ file, group, projectId: project.id })); event.dataTransfer.effectAllowed = "move"; }
  function drop(event: DragEvent) {
    if (!event.dataTransfer) return;
    if ((event.dataTransfer.types.includes("application/x-persistty-lower-terminal") || event.dataTransfer.types.includes("application/x-persistty-upper-terminal"))) {
      event.preventDefault(); event.stopPropagation();
      try {
        const terminal = decodeTerminal(JSON.parse(event.dataTransfer.getData("application/x-persistty-lower-terminal") || event.dataTransfer.getData("application/x-persistty-upper-terminal")));
        if (terminal.project_id === project.id) { onMoveToTop?.(terminal); const before = event.target instanceof Element ? event.target.closest("[data-terminal-id]")?.getAttribute("data-terminal-id") : null; if (before && before !== terminal.id) useWorkspaceView.getState().orderTerminal(project.id, terminal.id, before, visibleUpper.map(item => item.id)); }
      } catch { /* Unrelated drag data. */ }
      return;
    }
    if (!event.dataTransfer.types.includes("application/x-persistty-tab")) return;
    event.preventDefault(); event.stopPropagation();
    try {
      const value: unknown = JSON.parse(event.dataTransfer.getData("application/x-persistty-tab"));
      if (!value || typeof value !== "object" || !("file" in value) || !("group" in value)) return;
      const { file, group: source } = value;
      if (!("projectId" in value) || value.projectId !== project.id || (source !== 0 && source !== 1 && source !== 2 && source !== 3) || typeof file !== "object" || file === null || !("folderId" in file) || !("path" in file) || typeof file.folderId !== "string" || typeof file.path !== "string") return;
      const target = event.target instanceof Element ? event.target.closest("[data-file-key]")?.getAttribute("data-file-key") : null;
      const before = files.find(item => fileKey(item) === target);
      move(project.id, { folderId: file.folderId, path: file.path }, group, source, before);
    } catch { /* Unrelated drag data. */ }
  }
  const dropTarget = useDropTarget(mobile ? [] : ["application/x-persistty-tab", "application/x-persistty-lower-terminal", "application/x-persistty-upper-terminal"], drop);
  return <section ref={dropTarget} className="editor-group" aria-label={mobile ? "编辑器" : group === 0 ? "左侧编辑器" : group === 1 ? "右侧编辑器" : `第 ${group + 1} 组编辑器`}>
    <Tabs value={activeValue} onValueChange={value => {
      const terminal = visibleUpper.find(item => item.id === value);
      if (terminal) onActivateUpper?.(terminal.id);
      else { const file = files.find(item => fileKey(item) === value); if (file) { onActivateUpper?.(null); activateFile(file); } }
    }} className="editor-tab-layout"><div className="editor-tab-row"><TabsList variant="line" className="editor-tabs" aria-label="文件标签">
      {files.map((file, index) => <div key={fileKey(file)} className={`editor-tab ${active && fileKey(active) === fileKey(file) ? "active" : ""}`} data-file-key={fileKey(file)} draggable={!mobile} onDragStart={event => dragStart(event, file)}>
        <TabsTrigger value={fileKey(file)} className="editor-tab-label" onClick={() => { onActivateUpper?.(null); activateFile(file); }}><FileTypeIcon path={file.path} /><span className="truncate" title={editors.buffers.get(fileKey(file))?.dirty ? `${file.path} · 未保存` : file.path}>{file.path.split("/").at(-1)}</span></TabsTrigger>
        {index > 0 && !mobile && <Button size="icon-xs" variant="ghost" aria-label={`左移标签 ${file.path}`} onClick={() => move(project.id, file, group, group, files[index - 1])}>←</Button>}
        <Button size="icon" variant="ghost" className={`editor-tab-close ${editors.buffers.get(fileKey(file))?.dirty ? "is-dirty" : ""}`} aria-label={`关闭 ${file.path}`} title="关闭文件标签" onClick={() => {
          const buffer = editors.buffers.get(fileKey(file));
          const references = allOpenFiles(view).filter(item => buffer ? buffer.aliases.has(fileKey(item)) : fileKey(item) === fileKey(file)).length;
          if (references > 1) { closeFile(file); return; }
          void editors.protect(file).then(ok => { if (ok) closeFile(file); else setCloseError("草稿保存失败，文件仍保持打开，请导出内容。"); });
        }}>{editors.buffers.get(fileKey(file))?.dirty && <span className="editor-dirty-dot" role="img" aria-label="未保存" />}<X /></Button>
      </div>)}
      {visibleUpper.map(item => <TerminalTab key={item.id} terminal={item} region={visibleUpper} active={activeTerminal?.id === item.id} position="top" onActivate={() => onActivateUpper?.(item.id)} onRefresh={() => void refreshTerminals()} onMove={item => onMoveToBottom?.(item.id)} moves={[...groupIndexes.filter(i => i !== group).map(i => ({ label: `移到上方第 ${i + 1} 组`, run: (terminal: Terminal) => useWorkspaceView.getState().placeTerminal(project.id, terminal.id, { region: "top", group: i }) })), ...groupIndexes.map(i => ({ label: `移到下方第 ${i + 1} 组`, run: (terminal: Terminal) => useWorkspaceView.getState().placeTerminal(project.id, terminal.id, { region: "bottom", group: i }) }))]} />)}
    </TabsList><div className="editor-tab-actions">
      {active && !activeTerminal && <EditorFileActions project={project} file={active} />}
      {!mobile && active && !activeTerminal && <Button size="icon" variant="ghost" aria-label="向右移动文件" title="向右移动文件" onClick={() => move(project.id, active, ((group + 1) % 4) as GroupIndex, group)}><PanelRight /></Button>}
      {!mobile && active && visibleEditorCount(view) < 4 && <Button size="icon" variant="ghost" aria-label="向右拆分编辑器" title="向右拆分编辑器" onClick={() => split(project.id, active)}><PanelRight /></Button>}
      {!mobile && view?.split && group !== 0 && <Button size="icon" variant="ghost" aria-label="合并编辑器" title="合并编辑器" onClick={() => unsplit(project.id)}><PanelRight /></Button>}
    </div></div>
    {closeError && <p role="alert" className="terminal-inline-error">{closeError}</p>}
    {refreshError && <p role="alert" className="terminal-inline-error">{refreshError}</p>}
    <TabsContent value={activeValue} className="editor-tab-content">{activeTerminal ? <div className="editor-upper-terminal"><TerminalSession key={activeTerminal.id} terminal={activeTerminal} closeRequested={closeUpperId === activeTerminal.id} onCloseRequestHandled={() => onCloseUpper?.(null)} /></div> : active ? <FileEditor key={fileKey(active)} project={project} file={active} mobile={mobile} /> : <div className="editor-empty"><Files className="size-9" /><span>从资源管理器打开文件</span></div>}</TabsContent></Tabs>
  </section>;
}


function visibleEditorCount(view: import("./workspace-view").ProjectView | undefined): number { return groupIndexes.filter(i => i === 0 || !!view?.groups[i].length || Object.values(view?.terminals ?? {}).some(position => position.region === "top" && position.group === i)).length; }
