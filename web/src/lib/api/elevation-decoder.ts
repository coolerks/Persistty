import { decodeVersion, exact, id, optionalText, ProtocolError, type FileVersion } from "./decoder";

export type ElevationPrepared = { id: string; target_id: string; target_path: string; content_hash: string; expires_at: string; state: "prepared" };
export type ElevationResult = { id: string; state: "prepared" | "executing" | "applied" | "rejected" | "cancelled" | "expired" | "indeterminate"; code: string | null; version: FileVersion | null };
const nonce = (value: unknown): string => { const text = id(value); if (!/^[a-f0-9]{64}$/.test(text)) throw new ProtocolError(); return text; };
const targetID = (value: unknown): string => { const text = id(value); if (text.length > 64) throw new ProtocolError(); return text; };
export function decodeElevationPrepared(value: unknown): ElevationPrepared {
  const o = exact(value, ["id", "target_id", "target_path", "content_hash", "expires_at", "state"]);
  const target_path = optionalText(o.target_path), content_hash = optionalText(o.content_hash, 128), expires_at = optionalText(o.expires_at, 64);
  if ((!target_path.startsWith("/") || target_path.split("/").some(part => part === "." || part === "..") || /[\r\n]/.test(target_path)) || !/^sha256:[a-f0-9]{64}$/.test(content_hash) || !/^\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d(?:\.\d{1,9})?Z$/.test(expires_at) || !Number.isFinite(Date.parse(expires_at)) || new Date(expires_at).toISOString().slice(0, 19) !== expires_at.slice(0, 19) || o.state !== "prepared") throw new ProtocolError();
  return { id: nonce(o.id), target_id: targetID(o.target_id), target_path, content_hash, expires_at, state: "prepared" };
}
export function decodeElevationResult(value: unknown): ElevationResult {
  const o = exact(value, ["id", "state", "code", "version"]), state = o.state;
  if (state !== "prepared" && state !== "executing" && state !== "applied" && state !== "rejected" && state !== "cancelled" && state !== "expired" && state !== "indeterminate") throw new ProtocolError();
  const code = o.code === null ? null : optionalText(o.code, 64), version = o.version === null ? null : decodeVersion(o.version);
  if (state === "applied" ? code !== null || version === null : version !== null || (["prepared", "executing"].includes(state) ? code !== null : !code)) throw new ProtocolError();
  if (version && (!Number.isFinite(Date.parse(version.mtime)) || version.size > (8 << 20) || !/^sha256:[a-f0-9]{64}$/.test(version.etag))) throw new ProtocolError();
  return { id: nonce(o.id), state, code, version };
}
