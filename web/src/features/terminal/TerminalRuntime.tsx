import { useContext, useEffect, useLayoutEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { api } from "@/lib/api/client";
import { Empty, EmptyDescription, EmptyHeader } from "@/components/ui/empty";
import type { Terminal } from "@/lib/api/decoder";
import { TerminalSessionView } from "./TerminalSession";
import { RuntimeContext, type RuntimeEntry, type RuntimeScope } from "./runtime-context";

export function TerminalRuntimeProvider({ children }: { children: ReactNode }) {
  const entries = useRef(new Map<string, RuntimeEntry>());
  const listeners = useRef(new Set<(id: string) => void>());
  const requests = useRef(new Set<AbortController>());
  const active = useRef(true);
  const [, render] = useState(0);
  useEffect(() => {
    active.current = true;
    const pending = requests.current;
    return () => { active.current = false; for (const request of pending) request.abort(); pending.clear(); };
  }, []);
  const scope = useMemo<RuntimeScope>(() => ({
    ensure(terminal) {
      let entry = entries.current.get(terminal.id);
      if (!entry) {
        const element = document.createElement("div");
        element.className = "terminal-runtime-mount";
        entry = { terminal, element, closeRequested: false, handled: new Set() };
        entries.current.set(terminal.id, entry);
        render(value => value + 1);
      } else if (entry.terminal.state !== terminal.state) {
        entry.terminal = terminal;
        render(value => value + 1);
      }
      return entry;
    },
    requestClose(id) {
      const entry = entries.current.get(id);
      if (entry && !entry.closeRequested) { entry.closeRequested = true; render(value => value + 1); }
    },
    subscribe(listener) {
      listeners.current.add(listener);
      return () => { listeners.current.delete(listener); };
    },
  }), []);

  function handled(entry: RuntimeEntry) {
    entry.closeRequested = false;
    render(value => value + 1);
    for (const listener of entry.handled) listener();
  }

  function stateChanged(entry: RuntimeEntry) {
    for (const listener of listeners.current) listener(entry.terminal.id);
    const request = new AbortController();
    requests.current.add(request);
    void api.terminals(request.signal).then(items => {
      if (!active.current || request.signal.aborted) return;
      const terminal = items.find(item => item.id === entry.terminal.id);
      if (terminal) { entry.terminal = terminal; render(value => value + 1); }
    }).catch(() => { /* The owning list reports refresh failures. */ }).finally(() => requests.current.delete(request));
  }

  return <RuntimeContext.Provider value={scope}>{children}{Array.from(entries.current.values(), entry =>
    createPortal(entry.terminal.state === "running" ? <TerminalSessionView terminal={entry.terminal}
      closeRequested={entry.closeRequested} onCloseRequestHandled={() => handled(entry)} onStateChange={() => stateChanged(entry)} /> :
      <Empty><EmptyHeader><EmptyDescription>{entry.terminal.state === "terminated" ? "会话已结束；不会自动重新执行命令。" : "终端服务暂时不可用，请刷新终端列表。"}</EmptyDescription></EmptyHeader></Empty>, entry.element, entry.terminal.id))}
  </RuntimeContext.Provider>;
}

// The portal container is stable; changing its DOM parent preserves xterm and WS.
export function TerminalSession({ terminal, closeRequested, onCloseRequestHandled }: {
  terminal: Terminal; closeRequested: boolean; onCloseRequestHandled(): void;
}) {
  const scope = useContext(RuntimeContext);
  if (!scope) throw new Error("终端必须在工作区运行时内显示");
  const hostRef = useRef<HTMLDivElement>(null);
  const handledRef = useRef(onCloseRequestHandled);
  handledRef.current = onCloseRequestHandled;
  useLayoutEffect(() => {
    if (!scope || !hostRef.current) return;
    const host = hostRef.current;
    const entry = scope.ensure(terminal);
    const handled = () => handledRef.current();
    entry.handled.add(handled);
    host.append(entry.element);
    return () => {
      entry.handled.delete(handled);
      if (entry.element.parentElement === host) entry.element.remove();
    };
  }, [scope, terminal]);
  useLayoutEffect(() => { if (closeRequested) scope?.requestClose(terminal.id); }, [scope, terminal.id, closeRequested]);
  return <div className="terminal-runtime-host" ref={hostRef} />;
}
