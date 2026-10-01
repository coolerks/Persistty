import { decodeArchive, decodeDeletePreview, decodeDirectory, decodeEnvelope, decodeError, decodeFileContent, decodeFileListing, decodeFileMetadata, decodeImportResult, decodeList, decodeOperationResult, decodeProject, decodeSession, decodeTerminal, decodeTerminalHistory, decodeUploadState, ProtocolError } from "./decoder";
import { decodeTerminationBatch, decodeSaveResult, decodeInspection, type BatchTarget, type FileVersion } from "./decoder";

export type FileOperation = { kind: "create_file" | "create_directory" | "rename" | "copy" | "move" | "delete"; project_version: number; source_folder_id?: string; source_path?: string; target_folder_id?: string; target_path?: string; expected_version?: FileVersion; expected_identity?: string; delete_token?: string };
export type UploadInput = { project_id: string; folder_id: string; project_version: number; path: string; batch_id: string; size: number; sha256: string; expected_version?: FileVersion };

export class ApiError extends Error {
  constructor(public readonly status: number, public readonly code: string, message: string,
    public readonly requestId: string, public readonly retryAfter: number | null, public readonly terminalId: string | null = null) {
    super(message); this.name = "ApiError";
  }
}

export async function request<T>(path: string, decode: (value: unknown) => T, options: RequestInit = {}): Promise<T> {
  const response = await fetch(`/api/v1${path}`, { ...options, credentials: "same-origin", cache: "no-store", redirect: "error" });
  if (!response.ok) {
    const value: unknown = await response.json().catch(() => { throw new ProtocolError(); });
    const error = decodeError(value);
    const retry = response.headers.get("Retry-After");
    const seconds = retry === null ? NaN : /^\d+$/.test(retry) ? Number(retry) : Math.max(0, (Date.parse(retry) - Date.now()) / 1000);
    const retryAfter = Number.isFinite(seconds) && seconds >= 0 ? seconds : null;
    throw new ApiError(response.status, error.code, error.message, error.requestId, retryAfter, error.terminalId);
  }
  if (response.status === 204) return decode(undefined);
  const value: unknown = await response.json().catch(() => { throw new ProtocolError(); });
  return decodeEnvelope(value, decode);
}

export const api = {
  inspect: (projectId: string, folderId: string, version: number, path: string, signal: AbortSignal) => request(`/projects/${encodeURIComponent(projectId)}/folders/${encodeURIComponent(folderId)}/inspect?project_version=${version}&path=${encodeURIComponent(path)}`, decodeInspection, { signal }),
  session: (signal: AbortSignal) => request("/auth/session", decodeSession, { signal }),
  login: (password: string, signal: AbortSignal) => request("/auth/login", decodeSession,
    { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ password }), signal }),
  logout: (csrf: string, signal: AbortSignal) => request("/auth/logout", () => undefined,
    { method: "POST", headers: { "X-CSRF-Token": csrf }, signal }),
  projects: (signal: AbortSignal) => request("/projects", decodeList(decodeProject), { signal }),
  project: (projectId: string, signal: AbortSignal) => request(`/projects/${encodeURIComponent(projectId)}`, decodeProject, { signal }),
  directory: (path: string, signal: AbortSignal) => request(`/directories?path=${encodeURIComponent(path)}`, decodeDirectory, { signal }),
  entries: (projectId: string, folderId: string, version: number, path: string, cursor: string, signal: AbortSignal) => request(`/projects/${encodeURIComponent(projectId)}/folders/${encodeURIComponent(folderId)}/entries?project_version=${version}&path=${encodeURIComponent(path)}&cursor=${encodeURIComponent(cursor)}`, decodeFileListing, { signal }),
  content: (projectId: string, folderId: string, version: number, path: string, signal: AbortSignal) => request(`/projects/${encodeURIComponent(projectId)}/folders/${encodeURIComponent(folderId)}/content?project_version=${version}&path=${encodeURIComponent(path)}`, decodeFileContent, { signal }),
  metadata: (projectId: string, folderId: string, version: number, path: string, signal: AbortSignal) => request(`/projects/${encodeURIComponent(projectId)}/folders/${encodeURIComponent(folderId)}/metadata?project_version=${version}&path=${encodeURIComponent(path)}`, decodeFileMetadata, { signal }),
  saveContent: (projectId: string, folderId: string, input: { project_version: number; path: string; expected_version: FileVersion; content: string }, csrf: string, signal: AbortSignal) => request(`/projects/${encodeURIComponent(projectId)}/folders/${encodeURIComponent(folderId)}/content`, decodeSaveResult,
    { method: "PUT", headers: { "Content-Type": "application/json", "X-CSRF-Token": csrf }, body: JSON.stringify(input), signal }),
  fileOperation: (projectId: string, input: FileOperation, csrf: string, signal: AbortSignal) => request(`/projects/${encodeURIComponent(projectId)}/file-operations`, decodeOperationResult,
    { method: "POST", headers: { "Content-Type": "application/json", "X-CSRF-Token": csrf }, body: JSON.stringify(input), signal }),
  deletePreview: (projectId: string, folderId: string, version: number, path: string, csrf: string, signal: AbortSignal) => request(`/projects/${encodeURIComponent(projectId)}/delete-preview`, decodeDeletePreview,
    { method: "POST", headers: { "Content-Type": "application/json", "X-CSRF-Token": csrf }, body: JSON.stringify({ project_version: version, folder_id: folderId, path }), signal }),
  createUpload: (input: UploadInput, csrf: string, signal: AbortSignal) => request("/uploads", decodeUploadState,
    { method: "POST", headers: { "Content-Type": "application/json", "X-CSRF-Token": csrf }, body: JSON.stringify(input), signal }),
  uploadStatus: (id: string, signal: AbortSignal) => request(`/uploads/${encodeURIComponent(id)}`, decodeUploadState, { signal }),
  uploadChunk: (id: string, index: number, bytes: ArrayBuffer, sha256: string, csrf: string, signal: AbortSignal) => request(`/uploads/${encodeURIComponent(id)}/chunks/${index}`, decodeUploadState,
    { method: "PUT", headers: { "Content-Type": "application/octet-stream", "X-Chunk-SHA256": sha256, "X-CSRF-Token": csrf }, body: bytes, signal }),
  completeUpload: (id: string, csrf: string, signal: AbortSignal) => request(`/uploads/${encodeURIComponent(id)}/complete`, decodeImportResult,
    { method: "POST", headers: { "X-CSRF-Token": csrf }, signal }),
  cancelUpload: (id: string, csrf: string, signal: AbortSignal) => request(`/uploads/${encodeURIComponent(id)}`, () => undefined,
    { method: "DELETE", headers: { "X-CSRF-Token": csrf }, signal }),
  createArchive: (projectId: string, folderId: string, version: number, path: string, csrf: string, signal: AbortSignal) => request("/archives", decodeArchive,
    { method: "POST", headers: { "Content-Type": "application/json", "X-CSRF-Token": csrf }, body: JSON.stringify({ project_id: projectId, folder_id: folderId, project_version: version, path }), signal }),
  archiveStatus: (id: string, signal: AbortSignal) => request(`/archives/${encodeURIComponent(id)}`, decodeArchive, { signal }),
  cancelArchive: (id: string, csrf: string, signal: AbortSignal) => request(`/archives/${encodeURIComponent(id)}`, () => undefined,
    { method: "DELETE", headers: { "X-CSRF-Token": csrf }, signal }),
  createProject: (name: string, paths: string[], mainIndex: number, csrf: string, signal: AbortSignal) => request("/projects", decodeProject,
    { method: "POST", headers: { "Content-Type": "application/json", "X-CSRF-Token": csrf }, body: JSON.stringify({ name, folder_paths: paths, main_index: mainIndex }), signal }),
  updateProject: (project: { id: string; version: number }, change: { name: string; add_paths: string[]; remove_folder_ids: string[]; main_folder_id: string }, csrf: string, signal: AbortSignal) => request(`/projects/${encodeURIComponent(project.id)}`, decodeProject,
    { method: "PATCH", headers: { "Content-Type": "application/json", "X-CSRF-Token": csrf }, body: JSON.stringify({ expected_version: project.version, ...change }), signal }),
  updateProjectWithNewMain: (project: { id: string; version: number }, change: { name: string; add_paths: string[]; remove_folder_ids: string[]; main_added_index: number }, csrf: string, signal: AbortSignal) => request(`/projects/${encodeURIComponent(project.id)}`, decodeProject,
    { method: "PATCH", headers: { "Content-Type": "application/json", "X-CSRF-Token": csrf }, body: JSON.stringify({ expected_version: project.version, ...change }), signal }),
  deleteProject: (project: { id: string; version: number }, csrf: string, signal: AbortSignal) => request(`/projects/${encodeURIComponent(project.id)}`, () => undefined,
    { method: "DELETE", headers: { "Content-Type": "application/json", "X-CSRF-Token": csrf }, body: JSON.stringify({ expected_version: project.version }), signal }),
  terminals: (signal: AbortSignal) => request("/terminals", decodeList(decodeTerminal), { signal }),
  terminal: (id: string, signal: AbortSignal) => request(`/terminals/${encodeURIComponent(id)}`, decodeTerminal, { signal }),
  createTerminal: (input: { project_id: string; project_version: number; folder_id?: string; display_name?: string; cols?: number; rows?: number }, csrf: string, signal: AbortSignal) => request("/terminals", decodeTerminal,
    { method: "POST", headers: { "Content-Type": "application/json", "X-CSRF-Token": csrf }, body: JSON.stringify(input), signal }),
  terminalHistory: (id: string, signal: AbortSignal) => request(`/terminals/${encodeURIComponent(id)}/history`, decodeTerminalHistory, { signal }),
  renameTerminal: (id: string, expected: string, name: string, csrf: string, signal: AbortSignal) => request(`/terminals/${encodeURIComponent(id)}`, decodeTerminal,
    { method: "PATCH", headers: { "Content-Type": "application/json", "X-CSRF-Token": csrf }, body: JSON.stringify({ expected_display_name: expected, display_name: name }), signal }),
  terminateBatch: (members: BatchTarget[], csrf: string, signal: AbortSignal) => request("/terminals/termination-batches", decodeTerminationBatch,
    { method: "POST", headers: { "Content-Type": "application/json", "X-CSRF-Token": csrf }, body: JSON.stringify({ members }), signal }),
  terminationBatch: (id: string, signal: AbortSignal) => request(`/terminals/termination-batches/${encodeURIComponent(id)}`, decodeTerminationBatch, { signal }),
};

export function fileDownloadURL(projectId: string, folderId: string, version: number, path: string): string {
  return `/api/v1/projects/${encodeURIComponent(projectId)}/folders/${encodeURIComponent(folderId)}/download?project_version=${version}&path=${encodeURIComponent(path)}`;
}
export function archiveDownloadURL(id: string): string { return `/api/v1/archives/${encodeURIComponent(id)}/download`; }

export function errorMessage(error: unknown): string {
  if (error instanceof ApiError && error.status === 429) return `请求过于频繁。${error.retryAfter !== null ? `请在 ${Math.ceil(error.retryAfter)} 秒后重试。` : "请稍后重试。"}`;
  if (error instanceof Error && error.name !== "TypeError") return error.message;
  return "无法连接服务器，请检查网络后重试。";
}
