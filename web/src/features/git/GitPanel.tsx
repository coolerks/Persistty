import { useCallback, useEffect, useRef, useState } from "react";
import { Maximize, Minimize } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Empty, EmptyHeader, EmptyTitle, EmptyDescription } from "@/components/ui/empty";
import { Loading } from "@/components/Feedback";
import { useAuth } from "@/features/auth/auth-context";
import { useEditorScope } from "@/features/workspaces/editor-context";
import { ReadOnlyComparison } from "@/features/workspaces/ReadOnlyComparison";
import { errorMessage } from "@/lib/api/client";
import { searchGitAPI, type GitCompareInput } from "@/lib/api/search-git-client";
import type { Project } from "@/lib/api/decoder";
import type { Repository } from "@/lib/api/search-git-decoder";
export function GitPanel({ project, mobile, visible }: { project: Project; mobile: boolean; visible: boolean }) {
  const auth = useAuth(), scope = useEditorScope();
  const csrf = auth.state.status === "authenticated" ? auth.state.session.csrf_token : "";
  const [repositories, setRepositories] = useState<Repository[]>([]), [repoID, setRepoID] = useState("");
  const [status, setStatus] = useState<Awaited<ReturnType<typeof searchGitAPI.status>> | null>(null);
  const [history, setHistory] = useState<Awaited<ReturnType<typeof searchGitAPI.log>> | null>(null);
  const [refs, setRefs] = useState<Awaited<ReturnType<typeof searchGitAPI.refs>>["items"]>([]), [reference, setReference] = useState("");
  const [detail, setDetail] = useState<Awaited<ReturnType<typeof searchGitAPI.detail>> | null>(null);
  const [comparison, setComparison] = useState<Awaited<ReturnType<typeof searchGitAPI.compare>> | null>(null);
  const [bufferContent, setBufferContent] = useState<string | null>(null), [useBuffer, setUseBuffer] = useState(false);
  const [pending, setPending] = useState(false), [error, setError] = useState<string | null>(null), [tab, setTab] = useState("changes"), [changeKind, setChangeKind] = useState("head");
  const [discovering, setDiscovering] = useState(true);
  const [discoveryRevision, setDiscoveryRevision] = useState(0);
  const [fullscreen, setFullscreen] = useState(false), [comparisonMode, setComparisonMode] = useState("side-by-side");
  const controller = useRef<AbortController | null>(null);
  const repository = repositories.find(item => item.id === repoID);
  const run = useCallback(async (work: (signal: AbortSignal) => Promise<void>) => {
    controller.current?.abort(); const current = new AbortController(); controller.current = current; setPending(true); setError(null);
    try { await work(current.signal); } catch (cause) { if (!current.signal.aborted) setError(errorMessage(cause)); }
    finally { if (!current.signal.aborted) setPending(false); }
  }, []);
  useEffect(() => {
    if (!visible) { controller.current?.abort(); setPending(false); return; }
    const discovery = new AbortController(); setDiscovering(true); setError(null);
    void searchGitAPI.repositories(project.id, project.version, discovery.signal).then(result => {
      if (discovery.signal.aborted) return; setRepositories(result.items); setRepoID(previous => result.items.some(item => item.id === previous) ? previous : result.items[0]?.id ?? "");
      if (result.truncated) setError("仓库列表已截断，请缩小项目范围。");
    }).catch(cause => { if (!discovery.signal.aborted) setError(errorMessage(cause)); }).finally(() => { if (!discovery.signal.aborted) setDiscovering(false); });
    return () => { discovery.abort(); controller.current?.abort(); };
  }, [project.id, project.version, visible, discoveryRevision]);
  const refresh = useCallback(async (signal: AbortSignal) => {
    if (!repoID || repository?.state !== "available") return;
    if (tab === "changes") { const result = await searchGitAPI.status(project.id, project.version, repoID, signal); if (!signal.aborted) setStatus(result); }
    else if (tab === "history") { const result = await searchGitAPI.log(project.id, project.version, repoID, 0, signal); if (!signal.aborted) { setHistory(result); setDetail(null); } }
    else { const result = await searchGitAPI.refs(project.id, project.version, repoID, signal); if (!signal.aborted) { setRefs(result.items); setReference(previous => result.items.some(item => item.name === previous) ? previous : result.items[0]?.name ?? ""); } }
  }, [project.id, project.version, repoID, repository?.state, tab]);
  useEffect(() => { setComparison(null); setStatus(null); setHistory(null); setDetail(null); setRefs([]); if (visible && repoID && repository?.state === "available") void run(refresh); return () => controller.current?.abort(); }, [refresh, visible, run, repoID, repository?.state, discoveryRevision]);
  useEffect(() => { if (!visible || tab !== "changes" || !repoID) return; const update = () => { if (document.visibilityState === "visible" && !controller.current?.signal.aborted && !pending) void run(refresh); }; const timer = setInterval(update, 15000); window.addEventListener("focus", update); window.addEventListener("online", update); return () => { clearInterval(timer); window.removeEventListener("focus", update); window.removeEventListener("online", update); }; }, [visible, tab, repoID, pending, run, refresh]);
  const repositoryRoot = repository ? `${project.folders.find(item => item.id === repository.folder_id)?.path}/${repository.path ? repository.path + "/" : ""}` : "";
  const comparableBuffers = [...new Set(scope.buffers.values())].flatMap(buffer => {
    if (!repositoryRoot || buffer.state.status !== "ready") return [];
    for (const alias of buffer.aliases.values()) {
      const absolute = `${project.folders.find(folder => folder.id === alias.folderId)?.path}/${alias.path}`;
      if (absolute.startsWith(repositoryRoot)) return [{ buffer, path: absolute.slice(repositoryRoot.length) }];
    }
    return [];
  });
  function compare(path: string, kind: GitCompareInput["kind"] = "head") {
    if (!repository) return;
    void run(async signal => {
      const value = await searchGitAPI.compare(project.id, repoID, { project_version: project.version, path, kind, reference, commit_id: detail?.commit.id ?? "", parent_id: detail?.parent_id ?? "" }, csrf, signal);
      if (signal.aborted) return;
      const root = project.folders.find(item => item.id === repository.folder_id)?.path ?? "";
      const absolute = `${root}/${repository.path ? repository.path + "/" : ""}${path}`;
      const buffer = [...new Set(scope.buffers.values())].find(item => item.state.status === "ready" && [...item.aliases.values()].some(alias => `${project.folders.find(folder => folder.id === alias.folderId)?.path}/${alias.path}` === absolute));
      setBufferContent(kind === "staged" || kind === "commit" ? null : buffer?.state.content ?? null); setUseBuffer(false); setFullscreen(false); setComparisonMode("side-by-side"); setComparison(value);
    });
  }
  async function loadDetail(commit: string, parent = "") { await run(async signal => { const value = await searchGitAPI.detail(project.id, project.version, repoID, commit, parent, signal); if (!signal.aborted) setDetail(value); }); }
  return <section className="feature-panel" aria-label="只读 Git"><div className="flex items-center justify-between gap-2"><h2>只读 Git</h2><Button size="sm" variant="outline" disabled={pending || discovering} onClick={() => setDiscoveryRevision(value => value + 1)}>刷新</Button></div>
    <FieldGroup><Field><FieldLabel>仓库</FieldLabel><Select value={repoID} onValueChange={value => setRepoID(value ?? "")} disabled={pending || discovering || repositories.length === 0}><SelectTrigger aria-label="选择仓库"><SelectValue>{repository?.name ?? (discovering ? "正在查找仓库…" : "未发现 Git 仓库")}</SelectValue></SelectTrigger><SelectContent><SelectGroup>{repositories.map(item => <SelectItem key={item.id} value={item.id}>{item.name}</SelectItem>)}</SelectGroup></SelectContent></Select></Field></FieldGroup>
    {pending && <Loading />}{error && <Alert variant="destructive"><AlertDescription>{error}</AlertDescription></Alert>}
    {repository?.state === "unavailable" && <Alert><AlertDescription>{repository.reason}</AlertDescription></Alert>}
    {!pending && !discovering && !error && !repositories.length && <Empty><EmptyHeader><EmptyTitle>未发现 Git 仓库</EmptyTitle><EmptyDescription>仅查看已注册文件夹中的本地仓库。</EmptyDescription></EmptyHeader></Empty>}
    {repository?.state === "available" && <Tabs value={tab} onValueChange={setTab}><TabsList aria-label="Git 视图"><TabsTrigger value="changes">变更</TabsTrigger><TabsTrigger value="history">历史</TabsTrigger><TabsTrigger value="refs">本地引用</TabsTrigger></TabsList>
      <TabsContent value="changes">{status && <p className="break-all">{status?.branch || "分离 HEAD"} · {status?.head.slice(0, 12) || "尚无提交"}</p>}<Tabs value={changeKind} onValueChange={setChangeKind}><TabsList aria-label="比较基线"><TabsTrigger value="head">HEAD</TabsTrigger><TabsTrigger value="staged">暂存</TabsTrigger><TabsTrigger value="unstaged">未暂存</TabsTrigger></TabsList></Tabs>
        {((changeKind === "head" ? status?.total_paths.map(path => status.changes.find(item => item.path === path) ?? { path, old_path: "", index: "", worktree: "" }) : status?.changes.filter(item => changeKind === "staged" ? item.index !== " " && item.index !== "?" : item.worktree !== " ")) ?? []).map(item => <Button key={item.path} variant="ghost" className="git-file-button" disabled={pending} onClick={() => compare(item.path, changeKind === "staged" ? "staged" : changeKind === "unstaged" ? "unstaged" : "head")}><span className="truncate">{item.old_path ? item.old_path + " → " : ""}{item.path}</span><span>{item.index}{item.worktree}</span></Button>)}
        {status && !status.changes.length && <p>工作区没有变更。</p>}
      </TabsContent>
      <TabsContent value="history">{history && !history.items.length && <p>尚无提交历史。</p>}{history?.items.map(commit => <Button key={commit.id} variant="ghost" className="git-file-button" disabled={pending} onClick={() => void loadDetail(commit.id)}><span className="truncate">{commit.subject || "无标题"}</span><span>{commit.id.slice(0, 8)}</span></Button>)}{history && history.next_offset >= 0 && <Button variant="outline" disabled={pending} onClick={() => void run(async signal => { const next = await searchGitAPI.log(project.id, project.version, repoID, history.next_offset, signal, history.head); if (!signal.aborted) setHistory({ head: next.head, items: [...history.items, ...next.items], next_offset: next.next_offset }); })}>加载更多提交</Button>}
        {detail && <div><p className="break-all">{detail.commit.author} · {detail.commit.date}<br />{detail.commit.subject}</p>{detail.commit.parents.length > 1 && <Field><FieldLabel>比较父提交</FieldLabel><Select value={detail.parent_id} onValueChange={value => { if (value) void loadDetail(detail.commit.id, value); }} disabled={pending}><SelectTrigger aria-label="比较父提交"><SelectValue>{detail.parent_id.slice(0, 12)}</SelectValue></SelectTrigger><SelectContent><SelectGroup>{detail.commit.parents.map(parent => <SelectItem key={parent} value={parent}>{parent.slice(0, 12)}</SelectItem>)}</SelectGroup></SelectContent></Select></Field>}{detail.files.map(path => <Button key={path} variant="ghost" className="git-file-button" disabled={pending} onClick={() => compare(path, "commit")}>{path}</Button>)}</div>}
      </TabsContent>
      <TabsContent value="refs"><FieldGroup><Field><FieldLabel>本地分支或标签</FieldLabel><Select value={reference} onValueChange={value => setReference(value ?? "")} disabled={pending || refs.length === 0}><SelectTrigger aria-label="本地分支或标签"><SelectValue>{reference || "暂无本地引用"}</SelectValue></SelectTrigger><SelectContent><SelectGroup>{refs.map(ref => <SelectItem key={ref.name} value={ref.name}>{ref.name}</SelectItem>)}</SelectGroup></SelectContent></Select></Field></FieldGroup><p>选择编辑器已打开的文件，与此引用比较。</p>{comparableBuffers.map(({ buffer, path }) => <Button key={buffer.id} variant="ghost" className="git-file-button" disabled={pending || !reference} onClick={() => compare(path, "reference")}>{path}</Button>)}</TabsContent>
    </Tabs>}
    <Dialog open={comparison !== null} onOpenChange={open => { if (!open) setComparison(null); }}><DialogContent className="draft-dialog git-comparison-dialog" data-fullscreen={fullscreen}><DialogHeader className="pr-8 min-w-0"><DialogTitle className="break-all">只读比较 · {comparison?.old_path ? `${comparison.old_path} → ` : ""}{comparison?.path}</DialogTitle><DialogDescription>基线：{comparison?.baseline || "空树"}。此视图不会保存文件或执行 Git 写操作。</DialogDescription></DialogHeader>
      <div className="comparison-toolbar">{!mobile && <Tabs value={comparisonMode} onValueChange={setComparisonMode}><TabsList aria-label="差异显示方式"><TabsTrigger value="side-by-side">并排</TabsTrigger><TabsTrigger value="inline">行内</TabsTrigger></TabsList></Tabs>}<Button size="sm" variant="outline" aria-label={fullscreen ? "退出全屏比较" : "全屏比较"} onClick={() => setFullscreen(value => !value)}>{fullscreen ? <Minimize /> : <Maximize />}{fullscreen ? "退出全屏" : "全屏"}</Button></div>
      {bufferContent !== null && <Field orientation="horizontal"><Checkbox id="git-use-buffer" checked={useBuffer} onCheckedChange={setUseBuffer} /><FieldLabel htmlFor="git-use-buffer">比较当前编辑器输入（打开比较时的快照）</FieldLabel></Field>}
      <div className="comparison-body">{comparison && (comparison.binary ? <p>二进制文件暂不提供文本差异。</p> : <ReadOnlyComparison path={comparison.path} original={comparison.original} modified={useBuffer && bufferContent !== null ? bufferContent : comparison.modified} mobile={mobile} sideBySide={comparisonMode === "side-by-side"} />)}</div><DialogFooter><Button variant="outline" onClick={() => setComparison(null)}>关闭比较</Button></DialogFooter>
    </DialogContent></Dialog>
  </section>;
}
