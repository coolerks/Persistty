import { describe, expect, it } from "vitest";
import { commitGraph } from "./commit-graph";
import { gitFileTree } from "./file-tree";
describe("提交关系图", () => {
  it("合并分叉后汇合，不把两条分支绘制为线性历史", () => {
    const rows = commitGraph([{ id: "merge", parents: ["left", "right"] }, { id: "left", parents: ["root"] }, { id: "right", parents: ["root"] }, { id: "root", parents: [] }]);
    expect(rows[0]!.edges.map(edge => edge.to)).toEqual([0, 1]);
    expect(rows[1]!.edges).toContainEqual({ from: 1, to: 1, parent: "right", through: true });
    expect(rows[2]!.edges).toContainEqual({ from: 1, to: 0, parent: "root", through: false });
    expect(rows[3]!.edges).toEqual([]);
  });
  it("后续页面保留未出现的父节点连接，追加后沿同一关系汇合", () => {
    const first = [{ id: "merge", parents: ["a", "b"] }, { id: "a", parents: ["root"] }];
    const partial = commitGraph(first), complete = commitGraph([...first, { id: "b", parents: ["root"] }, { id: "root", parents: [] }]);
    expect(complete.slice(0, 2)).toEqual(partial);
    expect(complete[2]!.lane).toBe(1);
  });
});
describe("变更文件树", () => {
  it("同名文件按完整路径区分，目录优先，不丢改名来源与状态", () => {
    const files = [{ path: "b/same.ts", status: "M" }, { path: "a/same.ts", status: "R", oldPath: "old.ts" }, { path: "README.md" }];
    const tree = gitFileTree(files);
    expect(tree.map(node => node.name)).toEqual(["a", "b", "README.md"]);
    expect(tree[0]!.children[0]!.file).toEqual(files[1]);
    expect(tree[1]!.children[0]!.path).toBe("b/same.ts");
  });
});
