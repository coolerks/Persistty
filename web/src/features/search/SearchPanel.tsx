import { useEffect, useRef, useState } from "react";
import { ChevronRight, Ellipsis, Search } from "lucide-react";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import { FileTypeIcon } from "@/features/workspaces/FileTypeIcon";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Checkbox } from "@/components/ui/checkbox";
import { Toggle } from "@/components/ui/toggle";
import { Field, FieldGroup, FieldLabel, FieldSet, FieldLegend, FieldDescription } from "@/components/ui/field";
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Empty, EmptyHeader, EmptyTitle, EmptyDescription } from "@/components/ui/empty";
import { Loading } from "@/components/Feedback";
import { useAuth } from "@/features/auth/auth-context";
import { api, errorMessage } from "@/lib/api/client";
import { randomUUID } from "@/lib/random-uuid";
import { searchGitAPI } from "@/lib/api/search-git-client";
import type { Project } from "@/lib/api/decoder";
import type { ReplacePreview, SearchFile, SearchResult } from "@/lib/api/search-git-decoder";
import { useEditorScope } from "@/features/workspaces/editor-context";
import { sameVersion } from "@/features/workspaces/editor-session";
import { fileKey, type OpenFile } from "@/features/workspaces/workspace-view";
import { ReadOnlyComparison } from "@/features/workspaces/ReadOnlyComparison";
import { useEditorLocation } from "@/features/workspaces/editor-navigation";
export type SearchIntent = { folderId: string; path: string; replace: boolean; id: number };
const resultReasons: Record<string, string> = { unsaved_input: "本地输入或版本变化", cancelled: "已取消", not_selected: "未选择", version_changed: "版本变化", save_failed: "保存失败" };
function previewResultLabel(preview: ReplacePreview, id: string) { const result = preview.results.find(item => item.id === id); return result ? `${resultLabels[result.state]}${result.reason ? ` · ${resultReasons[result.reason] ?? result.reason}` : ""}` : preview.state === "cancelled" ? "已取消" : "待应用"; }
const resultLabels = { applied: "已替换", conflict: "版本冲突，未替换", skipped: "已跳过", error: "失败", cancelled: "已取消" };
const globs = (value: string) => value.split(";").map(item => item.trim()).filter(Boolean);
export function SearchPanel({ project, mobile, intent, visible, onOpen }: { project: Project; mobile: boolean; intent: SearchIntent | null; visible: boolean; onOpen(file: OpenFile): void }) {
  const auth = useAuth(), scope = useEditorScope();
  const csrf = auth.state.status === "authenticated" ? auth.state.session.csrf_token : "";
  const [pattern, setPattern] = useState(""), [replacement, setReplacement] = useState("");
  const [folder, setFolder] = useState(""), [path, setPath] = useState("");
  const [regex, setRegex] = useState(false), [caseSensitive, setCaseSensitive] = useState(false), [wholeWord, setWholeWord] = useState(false);
  const [detailsOpen, setDetailsOpen] = useState(false);
  const [include, setInclude] = useState(""), [exclude, setExclude] = useState("");
  const [result, setResult] = useState<SearchResult | null>(null), [selected, setSelected] = useState<Set<string>>(new Set());
  const [preview, setPreview] = useState<ReplacePreview | null>(null), [previewOpen, setPreviewOpen] = useState(false);
  const [selectedFiles, setSelectedFiles] = useState<Set<string>>(new Set());
  const [comparison, setComparison] = useState<Awaited<ReturnType<typeof searchGitAPI.previewComparison>> | null>(null);
  const [applying, setApplying] = useState(false);
  const [pending, setPending] = useState(false), [error, setError] = useState<string | null>(null);
  const patternInput = useRef<HTMLInputElement>(null);
  const closePreviewButton = useRef<HTMLButtonElement>(null);
  const queryKey = JSON.stringify([pattern, folder, path, regex, caseSensitive, wholeWord, include, exclude]);
  const [searchedKey, setSearchedKey] = useState("");
  useEffect(() => { if (visible) patternInput.current?.focus(); }, [visible, intent]);
  const request = useRef<AbortController | null>(null), epoch = useRef(0);
  const snapshots = useRef<{ search: string | null; preview: string | null }>({ search: null, preview: null });
  useEffect(() => {
    if (intent) { setFolder(intent.folderId); setPath(intent.path); setDetailsOpen(true); }
  }, [intent]);
  useEffect(() => () => {
    epoch.current++; request.current?.abort();
    const signal = new AbortController().signal;
    if (snapshots.current.search) void searchGitAPI.cancelSearch(project.id, snapshots.current.search, csrf, signal).catch(() => {});
    if (snapshots.current.preview) void searchGitAPI.cancelPreview(project.id, snapshots.current.preview, csrf, signal).catch(() => {});
  }, [project.id, project.version, csrf]);
  async function run(work: (signal: AbortSignal) => Promise<void>) {
    request.current?.abort(); const controller = new AbortController(); request.current = controller; const generation = ++epoch.current;
    setPending(true); setError(null);
    try { await work(controller.signal); } catch (cause) { if (!controller.signal.aborted && generation === epoch.current) setError(errorMessage(cause)); }
    finally { if (generation === epoch.current) setPending(false); }
  }
  function toggle(ids: string[], checked: boolean) { setSelected(previous => { const next = new Set(previous); for (const id of ids) if (checked) next.add(id); else next.delete(id); return next; }); }
  async function search() {
    await run(async signal => {
      if (snapshots.current.preview) await searchGitAPI.cancelPreview(project.id, snapshots.current.preview, csrf, signal);
      if (snapshots.current.search) await searchGitAPI.cancelSearch(project.id, snapshots.current.search, csrf, signal);
      snapshots.current = { search: null, preview: null }; setPreview(null); setComparison(null); setResult(null);
      const value = await searchGitAPI.search(project.id, { project_version: project.version, folder_id: folder, path, pattern, regex, case_sensitive: caseSensitive, whole_word: wholeWord, include: globs(include), exclude: globs(exclude) }, csrf, signal);
      if (signal.aborted) return; snapshots.current.search = value.id; setResult(value); setSearchedKey(queryKey); setSelected(new Set(value.files.flatMap(file => file.matches.map(match => match.id))));
    });
  }
  async function locate(file: SearchFile, match: SearchFile["matches"][number]) {
    await run(async signal => {
      const buffer = scope.buffers.get(fileKey({ folderId: file.folder_id, path: file.path }));
      const generation = buffer?.state.generation;
      const metadata = await api.metadata(project.id, file.folder_id, project.version, file.path, signal);
      if (signal.aborted) return;
      if (!sameVersion(metadata.version, file.version) || (buffer && (buffer.dirty || buffer.state.generation !== generation || !buffer.state.base || !sameVersion(buffer.state.base.version, file.version)))) throw new Error("文件或本地输入已变化，请重新搜索；当前输入已保留。");
      const target = { folderId: file.folder_id, path: file.path };
      useEditorLocation.setState({ location: { projectId: project.id, file: target, version: file.version, line: match.line, column: match.column, endColumn: match.end_column, id: randomUUID() } }); onOpen(target);
    });
  }
  async function makePreview() {
    if (!result) return;
    await run(async signal => {
      if (preview) await searchGitAPI.cancelPreview(project.id, preview.id, csrf, signal);
      const value = await searchGitAPI.preview(project.id, { project_version: project.version, search_id: result.id, selected_match_ids: [...selected], replacement }, csrf, signal);
      if (signal.aborted) return; snapshots.current.preview = value.id; setPreview(value); setSelectedFiles(new Set(value.files.map(file => file.id))); setComparison(null); setPreviewOpen(true);
    });
  }
  async function apply() {
    if (!preview) return;
    const protection = scope.prepareReplacement(preview.files.filter(file => selectedFiles.has(file.id)));
    setApplying(true);
    await run(async signal => {
      try {
        const value = await searchGitAPI.apply(project.id, preview.id, { project_version: project.version, selected_file_ids: [...selectedFiles], protected_file_ids: protection.protectedIDs }, csrf, signal);
        if (!signal.aborted) setPreview(value);
      } finally { await protection.release(); if (request.current?.signal === signal) setApplying(false); }
    });
  }
  async function cancelApply() {
    if (!preview || !applying) return;
    const active = request.current, generation = epoch.current;
    if (!active || active.signal.aborted) return;
    try {
      await searchGitAPI.cancelPreview(project.id, preview.id, csrf, active.signal);
      if (generation !== epoch.current || active.signal.aborted) return;
      active.abort(); setApplying(false);
      await run(async signal => {
        const value = await searchGitAPI.previewStatus(project.id, preview.id, signal);
        if (!signal.aborted) setPreview(value);
      });
    } catch (cause) { if (generation === epoch.current && !active.signal.aborted) setError(errorMessage(cause)); }
  }
  async function closePreview() {
    if (pending) return;
    if (preview && preview.state !== "applying") await run(async signal => { await searchGitAPI.cancelPreview(project.id, preview.id, csrf, signal); if (!signal.aborted) { setPreview(null); snapshots.current.preview = null; } });
    setPreviewOpen(false);
  }
  return <section className="feature-panel search-panel" aria-label="搜索与替换"><h2>搜索与替换</h2>
    <form className="search-form" onSubmit={event => { event.preventDefault(); void search(); }}>
      <Field className="search-pattern-field"><FieldLabel className="sr-only" htmlFor="search-pattern">搜索内容</FieldLabel><div className="search-input-row"><Input ref={patternInput} id="search-pattern" placeholder="搜索" value={pattern} onChange={event => setPattern(event.target.value)} maxLength={4096} required disabled={pending} /><div className="search-input-options"><Toggle size="sm" aria-label="区分大小写" title="区分大小写" pressed={caseSensitive} onPressedChange={setCaseSensitive} disabled={pending}>Aa</Toggle><Toggle size="sm" aria-label="全词匹配" title="全词匹配" pressed={wholeWord} onPressedChange={setWholeWord} disabled={pending}>ab</Toggle><Toggle size="sm" aria-label="正则表达式" title="正则表达式" pressed={regex} onPressedChange={setRegex} disabled={pending}>.*</Toggle></div></div></Field>
      <Collapsible open={detailsOpen} onOpenChange={setDetailsOpen} className="search-details">
        <div className="search-form-actions"><Button type="submit" size="sm" variant="ghost" disabled={pending || !pattern}><Search />搜索</Button><CollapsibleTrigger render={<Button type="button" size="icon-sm" variant="ghost" aria-label="搜索范围与文件过滤" title="搜索范围与文件过滤" />}><Ellipsis /></CollapsibleTrigger></div>
        <CollapsibleContent><FieldGroup className="search-filters">
          <Field><FieldLabel>搜索范围</FieldLabel><Select value={folder || "all"} onValueChange={value => { setFolder(value === "all" ? "" : value ?? ""); setPath(""); }} disabled={pending}><SelectTrigger aria-label="搜索范围"><SelectValue>{folder ? project.folders.find(item => item.id === folder)?.path : "整个项目"}</SelectValue></SelectTrigger><SelectContent><SelectGroup><SelectItem value="all">整个项目</SelectItem>{project.folders.map(item => <SelectItem key={item.id} value={item.id}>{item.path}</SelectItem>)}</SelectGroup></SelectContent></Select></Field>
          {folder && <Field><FieldLabel htmlFor="search-path">子目录（可选）</FieldLabel><Input id="search-path" value={path} onChange={event => setPath(event.target.value)} disabled={pending} /></Field>}
          <Field><FieldLabel htmlFor="search-include">包含文件</FieldLabel><Input id="search-include" value={include} onChange={event => setInclude(event.target.value)} disabled={pending} placeholder="src/**; *.go" title="用 ; 分隔 glob；忽略规则仍生效。" /></Field>
          <Field><FieldLabel htmlFor="search-exclude">排除文件</FieldLabel><Input id="search-exclude" value={exclude} onChange={event => setExclude(event.target.value)} disabled={pending} placeholder="例如 **/*.test.ts" /></Field>
        </FieldGroup></CollapsibleContent>
      </Collapsible>
    </form>
    {pending && <><Loading /><Button variant="outline" onClick={() => { request.current?.abort(); epoch.current++; setPending(false); if (preview?.state === "applying" || preview?.state === "ready") void searchGitAPI.cancelPreview(project.id, preview.id, csrf, new AbortController().signal).catch(cause => setError(errorMessage(cause))); }}>取消当前操作</Button></>}
    {error && <Alert variant="destructive"><AlertDescription>{error}</AlertDescription></Alert>}
    {result && <>
      <p role="status">{result.files.length} 个文件 · {result.files.reduce((sum, file) => sum + file.matches.length, 0)} 处匹配{result.truncated ? "（结果已截断）" : ""} · {result.skipped.length} 项跳过</p>
      {searchedKey !== queryKey && <p role="status">搜索条件已变化，请重新搜索后生成替换预览。</p>}
      <FieldGroup><Field><FieldLabel htmlFor="search-replacement">替换为</FieldLabel><Input id="search-replacement" value={replacement} onChange={event => setReplacement(event.target.value)} disabled={pending} /><FieldDescription>{regex ? "捕获组使用 $1 或 ${name}；$$ 表示美元符号。" : "按字面文本替换。"}</FieldDescription></Field><Button variant="outline" disabled={pending || selected.size === 0 || searchedKey !== queryKey} onClick={() => void makePreview()}>预览选中项替换（{selected.size}）</Button></FieldGroup>
      <FieldSet className="search-results"><FieldLegend className="sr-only">搜索结果</FieldLegend><FieldGroup>{result.files.map(file => <Collapsible key={file.id} defaultOpen className="search-result-file"><CollapsibleTrigger render={<Button variant="ghost" size="sm" className="search-file-heading" />} title={`${project.folders.find(item => item.id === file.folder_id)?.path}/${file.path}`}><ChevronRight className="search-file-chevron" /><FileTypeIcon path={file.path} /><span className="truncate">{file.path}</span><span className="search-match-count">{file.matches.length}</span></CollapsibleTrigger><CollapsibleContent><Field orientation="horizontal"><Checkbox aria-label={`选择文件 ${file.path}`} checked={file.matches.every(match => selected.has(match.id))} onCheckedChange={checked => toggle(file.matches.map(match => match.id), checked)} disabled={pending} /><FieldLabel>选择此文件全部匹配</FieldLabel></Field>{file.matches.map(match => <Field key={match.id} orientation="horizontal"><Checkbox aria-label={`选择 ${file.path}:${match.line}:${match.column}`} checked={selected.has(match.id)} onCheckedChange={checked => toggle([match.id], checked)} disabled={pending} /><Button variant="ghost" className="search-result-button" disabled={pending} onClick={() => void locate(file, match)}><span>{match.line}:{match.column}</span><span className="truncate">{match.preview}</span></Button></Field>)}</CollapsibleContent></Collapsible>)}</FieldGroup></FieldSet>
      {!result.files.length && <Empty><EmptyHeader><EmptyTitle>没有匹配结果</EmptyTitle><EmptyDescription>调整搜索内容或范围后重试。</EmptyDescription></EmptyHeader></Empty>}
      {result.skipped.length > 0 && <details><summary>查看跳过项目</summary>{result.skipped.map((item, index) => <p key={index} className="break-all">{item.path} · {item.reason}</p>)}</details>}
    </>}
    <Dialog open={previewOpen} onOpenChange={open => { if (!open) void closePreview(); }}><DialogContent className="draft-dialog" showCloseButton={!pending} initialFocus={closePreviewButton}><DialogHeader><DialogTitle>替换预览</DialogTitle><DialogDescription>逐文件比较后明确应用。未保存、保存中或暂停的文件会跳过；版本冲突不会覆盖文件。</DialogDescription></DialogHeader>
      {preview && <p role="status">{preview.state === "cancelled" ? "预览已取消，未提交的文件不会写入。" : preview.state === "applying" ? "替换仍在处理中，请查询逐文件状态。" : preview.state === "completed" ? "处理完成，请核对逐文件结果。" : "尚未写入文件。"}</p>}
      {preview && <div className="replace-preview-files"><FieldSet><FieldLegend>准备替换 {preview.files.length} 个文件</FieldLegend><FieldGroup>{preview.files.map(file => <Field key={file.id} orientation="horizontal"><Checkbox aria-label={`应用 ${file.path}`} checked={selectedFiles.has(file.id)} disabled={pending || preview.state !== "ready"} onCheckedChange={checked => setSelectedFiles(previous => { const next = new Set(previous); if (checked) next.add(file.id); else next.delete(file.id); return next; })} /><Button variant="ghost" className="truncate" disabled={pending} onClick={() => void run(async signal => { const value = await searchGitAPI.previewComparison(project.id, preview.id, file.id, signal); if (!signal.aborted) setComparison(value); })}>{file.path} · {file.count} 处</Button><span>{previewResultLabel(preview, file.id)}</span></Field>)}</FieldGroup></FieldSet></div>}
      {comparison && <ReadOnlyComparison {...comparison} mobile={mobile} />}
      {error && <Alert variant="destructive"><AlertDescription>{error}</AlertDescription></Alert>}
      <DialogFooter><Button ref={closePreviewButton} variant="outline" disabled={pending} onClick={() => void closePreview()}>关闭预览</Button>{applying && <Button variant="outline" onClick={() => void cancelApply()}>停止剩余替换</Button>}{preview && <Button variant="outline" disabled={pending} onClick={() => void run(async signal => { const value = await searchGitAPI.previewStatus(project.id, preview.id, signal); if (!signal.aborted) setPreview(value); })}>读取应用状态</Button>}<Button disabled={pending || !preview || preview.state !== "ready" || selectedFiles.size === 0} onClick={() => void apply()}>确认应用到 {selectedFiles.size} 个文件</Button></DialogFooter>
    </DialogContent></Dialog>
  </section>;
}
