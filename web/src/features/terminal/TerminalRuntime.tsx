import { useCallback, useContext, useEffect, useLayoutEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { api, ApiError, errorMessage } from "@/lib/api/client";
import { useAuth } from "@/features/auth/auth-context";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Empty, EmptyDescription, EmptyHeader } from "@/components/ui/empty";
import type { Terminal, TerminationBatch } from "@/lib/api/decoder";
import { TerminalSessionView } from "./TerminalSession";
import { RuntimeContext, type RuntimeActions, type RuntimeEntry, type RuntimeScope } from "./runtime-context";

export function TerminalRuntimeProvider({ children }: { children: ReactNode }) {
  const auth = useAuth();
  const authRef = useRef(auth); authRef.current = auth;
  const entries = useRef(new Map<string, RuntimeEntry>());
  const listeners = useRef(new Set<(id: string) => void>());
  const changes = useRef(new Set<(id: string) => void>());
  const requests = useRef(new Set<AbortController>());
  const active = useRef(true);
  const [, render] = useState(0);
  const [closing, setClosing] = useState<Terminal[] | null>(null);
  const [taking, setTaking] = useState<Terminal | null>(null);
  const [renaming, setRenaming] = useState<Terminal | null>(null);
  const [name, setName] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [batches, setBatches] = useState<Record<string, TerminationBatch>>({});
  const [now, setNow] = useState(Date.now());
  const readyRef = useRef<((terminal: Terminal) => Promise<RuntimeActions>) | null>(null);
  const notify = useCallback((id: string) => { for (const listener of listeners.current) listener(id); }, []);
  const tracked = useCallback(() => { const request = new AbortController(); requests.current.add(request); return request; }, []);
  const remember = useCallback((batch: TerminationBatch) => {
    setBatches(current => {
      if (batch.state === "cancelled") { const next = { ...current }; delete next[batch.request_id]; return next; }
      return { ...current, [batch.request_id]: batch };
    });
  }, []);
  const stateChanged = useCallback((entry: RuntimeEntry) => {
    notify(entry.terminal.id);
    for (const listener of changes.current) listener(entry.terminal.id);
    const request = tracked();
    void api.terminals(request.signal).then(items => {
      if (!active.current || request.signal.aborted) return;
      const terminal = items.find(item => item.id === entry.terminal.id);
      if (terminal) { entry.terminal = terminal; render(value => value + 1); notify(terminal.id); }
    }).catch(() => { /* The owning list reports refresh failures. */ }).finally(() => requests.current.delete(request));
  }, [notify, tracked]);
  const scope = useMemo<RuntimeScope>(() => ({
    ensure(terminal) {
      let entry = entries.current.get(terminal.id);
      if (!entry) {
        const element = document.createElement("div"); element.className = "terminal-runtime-mount";
        entry = { terminal, element, closeRequested: false, handled: new Set(),
          state: { connection: "connecting", role: "observer", viewerId: "", generation: 0, history: false } };
        entries.current.set(terminal.id, entry); render(value => value + 1);
      } else if (JSON.stringify(entry.terminal) !== JSON.stringify(terminal)) {
        entry.terminal = terminal; render(value => value + 1); notify(terminal.id);
      }
      return entry;
    },
    entry(id) { return entries.current.get(id); },
    requestClose(id) { const entry = entries.current.get(id); if (entry) { setError(null); setClosing([{ ...entry.terminal }]); } },
    close(terminals) { if (terminals.length) { setError(null); setClosing(terminals.map(item => ({ ...item }))); } },
    takeover(terminal) {
      setError(null);
      if (readyRef.current) void readyRef.current(terminal).then(actions => actions.takeover()).catch(reason => { if (active.current) setError(`${terminal.display_name}：${errorMessage(reason)}`); });
    },
    inputIntent(id) { const entry = entries.current.get(id); if (entry) { setError(null); setTaking({ ...entry.terminal }); } },
    rename(terminal) { setError(null); setRenaming({ ...terminal }); setName(terminal.display_name); },
    subscribe(listener) { listeners.current.add(listener); return () => { listeners.current.delete(listener); }; },
    subscribeChanges(listener) { changes.current.add(listener); return () => { changes.current.delete(listener); }; },
    register(id, actions) {
      const entry = entries.current.get(id);
      if (entry) { entry.actions = actions; notify(id); }
      return () => { if (entry?.actions === actions) { delete entry.actions; notify(id); } };
    },
    report(id, state) { const entry = entries.current.get(id); if (entry) { entry.state = state; notify(id); } },
    event(id, event) {
      const entry = entries.current.get(id);
      if (!entry) return;
      if (event.type === "error") setError(`${entry.terminal.display_name}：${event.message}`);
      const pending = event.type === "ready" ? event.pending_termination : event.type === "termination_pending" ? event : null;
      if (pending?.members) remember({ ...pending, members: pending.members, state: "pending", results: [] });
      if (event.type === "termination_cancelled") setBatches(current => { const next = { ...current }; delete next[event.request_id]; return next; });
      if (event.type === "termination_executed") {
        setBatches(current => {
          const previous = current[event.request_id];
          return previous && event.results ? { ...current, [event.request_id]: { ...previous, state: "completed", results: event.results } } : current;
        });
        stateChanged(entry);
      }
      if (event.type === "metadata") {
        entry.terminal = { ...entry.terminal, display_name: event.display_name };
        render(value => value + 1); notify(id);
        for (const listener of changes.current) listener(id);
      }
    },
  }), [notify, remember, stateChanged]);
  useEffect(() => {
    active.current = true;
    const pending = requests.current;
    return () => { active.current = false; for (const request of pending) request.abort(); pending.clear(); };
  }, []);
  useEffect(() => {
    if (!Object.values(batches).some(batch => batch.state === "pending" || batch.state === "executing")) return;
    const timer = window.setInterval(() => setNow(Date.now()), 250);
    return () => window.clearInterval(timer);
  }, [batches]);
  useEffect(() => {
    const waiting = Object.values(batches).filter(batch => batch.state === "pending" || batch.state === "executing");
    if (!waiting.length) return;
    const request = tracked();
    const requestSet = requests.current;
    let inFlight = false;
    const timer = window.setInterval(() => {
      if (inFlight) return;
      inFlight = true;
      void Promise.all(waiting.map(batch => api.terminationBatch(batch.request_id, request.signal).then(result => {
        if (!request.signal.aborted) { remember(result); if (result.state === "completed") for (const member of result.members) { const entry = entries.current.get(member.terminal_id); if (entry) stateChanged(entry); } }
      }).catch(reason => {
        if (request.signal.aborted) return;
        if (reason instanceof ApiError && reason.status === 404) {
          setBatches(current => { const next = { ...current }; delete next[batch.request_id]; return next; });
          for (const member of batch.members) { const entry = entries.current.get(member.terminal_id); if (entry) stateChanged(entry); }
          setError("服务端不再保留此终止请求，请刷新终端列表核实状态。不会自动重试终止。");
        } else setError(errorMessage(reason));
      }))).finally(() => { inFlight = false; });
    }, 1000);
    return () => { window.clearInterval(timer); request.abort(); requestSet.delete(request); };
  }, [batches, remember, stateChanged, tracked]);
  async function ready(terminal: Terminal): Promise<RuntimeActions> {
    scope.ensure(terminal);
    return new Promise((resolve, reject) => {
      let unsubscribe = () => {};
      const timer = window.setTimeout(() => { unsubscribe(); reject(new Error(`${terminal.display_name}：连接未就绪，未启动倒计时。`)); }, 6000);
      const check = () => {
        const entry = entries.current.get(terminal.id);
        if (!active.current) { window.clearTimeout(timer); unsubscribe(); reject(new Error("工作区已关闭。")); }
        else if (entry?.actions && entry.state.connection === "connected") { window.clearTimeout(timer); unsubscribe(); resolve(entry.actions); }
      };
      unsubscribe = scope.subscribe(check); check();
    });
  }
  async function perform() {
    if (busy || authRef.current.state.status !== "authenticated") return;
    setBusy(true); setError(null);
    const request = tracked();
    const acquired: string[] = [];
    try {
      if (renaming) {
        const item = await api.renameTerminal(renaming.id, renaming.display_name, name, authRef.current.state.session.csrf_token, request.signal);
        scope.ensure(item); notify(item.id); for (const listener of changes.current) listener(item.id); setRenaming(null);
      } else if (taking) {
        const actions = await ready(taking); await actions.takeover(); actions.focus(); setTaking(null);
      } else if (closing) {
        const targets = [];
        for (const terminal of closing) {
          try {
            const actions = await ready(terminal);
            await actions.takeover(); acquired.push(terminal.display_name); targets.push(actions.target());
          } catch (reason) { throw new Error(`${terminal.display_name}：${errorMessage(reason)}`, { cause: reason }); }
        }
        if (!active.current || request.signal.aborted) return;
        const batch = await api.terminateBatch(targets, authRef.current.state.session.csrf_token, request.signal);
        if (!request.signal.aborted) { remember(batch); setClosing(null); }
      }
    } catch (reason) {
      const failed = reason instanceof ApiError && reason.terminalId ? closing?.find(item => item.id === reason.terminalId)?.display_name : null;
      if (!request.signal.aborted && active.current) setError(`${failed ? `${failed}：` : ""}${errorMessage(reason)}${closing && acquired.length ? ` 已获取控制权：${acquired.join("、")}；未回滚控制权，未启动新的倒计时。` : ""}`);
    } finally { requests.current.delete(request); if (active.current) setBusy(false); }
  }
  function cancel(batch: TerminationBatch) {
    const accepted = batch.members.some(member => entries.current.get(member.terminal_id)?.actions?.cancel(batch.request_id));
    if (!accepted) setError("取消请求未发送，请检查终端连接。关闭对话框不会取消倒计时。");
  }
  readyRef.current = ready;
  const listed = Object.values(batches);
  return <RuntimeContext.Provider value={scope}>{children}{Array.from(entries.current.values(), entry =>
    createPortal(entry.terminal.state === "running" ? <TerminalSessionView terminal={entry.terminal} onStateChange={() => stateChanged(entry)} /> :
      <Empty><EmptyHeader><EmptyDescription>{entry.terminal.state === "terminated" ? "会话已结束；不会自动重新执行命令。" : "终端服务暂时不可用，请刷新终端列表。"}</EmptyDescription></EmptyHeader></Empty>, entry.element, entry.terminal.id))}
    {error && !closing && !taking && !renaming && !listed.length && <Alert variant="destructive" className="shrink-0"><AlertDescription>{error}</AlertDescription><Button variant="ghost" size="sm" onClick={() => setError(null)}>关闭</Button></Alert>}
    <Dialog open={Boolean(closing || taking || renaming)} onOpenChange={open => { if (!open && !busy) { setClosing(null); setTaking(null); setRenaming(null); setError(null); } }}>
      <DialogContent showCloseButton={!busy}><DialogHeader><DialogTitle>{renaming ? "重命名终端" : taking ? "接管终端？" : "终止终端？"}</DialogTitle>
        <DialogDescription>{renaming ? renaming.display_name : taking ? `${taking.display_name}：接管后可正常输入。触发此次提示的输入已丢弃，请重新输入。` : "关闭这些终端会终止其中正在运行的程序。必须先获取所有目标的控制权，再开始同一倒计时。"}</DialogDescription></DialogHeader>
        {closing && <ul className="terminal-dialog-list">{closing.map(item => <li key={item.id}>{item.display_name}</li>)}</ul>}
        {renaming && <FieldGroup><Field><FieldLabel htmlFor="terminal-name">终端名称</FieldLabel><Input id="terminal-name" value={name} disabled={busy} onChange={event => setName(event.target.value)} autoFocus /></Field></FieldGroup>}
        {error && <Alert variant="destructive"><AlertDescription>{error}</AlertDescription></Alert>}
        <DialogFooter><Button variant="outline" disabled={busy} onClick={() => { setClosing(null); setTaking(null); setRenaming(null); setError(null); }}>取消</Button>
          <Button variant={closing ? "destructive" : "default"} disabled={busy || Boolean(renaming && !name.trim())} onClick={() => void perform()}>{busy ? "处理中…" : renaming ? "保存" : taking ? "接管" : "接管并关闭"}</Button></DialogFooter>
      </DialogContent>
    </Dialog>
    <Dialog open={listed.length > 0} onOpenChange={() => {}}><DialogContent showCloseButton={false}><DialogHeader><DialogTitle>终端终止状态</DialogTitle><DialogDescription>倒计时期间任一查看端可以取消整批。隐藏面板不会终止会话。</DialogDescription></DialogHeader>
      <div className="terminal-dialog-list">{listed.map(batch => <section key={batch.request_id} className="terminal-batch-status"><strong>{batch.state === "completed" ? "执行结果" : batch.state === "executing" ? "正在执行终止" : `将在 ${Math.max(0, Math.ceil((Date.parse(batch.deadline) - now) / 1000))} 秒后终止`}</strong>
        <ul>{batch.members.map(member => <li key={member.terminal_id}>{member.display_name}{batch.state === "completed" && `：${({ running: "仍在运行", terminated: "已结束", unavailable: "状态未知，请刷新" })[batch.results.find(result => result.terminal_id === member.terminal_id)?.state ?? "unavailable"]}`}</li>)}</ul>
        {batch.state === "pending" ? <Button variant="outline" onClick={() => cancel(batch)}>取消终止</Button> : batch.state === "completed" ? <Button variant="outline" onClick={() => setBatches(current => { const next = { ...current }; delete next[batch.request_id]; return next; })}>关闭结果</Button> : null}
      </section>)}</div>{error && <Alert variant="destructive"><AlertDescription>{error}</AlertDescription></Alert>}
    </DialogContent></Dialog>
  </RuntimeContext.Provider>;
}

// Only the DOM parent changes; the portal's xterm and socket retain their identity.
export function TerminalSession({ terminal, closeRequested = false, onCloseRequestHandled }: {
  terminal: Terminal; closeRequested?: boolean; onCloseRequestHandled?(): void;
}) {
  const scope = useContext(RuntimeContext);
  if (!scope) throw new Error("终端必须在工作区运行时内显示");
  const hostRef = useRef<HTMLDivElement>(null);
  useLayoutEffect(() => {
    if (!hostRef.current) return;
    const host = hostRef.current; const entry = scope.ensure(terminal); host.append(entry.element);
    return () => { if (entry.element.parentElement === host) entry.element.remove(); };
  }, [scope, terminal]);
  useLayoutEffect(() => { if (closeRequested) { scope.requestClose(terminal.id); onCloseRequestHandled?.(); } }, [scope, terminal.id, closeRequested, onCloseRequestHandled]);
  return <div className="terminal-runtime-host" ref={hostRef} />;
}
