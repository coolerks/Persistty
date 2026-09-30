import { beforeEach, expect, it } from "vitest";
import { fileKey, useWorkspaceView } from "./workspace-view";

beforeEach(() => { useWorkspaceView.setState({ projects: {}, languageModes: {} }); localStorage.clear(); });

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


it("语言覆盖按项目与文件隔离，同文件拆分共享，最后视图关闭时清理且不持久化", () => {
  const state = () => useWorkspaceView.getState(); const file = { folderId: "root", path: "query.sql" };
  state().open("a", file); state().open("b", file); state().setLanguage("a", file, "pgsql");
  state().split("a", file); state().close("a", file, 0);
  expect(state().languageModes.a?.[fileKey(file)]).toBe("pgsql");
  expect(state().languageModes.b?.[fileKey(file)]).toBeUndefined();
  expect(localStorage.getItem("persistty.workspace-view.v1")).not.toContain("pgsql");
  state().close("a", file, 1); expect(state().languageModes.a?.[fileKey(file)]).toBeUndefined();
  state().open("a", file); expect(state().languageModes.a?.[fileKey(file)]).toBeUndefined();
});

it("语言覆盖随目录重命名迁移，删除与自动模式清理，未知模式拒绝", () => {
  const state = () => useWorkspaceView.getState(); const file = { folderId: "root", path: "old/query.sql" };
  const target = { folderId: "other", path: "new/query.sql" };
  state().open("a", file); state().setLanguage("a", file, "mysql");
  state().relocate("a", { folderId: "root", path: "old" }, { folderId: "other", path: "new" });
  expect(state().languageModes.a?.[fileKey(target)]).toBe("mysql");
  expect(state().languageModes.a?.[fileKey(file)]).toBeUndefined();
  state().setLanguage("a", target, "invalid"); expect(state().languageModes.a?.[fileKey(target)]).toBe("mysql");
  state().setLanguage("a", target, undefined); expect(state().languageModes.a?.[fileKey(target)]).toBeUndefined();
  state().setLanguage("a", target, "redshift"); state().remove("a", { folderId: "other", path: "new" });
  expect(state().languageModes.a).toEqual({});
});
