import { beforeEach, expect, it } from "vitest";
import { fileKey, useWorkspaceView } from "./workspace-view";

beforeEach(() => { useWorkspaceView.setState({ projects: {} }); localStorage.clear(); });

it("关闭标签只改变浏览器视图，拆分与移动维持单一活动项", () => {
  const first = { folderId: "root-a", path: "src/one.ts" };
  const second = { folderId: "root-b", path: "src/two.ts" };
  const state = () => useWorkspaceView.getState();
  state().open("project", first);
  state().open("project", second);
  state().split("project", second);
  expect(state().projects.project?.split).toBe(true);
  expect(state().projects.project?.groups[1]).toEqual([second]);
  state().move("project", second, 0);
  expect(state().projects.project?.split).toBe(false);
  expect(state().projects.project?.active[0]).toBe(fileKey(second));
  state().close("project", second, 0);
  expect(state().projects.project?.groups[0]).toEqual([first]);
  expect(state().projects.project?.active[0]).toBe(fileKey(first));
});

it("重命名目录会移动子文件标签，删除目录会关闭其视图", () => {
  const state = () => useWorkspaceView.getState();
  state().open("project", { folderId: "root", path: "old/sub/file.go" });
  state().relocate("project", { folderId: "root", path: "old" }, { folderId: "root", path: "new" });
  expect(state().projects.project?.groups[0][0]?.path).toBe("new/sub/file.go");
  expect(state().projects.project?.active[0]).toBe(fileKey({ folderId: "root", path: "new/sub/file.go" }));
  state().remove("project", { folderId: "root", path: "new" });
  expect(state().projects.project?.groups[0]).toEqual([]);
  expect(state().projects.project?.active[0]).toBeNull();
});
