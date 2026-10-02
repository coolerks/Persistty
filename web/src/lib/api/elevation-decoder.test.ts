import { expect, it } from "vitest";
import fixture from "../../../../tests/contracts/elevation.json";
import { decodeEnvelope, ProtocolError } from "./decoder";
import { decodeElevationPrepared, decodeElevationResult } from "./elevation-decoder";
it("W07 Go/TS 共用准备与全部结果 DTO", () => {
  expect(decodeEnvelope(fixture.prepared, decodeElevationPrepared).target_id).toBe("example");
  expect(fixture.results.map(row => decodeEnvelope(row, decodeElevationResult).state)).toEqual(["prepared", "executing", "applied", "rejected", "cancelled", "expired", "indeterminate"]);
});
it("拒绝请求错配、未知字段、状态版本矛盾及错误 hash", () => {
  for (const value of [{ ...fixture.prepared.data, id: "unknown" }, { ...fixture.prepared.data, content_hash: "sha256:short" }, { ...fixture.prepared.data, extra: true }, { ...fixture.prepared.data, target_path: "relative" }]) expect(() => decodeElevationPrepared(value)).toThrow(ProtocolError);
  const applied = fixture.results[2]!.data;
  for (const value of [{ ...applied, state: "unknown" }, { ...applied, code: "secret" }, { ...applied, version: null }, { ...applied, state: "rejected" }]) expect(() => decodeElevationResult(value)).toThrow(ProtocolError);
});
