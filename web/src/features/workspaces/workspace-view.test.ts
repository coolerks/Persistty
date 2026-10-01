import { beforeEach, expect, it } from "vitest";
import { fileKey, restoreViews, useWorkspaceView } from "./workspace-view";

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

it("四个左右分组可排序，手机操作不覆盖桌面标签", () => {
  const state = () => useWorkspaceView.getState(); const one = { folderId: "root", path: "one.go" }; const two = { folderId: "root", path: "two.go" };
  state().open("p", one); state().open("p", two); state().split("p", one); state().split("p", two); state().split("p", one); state().split("p", two);
  expect(state().projects.p?.groups.map(files => files.length)).toEqual([2, 1, 1, 1]);
  state().move("p", two, 0, 0, one); expect(state().projects.p?.groups[0]).toEqual([two, one]);
  state().openMobile("p", two); state().closeMobile("p", two); expect(state().projects.p?.groups[0]).toEqual([two, one]);
  state().unsplit("p"); expect(state().projects.p?.groups).toEqual([[two, one], [], [], []]);
});
it("终端位置仅保存 ID/组，移动/排序不持久化 runtime 或正文", () => {
  const state = () => useWorkspaceView.getState(); state().placeTerminal("p", "a", { region: "top", group: 2 }); state().placeTerminal("p", "b", { region: "bottom", group: 3 }); state().orderTerminal("p", "b", "a");
  expect(state().projects.p?.upperActive[2]).toBe("a"); expect(state().projects.p?.lowerCount).toBe(4); expect(state().projects.p?.terminalOrder).toEqual(["b", "a"]);
  state().unsplit("p"); expect(state().projects.p?.terminals.a).toEqual({ region: "top", group: 0 });
});

it("旧双组迁移、损坏记录、数量和路径边界受验证", () => {
  const old = { groups: [[{ folderId: "root", path: "a.ts" }], []], active: ["root\u0000a.ts", null], focused: 0, split: false };
  expect(restoreViews({ projects: { p: old } }).p?.groups).toHaveLength(4);
  expect(() => restoreViews({ projects: { p: { ...old, groups: [[{ folderId: "root", path: "../a" }], []] } } })).toThrow();
  expect(() => restoreViews({ projects: { p: { ...old, groups: [Array.from({ length: 101 }, (_, i) => ({ folderId: "root", path: `${i}.ts` })), []] } } })).toThrow();
  expect(() => restoreViews(null)).toThrow();
});
