import { StrictMode } from "react";
import { MemoryRouter } from "react-router";
import { act, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, expect, it, vi } from "vitest";
import fixture from "../../../tests/contracts/foundation.json";
import { App } from "./App";

afterEach(() => vi.unstubAllGlobals());
function sessionResponse() { return Response.json({ ...fixture.session, data: { ...fixture.session.data, expires_at: new Date(Date.now() + 86400000).toISOString() } }); }
function mount(path: string) { return render(<StrictMode><MemoryRouter initialEntries={[path]}><App /></MemoryRouter></StrictMode>); }

it("书签匿名跳登录，真实提交后返回原项目404；不会伪造项目", async () => {
  const fetch = vi.fn((path: string, options: RequestInit) => {
    if (path === "/api/v1/auth/session") return Promise.resolve(Response.json(fixture.unauthenticated, { status: 401 }));
    if (path === "/api/v1/auth/login") { expect(options.body).toBe(JSON.stringify({ password: "test-password" })); return Promise.resolve(sessionResponse()); }
    return Promise.resolve(Response.json(fixture.not_found, { status: 404 }));
  });
  vi.stubGlobal("fetch", fetch); mount("/projects/missing");
  const user = userEvent.setup();
  await user.type(await screen.findByLabelText("访问密码"), "test-password");
  await user.click(screen.getByRole("button", { name: "登录" }));
  expect(await screen.findByText("项目不存在")).toBeInTheDocument();
  expect(fetch.mock.calls.some(([path]) => path === "/api/v1/projects/missing")).toBe(true);
  expect(localStorage.length).toBe(0);
});
it("真实空列表与失败分离，退出不调用任何终端操作", async () => {
  const fetch = vi.fn((path: string) => {
    if (path.endsWith("/auth/session")) return Promise.resolve(sessionResponse());
    if (path.endsWith("/auth/logout")) return Promise.resolve(new Response(null, { status: 204 }));
    return Promise.resolve(Response.json(fixture.empty_projects));
  });
  vi.stubGlobal("fetch", fetch); mount("/projects");
  expect(await screen.findByText("暂无项目")).toBeInTheDocument();
  await userEvent.click(screen.getByRole("button", { name: "退出登录" }));
  expect(await screen.findByLabelText("访问密码")).toBeInTheDocument();
  expect(fetch.mock.calls.filter(([path]) => path.includes("/terminals"))).toHaveLength(0);
});
it("网络错误不可显示为空列表，能手动重试", async () => {
  let fail = true;
  vi.stubGlobal("fetch", vi.fn((path: string) => {
    if (path.endsWith("/auth/session")) return Promise.resolve(sessionResponse());
    if (fail) return Promise.reject(new TypeError("offline"));
    return Promise.resolve(Response.json(fixture.empty_projects));
  }));
  mount("/projects");
  expect(await screen.findByRole("alert")).toHaveTextContent("无法连接服务器");
  expect(screen.queryByText("暂无项目")).not.toBeInTheDocument();
  fail = false; await userEvent.click(screen.getByRole("button", { name: "重试" }));
  expect(await screen.findByText("暂无项目")).toBeInTheDocument();
});
it("项目打开位置弹窗支持取消和当前标签，直达不重复询问", async () => {
  vi.stubGlobal("fetch", vi.fn((path: string) => Promise.resolve(path.endsWith("/auth/session") ? sessionResponse() : Response.json(path.endsWith("/projects") ? fixture.projects : fixture.project))));
  mount("/projects");
  const user = userEvent.setup();
  await user.click(await screen.findByRole("button", { name: /示例项目/ }));
  expect(screen.getByRole("dialog", { name: "打开项目" })).toBeInTheDocument();
  expect(screen.getByRole("link", { name: "新标签页" })).toHaveAttribute("rel", "noopener noreferrer");
  await user.keyboard("{Escape}");
  await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
  await user.click(screen.getByRole("button", { name: /示例项目/ }));
  await user.click(screen.getByRole("button", { name: "当前标签页" }));
  expect(await screen.findByRole("heading", { name: "示例项目" })).toBeInTheDocument();
  expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
});
it("资源401停止读取并返回登录，未知终端不创建或attach", async () => {
  let authorized = true;
  vi.stubGlobal("fetch", vi.fn((path: string) => {
    if (path.endsWith("/auth/session")) return Promise.resolve(sessionResponse());
    if (!authorized) return Promise.resolve(Response.json(fixture.unauthenticated, { status: 401 }));
    return Promise.resolve(Response.json(fixture.empty_terminals));
  }));
  mount("/terminals/missing");
  expect(await screen.findByText("终端不存在")).toBeInTheDocument();
  authorized = false; await userEvent.click(screen.getByRole("button", { name: "刷新终端" }));
  expect(await screen.findByLabelText("访问密码")).toBeInTheDocument();
});

it("切换项目取消旧请求，忽略不遵守abort的迟到响应", async () => {
  const oldReplies: Array<(response: Response) => void> = [];
  const oldSignals: Array<AbortSignal> = [];
  vi.stubGlobal("fetch", vi.fn((path: string, options: RequestInit) => {
    if (path.endsWith("/auth/session")) return Promise.resolve(sessionResponse());
    if (path.endsWith("/projects/old")) {
      if (options.signal) oldSignals.push(options.signal);
      return new Promise<Response>(resolve => oldReplies.push(resolve));
    }
    return Promise.resolve(Response.json(path.endsWith("/projects") ? fixture.projects : fixture.project));
  }));
  mount("/projects/old");
  await waitFor(() => expect(oldReplies.length).toBeGreaterThan(0));
  const user = userEvent.setup();
  await user.click(screen.getByRole("link", { name: /^项目$/ }));
  await user.click(await screen.findByRole("button", { name: /示例项目/ }));
  await user.click(screen.getByRole("button", { name: "当前标签页" }));
  expect(await screen.findByRole("heading", { name: "示例项目" })).toBeInTheDocument();
  await act(async () => oldReplies.forEach(resolve => resolve(Response.json({ ...fixture.project, data: { ...fixture.project.data, id: "old", name: "旧响应" } }))));
  expect(oldSignals.every(signal => signal.aborted)).toBe(true);
  expect(screen.queryByText("旧响应")).not.toBeInTheDocument();
});
