import * as monaco from "monaco-editor";
declare global { interface Window { editorRecovery: { monaco: typeof monaco } } }
window.editorRecovery = { monaco };
import { createRoot } from "react-dom/client";
import { MemoryRouter } from "react-router";
import { AuthContext } from "../../src/features/auth/auth-context";
import { ProjectWorkbench } from "../../src/features/workspaces/ProjectWorkbench";
import "../../src/app/styles.css";

const project = { id: "interaction", name: "交互验收", version: 1, main_folder_id: "root", folders: [{ id: "root", path: "/fixture" }, { id: "other", path: "/other" }] };
createRoot(document.getElementById("root")!).render(<MemoryRouter><AuthContext.Provider value={{ state: { status: "authenticated", session: { authenticated: true, csrf_token: "fixture", expires_at: "2099-01-01T00:00:00Z" } }, login: async () => {}, logout: async () => {}, expire: () => {}, retry: () => {} }}>
  <div className="app-shell app-shell-workspace"><ProjectWorkbench project={project} onEdit={() => {}} /></div>
</AuthContext.Provider></MemoryRouter>);
