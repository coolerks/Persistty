import { act, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, expect, it, vi } from "vitest";
import fixture from "../../../../tests/contracts/foundation.json";
import { AuthProvider } from "./AuthProvider";
import { useAuth } from "./auth-context";

afterEach(() => { vi.unstubAllGlobals(); vi.useRealTimers(); });
function sessionResponse(csrf = "old-session", ttl = 60_000) {
  return Response.json({ ...fixture.session, data: { ...fixture.session.data, csrf_token: csrf, expires_at: new Date(Date.now() + ttl).toISOString() } });
}
function Probe({ loginSignal = new AbortController().signal }: { loginSignal?: AbortSignal }) {
  const auth = useAuth();
  return <>
    <output aria-label="认证状态">{auth.state.status === "authenticated" ? auth.state.session.csrf_token : auth.state.status}</output>
    <button onClick={() => { void auth.logout(new AbortController().signal); }}>退出</button>
    <button onClick={auth.expire}>到期</button>
    <button onClick={() => { void auth.login("test-only-password", loginSignal); }}>登录</button>
  </>;
}
it("过期后等待已取消旧登出结算再登录，不被迟到响应覆盖", async () => {
  let reply: ((response: Response) => void) | undefined;
  let logoutSignal: AbortSignal | null | undefined;
  const fetch = vi.fn((path: string, options: RequestInit) => {
    if (path.endsWith("/auth/session")) return Promise.resolve(sessionResponse());
    if (path.endsWith("/auth/login")) return Promise.resolve(sessionResponse("new-session"));
    logoutSignal = options.signal;
    return new Promise<Response>(resolve => { reply = resolve; });
  });
  vi.stubGlobal("fetch", fetch);
  render(<AuthProvider><Probe /></AuthProvider>);
  const user = userEvent.setup();
  await waitFor(() => expect(screen.getByLabelText("认证状态")).toHaveTextContent("old-session"));
  await user.click(screen.getByText("退出"));
  await user.click(screen.getByText("到期"));
  expect(logoutSignal?.aborted).toBe(true);
  await user.click(screen.getByText("登录"));
  expect(fetch.mock.calls.some(([path]) => path.endsWith("/auth/login"))).toBe(false);
  await act(async () => reply?.(new Response(null, { status: 204 })));
  await waitFor(() => expect(screen.getByLabelText("认证状态")).toHaveTextContent("new-session"));
  expect(screen.getByLabelText("认证状态")).toHaveTextContent("new-session");
});
it("取消登录后忽略不遵守abort的迟到响应", async () => {
  let reply: ((response: Response) => void) | undefined;
  vi.stubGlobal("fetch", vi.fn((path: string) => path.endsWith("/auth/session") ? Promise.resolve(Response.json(fixture.unauthenticated, { status: 401 })) : new Promise<Response>(resolve => { reply = resolve; })));
  const controller = new AbortController();
  render(<AuthProvider><Probe loginSignal={controller.signal} /></AuthProvider>);
  await waitFor(() => expect(screen.getByLabelText("认证状态")).toHaveTextContent("anonymous"));
  await userEvent.click(screen.getByText("登录"));
  controller.abort();
  await act(async () => reply?.(sessionResponse("canceled-session")));
  expect(screen.getByLabelText("认证状态")).toHaveTextContent("anonymous");
});
it("认证到期计时器退出，不请求任何资源写入", async () => {
  vi.useFakeTimers();
  const fetch = vi.fn().mockResolvedValue(sessionResponse("expiring-session", 1000));
  vi.stubGlobal("fetch", fetch);
  render(<AuthProvider><Probe /></AuthProvider>);
  await act(async () => { await Promise.resolve(); });
  expect(screen.getByLabelText("认证状态")).toHaveTextContent("expiring-session");
  await act(async () => { await vi.advanceTimersByTimeAsync(1000); });
  expect(screen.getByLabelText("认证状态")).toHaveTextContent("anonymous");
  expect(fetch).toHaveBeenCalledTimes(1);
});
