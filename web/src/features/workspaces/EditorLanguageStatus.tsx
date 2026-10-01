import type { Project } from "@/lib/api/decoder";
import { useEditorScope } from "./editor-context";
import { languageForFile } from "./file-language";
import { LanguageSelect } from "./LanguageSelect";
import { fileKey, useWorkspaceView, type OpenFile } from "./workspace-view";

export function EditorLanguageStatus({ project, file }: { project: Project; file: OpenFile | undefined }) {
  const scope = useEditorScope();
  const mode = useWorkspaceView(state => file ? state.languageModes[project.id]?.[fileKey(file)] : undefined);
  const buffer = file ? scope.buffers.get(fileKey(file)) : undefined;
  if (!file || buffer?.state.status !== "ready") return null;
  return <LanguageSelect mode={mode} detected={languageForFile(file.path, buffer.state.content)} onChange={value => useWorkspaceView.getState().setLanguage(project.id, file, value)} />;
}
