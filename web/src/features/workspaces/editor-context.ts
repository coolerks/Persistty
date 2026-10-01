import { createContext, useContext, useSyncExternalStore } from "react";
import type { EditorScope } from "./editor-session";
export const EditorContext = createContext<EditorScope | null>(null);
export function useEditorScope(): EditorScope {
  const scope = useContext(EditorContext);
  if (!scope) throw new Error("编辑器尚未初始化。");
  useSyncExternalStore(scope.subscribe, scope.snapshot, scope.snapshot);
  return scope;
}
