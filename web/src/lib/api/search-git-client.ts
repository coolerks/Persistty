import { request } from "./client";
import { decodeBaseline, decodeDetail, decodeGitComparison, decodeLog, decodePreview, decodeRefs, decodeReplaceComparison, decodeRepositories, decodeSearch, decodeStatus } from "./search-git-decoder";
const projectPath = (id: string) => `/projects/${encodeURIComponent(id)}`;
const repoPath = (project: string, repo: string) => `${projectPath(project)}/repositories/${encodeURIComponent(repo)}`;
const write = (body: unknown, csrf: string, signal: AbortSignal): RequestInit => ({ method: "POST", headers: { "Content-Type": "application/json", "X-CSRF-Token": csrf }, body: JSON.stringify(body), signal });
const remove = (csrf: string, signal: AbortSignal): RequestInit => ({ method: "DELETE", headers: { "X-CSRF-Token": csrf }, signal });
export type SearchQuery = { project_version: number; folder_id: string; path: string; pattern: string; regex: boolean; case_sensitive: boolean; whole_word: boolean; include: string[]; exclude: string[] };
export type GitCompareInput = { project_version: number; path: string; kind: "head" | "staged" | "unstaged" | "reference" | "commit"; reference: string; commit_id: string; parent_id: string };
export const searchGitAPI = {
  search: (p: string, input: SearchQuery, csrf: string, signal: AbortSignal) => request(`${projectPath(p)}/searches`, decodeSearch, write(input, csrf, signal)),
  cancelSearch: (p: string, id: string, csrf: string, signal: AbortSignal) => request(`${projectPath(p)}/searches/${id}`, () => undefined, remove(csrf, signal)),
  preview: (p: string, input: { project_version: number; search_id: string; selected_match_ids: string[]; replacement: string }, csrf: string, signal: AbortSignal) => request(`${projectPath(p)}/replace-previews`, decodePreview, write(input, csrf, signal)),
  previewStatus: (p: string, id: string, signal: AbortSignal) => request(`${projectPath(p)}/replace-previews/${id}`, decodePreview, { signal }),
  previewComparison: (p: string, id: string, file: string, signal: AbortSignal) => request(`${projectPath(p)}/replace-previews/${id}?file_id=${file}`, decodeReplaceComparison, { signal }),
  apply: (p: string, id: string, input: { project_version: number; selected_file_ids: string[]; protected_file_ids: string[] }, csrf: string, signal: AbortSignal) => request(`${projectPath(p)}/replace-previews/${id}/apply`, decodePreview, write(input, csrf, signal)),
  cancelPreview: (p: string, id: string, csrf: string, signal: AbortSignal) => request(`${projectPath(p)}/replace-previews/${id}`, () => undefined, remove(csrf, signal)),
  repositories: (p: string, version: number, signal: AbortSignal) => request(`${projectPath(p)}/repositories?project_version=${version}`, decodeRepositories, { signal }),
  status: (p: string, version: number, repo: string, signal: AbortSignal) => request(`${repoPath(p, repo)}/status?project_version=${version}`, decodeStatus, { signal }),
  refs: (p: string, version: number, repo: string, signal: AbortSignal) => request(`${repoPath(p, repo)}/refs?project_version=${version}`, decodeRefs, { signal }),
  log: (p: string, version: number, repo: string, offset: number, signal: AbortSignal, head = "") => request(`${repoPath(p, repo)}/log?project_version=${version}&offset=${offset}&head=${encodeURIComponent(head)}`, decodeLog, { signal }),
  detail: (p: string, version: number, repo: string, commit: string, parent: string, signal: AbortSignal) => request(`${repoPath(p, repo)}/commits/${commit}?project_version=${version}&parent_id=${encodeURIComponent(parent)}`, decodeDetail, { signal }),
  compare: (p: string, repo: string, input: GitCompareInput, csrf: string, signal: AbortSignal) => request(`${repoPath(p, repo)}/comparisons`, decodeGitComparison, write(input, csrf, signal)),
  baseline: (p: string, version: number, folder: string, path: string, signal: AbortSignal) => request(`${projectPath(p)}/git-baseline?project_version=${version}&folder_id=${folder}&path=${encodeURIComponent(path)}`, decodeBaseline, { signal }),
};
