import { array, decodeVersion, exact, id, integer, optionalText, ProtocolError } from "./decoder";
type Decoder<T> = (value: unknown) => T;
function shape<S extends Record<string, Decoder<unknown>>>(schema: S): Decoder<{ [K in keyof S]: ReturnType<S[K]> }> {
  return value => {
    const source = exact(value, Object.keys(schema));
    const result: Record<string, unknown> = {};
    for (const key of Object.keys(schema)) result[key] = schema[key]!(source[key]);
    return result as { [K in keyof S]: ReturnType<S[K]> };
  };
}
const str = (value: unknown) => optionalText(value);
const content = (value: unknown) => optionalText(value, 8 << 20);
const bool = (value: unknown) => { if (typeof value !== "boolean") throw new ProtocolError(); return value; };
const positive = (value: unknown) => integer(value, 1);
const list = <T>(decode: Decoder<T>, max: number): Decoder<T[]> => value => array(value, decode, max);
const nullable = <T>(decode: Decoder<T>): Decoder<T | null> => value => value === null ? null : decode(value);
const choice = <T extends string>(values: readonly T[]): Decoder<T> => value => { const found = values.find(item => item === value); if (!found) throw new ProtocolError(); return found; };
export const decodeSearchFile = shape({ id, folder_id: id, path: str, version: decodeVersion, matches: list(shape({ id, line: positive, column: positive, end_column: positive, preview: str }), 5000) });
export const decodeSearch = shape({ id, project_version: positive, files: list(decodeSearchFile, 2000), truncated: bool, skipped: list(shape({ folder_id: id, path: str, reason: str }), 1000), expires_at: str });
export const decodePreviewFile = shape({ id, folder_id: id, path: str, version: decodeVersion, count: positive, new_hash: str });
export const decodePreview = shape({ id, project_version: positive, files: list(decodePreviewFile, 2000), results: list(shape({ id, state: choice(["applied", "conflict", "skipped", "error", "cancelled"] as const), reason: str, version: nullable(decodeVersion) }), 2000), state: choice(["ready", "applying", "completed", "cancelled"] as const), expires_at: str });
export const decodeReplaceComparison = shape({ id, path: str, original: content, modified: content });
export const decodeRepository = shape({ id, folder_id: id, path: str, name: str, state: choice(["available", "unavailable"] as const), reason: str });
export const decodeRepositories = shape({ items: list(decodeRepository, 100), truncated: bool });
export const decodeStatus = shape({ repo_id: id, head: str, branch: str, total_paths: list(str, 5000), changes: list(shape({ path: str, old_path: str, index: str, worktree: str }), 10000) });
export const decodeRefs = shape({ items: list(shape({ name: str, commit_id: str }), 2000) });
export const decodeCommit = shape({ id, parents: list(str, 100), author: str, date: str, subject: (value: unknown) => optionalText(value, 64 << 10) });
export const decodeLog = shape({ head: str, items: list(decodeCommit, 50), next_offset: (value: unknown) => integer(value, -1) });
export const decodeFileStat = shape({ path: str, old_path: str, status: choice(["A", "M", "D", "T", "R", "C"] as const), additions: nullable(integer), deletions: nullable(integer) });
const githubURL = (value: unknown) => {
  const url = str(value);
  if (url && !/^https:\/\/github\.com\/(?!\.{1,2}\/)[A-Za-z0-9_.-]+\/(?!\.{1,2}\/)[A-Za-z0-9_.-]+\/commit\/(?:[a-f0-9]{40}|[a-f0-9]{64})$/.test(url)) throw new ProtocolError();
  return url;
};
const detailShape = shape({ commit: decodeCommit, parent_id: str, files: list(str, 5000), message: (value: unknown) => optionalText(value, 64 << 10), stats: list(decodeFileStat, 5000), github_url: githubURL });
export const decodeDetail = (value: unknown) => {
  const detail = detailShape(value);
  if (detail.stats.length !== detail.files.length || new Set(detail.files).size !== detail.files.length || detail.stats.some((stat, i) => stat.path !== detail.files[i] || (stat.additions === null) !== (stat.deletions === null))) throw new ProtocolError();
  if (detail.github_url && !detail.github_url.endsWith(`/commit/${detail.commit.id}`)) throw new ProtocolError();
  return detail;
};
export type CommitDetail = ReturnType<typeof decodeDetail>;
export type GitFileStat = ReturnType<typeof decodeFileStat>;
export const decodeGitComparison = shape({ old_path: str, repo_id: id, path: str, original: content, modified: content, binary: bool, baseline: str });
export const decodeBaseline = shape({ state: choice(["tracked", "untracked", "binary", "unavailable", "no_repository"] as const), repo_id: str, head: str, content, version: nullable(decodeVersion) });
export type SearchResult = ReturnType<typeof decodeSearch>;
export type SearchFile = ReturnType<typeof decodeSearchFile>;
export type ReplacePreview = ReturnType<typeof decodePreview>;
export type PreviewFile = ReturnType<typeof decodePreviewFile>;
export type Repository = ReturnType<typeof decodeRepository>;

const fileNamePath = (value: unknown) => {
  const path = str(value);
  if (!path || path.startsWith("/") || path.includes("\0") || path.split("/").some(part => !part || part === "." || part === ".." || part === ".git")) throw new ProtocolError();
  return path;
};
export const decodeFileNames = shape({ project_version: positive, items: list(shape({ folder_id: id, path: fileNamePath }), 100), truncated: bool });
export type FileNames = ReturnType<typeof decodeFileNames>;
