import { expect, it } from "vitest";
import fixture from "../../../../tests/fixtures/search-git.json";
import { decodeSearch, decodePreview, decodeRepositories, decodeStatus, decodeLog, decodeDetail, decodeBaseline, decodeGitComparison } from "./search-git-decoder";
import { ProtocolError } from "./decoder";
it("解码共享 Go/TypeScript 搜索、部分替换和只读 Git 协议", () => {
  expect(decodeSearch(fixture.search)).toEqual(fixture.search);
  expect(decodePreview(fixture.preview)).toEqual(fixture.preview);
  expect(decodeRepositories(fixture.repositories)).toEqual(fixture.repositories);
  expect(decodeStatus(fixture.status)).toEqual(fixture.status);
  expect(decodeLog(fixture.log).items[0]?.parents).toEqual([]);
  expect(decodeDetail(fixture.detail)).toEqual(fixture.detail);
  expect(decodeBaseline(fixture.baseline)).toEqual(fixture.baseline);
  expect(decodeGitComparison(fixture.comparison)).toEqual(fixture.comparison);
});
it("拒绝错误行数、文件映射、超长消息和不安全GitHub链接", () => {
  expect(() => decodeDetail({ ...fixture.detail, github_url: "javascript:alert(1)" })).toThrow(ProtocolError);
  expect(() => decodeDetail({ ...fixture.detail, github_url: "https://github.com.evil.invalid/owner/repo/commit/" + fixture.detail.commit.id })).toThrow(ProtocolError);
  expect(() => decodeDetail({ ...fixture.detail, message: "x".repeat((64 << 10) + 1) })).toThrow(ProtocolError);
  expect(() => decodeDetail({ ...fixture.detail, stats: [{ ...fixture.detail.stats[0], additions: -1 }] })).toThrow(ProtocolError);
  expect(() => decodeDetail({ ...fixture.detail, stats: [{ ...fixture.detail.stats[0], additions: null }] })).toThrow(ProtocolError);
  expect(() => decodeDetail({ ...fixture.detail, files: ["wrong.txt"] })).toThrow(ProtocolError);
});
it("拒绝新增字段、非整数坐标、空版本和伪造状态", () => {
  expect(() => decodeSearch({ ...fixture.search, force: true })).toThrow(ProtocolError);
  expect(() => decodeSearch({ ...fixture.search, project_version: 0 })).toThrow(ProtocolError);
  expect(() => decodeSearch({ ...fixture.search, files: [{ ...fixture.search.files[0], matches: [{ ...fixture.search.files[0]!.matches[0], column: 1.5 }] }] })).toThrow(ProtocolError);
  expect(() => decodePreview({ ...fixture.preview, results: [{ id: "x", state: "applied", reason: "", version: {} }] })).toThrow(ProtocolError);
  expect(() => decodeBaseline({ ...fixture.baseline, state: "clean" })).toThrow(ProtocolError);
});
