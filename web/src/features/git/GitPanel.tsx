import { useCallback, useEffect, useRef, useState } from "react";
import { GitBranch, Maximize, Minimize, RefreshCw } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
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
import { GitSections } from "./GitSections";
import { GitHistory } from "./GitHistory";
import { GitFiles, GitFileViewToggle, type FileView } from "./GitFiles";
import { refreshGitBaselines } from "./baseline-requests";
import { useCommitDetails } from "./useCommitDetails";
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
  const [pending, setPending] = useState(false), [error, setError] = useState<string | null>(null), [changeKind, setChangeKind] = useState("head");
  const [statusPending, setStatusPending] = useState(false), [statusError, setStatusError] = useState<string | null>(null);
  const [expanded, setExpanded] = useState("");
  const [changesView, setChangesView] = useState<FileView>("list"), [historyView, setHistoryView] = useState<FileView>("list");
  const [refsOpen, setRefsOpen] = useState(false);
  const statusController = useRef<AbortController | null>(null);
  const [discovering, setDiscovering] = useState(true);
  const [discoveryRevision, setDiscoveryRevision] = useState(0);
  const discoveryKey = JSON.stringify([project.id, project.version, discoveryRevision]);
  const [discoveredKey, setDiscoveredKey] = useState("");
  const baselineRevision = useRef(0);
  const [fullscreen, setFullscreen] = useState(false), [comparisonMode, setComparisonMode] = useState("side-by-side");
  const controller = useRef<AbortController | null>(null);
  const details = useCommitDetails(project, repoID, visible, discoveryRevision, pending || refsOpen || comparison !== null);
  const { cancelPreview } = details;
  const repository = repositories.find(item => item.id === repoID);
  const run = useCallback(async (work: (signal: AbortSignal) => Promise<void>) => {
    cancelPreview();
    statusController.current?.abort(); setStatusPending(false);
    controller.current?.abort(); const current = new AbortController(); controller.current = current; setPending(true); setError(null);
    try { await work(current.signal); return !current.signal.aborted; } catch (cause) { if (!current.signal.aborted) setError(errorMessage(cause)); return false; }
    finally { if (!current.signal.aborted) setPending(false); }
  }, [cancelPreview]);
  useEffect(() => {
    if (!visible) { controller.current?.abort(); statusController.current?.abort(); setPending(false); setStatusPending(false); return; }
    const discovery = new AbortController(); setDiscovering(true); setError(null);
    void searchGitAPI.repositories(project.id, project.version, discovery.signal).then(result => {
      if (discovery.signal.aborted) return; setDiscoveredKey(discoveryKey); setRepositories(result.items); setRepoID(previous => result.items.some(item => item.id === previous) ? previous : result.items[0]?.id ?? "");
      if (result.truncated) setError("仓库列表已截断，请缩小项目范围。");
    }).catch(cause => { if (!discovery.signal.aborted) setError(errorMessage(cause)); }).finally(() => { if (!discovery.signal.aborted) setDiscovering(false); });
    return () => { discovery.abort(); controller.current?.abort(); statusController.current?.abort(); };
  }, [project.id, project.version, visible, discoveryRevision, discoveryKey]);
  const refreshStatus = useCallback(async () => {
    if (!repoID || repository?.state !== "available") return;
    statusController.current?.abort(); const current = new AbortController(); statusController.current = current;
    setStatusPending(true); setStatusError(null);
    try { const result = await searchGitAPI.status(project.id, project.version, repoID, current.signal); if (!current.signal.aborted) setStatus(result); }
    catch (cause) { if (!current.signal.aborted) setStatusError(errorMessage(cause)); }
    finally { if (!current.signal.aborted) setStatusPending(false); }
  }, [project.id, project.version, repoID, repository?.state]);
  const refresh = useCallback(async (signal: AbortSignal) => {
    const result = await searchGitAPI.log(project.id, project.version, repoID, 0, signal);
    if (!signal.aborted) setHistory(result);
  }, [project.id, project.version, repoID]);
  useEffect(() => {
    setPending(false); setStatusPending(false); setStatusError(null); setComparison(null); setRefsOpen(false); setStatus(null); setHistory(null); setDetail(null); setExpanded(""); setRefs([]);
    let cancelled = false;
    let initialSignal: AbortSignal | null = null;
    if (visible && discoveredKey === discoveryKey && repoID && repository?.state === "available") {
      // Initial reads are serial, leaving the second tool slot for editor baseline.
      void run(signal => { initialSignal = signal; return refresh(signal); }).then(success => { if (cancelled || initialSignal?.aborted) return; if (success && discoveryRevision > baselineRevision.current) { baselineRevision.current = discoveryRevision; refreshGitBaselines(project.id); } void refreshStatus(); });
    }
    return () => { cancelled = true; controller.current?.abort(); statusController.current?.abort(); };
  }, [refresh, refreshStatus, visible, run, repoID, repository?.state, discoveryRevision, discoveredKey, discoveryKey, project.id]);
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
    setRefsOpen(false);
    void run(async signal => {
      const value = await searchGitAPI.compare(project.id, repoID, { project_version: project.version, path, kind, reference, commit_id: detail?.commit.id ?? "", parent_id: detail?.parent_id ?? "" }, csrf, signal);
      if (signal.aborted) return;
      const root = project.folders.find(item => item.id === repository.folder_id)?.path ?? "";
      const absolute = `${root}/${repository.path ? repository.path + "/" : ""}${path}`;
      const buffer = [...new Set(scope.buffers.values())].find(item => item.state.status === "ready" && [...item.aliases.values()].some(alias => `${project.folders.find(folder => folder.id === alias.folderId)?.path}/${alias.path}` === absolute));
      setBufferContent(kind === "staged" || kind === "commit" ? null : buffer?.state.content ?? null); setUseBuffer(false); setFullscreen(false); setComparisonMode("side-by-side"); setComparison(value);
    });
  }
  async function loadDetail(commit: string, parent = "") { await run(async signal => { const value = await details.read(commit, parent, signal); if (!signal.aborted) setDetail(value); }); }
  const changeFiles = ((changeKind === "head" ? status?.total_paths.map(path => status.changes.find(item => item.path === path) ?? { path, old_path: "", index: "", worktree: "" }) : status?.changes.filter(item => changeKind === "staged" ? item.index !== " " && item.index !== "?" : item.worktree !== " ")) ?? []).map(item => ({ path: item.path, oldPath: item.old_path, status: (changeKind === "staged" ? item.index : item.worktree.trim() || item.index.trim()) || "M" }));
  function openRefs() { setRefsOpen(true); void run(async signal => { const result = await searchGitAPI.refs(project.id, project.version, repoID, signal); if (!signal.aborted) { setRefs(result.items); setReference(previous => result.items.some(item => item.name === previous) ? previous : result.items[0]?.name ?? ""); } }); }
  return <section className="feature-panel git-panel" aria-label="只读 Git"><div className="flex flex-wrap items-center justify-between gap-2"><h2>只读 Git</h2><div className="flex gap-1"><Button size="sm" variant="ghost" disabled={discovering || repository?.state !== "available"} onClick={openRefs}>本地引用</Button><Button size="icon-sm" variant="ghost" aria-label="刷新仓库" title="刷新仓库" disabled={discovering} onClick={() => setDiscoveryRevision(value => value + 1)}><RefreshCw /></Button></div></div>
    <FieldGroup><Field><FieldLabel>仓库</FieldLabel><Select value={repoID} onValueChange={value => setRepoID(value ?? "")} disabled={discovering || repositories.length === 0}><SelectTrigger aria-label="选择仓库"><SelectValue>{repository?.name ?? (discovering ? "正在查找仓库…" : "未发现 Git 仓库")}</SelectValue></SelectTrigger><SelectContent><SelectGroup>{repositories.map(item => <SelectItem key={item.id} value={item.id}>{item.name}</SelectItem>)}</SelectGroup></SelectContent></Select></Field></FieldGroup>
    {error && !refsOpen && <Alert variant="destructive"><AlertDescription>{error}</AlertDescription></Alert>}
    {repository?.state === "unavailable" && <Alert><AlertDescription>{repository.reason}</AlertDescription></Alert>}
    {!discovering && !error && !repositories.length && <Empty><EmptyHeader><EmptyTitle>未发现 Git 仓库</EmptyTitle><EmptyDescription>仅查看已注册文件夹中的本地仓库。</EmptyDescription></EmptyHeader></Empty>}
    {repository?.state === "available" && <GitSections
      changesActions={<><GitFileViewToggle label="变更文件展示" value={changesView} onChange={setChangesView} /><Button variant="ghost" size="icon-sm" aria-label="刷新变更" title="刷新变更" disabled={pending || statusPending} onClick={() => void refreshStatus()}><RefreshCw /></Button></>}
      historyActions={<><GitBranch className="git-branch-icon" /><span className="git-branch-label truncate">{status?.branch || "历史"}</span><GitFileViewToggle label="历史文件展示" value={historyView} onChange={setHistoryView} /></>}
      changes={<>{statusPending && <Loading />}{statusError && <Alert variant="destructive"><AlertDescription>{statusError}</AlertDescription></Alert>}{status && <p className="git-branch-summary truncate" title={status.head}>{status.branch || "分离 HEAD"} · {status.head.slice(0, 12) || "尚无提交"}</p>}
        <Tabs value={changeKind} onValueChange={setChangeKind}><TabsList aria-label="比较基线"><TabsTrigger value="head">HEAD</TabsTrigger><TabsTrigger value="staged">暂存</TabsTrigger><TabsTrigger value="unstaged">未暂存</TabsTrigger></TabsList></Tabs>
        <GitFiles files={changeFiles} view={changesView} disabled={pending} onSelect={path => compare(path, changeKind === "staged" ? "staged" : changeKind === "unstaged" ? "unstaged" : "head")} />
        {status && !changeFiles.length && <p>工作区没有变更。</p>}{!status && !statusPending && !statusError && <p>点击刷新变更读取状态。</p>}
      </>}
      history={<><GitHistory history={history} detail={detail} expanded={expanded} branch={status?.branch ?? ""} view={historyView} pending={pending}
        preview={details.preview} onPreview={(commit, parent, open) => { if (open) { statusController.current?.abort(); setStatusPending(false); } details.show(commit, parent, open); }}
        onExpand={(commit, open) => { setExpanded(open ? commit : ""); if (open) { setDetail(null); void loadDetail(commit, history?.items.find(item => item.id === commit)?.parents[0] ?? ""); } else { controller.current?.abort(); setPending(false); } }}
        onParent={(commit, parent) => { setExpanded(commit); setDetail(null); void loadDetail(commit, parent); }} onCompare={path => compare(path, "commit")}
        onMore={() => { if (!history || pending || history.next_offset < 0) return Promise.resolve(false); return run(async signal => { const next = await searchGitAPI.log(project.id, project.version, repoID, history.next_offset, signal, history.head); if (!signal.aborted) setHistory({ head: next.head, items: [...history.items, ...next.items], next_offset: next.next_offset }); }); }} />{pending && <Loading />}</>}
    />}
    <Dialog open={refsOpen} onOpenChange={setRefsOpen}><DialogContent><DialogHeader><DialogTitle>本地引用比较</DialogTitle><DialogDescription>选择已打开文件，与本地分支或标签比较。</DialogDescription></DialogHeader>{error && <Alert variant="destructive"><AlertDescription>{error}</AlertDescription></Alert>}{pending && <Loading />}<FieldGroup><Field><FieldLabel>本地分支或标签</FieldLabel><Select value={reference} onValueChange={value => setReference(value ?? "")} disabled={pending || refs.length === 0}><SelectTrigger aria-label="本地分支或标签"><SelectValue>{reference || "暂无本地引用"}</SelectValue></SelectTrigger><SelectContent><SelectGroup>{refs.map(ref => <SelectItem key={ref.name} value={ref.name}>{ref.name}</SelectItem>)}</SelectGroup></SelectContent></Select></Field></FieldGroup><GitFiles files={comparableBuffers.map(item => ({ path: item.path }))} view="list" disabled={pending || !reference} onSelect={path => compare(path, "reference")} /><DialogFooter><Button variant="outline" onClick={() => setRefsOpen(false)}>关闭引用</Button></DialogFooter></DialogContent></Dialog>
    <Dialog open={comparison !== null} onOpenChange={open => { if (!open) setComparison(null); }}><DialogContent className="draft-dialog git-comparison-dialog" data-fullscreen={fullscreen}><DialogHeader className="pr-8 min-w-0"><DialogTitle className="break-all">只读比较 · {comparison?.old_path ? `${comparison.old_path} → ` : ""}{comparison?.path}</DialogTitle><DialogDescription>基线：{comparison?.baseline || "空树"}。此视图不会保存文件或执行 Git 写操作。</DialogDescription></DialogHeader>
      <div className="comparison-toolbar">{!mobile && <Tabs value={comparisonMode} onValueChange={setComparisonMode}><TabsList aria-label="差异显示方式"><TabsTrigger value="side-by-side">并排</TabsTrigger><TabsTrigger value="inline">行内</TabsTrigger></TabsList></Tabs>}<Button size="sm" variant="outline" aria-label={fullscreen ? "退出全屏比较" : "全屏比较"} onClick={() => setFullscreen(value => !value)}>{fullscreen ? <Minimize /> : <Maximize />}{fullscreen ? "退出全屏" : "全屏"}</Button></div>
      {bufferContent !== null && <Field orientation="horizontal"><Checkbox id="git-use-buffer" checked={useBuffer} onCheckedChange={setUseBuffer} /><FieldLabel htmlFor="git-use-buffer">比较当前编辑器输入（打开比较时的快照）</FieldLabel></Field>}
      <div className="comparison-body">{comparison && (comparison.binary ? <p>二进制文件暂不提供文本差异。</p> : <ReadOnlyComparison path={comparison.path} original={comparison.original} modified={useBuffer && bufferContent !== null ? bufferContent : comparison.modified} mobile={mobile} sideBySide={comparisonMode === "side-by-side"} />)}</div><DialogFooter><Button variant="outline" onClick={() => setComparison(null)}>关闭比较</Button></DialogFooter>
    </DialogContent></Dialog>
  </section>;
}
