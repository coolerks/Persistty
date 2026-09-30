import { createContext, useContext, useLayoutEffect, useRef } from "react";
import type { Terminal } from "@/lib/api/decoder";

export type RuntimeEntry = {
  terminal: Terminal;
  element: HTMLDivElement;
  closeRequested: boolean;
  handled: Set<() => void>;
};
export type RuntimeScope = {
  ensure(terminal: Terminal): RuntimeEntry;
  requestClose(id: string): void;
  subscribe(listener: (id: string) => void): () => void;
};
export const RuntimeContext = createContext<RuntimeScope | null>(null);

export function useTerminalStateChanges(listener: (id: string) => void) {
  const scope = useContext(RuntimeContext);
  const listenerRef = useRef(listener);
  listenerRef.current = listener;
  useLayoutEffect(() => scope?.subscribe(id => listenerRef.current(id)), [scope]);
}
