import { expect, it } from "vitest";
import { lineChanges } from "./line-changes";
it("HEAD 行标记区分添加、修改、删除及无变化", () => {
  expect(lineChanges("a\nb", "a\nb")).toEqual([]);
  expect(lineChanges("a\nb", "a\nx\nb")).toEqual([{ line: 2, kind: "added" }]);
  expect(lineChanges("a\nb", "a\nc")).toEqual([{ line: 2, kind: "modified" }]);
  expect(lineChanges("a\nb\nc", "a\nc")).toEqual([{ line: 2, kind: "deleted" }]);
});
it("大幅变化有界处理，所有标记落在现有行", () => {
  const marks = lineChanges(Array(1000).fill("before").join("\n"), Array(1000).fill("after").join("\n"));
  expect(marks).toHaveLength(1000); expect(marks.every(mark => mark.line >= 1 && mark.line <= 1000 && mark.kind === "modified")).toBe(true);
});
