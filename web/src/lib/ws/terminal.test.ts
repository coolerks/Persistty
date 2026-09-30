import { afterEach, describe, expect, it, vi } from "vitest";
import fixture from "../../../../tests/contracts/terminal-runtime.json";
import { ProtocolError } from "@/lib/api/decoder";
import { decodeTerminalEvent, TerminalSocket } from "./terminal";

afterEach(() => vi.unstubAllGlobals());

describe("W03 终端 WS 协议", () => {
  it("读取首帧、控制权与全端倒计时", () => {
    expect(decodeTerminalEvent(fixture.ready)).toMatchObject({ type: "ready", role: "controller", generation: 2 });
    expect(decodeTerminalEvent(fixture.observer_ready)).toMatchObject({ type: "ready", role: "observer" });
    expect(decodeTerminalEvent(fixture.control_after_takeover)).toMatchObject({ type: "control", generation: 3 });
    expect(decodeTerminalEvent(fixture.termination_pending)).toMatchObject({ type: "termination_pending" });
    expect(decodeTerminalEvent(fixture.termination_cancelled)).toMatchObject({ type: "termination_cancelled" });
    expect(decodeTerminalEvent(fixture.stale_generation_error)).toMatchObject({ type: "error", code: "stale_generation" });
  });
  it("拒绝未知协议、越界尺寸和缺少字段", () => {
    expect(() => decodeTerminalEvent({ ...fixture.ready, protocol: 1 })).toThrow(ProtocolError);
    expect(() => decodeTerminalEvent({ ...fixture.ready, cols: 0 })).toThrow(ProtocolError);
    expect(() => decodeTerminalEvent({ ...fixture.ready, unexpected: true })).toThrow(ProtocolError);
    expect(() => decodeTerminalEvent({ type: "control", generation: 4 })).toThrow(ProtocolError);
  });
  it("v3 使用统一成员与结果，v2 不接受新增字段", () => {
    expect(decodeTerminalEvent(fixture.ready_v3, 3)).toMatchObject({ protocol: 3, pending_termination: { members: fixture.pending_v3.members } });
    expect(decodeTerminalEvent(fixture.pending_v3, 3)).toMatchObject({ members: fixture.pending_v3.members });
    expect(decodeTerminalEvent(fixture.executed_v3, 3)).toMatchObject({ results: fixture.executed_v3.results });
    expect(decodeTerminalEvent(fixture.metadata_v3, 3)).toMatchObject({ display_name: "构建" });
    expect(() => decodeTerminalEvent(fixture.metadata_v3)).toThrow(ProtocolError);
    expect(() => decodeTerminalEvent({ ...fixture.pending_v3, members: [...fixture.pending_v3.members, fixture.pending_v3.members[0]] }, 3)).toThrow(ProtocolError);
  });
  it("离线立即拒绝输入，dispose 后忽略迟到帧且不重放", () => {
    class FakeSocket {
      static OPEN = 1;
      static instances: FakeSocket[] = [];
      readyState = 1;
      bufferedAmount = 0;
      binaryType = "";
      onmessage: ((event: { data: string }) => void) | null = null;
      onclose: ((event: { code: number }) => void) | null = null;
      close = vi.fn();
      send = vi.fn();
      constructor() { FakeSocket.instances.push(this); }
    }
    vi.stubGlobal("WebSocket", FakeSocket);
    const event = vi.fn();
    const closed = vi.fn();
    const socket = new TerminalSocket(fixture.ready.terminal_id, { event, closed, output: vi.fn() });
    const transport = FakeSocket.instances[0]!;
    transport!.onmessage!({ data: JSON.stringify({ ...fixture.ready, protocol: 3 }) });
    expect(socket.sendInput(new Uint8Array([3]))).toBe(true);
    window.dispatchEvent(new Event("offline"));
    expect(transport!.close).toHaveBeenCalledWith(1000, "view_closed");
    expect(socket.sendInput(new Uint8Array([3]))).toBe(false);
    expect(transport!.send).toHaveBeenCalledTimes(1);
    socket.dispose();
    transport!.onmessage!({ data: JSON.stringify(fixture.control_after_takeover) });
    transport!.onclose!({ code: 1000 });
    expect(event).toHaveBeenCalledTimes(1);
    expect(closed).toHaveBeenCalledExactlyOnceWith(1001);
  });
});
