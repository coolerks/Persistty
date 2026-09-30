import { useEffectiveTheme } from "@/features/settings/use-effective-theme";
import { Editor as MonacoEditor, loader } from "@monaco-editor/react";
import * as monaco from "monaco-editor";
import { editorURI, retainEditorModel } from "./editor-model-lifecycle";
import editorWorker from "monaco-editor/editor/editor.worker.js?worker";
import jsonWorker from "monaco-editor/language/json/json.worker.js?worker";
import cssWorker from "monaco-editor/language/css/css.worker.js?worker";
import htmlWorker from "monaco-editor/language/html/html.worker.js?worker";
import tsWorker from "monaco-editor/language/typescript/ts.worker.js?worker";

self.MonacoEnvironment = {
  getWorker(_moduleId, label) {
    if (label === "json") return new jsonWorker();
    if (["css", "scss", "less"].includes(label)) return new cssWorker();
    if (["html", "handlebars", "razor"].includes(label)) return new htmlWorker();
    if (label === "typescript" || label === "javascript") return new tsWorker();
    return new editorWorker();
  },
};
monaco.typescript.typescriptDefaults.setDiagnosticsOptions({ noSemanticValidation: true, noSyntaxValidation: true, noSuggestionDiagnostics: true });
monaco.typescript.javascriptDefaults.setDiagnosticsOptions({ noSemanticValidation: true, noSyntaxValidation: true, noSuggestionDiagnostics: true });
monaco.json.jsonDefaults.setDiagnosticsOptions({ validate: false, enableSchemaRequest: false });
monaco.css.cssDefaults.setDiagnosticsOptions({ validate: false });
monaco.css.scssDefaults.setDiagnosticsOptions({ validate: false });
monaco.css.lessDefaults.setDiagnosticsOptions({ validate: false });
monaco.html.htmlDefaults.setModeConfiguration({ ...monaco.html.htmlDefaults.modeConfiguration, diagnostics: false });
loader.config({ monaco });

export default function DesktopEditor({ projectId, folderId, path, content, language }: { projectId: string; folderId: string; path: string; content: string; language: string }) {
  const theme = useEffectiveTheme() === "dark" ? "vs-dark" : "vs";
  return <MonacoEditor path={editorURI(projectId, { folderId, path })} value={content} language={language} theme={theme} keepCurrentModel saveViewState onMount={editor => {
    const model = editor.getModel();
    if (model) retainEditorModel(projectId, { folderId, path }, () => { if (!model.isDisposed()) model.dispose(); });
  }} options={{ readOnly: true, minimap: { enabled: true }, automaticLayout: true, wordWrap: "on", fontSize: 13, scrollBeyondLastLine: false, glyphMargin: true }} />;
}
