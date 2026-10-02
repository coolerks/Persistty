import { request } from "./client";
import type { FileVersion } from "./decoder";
import { decodeElevationPrepared, decodeElevationResult } from "./elevation-decoder";

export const elevationAPI = {
  prepare: (projectId: string, folderId: string, input: { project_version: number; path: string; expected_version: FileVersion; content: string }, csrf: string, signal: AbortSignal) => request(`/projects/${encodeURIComponent(projectId)}/folders/${encodeURIComponent(folderId)}/elevation-requests`, decodeElevationPrepared, { method: "POST", headers: { "Content-Type": "application/json", "X-CSRF-Token": csrf }, body: JSON.stringify(input), signal }),
  execute: (id: string, content: string, password: string, csrf: string, signal: AbortSignal) => request(`/elevation-requests/${encodeURIComponent(id)}/execute`, decodeElevationResult, { method: "POST", headers: { "Content-Type": "application/json", "X-CSRF-Token": csrf }, body: JSON.stringify({ content, password }), signal }),
  status: (id: string, signal: AbortSignal) => request(`/elevation-requests/${encodeURIComponent(id)}`, decodeElevationResult, { signal }),
  cancel: (id: string, csrf: string, signal: AbortSignal) => request(`/elevation-requests/${encodeURIComponent(id)}`, decodeElevationResult, { method: "DELETE", headers: { "X-CSRF-Token": csrf }, signal }),
};
export type ElevationAPI = typeof elevationAPI;
