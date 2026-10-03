import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";
import { viewStorage } from "./view-storage";
import { isFileLanguage } from "./file-language";

export type OpenFile = { folderId: string; path: string };
export type GroupIndex = 0 | 1 | 2 | 3;
export const groupIndexes: GroupIndex[] = [0, 1, 2, 3];
export type TerminalPosition = { region: "top" | "bottom"; group: GroupIndex };
type Four<T> = [T, T, T, T];
const four = <T,>(fn: (i: GroupIndex) => T): Four<T> => [fn(0), fn(1), fn(2), fn(3)];
export type ProjectView = {
  groups: Four<OpenFile[]>; active: Four<string | null>; focused: GroupIndex; split: boolean;
  mobileFiles: OpenFile[]; mobileActive: string | null; mobileView: "files" | "editor" | "terminal" | "search" | "git"; sidebar: "files" | "search" | "git"; mobileTerminal: string | null;
  terminals: Record<string, TerminalPosition>; terminalOrder: string[]; upperActive: Four<string | null>; lowerActive: Four<string | null>; lowerCount: number;
  expanded: Record<string, true>;
};
type WorkspaceView = {
  projects: Record<string, ProjectView>; languageModes: Record<string, Record<string, string>>; recoveryError: string | null;
  setLanguage(projectId: string, file: OpenFile, mode: string | undefined): void;
  open(projectId: string, file: OpenFile, group?: GroupIndex): void;
  activate(projectId: string, file: OpenFile, group: GroupIndex): void;
  close(projectId: string, file: OpenFile, group: GroupIndex): void;
  split(projectId: string, file: OpenFile): void;
  move(projectId: string, file: OpenFile, target: GroupIndex, source?: GroupIndex, before?: OpenFile): void;
  unsplit(projectId: string): void;
  relocate(projectId: string, source: OpenFile, target: OpenFile): void;
  remove(projectId: string, source: OpenFile): void;
  openMobile(projectId: string, file: OpenFile): void;
  closeMobile(projectId: string, file: OpenFile): void;
  mobileView(projectId: string, view: ProjectView["mobileView"]): void;
  placeTerminal(projectId: string, id: string, position: TerminalPosition): void;
  selectTerminal(projectId: string, id: string | null, position: TerminalPosition | "mobile"): void;
  orderTerminal(projectId: string, id: string, before: string, visibleOrder?: string[]): void;
  splitTerminal(projectId: string): void;
  mergeTerminalGroup(projectId: string, group: GroupIndex): void;
  expand(projectId: string, file: OpenFile, expanded: boolean): void;
};
export const fileKey = (file: OpenFile) => `${file.folderId}\u0000${file.path}`;
export const emptyView = (): ProjectView => ({ groups: [[], [], [], []], active: [null, null, null, null], focused: 0, split: false, mobileFiles: [], mobileActive: null, mobileView: "files", sidebar: "files", mobileTerminal: null, terminals: {}, terminalOrder: [], upperActive: [null, null, null, null], lowerActive: [null, null, null, null], lowerCount: 1, expanded: {} });
const sameFile = (a: OpenFile, b: OpenFile) => fileKey(a) === fileKey(b);
export function allOpenFiles(view: ProjectView | undefined): OpenFile[] { return [...(view?.groups.flat() ?? []), ...(view?.mobileFiles ?? [])]; }
const matches = (file: OpenFile, source: OpenFile) => file.folderId === source.folderId && (file.path === source.path || file.path.startsWith(`${source.path}/`));
function mutate(state: Pick<WorkspaceView, "projects" | "languageModes">, projectId: string, change: (view: ProjectView, modes: Record<string, string>) => void) {
  const original = state.projects[projectId] ?? emptyView();
  const modes = { ...state.languageModes[projectId] };
  const view: ProjectView = { ...emptyView(), ...original, groups: four(i => [...(original.groups[i] ?? [])]), active: four(i => original.active[i] ?? null), mobileFiles: [...(original.mobileFiles ?? [])], terminals: Object.fromEntries(Object.entries(original.terminals ?? {}).map(([id, position]) => [id, { ...position }])), terminalOrder: [...(original.terminalOrder ?? [])], upperActive: [...(original.upperActive ?? [null, null, null, null])], lowerActive: [...(original.lowerActive ?? [null, null, null, null])], expanded: { ...original.expanded } };
  change(view, modes);
  view.split = view.groups.slice(1).some(files => files.length > 0) || Object.values(view.terminals).some(position => position.region === "top" && position.group > 0);
  const openKeys = new Set(allOpenFiles(view).map(fileKey));
  for (const key of Object.keys(modes)) if (!openKeys.has(key)) delete modes[key];
  return { projects: { ...state.projects, [projectId]: view }, languageModes: { ...state.languageModes, [projectId]: modes } };
}
const groupIndex = (value: unknown): value is GroupIndex => value === 0 || value === 1 || value === 2 || value === 3;
const identifier = (value: unknown): value is string => typeof value === "string" && /^[A-Za-z0-9_-]{1,128}$/.test(value);
const record = (value: unknown): value is Record<string, unknown> => typeof value === "object" && value !== null && !Array.isArray(value);
function storedFile(value: unknown): value is OpenFile {
  return record(value) && identifier(value.folderId) && typeof value.path === "string" && value.path.length > 0 && value.path.length <= 4096 && !value.path.includes("\0") && !value.path.startsWith("/") && value.path.split("/").every(part => part !== "" && part !== "." && part !== "..");
}
// Both migration and merge validate persisted state; an unknown schema never becomes a live view.
export function restoreViews(value: unknown): Record<string, ProjectView> {
  if (!record(value) || !record(value.projects) || Object.keys(value.projects).length > 100) throw new Error("工作台记录格式或数量无效，已使用默认视图；本地草稿未删除。");
  const projects: Record<string, ProjectView> = {};
  for (const [id, raw] of Object.entries(value.projects)) {
    if (!identifier(id) || !record(raw) || !Array.isArray(raw.groups) || raw.groups.length < 2 || raw.groups.length > 4 || !raw.groups.every(group => Array.isArray(group) && group.every(storedFile))) throw new Error("工作台文件标签记录无效，已使用默认视图。");
    const view = emptyView();
    view.groups = four(i => { const group = raw.groups as OpenFile[][]; return [...new Map((group[i] ?? []).map(file => [fileKey(file), file])).values()]; });
    if (view.groups.flat().length > 100) throw new Error("工作台文件标签超过上限，已使用默认视图。");
    const active: unknown[] = Array.isArray(raw.active) ? raw.active : [];
    view.active = four(i => { const files = view.groups[i]; const last = files.at(-1); return files.some(file => fileKey(file) === active[i]) ? String(active[i]) : last ? fileKey(last) : null; });
    view.focused = groupIndex(raw.focused) ? raw.focused : 0;
    if (raw.mobileFiles !== undefined) { if (!Array.isArray(raw.mobileFiles) || raw.mobileFiles.length > 100 || !raw.mobileFiles.every(storedFile)) throw new Error("手机文件标签记录无效。"); view.mobileFiles = raw.mobileFiles; }
    view.mobileActive = view.mobileFiles.some(file => fileKey(file) === raw.mobileActive) ? String(raw.mobileActive) : null;
    if (raw.mobileView === "files" || raw.mobileView === "editor" || raw.mobileView === "terminal" || raw.mobileView === "search" || raw.mobileView === "git") view.mobileView = raw.mobileView;
    if (raw.sidebar === "files" || raw.sidebar === "search" || raw.sidebar === "git") view.sidebar = raw.sidebar;
    view.mobileTerminal = identifier(raw.mobileTerminal) ? raw.mobileTerminal : null;
    if (record(raw.terminals)) {
      if (Object.keys(raw.terminals).length > 500) throw new Error("终端视图记录超过上限。");
      for (const [key, position] of Object.entries(raw.terminals)) { if (!identifier(key) || !record(position) || (position.region !== "top" && position.region !== "bottom") || !groupIndex(position.group)) throw new Error("终端位置记录无效。"); view.terminals[key] = { region: position.region, group: position.group }; }
    }
    view.terminalOrder = Array.isArray(raw.terminalOrder) ? [...new Set(raw.terminalOrder.filter(identifier))].slice(0, 500) : [];
    view.upperActive = four(i => Array.isArray(raw.upperActive) && identifier(raw.upperActive[i]) ? String(raw.upperActive[i]) : null);
    view.lowerActive = four(i => Array.isArray(raw.lowerActive) && identifier(raw.lowerActive[i]) ? String(raw.lowerActive[i]) : null);
    view.lowerCount = typeof raw.lowerCount === "number" && Number.isInteger(raw.lowerCount) && raw.lowerCount >= 1 && raw.lowerCount <= 4 ? raw.lowerCount : 1;
    for (const position of Object.values(view.terminals)) if (position.region === "bottom") view.lowerCount = Math.max(view.lowerCount, position.group + 1);
    if (record(raw.expanded) && Object.keys(raw.expanded).length <= 1000) for (const [key, expanded] of Object.entries(raw.expanded)) { const n = key.indexOf("\0"); if (expanded === true && storedFile({ folderId: key.slice(0, n), path: key.slice(n + 1) })) view.expanded[key] = true; }
    view.split = view.groups.slice(1).some(files => files.length > 0) || Object.values(view.terminals).some(position => position.region === "top" && position.group > 0);
    projects[id] = view;
  }
  return projects;
}
export const useWorkspaceView = create<WorkspaceView>()(persist((set) => ({
  projects: {}, languageModes: {}, recoveryError: null,
  setLanguage: (projectId, file, mode) => set(state => mutate(state, projectId, (_view, modes) => { if (mode === undefined) delete modes[fileKey(file)]; else if (isFileLanguage(mode)) modes[fileKey(file)] = mode; })),
  open: (projectId, file, group) => set(state => mutate(state, projectId, view => {
    const index = group ?? view.focused;
    if (!view.groups[index].some(item => sameFile(item, file)) && view.groups.flat().length < 100) view.groups[index].push(file);
    if (view.groups[index].some(item => sameFile(item, file))) view.active[index] = fileKey(file); view.focused = index;
  })),
  activate: (projectId, file, group) => set(state => mutate(state, projectId, view => { if (view.groups[group].some(item => sameFile(item, file))) { view.active[group] = fileKey(file); view.focused = group; } })),
  close: (projectId, file, group) => set(state => mutate(state, projectId, view => {
    view.groups[group] = view.groups[group].filter(item => !sameFile(item, file)); const last = view.groups[group].at(-1);
    if (view.active[group] === fileKey(file)) view.active[group] = last ? fileKey(last) : null;
    if (!view.groups[group].length) view.focused = groupIndexes.find(i => view.groups[i].length) ?? 0;
  })),
  split: (projectId, file) => set(state => mutate(state, projectId, view => {
    const index = groupIndexes.find(i => i > 0 && !view.groups[i].length && !Object.values(view.terminals).some(position => position.region === "top" && position.group === i));
    if (index === undefined || view.groups.flat().length >= 100) return;
    view.groups[index].push(file); view.active[index] = fileKey(file); view.focused = index;
  })),
  move: (projectId, file, target, source, before) => set(state => mutate(state, projectId, view => {
    const origin = source ?? groupIndexes.find(i => i !== target && view.groups[i].some(item => sameFile(item, file)));
    if (origin === undefined || !view.groups[origin].some(item => sameFile(item, file))) return;
    view.groups[origin] = view.groups[origin].filter(item => !sameFile(item, file)); const last = view.groups[origin].at(-1);
    if (view.active[origin] === fileKey(file)) view.active[origin] = last ? fileKey(last) : null;
    view.groups[target] = view.groups[target].filter(item => !sameFile(item, file));
    const index = before ? view.groups[target].findIndex(item => sameFile(item, before)) : -1;
    view.groups[target].splice(index < 0 ? view.groups[target].length : index, 0, file); view.active[target] = fileKey(file); view.focused = target;
  })),
  unsplit: projectId => set(state => mutate(state, projectId, view => {
    for (const index of groupIndexes.slice(1)) { for (const file of view.groups[index]) if (!view.groups[0].some(item => sameFile(item, file))) view.groups[0].push(file); view.groups[index] = []; view.active[index] = null; }
    for (const position of Object.values(view.terminals)) if (position.region === "top") position.group = 0;
    view.active[0] ??= view.groups[0].at(-1) ? fileKey(view.groups[0].at(-1)!) : null; view.focused = 0;
  })),
  relocate: (projectId, source, target) => set(state => mutate(state, projectId, (view, modes) => {
    const relocate = (item: OpenFile) => matches(item, source) ? { folderId: target.folderId, path: target.path + item.path.slice(source.path.length) } : item;
    for (const [key, mode] of Object.entries(modes)) { const n = key.indexOf("\0"); const file = { folderId: key.slice(0, n), path: key.slice(n + 1) }; if (matches(file, source)) { delete modes[key]; modes[fileKey(relocate(file))] = mode; } }
    for (const index of groupIndexes) { const active = view.groups[index].find(item => fileKey(item) === view.active[index]); view.groups[index] = [...new Map(view.groups[index].map(relocate).map(file => [fileKey(file), file])).values()]; if (active) view.active[index] = fileKey(relocate(active)); }
    const active = view.mobileFiles.find(item => fileKey(item) === view.mobileActive); view.mobileFiles = [...new Map(view.mobileFiles.map(relocate).map(file => [fileKey(file), file])).values()]; if (active) view.mobileActive = fileKey(relocate(active));
  })),
  remove: (projectId, source) => set(state => mutate(state, projectId, view => {
    for (const index of groupIndexes) { view.groups[index] = view.groups[index].filter(item => !matches(item, source)); if (!view.groups[index].some(item => fileKey(item) === view.active[index])) view.active[index] = view.groups[index].at(-1) ? fileKey(view.groups[index].at(-1)!) : null; }
    view.mobileFiles = view.mobileFiles.filter(item => !matches(item, source)); if (!view.mobileFiles.some(item => fileKey(item) === view.mobileActive)) view.mobileActive = view.mobileFiles.at(-1) ? fileKey(view.mobileFiles.at(-1)!) : null;
  })),
  openMobile: (projectId, file) => set(state => mutate(state, projectId, view => { if (!view.mobileFiles.some(item => sameFile(item, file)) && view.mobileFiles.length < 100) view.mobileFiles.push(file); view.mobileActive = fileKey(file); view.mobileView = "editor"; })),
  closeMobile: (projectId, file) => set(state => mutate(state, projectId, view => { view.mobileFiles = view.mobileFiles.filter(item => !sameFile(item, file)); if (view.mobileActive === fileKey(file)) view.mobileActive = view.mobileFiles.at(-1) ? fileKey(view.mobileFiles.at(-1)!) : null; })),
  mobileView: (projectId, mobileView) => set(state => mutate(state, projectId, view => { view.mobileView = mobileView; })),
  placeTerminal: (projectId, id, position) => set(state => mutate(state, projectId, view => { view.terminals[id] = position; if (!view.terminalOrder.includes(id)) view.terminalOrder.push(id); (position.region === "top" ? view.upperActive : view.lowerActive)[position.group] = id; if (position.region === "top") view.focused = position.group; else view.lowerCount = Math.max(view.lowerCount, position.group + 1); })),
  selectTerminal: (projectId, id, position) => set(state => mutate(state, projectId, view => { if (position === "mobile") view.mobileTerminal = id; else (position.region === "top" ? view.upperActive : view.lowerActive)[position.group] = id; })),
  orderTerminal: (projectId, id, before, visibleOrder = []) => set(state => mutate(state, projectId, view => { view.terminalOrder = [...new Set([...view.terminalOrder, ...visibleOrder])].filter(item => item !== id); const index = view.terminalOrder.indexOf(before); view.terminalOrder.splice(index < 0 ? view.terminalOrder.length : index, 0, id); })),
  splitTerminal: projectId => set(state => mutate(state, projectId, view => { view.lowerCount = Math.min(4, view.lowerCount + 1); })),
  mergeTerminalGroup: (projectId, group) => set(state => mutate(state, projectId, view => {
    if (view.lowerCount <= 1 || group >= view.lowerCount) return;
    const target = group === 0 ? 1 : groupIndexes[group - 1] ?? 0;
    const active = view.lowerActive;
    for (const position of Object.values(view.terminals)) if (position.region === "bottom") {
      if (position.group === group) position.group = target;
      if (position.group > group) position.group = groupIndexes[position.group - 1] ?? 0;
    }
    view.lowerCount--;
    view.lowerActive = four(i => i >= view.lowerCount ? null : active[i >= group ? i + 1 : i] ?? null);
    const mergedGroup = target > group ? groupIndexes[target - 1] ?? 0 : target;
    view.lowerActive[mergedGroup] ??= active[group];
  })),
  expand: (projectId, file, expanded) => set(state => mutate(state, projectId, view => { if (expanded && Object.keys(view.expanded).length < 1000) view.expanded[fileKey(file)] = true; else delete view.expanded[fileKey(file)]; })),
}), { name: "persistty.workspace-view.v1", version: 2, storage: createJSONStorage(() => viewStorage), partialize: state => ({ projects: state.projects }), migrate: (value, version) => { if (version !== 0 && version !== 1) throw new Error("工作台记录版本不支持；本地草稿未删除。"); return { projects: restoreViews(value) }; }, merge: (persisted, current) => { try { return persisted === undefined ? current : { ...current, projects: restoreViews(persisted) }; } catch (error) { return { ...current, recoveryError: error instanceof Error ? error.message : "工作台恢复失败。" }; } }, onRehydrateStorage: () => (_state, error) => { if (error) queueMicrotask(() => useWorkspaceView.setState({ recoveryError: "工作台记录无法恢复，已使用默认视图；本地草稿未删除。" })); } }));
