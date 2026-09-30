import { useEffect, useState } from "react";
import { createRoot } from "react-dom/client";
import * as monaco from "monaco-editor";
import DesktopEditor from "../../src/features/workspaces/DesktopEditor";
import { LanguageSelect } from "../../src/features/workspaces/LanguageSelect";
import { FileTypeIcon } from "../../src/features/workspaces/FileTypeIcon";
import { fileLanguages, languageForFile } from "../../src/features/workspaces/file-language";
import { fileKey, useWorkspaceView } from "../../src/features/workspaces/workspace-view";
import { editorURI, watchEditorModels } from "../../src/features/workspaces/editor-model-lifecycle";
import { languageSamples } from "./language-samples";
import "../../src/app/styles.css";

declare global { interface Window { editorAssets: { monaco: typeof monaco; samples: typeof languageSamples; ids: string[]; uri: typeof editorURI } } }
window.editorAssets = { monaco, samples: languageSamples, ids: fileLanguages.map(item => item.id), uri: editorURI };
const files = [{ folderId: "root", path: "query.sql" }, { folderId: "root", path: "view.ftl" }];
for (const file of files) useWorkspaceView.getState().open("fixture", file);
function Fixture() {
  useEffect(() => watchEditorModels("fixture"), []);
  const [index, setIndex] = useState(0); const file = files[index]!;
  const mode = useWorkspaceView(state => state.languageModes.fixture?.[fileKey(file)]);
  const setLanguage = useWorkspaceView(state => state.setLanguage);
  const content = file.path.endsWith("sql") ? Array.from({ length: 100 }, () => languageSamples.sql).join("\n") : languageSamples.freemarker2!;
  const detected = languageForFile(file.path, content);
  return <main style={{ height: "100vh", display: "flex", flexDirection: "column" }}>
    <div style={{ display: "flex", gap: 12 }}>
      {files.map((item, n) => <button key={item.path} onClick={() => setIndex(n)}>{item.path}</button>)}
      <button onClick={() => document.documentElement.classList.toggle("dark")}>切换主题</button>
      <LanguageSelect mode={mode} detected={detected} onChange={value => setLanguage("fixture", file, value)} />
      <FileTypeIcon path="file.yaml" /><FileTypeIcon path="file.yml" /><FileTypeIcon path="file.d.ts" /><FileTypeIcon path="file.ts" /><FileTypeIcon path="config.toml" />
    </div>
    <div style={{ flex: 1, minHeight: 0 }}><DesktopEditor key={file.path} projectId="fixture" folderId={file.folderId} path={file.path} content={content} language={mode ?? detected} /></div>
  </main>;
}
createRoot(document.getElementById("root")!).render(<Fixture />);
