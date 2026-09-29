import { useRef, useState } from "react";
import { FolderPlus, Plus, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { api, errorMessage } from "@/lib/api/client";
import type { Project } from "@/lib/api/decoder";
import { useAuth } from "@/features/auth/auth-context";
import { DirectoryPicker } from "./DirectoryPicker";

type Row = { id: string | null; path: string };

export function ProjectEditor({ project, onClose, onSaved }: { project: Project | undefined; onClose(): void; onSaved(project: Project): void }) {
  const auth = useAuth();
  const [name, setName] = useState(project?.name ?? "");
  const [rows, setRows] = useState<Row[]>(project?.folders.map(folder => ({ ...folder })) ?? []);
  const [mainPath, setMainPath] = useState(project?.folders.find(folder => folder.id === project.main_folder_id)?.path ?? "");
  const [picker, setPicker] = useState(false);
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const request = useRef<AbortController | null>(null);

  async function save() {
    if (pending || auth.state.status !== "authenticated") return;
    const controller = new AbortController(); request.current = controller;
    setPending(true); setError(null);
    try {
      let saved: Project;
      if (project) {
        const existing = rows.filter(row => row.id !== null);
        const added = rows.filter(row => row.id === null);
        const currentMain = existing.find(row => row.path === mainPath)?.id;
        if (currentMain) {
          saved = await api.updateProject(project, { name, add_paths: added.map(row => row.path), remove_folder_ids: project.folders.filter(folder => !existing.some(row => row.id === folder.id)).map(folder => folder.id), main_folder_id: currentMain }, auth.state.session.csrf_token, controller.signal);
        } else {
          saved = await api.updateProjectWithNewMain(project, { name, add_paths: added.map(row => row.path), remove_folder_ids: project.folders.filter(folder => !existing.some(row => row.id === folder.id)).map(folder => folder.id), main_added_index: added.findIndex(row => row.path === mainPath) }, auth.state.session.csrf_token, controller.signal);
        }
      } else {
        saved = await api.createProject(name, rows.map(row => row.path), rows.findIndex(row => row.path === mainPath), auth.state.session.csrf_token, controller.signal);
      }
      if (!controller.signal.aborted) onSaved(saved);
    } catch (cause: unknown) {
      if (!controller.signal.aborted) setError(errorMessage(cause));
    } finally { request.current = null; if (!controller.signal.aborted) setPending(false); }
  }

  return <><Dialog open onOpenChange={open => { if (!open && !pending && !picker) { request.current?.abort(); onClose(); } }}><DialogContent className="max-w-xl">
    <DialogHeader><DialogTitle>{project ? "编辑项目" : "新建项目"}</DialogTitle><DialogDescription>文件夹路径位于 Persistty 服务器。</DialogDescription></DialogHeader>
    <div className="grid gap-2"><Label htmlFor="project-name">项目名称</Label><Input id="project-name" value={name} maxLength={200} onChange={event => setName(event.target.value)} /></div>
    <div className="grid gap-2"><div className="flex items-center justify-between"><Label>文件夹</Label><Button size="sm" variant="outline" onClick={() => setPicker(true)}><Plus data-icon="inline-start" />添加</Button></div>
      <ul className="max-h-52 space-y-1 overflow-y-auto">{rows.map(row => <li key={row.path} className="flex min-w-0 items-center gap-2 border-b py-1">
        <input type="radio" name="main-folder" aria-label={`将 ${row.path} 设为主文件夹`} checked={mainPath === row.path} onChange={() => setMainPath(row.path)} />
        <span className="min-w-0 flex-1 truncate font-mono text-sm" title={row.path}>{row.path}</span>
        <Button type="button" variant="ghost" size="icon" aria-label={`移除 ${row.path}`} title="移除关联" onClick={() => { setRows(current => current.filter(item => item.path !== row.path)); if (mainPath === row.path) setMainPath(""); }}><Trash2 /></Button>
      </li>)}</ul>
      {rows.length === 0 && <div className="flex items-center gap-2 text-sm text-muted-foreground"><FolderPlus className="size-4" />尚未选择文件夹</div>}
    </div>
    {error && <p role="alert" className="text-sm text-destructive">{error}</p>}
    <DialogFooter><Button variant="outline" disabled={pending} onClick={onClose}>取消</Button><Button disabled={pending || !name.trim() || rows.length === 0 || !rows.some(row => row.path === mainPath)} onClick={() => void save()}>{pending ? "保存中" : "保存"}</Button></DialogFooter>
  </DialogContent></Dialog>
  <DirectoryPicker open={picker} onOpenChange={setPicker} onSelect={path => { if (!rows.some(row => row.path === path)) { setRows(current => [...current, { id: null, path }]); if (!mainPath) setMainPath(path); } }} />
  </>;
}
