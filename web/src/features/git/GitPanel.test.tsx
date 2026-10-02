import type { ReactNode } from "react";
import { act, cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, expect, test, vi } from "vitest";
import type { Project } from "@/lib/api/decoder";
import { searchGitAPI } from "@/lib/api/search-git-client";
import { GitPanel } from "./GitPanel";
vi.mock("@/features/auth/auth-context", () => ({ useAuth: () => ({ state: { status: "authenticated", session: { csrf_token: "fixture" } } }) }));
vi.mock("@/features/workspaces/editor-context", () => ({ useEditorScope: () => ({ buffers: new Map() }) }));
// Geometry is covered by real browsers; this test isolates request lifetimes.
vi.mock("./GitSections", () => ({ GitSections: ({ changes, history }: { changes: ReactNode; history: ReactNode }) => <>{changes}{history}</> }));
const project: Project = { id: "p", name: "Fixture", version: 1, main_folder_id: "f", folders: [{ id: "f", path: "/fixture" }] };
const commit = { id: "a".repeat(40), subject: "current commit", parents: [], author: "Fixture", date: "2026-10-02T00:00:00Z" };
const history = { head: commit.id, items: [commit], next_offset: -1 };
const status = { repo_id: "repo", branch: "main", head: commit.id, changes: [], total_paths: [] };
afterEach(() => { cleanup(); vi.restoreAllMocks(); });
function repositories() { vi.spyOn(searchGitAPI, "repositories").mockImplementation(async (_id, version) => ({ items: [{ id: `repo${version}`, folder_id: "f", path: "", name: `repo${version}`, state: "available", reason: "" }], truncated: false })); }
test("项目版本变更后迟到历史不能在旧仓库启动状态或覆盖新历史", async () => {
  repositories(); let finish: (value: typeof history) => void = () => {}; let oldSignal: AbortSignal | undefined;
  const load = vi.spyOn(searchGitAPI, "log").mockImplementation((_p, version, _repo, _offset, signal) => { if (version === 1) { oldSignal = signal; return new Promise(resolve => { finish = resolve; }); } return Promise.resolve(history); });
  const states = vi.spyOn(searchGitAPI, "status").mockResolvedValue(status);
  const view = render(<GitPanel project={project} mobile={false} visible />);
  await waitFor(() => expect(load).toHaveBeenCalledTimes(1));
  view.rerender(<GitPanel project={{ ...project, version: 2 }} mobile={false} visible />);
  await screen.findByRole("button", { name: /^current commit/ });
  await waitFor(() => expect(states).toHaveBeenCalledTimes(1));
  expect(oldSignal?.aborted).toBe(true);
  await act(async () => { finish({ ...history, items: [{ ...commit, subject: "obsolete commit" }] }); });
  expect(states).toHaveBeenCalledTimes(1); expect(states.mock.calls[0]?.slice(0, 3)).toEqual(["p", 2, "repo2"]);
  expect(screen.queryByRole("button", { name: /^obsolete commit/ })).not.toBeInTheDocument();
});
test("慢状态不能锁住提交展开，取消后迟到状态不回填或重试", async () => {
  repositories(); vi.spyOn(searchGitAPI, "log").mockResolvedValue(history);
  let finish: (value: typeof status) => void = () => {}; let signal: AbortSignal | undefined;
  const states = vi.spyOn(searchGitAPI, "status").mockImplementation((_p, _v, _r, current) => { signal = current; return new Promise(resolve => { finish = resolve; }); });
  vi.spyOn(searchGitAPI, "detail").mockResolvedValue({ commit, parent_id: "", files: [], stats: [], message: "current commit", github_url: "" });
  render(<GitPanel project={project} mobile={false} visible />);
  await waitFor(() => expect(states).toHaveBeenCalledTimes(1));
  await userEvent.click(await screen.findByRole("button", { name: /^current commit/ }));
  await screen.findByText("此父提交下没有文件变更。"); expect(signal?.aborted).toBe(true);
  await act(async () => { finish({ ...status, branch: "obsolete branch" }); });
  expect(screen.queryByText(/obsolete branch/)).not.toBeInTheDocument(); expect(states).toHaveBeenCalledTimes(1);
});
