import { decodeBatchMembers, decodeBatchResults, ProtocolError, type BatchMember, type BatchResult } from "@/lib/api/decoder";

export type TerminalRole = "controller" | "observer";
export type DeviceAttributes = "primary" | "secondary";
export type PendingTermination = { request_id: string; deadline: string; members?: BatchMember[] };
export type TerminalEvent =
  | { type: "ready"; protocol: 2 | 3; terminal_id: string; viewer_id: string; role: TerminalRole; generation: number; cols: number; rows: number; pending_termination: PendingTermination | null }
  | { type: "control"; role: TerminalRole; generation: number }
  | { type: "resized"; cols: number; rows: number }
  | ({ type: "termination_pending" } & PendingTermination)
  | { type: "termination_cancelled"; request_id: string }
  | { type: "termination_executed"; request_id: string; state: "running" | "terminated" | "unavailable"; results?: BatchResult[] }
  | { type: "metadata"; terminal_id: string; display_name: string }
  | { type: "error"; code: string; message: string };

function fail(): never { throw new ProtocolError(); }
function record(value: unknown): Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value) ? value as Record<string, unknown> : fail();
}
function exact(value: Record<string, unknown>, keys: string[]): void {
  if (Object.keys(value).length !== keys.length || !keys.every(key => Object.hasOwn(value, key))) fail();
}
function label(value: unknown): string {
  return typeof value === "string" && value.length > 0 && value.length <= 256 && !value.includes("\0") ? value : fail();
}
function generation(value: unknown): number {
  return typeof value === "number" && Number.isSafeInteger(value) && value >= 1 ? value : fail();
}
function dimension(value: unknown): number {
  return typeof value === "number" && Number.isSafeInteger(value) && value >= 1 && value <= 1000 ? value : fail();
}
function role(value: unknown): TerminalRole {
  return value === "controller" || value === "observer" ? value : fail();
}
function deadline(value: unknown): string {
  const text = label(value);
  return /^\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d(?:\.\d+)?Z$/.test(text) && Number.isFinite(Date.parse(text)) ? text : fail();
}
function pending(value: unknown, protocol: 2 | 3): PendingTermination | null {
  if (value === null) return null;
  const item = record(value);
  exact(item, protocol === 3 ? ["request_id", "deadline", "members"] : ["request_id", "deadline"]);
  return { request_id: label(item.request_id), deadline: deadline(item.deadline), ...(protocol === 3 ? { members: decodeBatchMembers(item.members) } : {}) };
}

export function decodeTerminalEvent(value: unknown, protocol: 2 | 3 = 2): TerminalEvent {
  const item = record(value);
  switch (item.type) {
    case "ready": {
      exact(item, ["type", "protocol", "terminal_id", "viewer_id", "role", "generation", "cols", "rows", "pending_termination"]);
      if (item.protocol !== protocol) fail();
      return { type: "ready", protocol, terminal_id: label(item.terminal_id), viewer_id: label(item.viewer_id),
        role: role(item.role), generation: generation(item.generation), cols: dimension(item.cols), rows: dimension(item.rows),
        pending_termination: pending(item.pending_termination, protocol) };
    }
    case "control":
      exact(item, ["type", "role", "generation"]);
      return { type: "control", role: role(item.role), generation: generation(item.generation) };
    case "resized":
      exact(item, ["type", "cols", "rows"]);
      return { type: "resized", cols: dimension(item.cols), rows: dimension(item.rows) };
    case "termination_pending":
      exact(item, protocol === 3 ? ["type", "request_id", "deadline", "members"] : ["type", "request_id", "deadline"]);
      return { type: "termination_pending", request_id: label(item.request_id), deadline: deadline(item.deadline), ...(protocol === 3 ? { members: decodeBatchMembers(item.members) } : {}) };
    case "termination_cancelled":
      exact(item, ["type", "request_id"]);
      return { type: "termination_cancelled", request_id: label(item.request_id) };
    case "termination_executed":
      exact(item, protocol === 3 ? ["type", "request_id", "state", "results"] : ["type", "request_id", "state"]);
      if (item.state !== "running" && item.state !== "terminated" && item.state !== "unavailable") fail();
      return { type: "termination_executed", request_id: label(item.request_id), state: item.state, ...(protocol === 3 ? { results: decodeBatchResults(item.results) } : {}) };
    case "metadata":
      if (protocol !== 3) fail();
      exact(item, ["type", "terminal_id", "display_name"]);
      return { type: "metadata", terminal_id: label(item.terminal_id), display_name: label(item.display_name) };
    case "error":
      exact(item, ["type", "code", "message"]);
      return { type: "error", code: label(item.code), message: label(item.message) };
    default:
      return fail();
  }
}

type Handlers = {
  event(event: TerminalEvent): void;
  output(bytes: Uint8Array): void;
  closed(code: number): void;
};

export class TerminalSocket {
  private readonly socket: WebSocket;
  private ready = false;
  private role: TerminalRole = "observer";
  private currentGeneration = 0;
  private disposed = false;
  private readonly offline = () => {
    if (this.disposed) return;
    this.dispose();
    this.handlers.closed(1001);
  };

  constructor(id: string, private readonly handlers: Handlers) {
    const scheme = window.location.protocol === "https:" ? "wss:" : "ws:";
    this.socket = new WebSocket(`${scheme}//${window.location.host}/api/v1/terminals/${encodeURIComponent(id)}/stream?protocol=3`);
    window.addEventListener("offline", this.offline);
    this.socket.binaryType = "arraybuffer";
    this.socket.onmessage = event => {
      if (this.disposed) return;
      try {
        if (typeof event.data === "string") {
          const value: unknown = JSON.parse(event.data);
          const message = decodeTerminalEvent(value, 3);
          if (!this.ready && message.type !== "ready") fail();
          if (message.type === "ready") {
            if (this.ready || message.terminal_id !== id) fail();
            this.ready = true;
            this.role = message.role;
            this.currentGeneration = message.generation;
          } else if (message.type === "control") {
            this.role = message.role;
            this.currentGeneration = message.generation;
          }
          this.handlers.event(message);
        } else if (event.data instanceof ArrayBuffer && this.ready) {
          this.handlers.output(new Uint8Array(event.data));
        } else fail();
      } catch {
        this.socket.close(1008, "invalid_terminal_frame");
      }
    };
    this.socket.onclose = event => {
      window.removeEventListener("offline", this.offline);
      if (!this.disposed) this.handlers.closed(event.code);
    };
  }

  sendInput(bytes: Uint8Array): boolean {
    if (!this.canControl() || bytes.length < 1 || bytes.length > 65536 || this.socket.bufferedAmount > 1 << 20) return false;
    const frame = new Uint8Array(bytes.length + 8);
    new DataView(frame.buffer).setBigUint64(0, BigInt(this.currentGeneration));
    frame.set(bytes, 8);
    this.socket.send(frame);
    return true;
  }

  sendDeviceAttributes(kind: DeviceAttributes): boolean {
    if (this.disposed || !this.ready || this.socket.readyState !== WebSocket.OPEN || this.socket.bufferedAmount > 1 << 20) return false;
    this.socket.send(JSON.stringify({ type: "device_attributes", kind }));
    return true;
  }

  sendControl(type: "takeover" | "resize" | "terminate" | "cancel_termination", extra: { cols?: number; rows?: number; request_id?: string } = {}): boolean {
    if (!this.ready || this.socket.readyState !== WebSocket.OPEN || this.socket.bufferedAmount > 1 << 20) return false;
    if ((type === "resize" || type === "terminate") && this.role !== "controller") return false;
    if (type === "cancel_termination" && !extra.request_id) return false;
    const data = type === "cancel_termination" ? { type, request_id: extra.request_id } :
      type === "resize" ? { type, generation: this.currentGeneration, cols: extra.cols, rows: extra.rows } :
        { type, generation: this.currentGeneration };
    this.socket.send(JSON.stringify(data));
    return true;
  }

  canControl(): boolean {
    return !this.disposed && this.ready && this.role === "controller" && this.socket.readyState === WebSocket.OPEN;
  }

  dispose(): void {
    this.disposed = true;
    this.ready = false;
    window.removeEventListener("offline", this.offline);
    this.socket.close(1000, "view_closed");
  }

  reconnect(): void {
    this.ready = false;
    this.socket.close(1013, "renderer_backpressure");
  }
}
