import { lazy, Suspense, useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Failure, Loading } from "@/components/Feedback";
import { ApiError } from "@/lib/api/client";
import type { Project } from "@/lib/api/decoder";
import { useEditorScope } from "./editor-context";
import { editorText } from "./editor-text";
import { downloadDraft, type FileDraft } from "./editor-drafts";
import { FilePreview } from "./FilePreview";
import { sameVersion } from "./editor-session";
import { languageForFile } from "./file-language";
import { fileKey, useWorkspaceView, type OpenFile } from "./workspace-view";
const DesktopEditor = lazy(() => import("./DesktopEditor"));
const DesktopDiff = lazy(() => import("./DesktopDiff"));
const labels = { saved: "已保存", pending: "待保存", saving: "保存中", conflict: "冲突", failed: "保存失败", paused: "已暂停自动保存" };

export function FileEditor({ project, file, mobile }: { project: Project; file: OpenFile; mobile: boolean }) {
  const scope = useEditorScope();
  useEffect(() => { scope.open(file); }, [scope, file]);
  const buffer = scope.buffers.get(fileKey(file));
  const mode = useWorkspaceView(state => state.languageModes[project.id]?.[fileKey(file)]);
  const [draft, setDraft] = useState<FileDraft | null>(null);
  const [source, setSource] = useState(false);
  const [compare, setCompare] = useState(false);
  const state = buffer?.state;
  const detected = languageForFile(file.path, state?.content ?? "");
  const language = mode ?? detected;
  const baseChanged = !!draft && !!state?.base && !sameVersion(draft.base.version, state.base.version);
  const ready = state?.status === "ready";
  const svg = ready && /^\s*(?:<\?xml[^>]*>\s*)?<svg[\s>]/.test(editorText(state.content));
  return <div className="file-editor"><div className="editor-breadcrumb">
    <span className="truncate" title={file.path}>{project.folders.find(folder => folder.id === file.folderId)?.path}/{file.path}</span>
    {svg && <Button size="sm" variant="outline" onClick={() => setSource(value => !value)}>{source ? "查看图片" : "编辑源码"}</Button>}
    {ready && ["conflict", "failed", "paused"].includes(state.saveState) && <Badge className="editor-save-warning" role="status" aria-live="polite" variant={state.saveState === "paused" ? "secondary" : "destructive"}>{labels[state.saveState]}</Badge>}
  </div>
    {ready && !["conflict", "failed", "paused"].includes(state.saveState) && <span className="sr-only" role="status" aria-live="polite">{labels[state.saveState]}</span>}
    {(!state || state.status === "loading") && <div className="editor-feedback"><Loading /></div>}
    {state?.status === "error" && <div className="editor-feedback">{state.error instanceof ApiError && (state.error.status === 415 || state.error.status === 413) ? <FilePreview key={`${project.version}:${fileKey(file)}`} project={project} file={file} /> : <Failure error={state.error} retry={() => void buffer?.load()} />}</div>}
    {ready && <>
      {state.draftError && <Alert variant="destructive"><AlertDescription>{state.draftError}<Button size="sm" variant="outline" onClick={() => downloadDraft(state.content, file.path)}>导出当前输入</Button></AlertDescription></Alert>}
      {state.error !== null && <Alert variant="destructive"><AlertDescription>{state.error instanceof Error ? state.error.message : "无法保存文件。"}<div className="flex gap-2"><Button size="sm" variant="outline" onClick={() => { void buffer?.compare().then(() => setCompare(true)); }}>查看并处理</Button><Button size="sm" variant="outline" onClick={() => downloadDraft(state.content, file.path)}>导出输入</Button></div></AlertDescription></Alert>}
      {state.drafts.length > 0 && <Alert><AlertDescription>此文件有 {state.drafts.length} 份本地草稿。{state.drafts.map((item, index) => <Button key={item.id} size="sm" variant="outline" onClick={() => setDraft(item)}>查看草稿 {index + 1}</Button>)}</AlertDescription></Alert>}
      {state.saveState === "paused" && <Alert><AlertDescription>恢复内容尚未写入服务器。<Button size="sm" variant="outline" onClick={() => void buffer?.save(true)}>保存恢复内容</Button></AlertDescription></Alert>}
      {svg && !source ? <FilePreview project={project} file={file} /> : <div className="editor-surface" onKeyDown={event => { if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === "s") { event.preventDefault(); if (state.saveState !== "conflict") void buffer?.save(true); } }}>
        {mobile ? <textarea aria-label={`${file.path} 内容`} value={editorText(state.content)} onChange={event => buffer?.change(event.target.value)} spellCheck={false} /> : <Suspense fallback={<Loading />}><DesktopEditor projectId={project.id} folderId={file.folderId} path={file.path} modelURI={buffer?.uri} content={editorText(state.content)} language={language} onChange={(value, changes) => buffer?.change(value, changes)} isModelOpen={() => !!buffer && scope.isOpen(buffer)} /></Suspense>}
      </div>}
    </>}
    <Dialog open={draft !== null} onOpenChange={open => { if (!open) setDraft(null); }}><DialogContent className="draft-dialog"><DialogHeader><DialogTitle>本地草稿</DialogTitle><DialogDescription>{draft ? new Date(draft.updatedAt).toLocaleString() : ""} · {baseChanged ? "服务器内容已经变化" : "服务器基线未变化"}。恢复后仍需明确保存。</DialogDescription></DialogHeader>
      {!mobile && draft && state?.base && <div className="draft-diff"><Suspense fallback={<Loading />}><DesktopDiff original={state.base.content} modified={draft.content} language={language} /></Suspense></div>}
      {mobile && baseChanged && <p>请导出草稿，在电脑上比较和处理。</p>}
      <DialogFooter><Button variant="outline" onClick={() => setDraft(null)}>保留并关闭</Button><Button variant="outline" onClick={() => { if (draft) downloadDraft(draft.content, draft.file.path); }}>导出</Button><Button variant="outline" onClick={() => { if (draft) void buffer?.discardDraft(draft).then(() => setDraft(null)); }}>丢弃此草稿</Button>{(!mobile || !baseChanged) && <Button onClick={() => { if (draft) buffer?.restore(draft); setDraft(null); }}>恢复到编辑器</Button>}</DialogFooter>
    </DialogContent></Dialog>
    <Dialog open={compare && !!state?.comparison} onOpenChange={setCompare}><DialogContent className="draft-dialog"><DialogHeader><DialogTitle>文件冲突</DialogTitle><DialogDescription>服务器内容与本地输入分别保留；采用当前服务器版本保存后，后端仍会复验。</DialogDescription></DialogHeader>
      {!mobile && state?.comparison && <div className="draft-diff"><Suspense fallback={<Loading />}><DesktopDiff original={state.comparison.content} modified={state.content} language={language} /></Suspense></div>}
      <DialogFooter><Button variant="outline" onClick={() => setCompare(false)}>取消</Button><Button variant="outline" onClick={() => { if (state) downloadDraft(state.content, file.path); }}>导出输入</Button><Button variant="outline" onClick={() => void buffer?.reloadCompared().then(() => setCompare(false))}>保留草稿并加载服务器</Button>{!mobile && <Button onClick={() => void buffer?.save(true).then(() => setCompare(false))}>确认保存本地内容</Button>}</DialogFooter>
    </DialogContent></Dialog>
  </div>;
}
