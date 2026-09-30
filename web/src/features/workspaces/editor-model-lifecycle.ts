import { fileKey, useWorkspaceView, type OpenFile } from "./workspace-view";

type RetainedModel = { projectId: string; file: OpenFile; dispose(): void };
const models = new Map<string, RetainedModel>();
const scopes = new Map<string, number>();

export function editorURI(projectId: string, file: OpenFile): string {
  return `persistty:///${encodeURIComponent(projectId)}/${encodeURIComponent(file.folderId)}/${file.path.split("/").map(encodeURIComponent).join("/")}`;
}

export function retainEditorModel(projectId: string, file: OpenFile, dispose: () => void): void {
  models.set(editorURI(projectId, file), { projectId, file, dispose });
}

function prune(projectId: string): void {
  const view = useWorkspaceView.getState().projects[projectId];
  const open = new Set(view?.groups.flat().map(fileKey));
  for (const [uri, model] of models) {
    if (model.projectId === projectId && (!scopes.has(projectId) || !open.has(fileKey(model.file)))) {
      model.dispose(); models.delete(uri);
    }
  }
}

// Defer disposal until React has detached editors; StrictMode's remount retains the scope.
export function watchEditorModels(projectId: string): () => void {
  scopes.set(projectId, (scopes.get(projectId) ?? 0) + 1);
  const unsubscribe = useWorkspaceView.subscribe(() => { setTimeout(() => prune(projectId), 0); });
  return () => {
    unsubscribe();
    const count = (scopes.get(projectId) ?? 1) - 1;
    if (count) scopes.set(projectId, count); else scopes.delete(projectId);
    setTimeout(() => prune(projectId), 0);
  };
}
