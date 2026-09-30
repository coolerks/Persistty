import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";
import { isFileLanguage } from "./file-language";

export type OpenFile = { folderId: string; path: string };
type ProjectView = { groups: [OpenFile[], OpenFile[]]; active: [string | null, string | null]; focused: 0 | 1; split: boolean };
type WorkspaceView = {
  projects: Record<string, ProjectView>;
  languageModes: Record<string, Record<string, string>>;
  setLanguage(projectId: string, file: OpenFile, mode: string | undefined): void;
  open(projectId: string, file: OpenFile, group?: 0 | 1): void;
  activate(projectId: string, file: OpenFile, group: 0 | 1): void;
  close(projectId: string, file: OpenFile, group: 0 | 1): void;
  split(projectId: string, file: OpenFile): void;
  move(projectId: string, file: OpenFile, target: 0 | 1): void;
  unsplit(projectId: string): void;
  relocate(projectId: string, source: OpenFile, target: OpenFile): void;
  remove(projectId: string, source: OpenFile): void;
};

export const fileKey = (file: OpenFile) => `${file.folderId}\u0000${file.path}`;
const emptyView = (): ProjectView => ({ groups: [[], []], active: [null, null], focused: 0, split: false });
const sameFile = (a: OpenFile, b: OpenFile) => fileKey(a) === fileKey(b);

function mutate(state: Pick<WorkspaceView, "projects" | "languageModes">, projectId: string, change: (view: ProjectView, modes: Record<string, string>) => void) {
  const original = state.projects[projectId] ?? emptyView();
  const modes = { ...state.languageModes[projectId] };
  const view: ProjectView = { groups: [[...original.groups[0]], [...original.groups[1]]], active: [...original.active], focused: original.focused, split: original.split };
  change(view, modes);
  const openKeys = new Set(view.groups.flat().map(fileKey));
  for (const key of Object.keys(modes)) if (!openKeys.has(key)) delete modes[key];
  return { projects: { ...state.projects, [projectId]: view }, languageModes: { ...state.languageModes, [projectId]: modes } };
}

export const useWorkspaceView = create<WorkspaceView>()(persist((set) => ({
  projects: {},
  languageModes: {},
  setLanguage: (projectId, file, mode) => set(state => mutate(state, projectId, (_view, modes) => {
    if (mode === undefined) delete modes[fileKey(file)];
    else if (isFileLanguage(mode)) modes[fileKey(file)] = mode;
  })),
  open: (projectId, file, group) => set(state => mutate(state, projectId, view => {
    const index = group ?? view.focused;
    if (!view.groups[index].some(item => sameFile(item, file))) view.groups[index].push(file);
    view.active[index] = fileKey(file); view.focused = index;
    if (index === 1) view.split = true;
  })),
  activate: (projectId, file, group) => set(state => mutate(state, projectId, view => {
    if (!view.groups[group].some(item => sameFile(item, file))) return;
    view.active[group] = fileKey(file); view.focused = group;
  })),
  close: (projectId, file, group) => set(state => mutate(state, projectId, view => {
    view.groups[group] = view.groups[group].filter(item => !sameFile(item, file));
    const last = view.groups[group].at(-1);
    if (view.active[group] === fileKey(file)) view.active[group] = last ? fileKey(last) : null;
    if (group === 1 && view.groups[1].length === 0) { view.split = false; view.focused = 0; }
  })),
  split: (projectId, file) => set(state => mutate(state, projectId, view => {
    if (!view.groups[1].some(item => sameFile(item, file))) view.groups[1].push(file);
    view.active[1] = fileKey(file); view.focused = 1; view.split = true;
  })),
  move: (projectId, file, target) => set(state => mutate(state, projectId, view => {
    const source: 0 | 1 = target === 0 ? 1 : 0;
    if (!view.groups[source].some(item => sameFile(item, file))) return;
    view.groups[source] = view.groups[source].filter(item => !sameFile(item, file));
    const last = view.groups[source].at(-1);
    if (view.active[source] === fileKey(file)) view.active[source] = last ? fileKey(last) : null;
    if (!view.groups[target].some(item => sameFile(item, file))) view.groups[target].push(file);
    view.active[target] = fileKey(file); view.focused = target;
    view.split = view.groups[1].length > 0;
  })),
  unsplit: projectId => set(state => mutate(state, projectId, view => {
    for (const file of view.groups[1]) if (!view.groups[0].some(item => sameFile(item, file))) view.groups[0].push(file);
    view.active[0] = view.active[1] ?? view.active[0]; view.groups[1] = []; view.active[1] = null; view.focused = 0; view.split = false;
  })),
  relocate: (projectId, source, target) => set(state => mutate(state, projectId, (view, modes) => {
    for (const [key, mode] of Object.entries(modes)) {
      const separator = key.indexOf("\u0000");
      const folderId = key.slice(0, separator); const path = key.slice(separator + 1);
      if (folderId === source.folderId && (path === source.path || path.startsWith(`${source.path}/`))) {
        delete modes[key];
        modes[fileKey({ folderId: target.folderId, path: target.path + path.slice(source.path.length) })] = mode;
      }
    }
    for (const index of [0, 1] as const) {
      const active = view.groups[index].find(item => fileKey(item) === view.active[index]);
      view.groups[index] = view.groups[index].map(item => item.folderId === source.folderId && (item.path === source.path || item.path.startsWith(`${source.path}/`)) ? { folderId: target.folderId, path: target.path + item.path.slice(source.path.length) } : item);
      if (active) {
        const updated = active.folderId === source.folderId && (active.path === source.path || active.path.startsWith(`${source.path}/`)) ? { folderId: target.folderId, path: target.path + active.path.slice(source.path.length) } : active;
        view.active[index] = fileKey(updated);
      }
    }
  })),
  remove: (projectId, source) => set(state => mutate(state, projectId, view => {
    for (const index of [0, 1] as const) {
      view.groups[index] = view.groups[index].filter(item => item.folderId !== source.folderId || (item.path !== source.path && !item.path.startsWith(`${source.path}/`)));
      const last = view.groups[index].at(-1);
      if (!view.groups[index].some(item => fileKey(item) === view.active[index])) view.active[index] = last ? fileKey(last) : null;
    }
    if (view.groups[1].length === 0) { view.split = false; view.focused = 0; }
  })),
}), { name: "persistty.workspace-view.v1", storage: createJSONStorage(() => localStorage), partialize: state => ({ projects: state.projects }) }));
