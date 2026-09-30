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
export type DirectoryListing = { path: string; parent: string; items: string[] };
export type FileEntry = { name: string; kind: "file" | "directory" | "unsupported"; size: number; mtime: string; identity: string };
export type FileListing = { items: FileEntry[]; next_cursor: string; project_version: number };
export type FileContent = { content: string; kind: "text"; version: { mtime: string; size: number; etag: string; identity: string } };
export type FileVersion = FileContent["version"];
export type FileMetadata = { kind: "file"; version: FileVersion };
export type FileOperationResult = { state: "applied" | "partial"; source_removed: boolean; target_created: boolean; failure_code: string };
export type DeletePreview = { path: string; kind: "file" | "directory"; count: number; bytes: number; token: string };
export type ImportResult = { state: "uploaded" | "skipped"; version: FileVersion };
export type UploadState = { id: string; status: "pending" | "completed" | "skipped"; size: number; chunk_bytes: number; received: number[]; expires_at: string; result: ImportResult | null };
export type ArchiveRecord = { id: string; project_id: string; folder_id: string; project_version: number; relative_path: string; created_at: string; expires_at: string; status: "pending" | "ready" | "failed" | "cancelled"; size: number; error_code: string };
export type Terminal = { id: string; display_name: string; project_id: string | null; working_directory: string; state: "running" | "terminated" | "unavailable" };
export type TerminalHistory = { content_base64: string; history_size: number; returned_lines: number; alternate_on: boolean; cols: number; rows: number; truncated: boolean };
export type BatchTarget = { terminal_id: string; viewer_id: string; generation: number };
export type BatchMember = { terminal_id: string; display_name: string };
export type BatchResult = { terminal_id: string; state: Terminal["state"] };
export type TerminationBatch = { request_id: string; deadline: string; members: BatchMember[]; state: "pending" | "executing" | "cancelled" | "completed"; results: BatchResult[] };

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
export function decodeDirectory(value: unknown): DirectoryListing {
  const object = exact(value, ["path", "parent", "items"]);
  const path = text(object.path);
  const parent = text(object.parent);
  if (!path.startsWith("/") || !parent.startsWith("/")) fail();
  const items = array(object.items, item => {
    const name = text(item, 255);
    if (name === "." || name === ".." || name.includes("/")) fail();
    return name;
  }, 10000);
  return { path, parent, items };
}
export function decodeFileListing(value: unknown): FileListing {
  const object = exact(value, ["items", "next_cursor", "project_version"]);
  const items = array(object.items, (value): FileEntry => {
    const entry = exact(value, ["name", "kind", "size", "mtime", "identity"]);
    const name = text(entry.name, 255);
    if (name === "." || name === ".." || name.includes("/")) fail();
    const kind = entry.kind;
    if (kind !== "file" && kind !== "directory" && kind !== "unsupported") fail();
    if (typeof entry.size !== "number" || !Number.isSafeInteger(entry.size) || entry.size < 0) fail();
    return { name, kind, size: entry.size, mtime: text(entry.mtime, 64), identity: text(entry.identity, 128) };
  }, 200);
  if (typeof object.project_version !== "number" || !Number.isSafeInteger(object.project_version) || object.project_version < 1) fail();
  if (typeof object.next_cursor !== "string" || object.next_cursor.length > 80) fail();
  return { items, next_cursor: object.next_cursor, project_version: object.project_version };
}
export function decodeFileContent(value: unknown): FileContent {
  const object = exact(value, ["content", "version", "kind"]);
  if (object.kind !== "text" || typeof object.content !== "string") fail();
  return { content: object.content, kind: "text", version: decodeVersion(object.version) };
}
function integer(value: unknown, min = 0): number {
  return typeof value === "number" && Number.isSafeInteger(value) && value >= min ? value : fail();
}
function optionalText(value: unknown, max = 4096): string {
  return typeof value === "string" && value.length <= max && !value.includes("\0") ? value : fail();
}
function decodeVersion(value: unknown): FileVersion {
  const item = exact(value, ["mtime", "size", "etag", "identity"]);
  return { mtime: text(item.mtime, 64), size: integer(item.size), etag: text(item.etag, 128), identity: text(item.identity, 128) };
}
export function decodeFileMetadata(value: unknown): FileMetadata {
  const item = exact(value, ["kind", "version"]);
  if (item.kind !== "file") fail();
  return { kind: "file", version: decodeVersion(item.version) };
}
export function decodeOperationResult(value: unknown): FileOperationResult {
  const item = exact(value, ["state", "source_removed", "target_created", "failure_code"]);
  if (item.state !== "applied" && item.state !== "partial") fail();
  if (typeof item.source_removed !== "boolean" || typeof item.target_created !== "boolean") fail();
  return { state: item.state, source_removed: item.source_removed, target_created: item.target_created, failure_code: optionalText(item.failure_code, 128) };
}
export function decodeDeletePreview(value: unknown): DeletePreview {
  const item = exact(value, ["path", "kind", "count", "bytes", "token"]);
  if (item.kind !== "file" && item.kind !== "directory") fail();
  return { path: text(item.path), kind: item.kind, count: integer(item.count, 1), bytes: integer(item.bytes), token: text(item.token, 4096) };
}
export function decodeImportResult(value: unknown): ImportResult {
  const item = exact(value, ["state", "version"]);
  if (item.state !== "uploaded" && item.state !== "skipped") fail();
  return { state: item.state, version: decodeVersion(item.version) };
}
export function decodeUploadState(value: unknown): UploadState {
  const item = exact(value, ["id", "status", "size", "chunk_bytes", "received", "expires_at", "result"]);
  if (item.status !== "pending" && item.status !== "completed" && item.status !== "skipped") fail();
  const received = array(item.received, value => integer(value), 100000);
  return { id: id(item.id), status: item.status, size: integer(item.size), chunk_bytes: integer(item.chunk_bytes, 1), received, expires_at: text(item.expires_at, 64), result: item.result === null ? null : decodeImportResult(item.result) };
}
export function decodeArchive(value: unknown): ArchiveRecord {
  const item = exact(value, ["id", "project_id", "folder_id", "project_version", "relative_path", "created_at", "expires_at", "status", "size", "error_code"]);
  if (item.status !== "pending" && item.status !== "ready" && item.status !== "failed" && item.status !== "cancelled") fail();
  return { id: id(item.id), project_id: id(item.project_id), folder_id: id(item.folder_id), project_version: integer(item.project_version, 1), relative_path: optionalText(item.relative_path), created_at: text(item.created_at, 64), expires_at: text(item.expires_at, 64), status: item.status, size: integer(item.size), error_code: optionalText(item.error_code, 128) };
}
export function decodeTerminal(value: unknown): Terminal {
  const object = exact(value, ["id", "display_name", "project_id", "working_directory", "state"]);
  if (object.state !== "running" && object.state !== "terminated" && object.state !== "unavailable") fail();
  const working_directory = text(object.working_directory);
  if (!working_directory.startsWith("/")) fail();
  return { id: id(object.id), display_name: text(object.display_name, 256), project_id: object.project_id === null ? null : id(object.project_id), working_directory, state: object.state };
}
export function decodeBatchMembers(value: unknown): BatchMember[] {
  const members = array(value, value => {
    const item = exact(value, ["terminal_id", "display_name"]);
    return { terminal_id: id(item.terminal_id), display_name: text(item.display_name, 200) };
  }, 200);
  if (members.length === 0 || new Set(members.map(item => item.terminal_id)).size !== members.length) fail();
  return members;
}
export function decodeBatchResults(value: unknown): BatchResult[] {
  const results = array<BatchResult>(value, value => {
    const item = exact(value, ["terminal_id", "state"]);
    if (item.state !== "running" && item.state !== "terminated" && item.state !== "unavailable") fail();
    return { terminal_id: id(item.terminal_id), state: item.state };
  }, 200);
  if (new Set(results.map(item => item.terminal_id)).size !== results.length) fail();
  return results;
}
export function decodeTerminationBatch(value: unknown): TerminationBatch {
  const item = exact(value, ["request_id", "deadline", "members", "state", "results"]);
  if (item.state !== "pending" && item.state !== "executing" && item.state !== "cancelled" && item.state !== "completed") fail();
  const deadline = text(item.deadline, 64);
  if (!/^\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d(?:\.\d+)?Z$/.test(deadline) || !Number.isFinite(Date.parse(deadline))) fail();
  const members = decodeBatchMembers(item.members);
  const results = decodeBatchResults(item.results);
  if (item.state === "completed" ? results.length !== members.length || results.some(result => !members.some(member => member.terminal_id === result.terminal_id)) : results.length !== 0) fail();
  return { request_id: id(item.request_id), deadline, members, state: item.state, results };
}
export function decodeTerminalHistory(value: unknown): TerminalHistory {
  const object = exact(value, ["content_base64", "history_size", "returned_lines", "alternate_on", "cols", "rows", "truncated"]);
  if (typeof object.content_base64 !== "string" || object.content_base64.length > 12_000_000 ||
    !/^(?:[A-Za-z0-9+/]{4})*(?:[A-Za-z0-9+/]{2}==|[A-Za-z0-9+/]{3}=)?$/.test(object.content_base64) ||
    typeof object.alternate_on !== "boolean" || typeof object.truncated !== "boolean") fail();
  const cols = integer(object.cols, 1);
  const rows = integer(object.rows, 1);
  if (cols > 1000 || rows > 1000) fail();
  return { content_base64: object.content_base64, history_size: integer(object.history_size),
    returned_lines: integer(object.returned_lines), alternate_on: object.alternate_on, cols, rows, truncated: object.truncated };
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
  const hasDetails = typeof object.error === "object" && object.error !== null && Object.hasOwn(object.error, "details");
  const error = exact(object.error, hasDetails ? ["code", "message", "details"] : ["code", "message"]);
  const terminalId = hasDetails ? id(exact(error.details, ["terminal_id"]).terminal_id) : null;
  return { code: text(error.code, 128), message: text(error.message), requestId: text(object.request_id, 128), terminalId };
}
