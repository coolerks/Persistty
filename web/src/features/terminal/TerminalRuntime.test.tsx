import { StrictMode, useContext, useEffect } from "react";
import { act, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { Terminal } from "@/lib/api/decoder";
import { TerminalRuntimeProvider, TerminalSession } from "./TerminalRuntime";
import { AuthContext } from "@/features/auth/auth-context";
import { RuntimeContext } from "./runtime-context";
import { api } from "@/lib/api/client";
import userEvent from "@testing-library/user-event";
import { Tabs, TabsList } from "@/components/ui/tabs";
import { TerminalTab } from "./TerminalTab";
import { terminalVisible, useTerminalView } from "./terminal-view";

const lifecycle = vi.hoisted(() => ({ mounted: vi.fn(), disposed: vi.fn(), takeover: vi.fn<(id: string) => Promise<void>>() }));
vi.mock("./TerminalSession", () => ({ TerminalSessionView: ({ terminal }: { terminal: Terminal }) => {
  const scope = useContext(RuntimeContext);
  useEffect(() => { lifecycle.mounted(terminal.id); return () => lifecycle.disposed(terminal.id); }, [terminal.id]);
  useEffect(() => {
    if (!scope) return;
    scope.report(terminal.id, { connection: "connected", role: "observer", viewerId: `v-${terminal.id}`, generation: 1, history: false });
    return scope.register(terminal.id, { takeover: () => lifecycle.takeover(terminal.id), target: () => ({ terminal_id: terminal.id, viewer_id: `v-${terminal.id}`, generation: 2 }), cancel: () => true, retry: vi.fn(), history: vi.fn(), refreshHistory: vi.fn(), focus: vi.fn() });
  }, [scope, terminal.id]);
  return <div data-testid="runtime">{terminal.id}</div>;
} }));

const terminal: Terminal = { id: "t1", project_id: "p1", display_name: "终端", working_directory: "/workspace", state: "running" };

function Fixture({ top, visible = true }: { top: boolean; visible?: boolean }) {
  return <AuthContext.Provider value={{ state: { status: "anonymous" }, login: vi.fn(), logout: vi.fn(), expire: vi.fn(), retry: vi.fn() }}><TerminalRuntimeProvider><section data-testid="top">{top && visible && <TerminalSession terminal={terminal} closeRequested={false} onCloseRequestHandled={() => {}} />}</section>
    <section data-testid="bottom">{!top && visible && <TerminalSession terminal={terminal} closeRequested={false} onCloseRequestHandled={() => {}} />}</section></TerminalRuntimeProvider></AuthContext.Provider>;
}

const second: Terminal = { ...terminal, id: "t2", display_name: "终端1" };
const ended: Terminal = { ...terminal, id: "ended", display_name: "已结束", state: "terminated" };
const unavailable: Terminal = { ...terminal, id: "unknown", display_name: "不可用", state: "unavailable" };
function EndedFixture({ position }: { position: "top" | "bottom" }) {
  const dismissed = useTerminalView(state => state.dismissed);
  const items = [ended, unavailable].filter(item => terminalVisible(item, dismissed));
  return <Tabs value="ended"><TabsList>{items.map(item => <TerminalTab key={item.id} terminal={item} region={items} active={item.id === "ended"} onActivate={vi.fn()} onRefresh={vi.fn()} position={position} />)}</TabsList></Tabs>;
}
function MixedFixture() {
  const scope = useContext(RuntimeContext);
  return <><button onClick={() => scope?.close([terminal, ended, unavailable])}>混合关闭</button><TerminalSession terminal={terminal} /><TerminalSession terminal={ended} /></>;
}
function CloseFixture() {
  const scope = useContext(RuntimeContext);
  return <><button onClick={() => scope?.close([terminal, second])}>测试关闭</button><TerminalSession terminal={terminal} /><TerminalSession terminal={second} /></>;
}

describe("工作台范围终端运行时", () => {
  for (const position of ["top", "bottom"] as const) it(`${position} 已结束标签 X 可关闭且零终止/接管请求，未知状态仍保护`, async () => {
    useTerminalView.setState({ dismissed: {} }); lifecycle.takeover.mockClear();
    const terminate = vi.spyOn(api, "terminateBatch");
    render(<AuthContext.Provider value={{ state: { status: "anonymous" }, login: vi.fn(), logout: vi.fn(), expire: vi.fn(), retry: vi.fn() }}><TerminalRuntimeProvider><EndedFixture position={position} /></TerminalRuntimeProvider></AuthContext.Provider>);
    const user = userEvent.setup();
    expect(screen.getByRole("button", { name: "关闭终端 不可用" })).toBeDisabled();
    await user.click(screen.getByRole("button", { name: "关闭终端 已结束" }));
    expect(screen.queryByRole("tab", { name: /已结束/ })).not.toBeInTheDocument();
    expect(screen.getByRole("tab", { name: /不可用/ })).toBeInTheDocument();
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    expect(terminate).not.toHaveBeenCalled(); expect(lifecycle.takeover).not.toHaveBeenCalled();
  });
  it("混合关闭仅为 running 弹确认，terminated 关闭视图而 unavailable 保留", async () => {
    useTerminalView.setState({ dismissed: {} }); lifecycle.takeover.mockClear();
    const terminate = vi.spyOn(api, "terminateBatch");
    render(<AuthContext.Provider value={{ state: { status: "anonymous" }, login: vi.fn(), logout: vi.fn(), expire: vi.fn(), retry: vi.fn() }}><TerminalRuntimeProvider><MixedFixture /></TerminalRuntimeProvider></AuthContext.Provider>);
    const user = userEvent.setup(); await user.click(screen.getByRole("button", { name: "混合关闭" }));
    expect(screen.getByRole("dialog")).toHaveTextContent("终端");
    expect(screen.getByRole("dialog")).not.toHaveTextContent("已结束");
    expect(screen.getByRole("dialog")).not.toHaveTextContent("不可用");
    expect(useTerminalView.getState().dismissed).toEqual({ ended: true });
    await user.click(screen.getByRole("button", { name: "取消" }));
    expect(terminate).not.toHaveBeenCalled(); expect(lifecycle.takeover).not.toHaveBeenCalled();
  });
  it("宿主移动和隐藏不销毁实例，离开工作台才释放", async () => {
    lifecycle.mounted.mockClear(); lifecycle.disposed.mockClear();
    const view = render(<StrictMode><Fixture top={false} /></StrictMode>);
    const runtime = await screen.findByTestId("runtime");
    const mounts = lifecycle.mounted.mock.calls.length;
    const disposals = lifecycle.disposed.mock.calls.length;
    view.rerender(<StrictMode><Fixture top /></StrictMode>);
    expect(screen.getByTestId("top")).toContainElement(runtime);
    view.rerender(<StrictMode><Fixture top visible={false} /></StrictMode>);
    expect(document.body).not.toContainElement(runtime);
    view.rerender(<StrictMode><Fixture top={false} /></StrictMode>);
    await waitFor(() => expect(screen.getByTestId("bottom")).toContainElement(runtime));
    expect(lifecycle.mounted).toHaveBeenCalledTimes(mounts);
    expect(lifecycle.disposed).toHaveBeenCalledTimes(disposals);
    act(() => view.unmount());
    expect(lifecycle.disposed).toHaveBeenCalledTimes(disposals + 1);
  });
  it("部分接管失败不发起批次，展示已接管对象，不自动重试", async () => {
    const terminate = vi.spyOn(api, "terminateBatch");
    lifecycle.takeover.mockImplementation(async id => { if (id === "t2") throw new Error("接管被拒绝"); });
    const view = render(<AuthContext.Provider value={{ state: { status: "authenticated", session: { authenticated: true, csrf_token: "test", expires_at: "2099-01-01T00:00:00Z" } }, login: vi.fn(), logout: vi.fn(), expire: vi.fn(), retry: vi.fn() }}><TerminalRuntimeProvider><CloseFixture /></TerminalRuntimeProvider></AuthContext.Provider>);
    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: "测试关闭" }));
    expect(screen.getByRole("dialog")).toHaveTextContent("终端1");
    expect(lifecycle.takeover).not.toHaveBeenCalled();
    await user.click(screen.getByRole("button", { name: "接管并关闭" }));
    expect(await screen.findByText(/终端1：接管被拒绝/)).toHaveTextContent("已获取控制权：终端；未回滚控制权，未启动新的倒计时");
    expect(lifecycle.takeover.mock.calls.map(call => call[0])).toEqual(["t1", "t2"]);
    expect(terminate).not.toHaveBeenCalled();
    await user.click(screen.getByRole("button", { name: "取消" }));
    expect(lifecycle.takeover).toHaveBeenCalledTimes(2);
    view.unmount(); terminate.mockRestore();
  });
});
