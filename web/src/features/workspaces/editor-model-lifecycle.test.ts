import { beforeEach, expect, it, vi } from "vitest";
import { useWorkspaceView } from "./workspace-view";
import { editorURI, retainEditorModel, watchEditorModels } from "./editor-model-lifecycle";

beforeEach(() => { useWorkspaceView.setState({ projects: {}, languageModes: {} }); });
it("同文件多个视图共享 model，关闭最后视图与离开项目释放", async () => {
  const state = () => useWorkspaceView.getState(); const file = { folderId: "root", path: "file.go" };
  state().open("project", file); const stop = watchEditorModels("project"); const dispose = vi.fn();
  retainEditorModel("project", file, dispose); state().split("project", file); state().setLanguage("project", file, "go");
  state().close("project", file, 0); await new Promise(resolve => setTimeout(resolve, 10)); expect(dispose).not.toHaveBeenCalled();
  state().close("project", file, 1); await vi.waitFor(() => expect(dispose).toHaveBeenCalledOnce());
  state().open("project", file); const another = vi.fn(); retainEditorModel("project", file, another);
  stop(); await vi.waitFor(() => expect(another).toHaveBeenCalledOnce());
});
it("StrictMode scope 清理后立即恢复不会释放仍打开的 model", async () => {
  const file = { folderId: "root", path: "a b/中文.go" }; useWorkspaceView.getState().open("project", file);
  const stop = watchEditorModels("project"); const dispose = vi.fn(); retainEditorModel("project", file, dispose);
  stop(); const remount = watchEditorModels("project"); await new Promise(resolve => setTimeout(resolve, 10)); expect(dispose).not.toHaveBeenCalled();
  expect(editorURI("project", file)).toBe("persistty:///project/root/a%20b/%E4%B8%AD%E6%96%87.go");
  remount(); await vi.waitFor(() => expect(dispose).toHaveBeenCalledOnce());
});
