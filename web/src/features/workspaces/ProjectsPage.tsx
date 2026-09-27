import { useState } from "react";
import { Link, useNavigate, useParams } from "react-router";
import { Folder, ArrowUpRight, RefreshCw, ExternalLink, PanelTop } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Empty, EmptyHeader, EmptyTitle, EmptyMedia, EmptyDescription } from "@/components/ui/empty";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from "@/components/ui/dialog";
import { api, ApiError } from "@/lib/api/client";
import { useResource } from "@/lib/api/use-resource";
import type { Project } from "@/lib/api/decoder";
import { Failure, Loading } from "@/components/Feedback";

export function ProjectsPage() {
  const { resource, refresh } = useResource(api.projects);
  const [selected, setSelected] = useState<Project | null>(null);
  const navigate = useNavigate();
  return <main className="page-main">
    <div className="page-heading"><h1>项目</h1><Button size="icon" variant="ghost" aria-label="刷新项目" title="刷新项目" disabled={resource.status === "loading"} onClick={refresh}><RefreshCw /></Button></div>
    {resource.status === "loading" && <Loading />}
    {resource.status === "error" && <Failure error={resource.error} retry={refresh} />}
    {resource.status === "ready" && (resource.data.length === 0 ? <Empty><EmptyHeader><EmptyMedia variant="icon"><Folder /></EmptyMedia><EmptyTitle>暂无项目</EmptyTitle></EmptyHeader></Empty> :
      <ul className="resource-list">{resource.data.map(project => <li key={project.id}>
        <Button variant="ghost" className="resource-row" onClick={() => setSelected(project)}>
          <Folder data-icon="inline-start" /><span className="resource-text"><span>{project.name}</span><span className="resource-path">{project.folders.find(folder => folder.id === project.main_folder_id)?.path}</span></span>
          <Badge variant="secondary">{project.folders.length} 个文件夹</Badge><ArrowUpRight data-icon="inline-end" />
        </Button>
      </li>)}</ul>)}
    <Dialog open={selected !== null} onOpenChange={open => { if (!open) setSelected(null); }}>
      <DialogContent><DialogHeader><DialogTitle>打开项目</DialogTitle><DialogDescription>{selected?.name}</DialogDescription></DialogHeader>
        <DialogFooter><Button variant="outline" onClick={() => { if (selected) { const id = selected.id; setSelected(null); navigate(`/projects/${encodeURIComponent(id)}`); } }}><PanelTop data-icon="inline-start" />当前标签页</Button>
          {selected && <Button asChild><a href={`/projects/${encodeURIComponent(selected.id)}`} target="_blank" rel="noopener noreferrer" onClick={() => setSelected(null)}><ExternalLink data-icon="inline-start" />新标签页</a></Button>}
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
  const { resource, refresh } = useResource(load);
  return <main className="page-main">
    {resource.status === "loading" && <Loading />}
    {resource.status === "error" && (resource.error instanceof ApiError && resource.error.status === 404 ?
      <Empty><EmptyHeader><EmptyMedia variant="icon"><Folder /></EmptyMedia><EmptyTitle>项目不存在</EmptyTitle><EmptyDescription>此项目已移除或地址无效。</EmptyDescription></EmptyHeader><div className="flex flex-wrap justify-center gap-2"><Button asChild variant="outline"><Link to="/projects">项目面板</Link></Button><Button asChild><Link to="/terminals">终端</Link></Button></div></Empty> : <Failure error={resource.error} retry={refresh} />)}
    {resource.status === "ready" && <>
      <div className="page-heading"><h1>{resource.data.name}</h1><Badge variant="outline">{resource.data.folders.length} 个文件夹</Badge></div>
      <ul className="folder-list">{resource.data.folders.map(folder => <li key={folder.id}><Folder className="size-4 shrink-0" /><span className="break-all font-mono text-sm">{folder.path}</span>{folder.id === resource.data.main_folder_id && <Badge variant="secondary">主文件夹</Badge>}</li>)}</ul>
      <Button asChild variant="outline"><Link to="/terminals">查看终端<ArrowUpRight data-icon="inline-end" /></Link></Button>
    </>}
  </main>;
}
