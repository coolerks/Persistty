import { api, ApiError, errorMessage } from "@/lib/api/client";
import type { FileContent, FileVersion, Project } from "@/lib/api/decoder";
import { editorURI } from "./editor-model-lifecycle";
import { draftStorage, type DraftStorage, type FileDraft } from "./editor-drafts";
import { applyEditorChanges, applyEditorText, type TextChange } from "./editor-text";
import { allOpenFiles, fileKey, useWorkspaceView, type OpenFile } from "./workspace-view";
import { elevationAPI, type ElevationAPI } from "@/lib/api/elevation-client";
import type { ElevationPrepared, ElevationResult } from "@/lib/api/elevation-decoder";

export type ElevationAttempt = { buffer: FileBuffer; epoch: number; content: string; generation: number; expected: FileVersion; projectVersion: number; file: OpenFile; csrf: string; request: ElevationPrepared | null; result: ElevationResult | null; phase: "preparing" | "prepared" | "executing" | "result" | "unknown"; error: unknown; querying: boolean; controller: AbortController };

export type SaveState = "saved" | "pending" | "saving" | "conflict" | "failed" | "paused";
export type BufferState = { status: "loading" | "ready" | "error"; content: string; base: FileContent | null; generation: number; saveState: SaveState; error: unknown; draftError: string | null; drafts: FileDraft[]; comparison: FileContent | null };
export type EditorAPI = Pick<typeof api, "content" | "saveContent" | "metadata">;
export const sameVersion = (a: FileVersion, b: FileVersion) => a.etag === b.etag && a.identity === b.identity && a.mtime === b.mtime && a.size === b.size;
export class FileBuffer {
  readonly id = crypto.randomUUID();
  readonly uri: string;
  readonly aliases = new Map<string, OpenFile>();
  state: BufferState = { status: "loading", content: "", base: null, generation: 0, saveState: "saved", error: null, draftError: null, drafts: [], comparison: null };
  private timer: ReturnType<typeof setTimeout> | undefined;
  private request: AbortController | undefined;
  private saving = false;
  private epoch = 0;
  private draftQueue: Promise<void> = Promise.resolve();
  private suspended = false;
  private replacementHold = 0;
  sourceRoot: string;
  constructor(readonly scope: EditorScope, public file: OpenFile) {
    this.sourceRoot = scope.project.folders.find(folder => folder.id === file.folderId)?.path ?? "";
    this.aliases.set(fileKey(file), file); this.uri = editorURI(scope.project.id, file);
  }
  get dirty(): boolean { return !!this.state.base && this.state.content !== this.state.base.content; }
  get draftId(): string { return `${this.scope.viewId}:${this.id}`; }
  update(change: Partial<BufferState>): void { this.state = { ...this.state, ...change }; this.scope.emit(); }
  async load(): Promise<void> {
    const epoch = ++this.epoch; this.request?.abort(); const controller = new AbortController(); this.request = controller;
    try {
      const snapshot = await this.scope.client.content(this.scope.project.id, this.file.folderId, this.scope.project.version, this.file.path, controller.signal);
      if (epoch !== this.epoch || controller.signal.aborted) return;
      const existing = this.scope.findIdentity(snapshot.version.identity, this);
      if (existing) { for (const file of this.aliases.values()) this.scope.bind(file, existing); this.scope.emit(); return; }
      let drafts: FileDraft[] = []; let draftError: string | null = null;
      try { drafts = (await this.scope.storage.list(this.scope.project.id)).filter(draft => draft.base.version.identity === snapshot.version.identity || (draft.file.folderId === this.file.folderId && draft.file.path === this.file.path)); }
      catch (reason) { draftError = errorMessage(reason); }
      if (epoch !== this.epoch || controller.signal.aborted) return;
      const merged = this.scope.findIdentity(snapshot.version.identity, this);
      if (merged) { for (const file of this.aliases.values()) this.scope.bind(file, merged); this.scope.emit(); return; }
      this.update({ status: "ready", base: snapshot, content: snapshot.content, saveState: "saved", error: null, drafts, draftError });
    } catch (error) { if (epoch === this.epoch && !controller.signal.aborted) this.update({ status: "error", error }); }
  }
  change(text: string, changes?: readonly TextChange[]): void {
    if (this.state.status !== "ready") return;
    const content = changes ? applyEditorChanges(this.state.content, changes) : applyEditorText(this.state.content, text);
    if (content === this.state.content) return;
    this.update({ content, generation: this.state.generation + 1, saveState: this.suspended ? (this.state.saveState === "saved" ? "paused" : this.state.saveState) : "pending" });
    void this.persist(); this.schedule();
  }
  private schedule(): void {
    clearTimeout(this.timer);
    if (!this.dirty || this.suspended || this.saving || !this.scope.csrf) return;
    this.timer = setTimeout(() => { void this.save(); }, 1000);
  }
  persist(): Promise<boolean> {
    const { base, content, generation } = this.state;
    if (!base || !this.dirty) return Promise.resolve(true);
    const draft: FileDraft = { schema: 1, id: this.draftId, projectId: this.scope.project.id, file: this.file, sourceRoot: this.sourceRoot, viewId: this.scope.viewId, generation, base, content, updatedAt: Date.now() };
    const result = this.draftQueue.then(async () => {
      try { await this.scope.storage.put(draft); this.update({ draftError: null }); return true; }
      catch (reason) { this.update({ draftError: errorMessage(reason) }); return false; }
    });
    this.draftQueue = result.then(() => undefined); return result;
  }
  async save(explicit = false): Promise<void> {
    clearTimeout(this.timer);
    const base = this.state.base;
    if (!base || !this.dirty || this.saving || this.scope.elevation?.buffer === this || this.replacementHold > 0 || !this.scope.csrf || (this.suspended && !explicit)) return;
    if (this.state.saveState === "conflict" && !this.state.comparison) return;
    const epoch = this.epoch; const content = this.state.content; const generation = this.state.generation;
    const expected = this.state.comparison ?? base;
    const controller = new AbortController(); this.request = controller; this.saving = true;
    this.update({ saveState: "saving", error: null });
    try {
      const result = await this.scope.client.saveContent(this.scope.project.id, this.file.folderId, { project_version: this.scope.project.version, path: this.file.path, expected_version: expected.version, content }, this.scope.csrf, controller.signal);
      if (epoch !== this.epoch || controller.signal.aborted) return;
      this.suspended = false;
      this.update({ base: { content, version: result.version, kind: "text" }, comparison: null, saveState: this.state.content === content ? "saved" : "pending" });
      // Wait queued draft writes before removing only this view's submitted revision.
      await this.draftQueue;
      try { await this.scope.storage.remove(this.draftId, generation); }
      catch (reason) { this.update({ draftError: errorMessage(reason) }); }
      if (this.dirty) await this.persist();
    } catch (error) {
      if (epoch !== this.epoch || controller.signal.aborted) return;
      this.suspended = true;
      this.update({ saveState: error instanceof ApiError && error.status === 409 ? "conflict" : "failed", error, comparison: null });
      await this.persist();
    } finally { this.saving = false; if (epoch === this.epoch) this.schedule(); }
  }
  get canElevate(): boolean { return this.dirty && this.state.status === "ready" && !this.saving && this.state.error instanceof ApiError && this.state.error.code === "permission_denied"; }
  async beginElevation(): Promise<void> {
    if (!this.canElevate || this.scope.elevation || !this.scope.csrf || !this.state.base) return;
    this.suspend();
    const attempt: ElevationAttempt = { buffer: this, epoch: this.epoch, content: this.state.content, generation: this.state.generation, expected: { ...(this.state.comparison ?? this.state.base).version }, projectVersion: this.scope.project.version, file: { ...this.file }, csrf: this.scope.csrf, request: null, result: null, phase: "preparing", error: null, querying: false, controller: new AbortController() };
    this.scope.elevation = attempt; this.scope.emit();
    try {
      const prepared = await this.scope.elevationClient.prepare(this.scope.project.id, attempt.file.folderId, { project_version: attempt.projectVersion, path: attempt.file.path, expected_version: attempt.expected, content: attempt.content }, attempt.csrf, attempt.controller.signal);
      if (!this.currentElevation(attempt)) { void this.scope.elevationClient.cancel(prepared.id, attempt.csrf, new AbortController().signal).catch(() => {}); return; }
      attempt.request = prepared; attempt.phase = "prepared"; this.scope.emit();
    } catch (error) { if (this.currentElevation(attempt)) { attempt.phase = "result"; attempt.error = error; this.scope.emit(); } }
  }
  private currentElevation(attempt: ElevationAttempt): boolean { return this.scope.elevation === attempt && this.epoch === attempt.epoch && this.scope.project.version === attempt.projectVersion && this.scope.csrf === attempt.csrf && fileKey(this.file) === fileKey(attempt.file); }
  async executeElevation(password: string): Promise<void> {
    const attempt = this.scope.elevation;
    if (!attempt || attempt.buffer !== this || !attempt.request || attempt.phase !== "prepared" || !this.currentElevation(attempt)) return;
    if (Date.parse(attempt.request.expires_at) <= Date.now()) { this.expireElevation(); return; }
    attempt.phase = "executing"; attempt.error = null; this.saving = true; this.update({ saveState: "saving" });
    try {
      const pending = this.scope.elevationClient.execute(attempt.request.id, attempt.content, password, attempt.csrf, attempt.controller.signal); password = "";
      const result = await pending;
      if (this.currentElevation(attempt)) await this.acceptElevationResult(attempt, result);
    } catch (error) { if (this.currentElevation(attempt)) { attempt.phase = "unknown"; attempt.error = error; this.update({ saveState: "failed", error: new Error("提权结果尚未确认，请查询结果；输入已保留。") }); } }
    finally { this.saving = false; }
  }
  private async acceptElevationResult(attempt: ElevationAttempt, result: ElevationResult): Promise<void> {
    if (!attempt.request || result.id !== attempt.request.id) throw new Error("提权响应与当前请求不一致。");
    attempt.result = result; attempt.phase = ["executing", "indeterminate"].includes(result.state) ? "unknown" : "result";
    if (result.state === "applied" && result.version) {
      const dirty = this.state.content !== attempt.content; this.suspended = dirty;
      this.update({ base: { content: attempt.content, version: result.version, kind: "text" }, comparison: null, error: null, saveState: dirty ? "paused" : "saved" });
      await this.draftQueue;
      if (!this.currentElevation(attempt)) return;
      try { await this.scope.storage.remove(this.draftId, attempt.generation); } catch (reason) { this.update({ draftError: errorMessage(reason) }); }
      if (this.dirty) await this.persist();
    } else {
      const conflict = result.code === "conflict";
      this.update({ saveState: conflict ? "conflict" : "failed", ...(conflict ? { error: new ApiError(409, "conflict", "文件已变化，请先比较后重新保存。", result.id, null), comparison: null } : {}) });
      await this.persist();
    }
    this.scope.emit();
  }
  async queryElevation(): Promise<void> {
    const attempt = this.scope.elevation;
    if (!attempt || attempt.buffer !== this || !attempt.request || attempt.phase !== "unknown" || attempt.querying || !this.currentElevation(attempt)) return;
    attempt.error = null; attempt.querying = true; this.scope.emit();
    try { const result = await this.scope.elevationClient.status(attempt.request.id, attempt.controller.signal); if (this.currentElevation(attempt)) await this.acceptElevationResult(attempt, result); }
    catch (error) { if (this.currentElevation(attempt)) { attempt.error = error; this.scope.emit(); } }
    finally { attempt.querying = false; if (this.currentElevation(attempt)) this.scope.emit(); }
  }
  expireElevation(): void { const attempt = this.scope.elevation; if (attempt?.buffer === this && attempt.phase === "prepared" && attempt.request) { attempt.phase = "result"; attempt.result = { id: attempt.request.id, state: "expired", code: "expired", version: null }; this.scope.emit(); } }
  dismissElevation(): void {
    const attempt = this.scope.elevation; if (attempt?.buffer !== this) return;
    attempt.controller.abort(); this.scope.elevation = null;
    if (attempt.request && !["applied", "rejected", "cancelled", "expired"].includes(attempt.result?.state ?? "")) void this.scope.elevationClient.cancel(attempt.request.id, attempt.csrf, new AbortController().signal).catch(() => {});
    if (attempt.phase === "executing" || attempt.phase === "unknown") this.update({ saveState: "failed", error: new Error("提权结果尚未确认，请重新读取服务器并比较，输入已保留。") });
    if (attempt.result?.state === "applied" && !this.suspended) this.schedule();
    this.scope.emit();
  }
  suspend(): void { this.dismissElevation(); this.suspended = true; clearTimeout(this.timer); this.request?.abort(); this.epoch++; if (this.state.status === "ready" && ["pending", "saving"].includes(this.state.saveState)) this.update({ saveState: "paused" }); }
  holdReplacement(version: FileVersion): (() => Promise<void>) | null {
    if (this.state.status !== "ready" || this.dirty || this.saving || this.suspended || this.replacementHold || this.state.saveState !== "saved" || !this.state.base || !sameVersion(this.state.base.version, version)) return null;
    this.replacementHold++; this.suspend();
    let released = false;
    return async () => { if (released) return; released = true; await this.refresh(); this.replacementHold--; if (this.dirty) { this.update({ saveState: "paused" }); await this.persist(); } else this.resume(); };
  }
  resume(): void { if (this.state.saveState === "saved" || this.state.saveState === "paused") { this.suspended = false; this.update({ saveState: this.dirty ? "pending" : "saved" }); this.schedule(); } }
  async refresh(): Promise<void> {
    if (this.state.status !== "ready" || this.saving) return;
    const epoch = this.epoch;
    const base = this.state.base;
    try {
      const snapshot = await this.scope.client.content(this.scope.project.id, this.file.folderId, this.scope.project.version, this.file.path, new AbortController().signal);
      if (epoch !== this.epoch || this.saving || !base || !this.state.base || !sameVersion(base.version, this.state.base.version) || sameVersion(snapshot.version, this.state.base.version)) return;
      if (this.dirty || this.saving) { this.suspend(); this.update({ saveState: "conflict", error: new Error("文件已在外部修改，本地输入已保留。"), comparison: null }); await this.persist(); }
      else this.update({ base: snapshot, content: snapshot.content, saveState: "saved", error: null });
    } catch (error) { if (epoch === this.epoch && !this.saving && base === this.state.base) { this.suspended = true; clearTimeout(this.timer); this.update({ saveState: "failed", error }); } }
  }
  async compare(): Promise<void> {
    this.suspend(); const epoch = this.epoch;
    try { const comparison = await this.scope.client.content(this.scope.project.id, this.file.folderId, this.scope.project.version, this.file.path, new AbortController().signal); if (epoch === this.epoch) this.update({ comparison, error: null }); }
    catch (error) { if (epoch === this.epoch) this.update({ error }); }
  }
  restore(draft: FileDraft): void {
    this.suspend(); this.update({ content: draft.content, generation: this.state.generation + 1, saveState: "paused", comparison: this.state.base }); void this.persist();
  }
  async discardDraft(draft: FileDraft): Promise<void> {
    try { await this.scope.storage.remove(draft.id, draft.generation); this.update({ drafts: this.state.drafts.filter(item => item.id !== draft.id) }); }
    catch (reason) { this.update({ draftError: errorMessage(reason) }); }
  }
  async reloadCompared(): Promise<void> {
    const comparison = this.state.comparison;
    if (!comparison || !await this.persist()) return;
    this.suspended = false; this.update({ base: comparison, content: comparison.content, comparison: null, saveState: "saved", error: null });
  }
}
export class EditorScope {
  elevation: ElevationAttempt | null = null;
  readonly viewId = crypto.randomUUID();
  readonly buffers = new Map<string, FileBuffer>();
  private listeners = new Set<() => void>();
  revision = 0;
  private configuration = 0;
  csrf = "";
  constructor(public project: Project, readonly storage: DraftStorage = draftStorage, readonly client: EditorAPI = api, readonly elevationClient: ElevationAPI = elevationAPI) {}
  emit = (): void => { this.revision++; for (const listener of this.listeners) listener(); };
  subscribe = (listener: () => void): (() => void) => { this.listeners.add(listener); return () => this.listeners.delete(listener); };
  snapshot = (): number => this.revision;
  open(file: OpenFile): FileBuffer {
    let buffer = this.buffers.get(fileKey(file));
    if (!buffer) { buffer = new FileBuffer(this, file); this.buffers.set(fileKey(file), buffer); void buffer.load(); }
    return buffer;
  }
  bind(file: OpenFile, buffer: FileBuffer): void { buffer.aliases.set(fileKey(file), file); this.buffers.set(fileKey(file), buffer); }
  findIdentity(identity: string, except: FileBuffer): FileBuffer | undefined { return [...new Set(this.buffers.values())].find(buffer => buffer !== except && buffer.state.base?.version.identity === identity); }
  isOpen(buffer: FileBuffer): boolean { return allOpenFiles(useWorkspaceView.getState().projects[this.project.id]).some(file => buffer.aliases.has(fileKey(file))); }
  closeInactive(): void {
    let removed = false;
    for (const buffer of new Set(this.buffers.values())) if (!this.isOpen(buffer)) {
      buffer.suspend(); void buffer.persist();
      // Clean closed files must reload on reopen; do not retain a suspended cache forever.
      if (!buffer.dirty) for (const key of buffer.aliases.keys()) { this.buffers.delete(key); removed = true; }
    }
    if (removed) this.emit();
  }
  async protect(file?: OpenFile): Promise<boolean> {
    const buffers = [...new Set(this.buffers.values())].filter(buffer => !file || [...buffer.aliases.values()].some(item => item.folderId === file.folderId && (item.path === file.path || item.path.startsWith(file.path + "/"))));
    const results = await Promise.all(buffers.map(async buffer => { buffer.suspend(); return buffer.persist(); })); return results.every(Boolean);
  }
  async refresh(): Promise<void> { await Promise.all([...new Set(this.buffers.values())].filter(buffer => this.isOpen(buffer)).map(buffer => buffer.refresh())); }
  prepareReplacement(files: readonly { id: string; folder_id: string; path: string; version: FileVersion }[]): { protectedIDs: string[]; release(): Promise<void> } {
    const protectedIDs: string[] = [], releases: (() => Promise<void>)[] = [];
    for (const file of files) {
      const matching = [...new Set(this.buffers.values())].filter(buffer => buffer.state.base?.version.identity === file.version.identity || buffer.aliases.has(fileKey({ folderId: file.folder_id, path: file.path })));
      for (const buffer of matching) {
        const release = buffer.holdReplacement(file.version);
        if (!release) protectedIDs.push(file.id); else releases.push(release);
      }
    }
    return { protectedIDs: [...new Set(protectedIDs)], release: async () => { await Promise.all(releases.map(release => release())); } };
  }
  async configure(project: Project, csrf: string): Promise<void> {
    if (project.version < this.project.version) return;
    if (project.version === this.project.version && csrf === this.csrf) return;
    const configuration = ++this.configuration;
    const versionChanged = project.version !== this.project.version;
    this.csrf = csrf;
    if (!versionChanged) { if ([...this.buffers.values()].some(buffer => buffer.dirty)) await this.protect(); return; }
    await this.protect(); if (configuration !== this.configuration) return; this.project = project;
    for (const buffer of new Set(this.buffers.values())) {
      let candidate = [...buffer.aliases.values()].find(file => project.folders.some(folder => folder.id === file.folderId));
      if (!candidate) {
        const absolute = buffer.sourceRoot.replace(/\/$/, "") + "/" + buffer.file.path;
        const folder = [...project.folders].sort((a, b) => b.path.length - a.path.length).find(folder => absolute.startsWith(folder.path.replace(/\/$/, "") + "/"));
        if (folder) candidate = { folderId: folder.id, path: absolute.slice(folder.path.replace(/\/$/, "").length + 1) };
      }
      if (candidate && !buffer.state.base) { buffer.file = candidate; buffer.sourceRoot = project.folders.find(folder => folder.id === candidate.folderId)?.path ?? buffer.sourceRoot; this.bind(candidate, buffer); void buffer.load(); continue; }
      if (candidate && buffer.state.base) {
        try {
          const meta = await this.client.metadata(project.id, candidate.folderId, project.version, candidate.path, new AbortController().signal);
          if (configuration !== this.configuration) return;
          if (!sameVersion(meta.version, buffer.state.base.version)) throw new Error("配置变化后文件内容或身份已改变，请导出并人工处理。");
          const previous = buffer.file; const aliases = [...buffer.aliases.values()];
          buffer.file = candidate; buffer.sourceRoot = project.folders.find(folder => folder.id === candidate.folderId)?.path ?? buffer.sourceRoot;
          this.bind(candidate, buffer);
          if (!await buffer.persist()) throw new Error("草稿关联迁移失败，原草稿与输入已保留。");
          useWorkspaceView.getState().relocate(project.id, previous, candidate);
          for (const alias of aliases) if (!project.folders.some(folder => folder.id === alias.folderId)) { useWorkspaceView.getState().relocate(project.id, alias, candidate); this.buffers.delete(fileKey(alias)); buffer.aliases.delete(fileKey(alias)); }
          if (buffer.dirty) buffer.update({ saveState: "paused", comparison: buffer.state.base }); else buffer.resume();
        } catch (error) { if (configuration !== this.configuration) return; buffer.update({ saveState: "failed", error }); }
      } else buffer.update({ status: buffer.state.base ? "ready" : "error", saveState: "failed", error: new Error("文件夹关联已移除，输入与草稿已保留，请导出内容。") });
    }
  }
  async relocate(source: OpenFile, target: OpenFile): Promise<void> {
    for (const buffer of new Set(this.buffers.values())) {
      let affected = false;
      for (const alias of [...buffer.aliases.values()]) {
        if (alias.folderId !== source.folderId || (alias.path !== source.path && !alias.path.startsWith(source.path + "/"))) continue;
        affected = true;
        const next = { folderId: target.folderId, path: target.path + alias.path.slice(source.path.length) };
        this.buffers.delete(fileKey(alias)); buffer.aliases.delete(fileKey(alias)); this.bind(next, buffer);
        if (fileKey(buffer.file) === fileKey(alias)) { buffer.file = next; buffer.sourceRoot = this.project.folders.find(folder => folder.id === next.folderId)?.path ?? ""; }
      }
      if (affected) { await buffer.persist(); await buffer.refresh(); buffer.resume(); }
    }
    this.emit();
  }
  dispose(): void { this.configuration++; for (const buffer of new Set(this.buffers.values())) { buffer.suspend(); void buffer.persist(); } }
}
export const editorScopes = new Map<string, EditorScope>();
