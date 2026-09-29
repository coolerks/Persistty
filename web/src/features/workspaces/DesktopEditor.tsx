import { useEffect, useState } from "react";
import { Editor as MonacoEditor, loader } from "@monaco-editor/react";
import * as monaco from "monaco-editor";
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
monaco.typescript.typescriptDefaults.setDiagnosticsOptions({ noSemanticValidation: true, noSyntaxValidation: true });
monaco.typescript.javascriptDefaults.setDiagnosticsOptions({ noSemanticValidation: true, noSyntaxValidation: true });
loader.config({ monaco });

function useMonacoTheme(): "vs" | "vs-dark" {
  const [theme, setTheme] = useState<"vs" | "vs-dark">(() => document.documentElement.classList.contains("dark") ? "vs-dark" : "vs");
  useEffect(() => {
    const observer = new MutationObserver(() => setTheme(document.documentElement.classList.contains("dark") ? "vs-dark" : "vs"));
    observer.observe(document.documentElement, { attributes: true, attributeFilter: ["class"] });
    return () => observer.disconnect();
  }, []);
  return theme;
}

function languageFor(path: string): string {
  const name = path.split("/").at(-1)?.toLowerCase() ?? "";
  if (name === "dockerfile") return "dockerfile";
  const suffix = name.split(".").at(-1);
  return ({ ts: "typescript", tsx: "typescript", js: "javascript", jsx: "javascript", json: "json", css: "css", html: "html", go: "go", py: "python", sh: "shell", md: "markdown", yaml: "yaml", yml: "yaml", sql: "sql", xml: "xml" } as Record<string, string>)[suffix ?? ""] ?? "plaintext";
}

export default function DesktopEditor({ projectId, folderId, path, content }: { projectId: string; folderId: string; path: string; content: string }) {
  const theme = useMonacoTheme();
  return <MonacoEditor path={`persistty:///${encodeURIComponent(projectId)}/${encodeURIComponent(folderId)}/${path.split("/").map(encodeURIComponent).join("/")}`} value={content} language={languageFor(path)} theme={theme} saveViewState options={{ readOnly: true, minimap: { enabled: true }, automaticLayout: true, wordWrap: "on", fontSize: 13, scrollBeyondLastLine: false, glyphMargin: true }} />;
}
