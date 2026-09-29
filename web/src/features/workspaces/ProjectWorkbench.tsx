import { lazy, Suspense, useEffect, useState } from "react";
import { Link } from "react-router";
import { Group, Panel, Separator, useDefaultLayout, usePanelRef } from "react-resizable-panels";
import { Files, FolderKanban, PanelBottom, PanelLeft, PanelRight, TerminalSquare, X, RefreshCw, FileText, Download } from "lucide-react";
import { Button, buttonVariants } from "@/components/ui/button";
import { api, ApiError, fileDownloadURL } from "@/lib/api/client";
import { useResource } from "@/lib/api/use-resource";
import type { Project } from "@/lib/api/decoder";
import { Failure, Loading } from "@/components/Feedback";
import { Explorer } from "./Explorer";
import { fileKey, useWorkspaceView } from "./workspace-view";
import type { OpenFile } from "./workspace-view";

const DesktopEditor = lazy(() => import("./DesktopEditor"));

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
  const mobile = useMobile();
  const view = useWorkspaceView(state => state.projects[project.id]);
  const open = useWorkspaceView(state => state.open);
  const [mobileView, setMobileView] = useState<"files" | "editor" | "terminal">("files");
  const sidebarRef = usePanelRef();
  const terminalRef = usePanelRef();
  const sidebarLayout = useDefaultLayout({ id: `persistty-sidebar-v1-${project.id}`, storage: localStorage });
  const verticalLayout = useDefaultLayout({ id: `persistty-vertical-v1-${project.id}`, storage: localStorage });
  const splitLayout = useDefaultLayout({ id: `persistty-editor-split-v1-${project.id}`, storage: localStorage });
  const focused = view?.focused ?? 0;
  const activeKey = view?.active[focused] ?? null;
  const activeFile = view?.groups[focused].find(file => fileKey(file) === activeKey);
  const explorer = <Explorer project={project} onFile={(folderId, path) => { open(project.id, { folderId, path }, mobile ? 0 : undefined); if (mobile) setMobileView("editor"); }} />;
  const editors = <div className="editor-area">
    {view?.split && !mobile ? <Group orientation="horizontal" defaultLayout={splitLayout.defaultLayout} onLayoutChanged={splitLayout.onLayoutChanged}>
      <Panel id="editor-left" defaultSize="50%" minSize="22%"><EditorGroup project={project} group={0} /></Panel>
      <Separator className="workbench-separator vertical" />
      <Panel id="editor-right" defaultSize="50%" minSize="22%"><EditorGroup project={project} group={1} /></Panel>
    </Group> : <EditorGroup project={project} group={mobile ? focused : 0} mobile={mobile} />}
  </div>;
  return <main className="workbench" aria-label={`${project.name} 工作台`}>
    <div className="workbench-title"><h1 className="workbench-title-name">{project.name}</h1><Button size="icon" variant="ghost" aria-label="编辑项目" title="编辑项目" onClick={onEdit}><FolderKanban /></Button></div>
    {mobile ? <>
      <nav className="mobile-workbench-nav" aria-label="工作区视图">
        <Button variant={mobileView === "files" ? "secondary" : "ghost"} onClick={() => setMobileView("files")}><Files />文件</Button>
        <Button variant={mobileView === "editor" ? "secondary" : "ghost"} onClick={() => setMobileView("editor")}><FileText />编辑器</Button>
        <Button variant={mobileView === "terminal" ? "secondary" : "ghost"} onClick={() => setMobileView("terminal")}><TerminalSquare />终端</Button>
      </nav>
      <div className="mobile-workbench-content">{mobileView === "files" ? explorer : mobileView === "editor" ? editors : <TerminalPane />}</div>
    </> : <div className="workbench-body">
      <nav className="activity-bar" aria-label="活动栏">
        <Button variant="ghost" size="icon" aria-label="资源管理器" title="资源管理器" onClick={() => sidebarRef.current?.isCollapsed() ? sidebarRef.current?.expand() : sidebarRef.current?.collapse()}><Files /></Button>
        <Link className={buttonVariants({ variant: "ghost", size: "icon" })} to="/projects" aria-label="项目面板" title="项目面板"><FolderKanban /></Link>
        <Button variant="ghost" size="icon" aria-label="终端面板" title="终端面板" onClick={() => terminalRef.current?.isCollapsed() ? terminalRef.current?.expand() : terminalRef.current?.collapse()}><TerminalSquare /></Button>
      </nav>
      <Group orientation="horizontal" defaultLayout={sidebarLayout.defaultLayout} onLayoutChanged={sidebarLayout.onLayoutChanged}>
        <Panel id="sidebar" panelRef={sidebarRef} defaultSize="24%" minSize={200} maxSize="45%" collapsible collapsedSize={0} className="workbench-sidebar">{explorer}</Panel>
        <Separator className="workbench-separator vertical" />
        <Panel id="main" minSize="40%"><Group orientation="vertical" defaultLayout={verticalLayout.defaultLayout} onLayoutChanged={verticalLayout.onLayoutChanged}>
          <Panel id="editors" defaultSize="62%" minSize={160}>{editors}</Panel>
          <Separator className="workbench-separator horizontal" />
          <Panel id="terminal" panelRef={terminalRef} defaultSize="38%" minSize={110} collapsible collapsedSize={0}><TerminalPane onHide={() => terminalRef.current?.collapse()} /></Panel>
        </Group></Panel>
      </Group>
    </div>}
    <footer className="workbench-status"><span>{project.folders.find(folder => folder.id === project.main_folder_id)?.path ?? project.name}</span><span>{activeFile ? `${activeFile.path} · UTF-8` : `${project.folders.length} 个文件夹`}</span><div className="status-controls"><Button size="icon" variant="ghost" aria-label="切换资源管理器" title="切换资源管理器" onClick={() => mobile ? setMobileView("files") : sidebarRef.current?.isCollapsed() ? sidebarRef.current?.expand() : sidebarRef.current?.collapse()}><PanelLeft /></Button><Button size="icon" variant="ghost" aria-label="切换终端面板" title="切换终端面板" onClick={() => mobile ? setMobileView("terminal") : terminalRef.current?.isCollapsed() ? terminalRef.current?.expand() : terminalRef.current?.collapse()}><PanelBottom /></Button></div></footer>
  </main>;
}

function EditorGroup({ project, group, mobile = false }: { project: Project; group: 0 | 1; mobile?: boolean }) {
  const view = useWorkspaceView(state => state.projects[project.id]);
  const activate = useWorkspaceView(state => state.activate);
  const close = useWorkspaceView(state => state.close);
  const split = useWorkspaceView(state => state.split);
  const move = useWorkspaceView(state => state.move);
  const unsplit = useWorkspaceView(state => state.unsplit);
  const files = view?.groups[group] ?? [];
  const active = files.find(file => fileKey(file) === view?.active[group]);
  function dragStart(event: React.DragEvent, file: OpenFile) { event.dataTransfer.setData("application/x-persistty-tab", JSON.stringify({ file, group })); event.dataTransfer.effectAllowed = "move"; }
  function drop(event: React.DragEvent) {
    if (!event.dataTransfer.types.includes("application/x-persistty-tab")) return;
    event.preventDefault();
    try {
      const value: unknown = JSON.parse(event.dataTransfer.getData("application/x-persistty-tab"));
      if (!value || typeof value !== "object" || !("file" in value) || !("group" in value)) return;
      const { file, group: source } = value;
      if ((source !== 0 && source !== 1) || typeof file !== "object" || file === null || !("folderId" in file) || !("path" in file) || typeof file.folderId !== "string" || typeof file.path !== "string") return;
      if (source !== group) move(project.id, { folderId: file.folderId, path: file.path }, group);
    } catch { /* Unrelated drag data. */ }
  }
  return <section className="editor-group" aria-label={mobile ? "编辑器" : group === 0 ? "左侧编辑器" : "右侧编辑器"} onDragOver={event => { if (!mobile && event.dataTransfer.types.includes("application/x-persistty-tab")) event.preventDefault(); }} onDrop={mobile ? undefined : drop}>
    <div className="editor-tabs" role="tablist" aria-label="文件标签">
      {files.map(file => <div key={fileKey(file)} className={`editor-tab ${active && fileKey(active) === fileKey(file) ? "active" : ""}`} draggable onDragStart={event => dragStart(event, file)}>
        <Button variant="ghost" role="tab" aria-selected={active ? fileKey(active) === fileKey(file) : false} className="editor-tab-label" onClick={() => activate(project.id, file, group)}><FileText className="size-4 shrink-0" /><span className="truncate">{file.path.split("/").at(-1)}</span></Button>
        <Button size="icon" variant="ghost" className="editor-tab-close" aria-label={`关闭 ${file.path}`} title="关闭文件标签" onClick={() => close(project.id, file, group)}><X /></Button>
      </div>)}
      <span className="editor-tab-spacer" />
      {!mobile && active && !view?.split && <Button size="icon" variant="ghost" aria-label="向右拆分编辑器" title="向右拆分编辑器" onClick={() => split(project.id, active)}><PanelRight /></Button>}
      {!mobile && view?.split && group === 1 && <Button size="icon" variant="ghost" aria-label="合并编辑器" title="合并编辑器" onClick={() => unsplit(project.id)}><PanelRight /></Button>}
    </div>
    {active ? <FileEditor key={fileKey(active)} project={project} file={active} /> : <div className="editor-empty"><Files className="size-9" /><span>从资源管理器打开文件</span></div>}
  </section>;
}

function FileEditor({ project, file }: { project: Project; file: OpenFile }) {
  const mobile = useMobile();
  const [load] = useState(() => (signal: AbortSignal) => api.content(project.id, file.folderId, project.version, file.path, signal));
  const { resource, refresh } = useResource(load);
  return <div className="file-editor"><div className="editor-breadcrumb"><span className="truncate">{project.folders.find(folder => folder.id === file.folderId)?.path}/{file.path}</span><Button size="icon" variant="ghost" aria-label="刷新文件" title="刷新文件" onClick={refresh}><RefreshCw /></Button><a className={buttonVariants({ variant: "ghost", size: "icon" })} aria-label="下载文件" title="下载文件" href={fileDownloadURL(project.id, file.folderId, project.version, file.path)}><Download /></a></div>
    {resource.status === "loading" && <div className="editor-feedback"><Loading /></div>}
    {resource.status === "error" && <div className="editor-feedback">{resource.error instanceof ApiError && resource.error.status === 415 ? <p>二进制或超出编辑大小的文件，请下载查看。</p> : <Failure error={resource.error} retry={refresh} />}</div>}
    {resource.status === "ready" && <div className="editor-surface">{mobile ? <textarea aria-label={`${file.path} 内容`} readOnly value={resource.data.content} spellCheck={false} /> : <Suspense fallback={<Loading />}><DesktopEditor projectId={project.id} folderId={file.folderId} path={file.path} content={resource.data.content} /></Suspense>}</div>}
  </div>;
}

function TerminalPane({ onHide }: { onHide?(): void }) {
  return <section className="terminal-pane" aria-label="终端面板"><div className="terminal-heading"><TerminalSquare className="size-4" /><strong>终端</strong><span className="terminal-heading-spacer" />{onHide && <Button size="icon" variant="ghost" aria-label="收起终端面板" title="收起终端面板" onClick={onHide}><X /></Button>}</div><div className="terminal-unavailable"><TerminalSquare className="size-6" /><span>终端服务尚未接入。已有会话不会因关闭此面板而终止。</span></div></section>;
}
