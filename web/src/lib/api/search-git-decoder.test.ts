import { expect, it } from "vitest";
import fixture from "../../../../tests/fixtures/search-git.json";
import { decodeSearch, decodePreview, decodeRepositories, decodeStatus, decodeLog, decodeBaseline, decodeGitComparison } from "./search-git-decoder";
import { ProtocolError } from "./decoder";
it("解码共享 Go/TypeScript 搜索、部分替换和只读 Git 协议", () => {
  expect(decodeSearch(fixture.search)).toEqual(fixture.search);
  expect(decodePreview(fixture.preview)).toEqual(fixture.preview);
  expect(decodeRepositories(fixture.repositories)).toEqual(fixture.repositories);
  expect(decodeStatus(fixture.status)).toEqual(fixture.status);
  expect(decodeLog(fixture.log).items[0]?.parents).toEqual([]);
  expect(decodeBaseline(fixture.baseline)).toEqual(fixture.baseline);
  expect(decodeGitComparison(fixture.comparison)).toEqual(fixture.comparison);
});
it("拒绝新增字段、非整数坐标、空版本和伪造状态", () => {
  expect(() => decodeSearch({ ...fixture.search, force: true })).toThrow(ProtocolError);
  expect(() => decodeSearch({ ...fixture.search, project_version: 0 })).toThrow(ProtocolError);
  expect(() => decodeSearch({ ...fixture.search, files: [{ ...fixture.search.files[0], matches: [{ ...fixture.search.files[0]!.matches[0], column: 1.5 }] }] })).toThrow(ProtocolError);
  expect(() => decodePreview({ ...fixture.preview, results: [{ id: "x", state: "applied", reason: "", version: {} }] })).toThrow(ProtocolError);
  expect(() => decodeBaseline({ ...fixture.baseline, state: "clean" })).toThrow(ProtocolError);
});
