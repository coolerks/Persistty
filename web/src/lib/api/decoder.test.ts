import { describe, expect, it } from "vitest";
import fixture from "../../../../tests/contracts/foundation.json";
import { decodeEnvelope, decodeError, decodeList, decodeProject, decodeSession, decodeTerminal, ProtocolError } from "./decoder";

describe("Go/TypeScript 共同 API fixture", () => {
  it("读取全部成功和错误形状", () => {
    expect(decodeEnvelope(fixture.session, decodeSession).authenticated).toBe(true);
    expect(decodeEnvelope(fixture.projects, decodeList(decodeProject))).toHaveLength(1);
    expect(decodeEnvelope(fixture.project, decodeProject).name).toBe("示例项目");
    expect(decodeEnvelope(fixture.empty_projects, decodeList(decodeProject))).toEqual([]);
    expect(decodeEnvelope(fixture.terminals, decodeList(decodeTerminal))[0]?.project_id).toBeNull();
    expect(decodeEnvelope(fixture.empty_terminals, decodeList(decodeTerminal))).toEqual([]);
    for (const error of [fixture.unauthenticated, fixture.not_found, fixture.invalid_request]) expect(decodeError(error).requestId).toBe("test-request");
  });
  it.each([null, {}, { ...fixture.project.data, version: 0 }, { ...fixture.project.data, version: Number.MAX_SAFE_INTEGER + 1 },
    { ...fixture.project.data, folders: [] }, { ...fixture.project.data, main_folder_id: "missing" },
    { ...fixture.project.data, folders: [...fixture.project.data.folders, ...fixture.project.data.folders] },
    { ...fixture.project.data, extra: true }, { ...fixture.project.data, name: null }, { ...fixture.project.data, id: "../bad" }])("拒绝非法项目 %#", value => {
    expect(() => decodeProject(value)).toThrow(ProtocolError);
  });
  it("拒绝未知状态、无UTC时间、越界列表和缺失信封", () => {
    expect(() => decodeTerminal({ ...fixture.terminals.data.items[0], state: "running" })).toThrow(ProtocolError);
    expect(() => decodeSession({ ...fixture.session.data, expires_at: "2026-10-04" })).toThrow(ProtocolError);
    expect(() => decodeSession({ ...fixture.session.data, expires_at: "2026-02-31T12:00:00Z" })).toThrow(ProtocolError);
    expect(() => decodeList(decodeProject)({ items: Array.from({ length: 201 }, () => fixture.project.data) })).toThrow(ProtocolError);
    expect(() => decodeList(decodeProject)({ items: [fixture.project.data, fixture.project.data] })).toThrow(ProtocolError);
    expect(() => decodeEnvelope({ data: fixture.project.data }, decodeProject)).toThrow(ProtocolError);
  });
});
