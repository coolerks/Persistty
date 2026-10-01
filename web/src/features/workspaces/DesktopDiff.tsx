import { useEffect, useRef } from "react";
import * as monaco from "monaco-editor";
import "./DesktopEditor";
import { useEffectiveTheme } from "@/features/settings/use-effective-theme";
import { editorText } from "./editor-text";
export default function DesktopDiff({ original, modified, language }: { original: string; modified: string; language: string }) {
  const host = useRef<HTMLDivElement>(null);
  const view = useRef<{ editor: monaco.editor.IStandaloneDiffEditor; original: monaco.editor.ITextModel; modified: monaco.editor.ITextModel } | null>(null);
  const initial = useRef({ original, modified, language });
  const theme = useEffectiveTheme() === "dark" ? "vs-dark" : "vs";
  useEffect(() => {
    if (!host.current) return;
    const editor = monaco.editor.createDiffEditor(host.current, { readOnly: true, originalEditable: false, automaticLayout: true, fontFamily: "Persistty Nerd Mono, monospace", fontSize: 13, minimap: { enabled: false } });
    const original = monaco.editor.createModel(editorText(initial.current.original), initial.current.language);
    const modified = monaco.editor.createModel(editorText(initial.current.modified), initial.current.language);
    view.current = { editor, original, modified }; const viewModel = editor.createViewModel({ original, modified }); editor.setModel(viewModel);
    let active = true;
    void document.fonts.load('13px "Persistty Nerd Mono"').then(() => { if (active) { monaco.editor.remeasureFonts(); editor.layout(); } });
    return () => { active = false; view.current = null; editor.setModel(null); viewModel.dispose(); editor.dispose(); original.dispose(); modified.dispose(); };
  }, []);
  useEffect(() => {
    const current = view.current; if (!current) return;
    if (current.original.getValue() !== editorText(original)) current.original.setValue(editorText(original));
    if (current.modified.getValue() !== editorText(modified)) current.modified.setValue(editorText(modified));
    monaco.editor.setModelLanguage(current.original, language); monaco.editor.setModelLanguage(current.modified, language);
  }, [original, modified, language]);
  useEffect(() => { monaco.editor.setTheme(theme); }, [theme]);
  return <div ref={host} className="h-full min-h-0" />;
}
