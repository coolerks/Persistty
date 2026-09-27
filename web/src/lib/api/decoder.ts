export class ProtocolError extends Error {
  constructor() { super("服务器响应格式无效，请重试。"); this.name = "ProtocolError"; }
}

function fail(): never { throw new ProtocolError(); }
function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}
function record(value: unknown): Record<string, unknown> { return isRecord(value) ? value : fail(); }
function exact(value: unknown, keys: string[]): Record<string, unknown> {
  const object = record(value);
  if (Object.keys(object).length !== keys.length || !keys.every(key => Object.hasOwn(object, key))) fail();
  return object;
}
function text(value: unknown, max = 4096): string {
  return typeof value === "string" && value.length > 0 && value.length <= max && !value.includes("\0") ? value : fail();
}
function id(value: unknown): string {
  const result = text(value, 128);
  return /^[A-Za-z0-9_-]+$/.test(result) ? result : fail();
}
export type Session = { authenticated: true; expires_at: string; csrf_token: string };
export type Folder = { id: string; path: string };
export type Project = { id: string; name: string; version: number; main_folder_id: string; folders: Folder[] };
export type Terminal = { id: string; display_name: string; project_id: string | null; working_directory: string; state: "unavailable" };

export function decodeEnvelope<T>(value: unknown, decode: (data: unknown) => T): T {
  const object = exact(value, ["data", "request_id"]);
  text(object.request_id, 128);
  return decode(object.data);
}
export function decodeSession(value: unknown): Session {
  const object = exact(value, ["authenticated", "expires_at", "csrf_token"]);
  if (object.authenticated !== true) fail();
  const expires_at = text(object.expires_at, 64);
  if (!/^\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d(?:\.\d{1,9})?Z$/.test(expires_at) || !Number.isFinite(Date.parse(expires_at)) || new Date(expires_at).toISOString().slice(0, 19) !== expires_at.slice(0, 19)) fail();
  return { authenticated: true, expires_at, csrf_token: text(object.csrf_token, 256) };
}
export function decodeProject(value: unknown): Project {
  const object = exact(value, ["id", "name", "version", "main_folder_id", "folders"]);
  const folders = array(object.folders, item => {
    const folder = exact(item, ["id", "path"]);
    const path = text(folder.path);
    if (!path.startsWith("/")) fail();
    return { id: id(folder.id), path };
  }, 200);
  if (!folders.length || new Set(folders.map(folder => folder.id)).size !== folders.length) fail();
  const main_folder_id = id(object.main_folder_id);
  if (!folders.some(folder => folder.id === main_folder_id)) fail();
  const version = object.version;
  if (typeof version !== "number" || !Number.isSafeInteger(version) || version < 1) fail();
  return { id: id(object.id), name: text(object.name, 256), version, main_folder_id, folders };
}
export function decodeTerminal(value: unknown): Terminal {
  const object = exact(value, ["id", "display_name", "project_id", "working_directory", "state"]);
  if (object.state !== "unavailable") fail();
  const working_directory = text(object.working_directory);
  if (!working_directory.startsWith("/")) fail();
  return { id: id(object.id), display_name: text(object.display_name, 256), project_id: object.project_id === null ? null : id(object.project_id), working_directory, state: "unavailable" };
}
function array<T>(value: unknown, decode: (item: unknown) => T, max: number): T[] {
  if (!Array.isArray(value) || value.length > max) fail();
  return value.map(decode);
}
export function decodeList<T extends { id: string }>(decode: (item: unknown) => T, max = 200) {
  return (value: unknown): T[] => {
    const object = exact(value, ["items"]);
    const items = array(object.items, decode, max);
    if (new Set(items.map(item => item.id)).size !== items.length) fail();
    return items;
  };
}
export function decodeError(value: unknown) {
  const object = exact(value, ["error", "request_id"]);
  const error = exact(object.error, ["code", "message"]);
  return { code: text(error.code, 128), message: text(error.message), requestId: text(object.request_id, 128) };
}
