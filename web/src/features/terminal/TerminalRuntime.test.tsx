import { StrictMode, useEffect } from "react";
import { act, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { Terminal } from "@/lib/api/decoder";
import { TerminalRuntimeProvider, TerminalSession } from "./TerminalRuntime";

const lifecycle = vi.hoisted(() => ({ mounted: vi.fn(), disposed: vi.fn() }));
vi.mock("./TerminalSession", () => ({ TerminalSessionView: ({ terminal }: { terminal: Terminal }) => {
  useEffect(() => { lifecycle.mounted(terminal.id); return () => lifecycle.disposed(terminal.id); }, [terminal.id]);
  return <div data-testid="runtime">{terminal.id}</div>;
} }));

const terminal: Terminal = { id: "t1", project_id: "p1", display_name: "终端", working_directory: "/workspace", state: "running" };

function Fixture({ top, visible = true }: { top: boolean; visible?: boolean }) {
  return <TerminalRuntimeProvider><section data-testid="top">{top && visible && <TerminalSession terminal={terminal} closeRequested={false} onCloseRequestHandled={() => {}} />}</section>
    <section data-testid="bottom">{!top && visible && <TerminalSession terminal={terminal} closeRequested={false} onCloseRequestHandled={() => {}} />}</section></TerminalRuntimeProvider>;
}

describe("工作台范围终端运行时", () => {
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
});
