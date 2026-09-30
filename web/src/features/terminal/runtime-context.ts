import { createContext, useContext, useLayoutEffect, useRef, useReducer } from "react";
import type { BatchTarget, Terminal } from "@/lib/api/decoder";
import type { TerminalEvent, TerminalRole } from "@/lib/ws/terminal";

export type RuntimeState = { connection: "connecting" | "connected" | "reconnecting" | "disconnected"; role: TerminalRole; viewerId: string; generation: number; history: boolean };
export type RuntimeActions = { takeover(): Promise<void>; target(): BatchTarget; cancel(id: string): boolean; retry(): void; history(): void; refreshHistory(): void; focus(): void };

export type RuntimeEntry = {
  terminal: Terminal;
  element: HTMLDivElement;
  closeRequested: boolean;
  handled: Set<() => void>;
  state: RuntimeState;
  actions?: RuntimeActions;
};
export type RuntimeScope = {
  ensure(terminal: Terminal): RuntimeEntry;
  requestClose(id: string): void;
  subscribe(listener: (id: string) => void): () => void;
  subscribeChanges(listener: (id: string) => void): () => void;
  entry(id: string): RuntimeEntry | undefined;
  register(id: string, actions: RuntimeActions): () => void;
  report(id: string, state: RuntimeState): void;
  event(id: string, event: TerminalEvent): void;
  takeover(terminal: Terminal): void;
  close(terminals: Terminal[]): void;
  rename(terminal: Terminal): void;
  inputIntent(id: string): void;
};
export const RuntimeContext = createContext<RuntimeScope | null>(null);

export function useTerminalRuntime(id?: string) {
  const scope = useContext(RuntimeContext);
  if (!scope) throw new Error("终端必须在工作区运行时内显示");
  const [, update] = useReducer(value => value + 1, 0);
  useLayoutEffect(() => scope.subscribe(changed => { if (!id || id === changed) update(); }), [scope, id]);
  return { scope, entry: id ? scope.entry(id) : undefined };
}

export function useTerminalStateChanges(listener: (id: string) => void) {
  const scope = useContext(RuntimeContext);
  const listenerRef = useRef(listener);
  listenerRef.current = listener;
  useLayoutEffect(() => scope?.subscribeChanges(id => listenerRef.current(id)), [scope]);
}
