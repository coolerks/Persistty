import { useEffect, useState } from "react";
import { Link, useNavigate, useParams } from "react-router";
import { Folder, ArrowUpRight, RefreshCw, ExternalLink, PanelTop, Plus, Pencil, Trash2 } from "lucide-react";
import { Button, buttonVariants } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Empty, EmptyHeader, EmptyTitle, EmptyMedia, EmptyDescription } from "@/components/ui/empty";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from "@/components/ui/dialog";
import { api, ApiError } from "@/lib/api/client";
import { useResource } from "@/lib/api/use-resource";
import type { Project } from "@/lib/api/decoder";
import { Failure, Loading } from "@/components/Feedback";
import { useAuth } from "@/features/auth/auth-context";
import { errorMessage } from "@/lib/api/client";
import { ProjectEditor } from "./ProjectEditor";
import { DraftRecovery } from "./DraftRecovery";
import { ProjectWorkbench } from "./ProjectWorkbench";

export function ProjectsPage() {
  const { resource, refresh } = useResource(api.projects);
  const [selected, setSelected] = useState<Project | null>(null);
  const [editing, setEditing] = useState<Project | "new" | null>(null);
  const [removing, setRemoving] = useState<Project | null>(null);
  const [mutationError, setMutationError] = useState<string | null>(null);
  const [pending, setPending] = useState(false);
  const auth = useAuth();
  const navigate = useNavigate();
  async function remove() {
    if (!removing || auth.state.status !== "authenticated" || pending) return;
    setPending(true); setMutationError(null);
    try {
      await api.deleteProject(removing, auth.state.session.csrf_token, new AbortController().signal);
      setRemoving(null); refresh();
    } catch (error: unknown) { setMutationError(errorMessage(error)); }
    finally { setPending(false); }
  }
  return <main className="page-main projects-page">
    <header className="projects-welcome"><h1>Persistty</h1><p>打开项目，继续工作。</p></header>
    <div className="projects-columns">
      <section className="projects-start" aria-labelledby="projects-start-title"><h2 id="projects-start-title">开始使用</h2>
        <Button variant="ghost" onClick={() => setEditing("new")}><Plus />新建项目</Button>
        <p>将服务器上的文件夹组织为项目。</p>
      </section>
    <section className="projects-panel" aria-label="项目列表">
    <div className="page-heading"><h2>已有项目</h2><Button size="icon-sm" variant="ghost" aria-label="刷新项目" title="刷新项目" disabled={resource.status === "loading"} onClick={refresh}><RefreshCw /></Button></div>
    {resource.status === "loading" && <Loading />}
    {resource.status === "error" && <Failure error={resource.error} retry={refresh} />}
    {resource.status === "ready" && (resource.data.length === 0 ? <Empty><EmptyHeader><EmptyMedia variant="icon"><Folder /></EmptyMedia><EmptyTitle>暂无项目</EmptyTitle></EmptyHeader></Empty> :
      <ul className="resource-list project-list">{resource.data.map(project => <li key={project.id} className="project-item">
        <Button variant="ghost" className="resource-row" aria-label={`打开 ${project.name}`} onClick={() => setSelected(project)}>
          <span className="project-icon"><Folder /></span><span className="resource-text"><span>{project.name}</span><span className="resource-path">{project.folders.find(folder => folder.id === project.main_folder_id)?.path}</span></span>
          <Badge variant="secondary">{project.folders.length} 个文件夹</Badge><ArrowUpRight data-icon="inline-end" />
        </Button><Button size="icon" variant="ghost" title="编辑项目" aria-label={`编辑 ${project.name}`} onClick={() => setEditing(project)}><Pencil /></Button><Button size="icon" variant="ghost" title="移除项目" aria-label={`移除 ${project.name}`} onClick={() => setRemoving(project)}><Trash2 /></Button>
      </li>)}</ul>)}
    </section></div>
    {editing && <ProjectEditor key={editing === "new" ? "new" : editing.id} project={editing === "new" ? undefined : editing} onClose={() => setEditing(null)} onSaved={() => { setEditing(null); refresh(); }} />}
    <Dialog open={removing !== null} onOpenChange={open => { if (!open && !pending) { setRemoving(null); setMutationError(null); } }}><DialogContent><DialogHeader><DialogTitle>移除项目</DialogTitle><DialogDescription>仅移除“{removing?.name}”的项目配置，不删除磁盘文件或终止已有终端。</DialogDescription></DialogHeader>
      {mutationError && <p role="alert" className="text-sm text-destructive">{mutationError}</p>}
      <DialogFooter><Button variant="outline" disabled={pending} onClick={() => setRemoving(null)}>取消</Button><Button variant="destructive" disabled={pending} onClick={() => void remove()}>移除项目</Button></DialogFooter>
    </DialogContent></Dialog>
    <Dialog open={selected !== null} onOpenChange={open => { if (!open) setSelected(null); }}>
      <DialogContent><DialogHeader><DialogTitle>打开项目</DialogTitle><DialogDescription>{selected?.name}</DialogDescription></DialogHeader>
        <DialogFooter><Button variant="outline" onClick={() => { if (selected) { const id = selected.id; setSelected(null); navigate(`/projects/${encodeURIComponent(id)}`); } }}><PanelTop data-icon="inline-start" />当前标签页</Button>
          {selected && <a className={buttonVariants()} href={`/projects/${encodeURIComponent(selected.id)}`} target="_blank" rel="noopener noreferrer" onClick={() => setSelected(null)}><ExternalLink data-icon="inline-start" />新标签页</a>}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </main>;
}

export function ProjectRoute() {
  const { projectId = "" } = useParams();
  return <ProjectPage key={projectId} projectId={projectId} />;
}
function ProjectPage({ projectId }: { projectId: string }) {
  const [load] = useState(() => (signal: AbortSignal) => api.project(projectId, signal));
  const { resource, refresh, refreshQuietly } = useResource(load);
  useEffect(() => { const timer = setInterval(refreshQuietly, 15000); return () => clearInterval(timer); }, [refreshQuietly]);
  const [editing, setEditing] = useState(false);
  if (resource.status === "loading") return <main className="page-main"><Link className={buttonVariants({ variant: "ghost" })} to="/projects">Persistty</Link><Loading /></main>;
  if (resource.status === "error") return <main className="page-main">{resource.error instanceof ApiError && resource.error.status === 404 ?
    <Empty><EmptyHeader><EmptyMedia variant="icon"><Folder /></EmptyMedia><EmptyTitle>项目不存在</EmptyTitle><EmptyDescription>此项目已移除或地址无效。</EmptyDescription></EmptyHeader><div className="flex flex-wrap justify-center gap-2"><Link className={buttonVariants({ variant: "outline" })} to="/projects">项目面板</Link><Link className={buttonVariants()} to="/terminals">终端</Link></div><DraftRecovery projectId={projectId} /></Empty> : <Failure error={resource.error} retry={refresh} />}</main>;
  return <><ProjectWorkbench project={resource.data} onEdit={() => setEditing(true)} />
    {editing && <ProjectEditor project={resource.data} onClose={() => setEditing(false)} onSaved={() => { setEditing(false); refreshQuietly(); }} />}
  </>;
}
