import { useEffect, useMemo, useRef } from "react";
import { lineChanges } from "@/features/git/line-changes";
import { editorText } from "./editor-text";
import type { EditorLocation } from "./editor-navigation";
import { useEffectiveTheme } from "@/features/settings/use-effective-theme";
import { Editor as MonacoEditor, loader } from "@monaco-editor/react";
import * as monaco from "monaco-editor";
import type { TextChange } from "./editor-text";
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

export default function DesktopEditor({ projectId, folderId, path, content, language, modelURI, onChange, isModelOpen, baseline, location }: { projectId: string; folderId: string; path: string; content: string; language: string; modelURI?: string | undefined; onChange?(value: string, changes: readonly TextChange[]): void; isModelOpen?(): boolean; baseline?: string | undefined; location?: EditorLocation | undefined }) {
  const theme = useEffectiveTheme() === "dark" ? "vs-dark" : "vs";
  const editorRef = useRef<monaco.editor.IStandaloneCodeEditor | null>(null);
  const decorations = useRef<monaco.editor.IEditorDecorationsCollection | null>(null);
  const selected = useRef("");
  const marks = useMemo(() => baseline === undefined ? [] : lineChanges(editorText(baseline), content), [baseline, content]);
  const update = () => {
    const editor = editorRef.current; if (!editor) return;
    decorations.current ??= editor.createDecorationsCollection();
    decorations.current.set(marks.map(mark => ({ range: new monaco.Range(mark.line, 1, mark.line, 1), options: { isWholeLine: true, linesDecorationsClassName: `git-line-${mark.kind}`, hoverMessage: { value: "HEAD 与当前编辑器输入的差异（有界行比较）" } } })));
    if (location && selected.current !== location.id) { selected.current = location.id; editor.setSelection(new monaco.Range(location.line, location.column, location.line, location.endColumn)); editor.revealLineInCenter(location.line); editor.focus(); }
  };
  const updateRef = useRef(update); updateRef.current = update;
  useEffect(() => { updateRef.current(); }, [marks, location]);
  useEffect(() => () => { decorations.current?.clear(); editorRef.current = null; }, []);
  return <MonacoEditor path={modelURI ?? editorURI(projectId, { folderId, path })} value={content} language={language} theme={theme} keepCurrentModel saveViewState onChange={(value, event) => onChange?.(value ?? "", event.changes)} onMount={editor => {
    editorRef.current = editor; decorations.current = null; updateRef.current();
    const model = editor.getModel();
    if (model) retainEditorModel(projectId, { folderId, path }, () => { if (!model.isDisposed()) model.dispose(); }, isModelOpen, modelURI);
    void document.fonts?.load('13px "Persistty Nerd Mono"').then(() => { if (!model?.isDisposed()) { monaco.editor.remeasureFonts(); editor.layout(); } }).catch(() => { /* Keep the measured fallback font. */ });
  }} options={{ readOnly: !onChange, fontFamily: "Persistty Nerd Mono, ui-monospace, SFMono-Regular, Menlo, Consolas, monospace", minimap: { enabled: true }, automaticLayout: true, wordWrap: "on", fontSize: 13, scrollBeyondLastLine: false, glyphMargin: true }} />;
}
