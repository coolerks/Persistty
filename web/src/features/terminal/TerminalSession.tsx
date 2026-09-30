import { useEffect, useRef, useState } from "react";
import { Terminal as XTerm } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { WebLinksAddon } from "@xterm/addon-web-links";
import { ArrowDown, ArrowLeft, ArrowRight, ArrowUp, History, Radio, RotateCw, ScrollText, Square, X } from "lucide-react";
import "@xterm/xterm/css/xterm.css";
import { Button } from "@/components/ui/button";
import { Toggle } from "@/components/ui/toggle";
import { Alert, AlertAction, AlertDescription } from "@/components/ui/alert";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { api, ApiError, errorMessage } from "@/lib/api/client";
import type { Terminal } from "@/lib/api/decoder";
import { useAuth } from "@/features/auth/auth-context";
import { type PendingTermination, type TerminalEvent, TerminalSocket, type TerminalRole } from "@/lib/ws/terminal";

const encoder = new TextEncoder();

function colors(element: HTMLElement) {
  const style = getComputedStyle(element);
  return { background: style.getPropertyValue("--background").trim(), foreground: style.getPropertyValue("--foreground").trim(),
    cursor: style.getPropertyValue("--primary").trim(), selectionBackground: style.getPropertyValue("--accent").trim() };
}

function openURL(value: string): void {
  try {
    const url = new URL(value);
    if (url.protocol === "http:" || url.protocol === "https:") window.open(url.href, "_blank", "noopener,noreferrer");
  } catch { /* Terminal output is not a valid URL. */ }
}

export function TerminalSessionView({ terminal, closeRequested, onCloseRequestHandled, onStateChange }: {
  terminal: Terminal; closeRequested: boolean; onCloseRequestHandled(): void; onStateChange(): void;
}) {
  const { expire } = useAuth();
  const hostRef = useRef<HTMLDivElement>(null);
  const historyHostRef = useRef<HTMLDivElement>(null);
  const xtermRef = useRef<XTerm | null>(null);
  const fitRef = useRef<FitAddon | null>(null);
  const socketRef = useRef<TerminalSocket | null>(null);
  const retryRef = useRef<(() => void) | null>(null);
  const onStateChangeRef = useRef(onStateChange);
  onStateChangeRef.current = onStateChange;
  const [connection, setConnection] = useState<"connecting" | "connected" | "reconnecting" | "disconnected">("connecting");
  const [role, setRole] = useState<TerminalRole>("observer");
  const [pending, setPending] = useState<PendingTermination | null>(null);
  const [confirm, setConfirm] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [historyVisible, setHistoryVisible] = useState(false);
  const [historyBytes, setHistoryBytes] = useState<Uint8Array | null>(null);
  const [historyLines, setHistoryLines] = useState(0);
  const [historyError, setHistoryError] = useState<string | null>(null);
  const [historyLoading, setHistoryLoading] = useState(false);
  const [historyRevision, setHistoryRevision] = useState(0);
  const [externalLink, setExternalLink] = useState<string | null>(null);
  const [now, setNow] = useState(Date.now());
  const [ctrl, setCtrl] = useState(false);
  const [alt, setAlt] = useState(false);
  const ctrlRef = useRef(false);
  const altRef = useRef(false);

  useEffect(() => { if (closeRequested) { setConfirm(true); onCloseRequestHandled(); } }, [closeRequested, onCloseRequestHandled]);
  useEffect(() => {
    if (!pending) return;
    const timer = window.setInterval(() => setNow(Date.now()), 250);
    return () => window.clearInterval(timer);
  }, [pending]);

  useEffect(() => {
    const host = hostRef.current;
    if (!host) return;
    let active = true;
    let retryTimer: number | undefined;
    let attempts = 0;
    let queuedBytes = 0;
    let streamVersion = 0;
    const probe = new AbortController();
    const xterm = new XTerm({ scrollback: 0, convertEol: false, fontSize: 13,
      fontFamily: "ui-monospace, SFMono-Regular, Menlo, Consolas, monospace", theme: colors(host), disableStdin: true });
    const fit = new FitAddon();
    const links = new WebLinksAddon((event, uri) => {
      try {
        const parsed = new URL(uri);
        if (parsed.protocol !== "http:" && parsed.protocol !== "https:") return;
        if (window.matchMedia("(max-width: 760px)").matches) setExternalLink(parsed.href);
        else if (event.ctrlKey) openURL(parsed.href);
      } catch { /* Ignore malformed terminal output. */ }
    });
    xterm.loadAddon(fit);
    xterm.loadAddon(links);
    xterm.open(host);
    xtermRef.current = xterm;
    fitRef.current = fit;
    const resize = () => {
      if (host.clientWidth < 20 || host.clientHeight < 20) return;
      fit.fit();
      socketRef.current?.sendControl("resize", { cols: xterm.cols, rows: xterm.rows });
    };
    const observer = new ResizeObserver(resize);
    observer.observe(host);
    void document.fonts.ready.then(() => { if (active) resize(); });
    const themeObserver = new MutationObserver(() => { xterm.options.theme = colors(host); });
    themeObserver.observe(document.documentElement, { attributes: true, attributeFilter: ["class"] });
    const applyInput = (value: string) => {
      let bytes = encoder.encode(value);
      if (value.length === 1 && ctrlRef.current) {
        const letter = value.toUpperCase().charCodeAt(0);
        if (letter >= 64 && letter <= 95) bytes = new Uint8Array([letter & 31]);
        ctrlRef.current = false; setCtrl(false);
      }
      if (altRef.current) {
        const prefixed = new Uint8Array(bytes.length + 1);
        prefixed[0] = 27; prefixed.set(bytes, 1); bytes = prefixed;
        altRef.current = false; setAlt(false);
      }
      if (!socketRef.current?.sendInput(bytes)) setError("当前端没有控制权，或连接暂时不可用。")
    };
    const input = xterm.onData(applyInput);
    const binary = xterm.onBinary(value => {
      const bytes = Uint8Array.from(value, char => char.charCodeAt(0) & 255);
      if (!socketRef.current?.sendInput(bytes)) setError("当前端没有控制权，或连接暂时不可用。")
    });
    const connect = () => {
      if (!active) return;
      setConnection(attempts === 0 ? "connecting" : "reconnecting");
      const socket = new TerminalSocket(terminal.id, {
        event(event: TerminalEvent) {
          if (!active) return;
          if (event.type === "ready") {
            streamVersion++;
            queuedBytes = 0;
            xterm.reset();
            xterm.options.disableStdin = event.role !== "controller";
            setRole(event.role);
            setPending(event.pending_termination);
            setConnection("connected");
            setError(null);
            attempts = 0;
            window.setTimeout(resize, 0);
          } else if (event.type === "control") {
            xterm.options.disableStdin = event.role !== "controller";
            setRole(event.role);
            if (event.role !== "controller") { ctrlRef.current = false; altRef.current = false; setCtrl(false); setAlt(false); }
            else resize();
          } else if (event.type === "termination_pending") {
            setPending({ request_id: event.request_id, deadline: event.deadline });
            setNow(Date.now());
          } else if (event.type === "termination_cancelled") {
            setPending(current => current?.request_id === event.request_id ? null : current);
            setConfirm(false);
          } else if (event.type === "termination_executed") {
            setPending(null);
            setConfirm(false);
            onStateChangeRef.current();
          } else if (event.type === "error") setError(event.message);
        },
        output(bytes) {
          if (!active) return;
          // A stalled renderer reconnects from tmux instead of retaining unbounded output.
          if (queuedBytes + bytes.length > 1 << 20) { socket.reconnect(); return; }
          const version = streamVersion;
          queuedBytes += bytes.length;
          xterm.write(bytes, () => { if (version === streamVersion) queuedBytes -= bytes.length; });
        },
        closed(code) {
          if (!active) return;
          socketRef.current = null;
          setConnection("disconnected");
          xterm.options.disableStdin = true;
          setRole("observer");
          if (code === 1008) { setError("终端授权已失效，请重新登录。"); expire(); return; }
          if (code === 1013) onStateChangeRef.current();
          const check = new AbortController();
          probe.signal.addEventListener("abort", () => check.abort(), { once: true });
          void api.session(check.signal).then(() => {
            if (!active) return;
            if (attempts >= 5) { setError("终端连接未能恢复，请手动重试。"); onStateChangeRef.current(); return; }
            attempts++;
            const backoff = Math.min(8000, 500 * 2 ** (attempts - 1));
            retryTimer = window.setTimeout(connect, backoff + Math.floor(Math.random() * 250));
          }).catch((reason: unknown) => {
            if (!active || check.signal.aborted) return;
            if (reason instanceof ApiError && reason.status === 401) { expire(); setError("登录已失效。"); }
            else if (attempts < 5) { attempts++; retryTimer = window.setTimeout(connect, Math.min(8000, 500 * 2 ** (attempts - 1))); }
            else setError(errorMessage(reason));
          });
        },
      });
      socketRef.current = socket;
    };
    retryRef.current = () => {
      if (retryTimer !== undefined) window.clearTimeout(retryTimer);
      socketRef.current?.dispose();
      attempts = 0;
      connect();
    };
    queueMicrotask(connect);
    return () => {
      active = false;
      probe.abort();
      if (retryTimer !== undefined) window.clearTimeout(retryTimer);
      socketRef.current?.dispose();
      socketRef.current = null;
      observer.disconnect();
      themeObserver.disconnect();
      input.dispose(); binary.dispose(); links.dispose(); fit.dispose(); xterm.dispose();
      xtermRef.current = null; fitRef.current = null;
      retryRef.current = null;
    };
  }, [terminal.id, expire]);

  useEffect(() => {
    if (!historyVisible) return;
    const controller = new AbortController();
    setHistoryLoading(true);
    setHistoryError(null);
    api.terminalHistory(terminal.id, controller.signal).then(snapshot => {
      if (controller.signal.aborted) return;
      const binary = atob(snapshot.content_base64);
      setHistoryBytes(Uint8Array.from(binary, char => char.charCodeAt(0)));
      setHistoryLines(snapshot.returned_lines);
      setHistoryLoading(false);
    }).catch((reason: unknown) => {
      if (!controller.signal.aborted) { setHistoryError(errorMessage(reason)); setHistoryLoading(false); }
    });
    return () => controller.abort();
  }, [terminal.id, historyVisible, historyRevision]);

  useEffect(() => {
    if (!historyVisible || !historyBytes || !historyHostRef.current) return;
    const host = historyHostRef.current;
    const terminal = new XTerm({ scrollback: historyLines, fontSize: 13,
      fontFamily: "ui-monospace, SFMono-Regular, Menlo, Consolas, monospace", theme: colors(host), disableStdin: true });
    const fit = new FitAddon();
    terminal.loadAddon(fit);
    terminal.open(host);
    const observer = new ResizeObserver(() => { if (host.clientWidth > 20 && host.clientHeight > 20) fit.fit(); });
    observer.observe(host);
    fit.fit();
    terminal.write(historyBytes, () => terminal.scrollToBottom());
    return () => { observer.disconnect(); fit.dispose(); terminal.dispose(); };
  }, [historyVisible, historyBytes, historyLines]);

  const seconds = pending ? Math.max(0, Math.ceil((Date.parse(pending.deadline) - now) / 1000)) : 0;
  const controlled = connection === "connected" && role === "controller";
  function shortcut(value: string) {
    if (!controlled || !socketRef.current?.sendInput(encoder.encode(value))) setError("请先接管终端，或等待连接恢复。")
    xtermRef.current?.focus();
  }
  function toggleCtrl(pressed: boolean) { ctrlRef.current = pressed; setCtrl(pressed); xtermRef.current?.focus(); }
  function toggleAlt(pressed: boolean) { altRef.current = pressed; setAlt(pressed); xtermRef.current?.focus(); }
  function requestTermination() {
    if (!socketRef.current?.sendControl("terminate")) setError("请先接管终端，再终止会话。");
    else setConfirm(false);
  }
  function cancelTermination() {
    if (pending && !socketRef.current?.sendControl("cancel_termination", { request_id: pending.request_id })) setError("取消请求未发送，请检查连接。")
  }

  return <div className="terminal-runtime" data-terminal-id={terminal.id}>
    <div className="terminal-runtime-toolbar">
      <span className={`terminal-connection ${connection}`}><Radio />{connectionLabel(connection)}</span>
      <span className="terminal-role">{role === "controller" ? "控制中" : "只读观察"}</span>
      {!controlled && connection === "connected" && <Button size="sm" variant="outline" onClick={() => socketRef.current?.sendControl("takeover")}>接管</Button>}
      {connection === "disconnected" && <Button size="icon-sm" variant="ghost" aria-label="重试连接" title="重试连接" onClick={() => retryRef.current?.()}><RotateCw /></Button>}
      <span className="terminal-heading-spacer" />
      <Button size="icon-sm" variant={historyVisible ? "secondary" : "ghost"} aria-label={historyVisible ? "返回实时终端" : "查看终端历史"}
        title={historyVisible ? "返回实时终端" : "查看终端历史"} onClick={() => setHistoryVisible(value => !value)}>{historyVisible ? <Radio /> : <ScrollText />}</Button>
      {historyVisible && <Button size="icon-sm" variant="ghost" aria-label="刷新终端历史" title="刷新终端历史" onClick={() => setHistoryRevision(value => value + 1)}><RotateCw /></Button>}
      <Button size="icon-sm" variant="ghost" aria-label="终止当前终端" title="终止当前终端" disabled={!controlled} onClick={() => setConfirm(true)}><Square /></Button>
    </div>
    {error && <Alert variant="destructive" className="shrink-0"><AlertDescription>{error}</AlertDescription><AlertAction><Button size="icon-xs" variant="ghost" aria-label="关闭错误" onClick={() => setError(null)}><X /></Button></AlertAction></Alert>}
    <div className="terminal-live" ref={hostRef} style={{ display: historyVisible ? "none" : undefined }} aria-label="实时终端" />
    {historyVisible && <div className="terminal-history"><div className="terminal-history-heading"><History />普通历史快照{historyLoading && <span>读取中…</span>}{historyError && <span role="alert">{historyError}</span>}</div><div className="terminal-history-surface" ref={historyHostRef} aria-label="终端历史" /></div>}
    <div className="terminal-shortcuts" aria-label="手机终端快捷键">
      <Toggle size="sm" variant="outline" disabled={!controlled} pressed={ctrl} onPressedChange={toggleCtrl}>Ctrl</Toggle>
      <Toggle size="sm" variant="outline" disabled={!controlled} pressed={alt} onPressedChange={toggleAlt}>Alt</Toggle>
      <Button size="sm" variant="outline" disabled={!controlled} onClick={() => shortcut("\x1b")}>Esc</Button>
      <Button size="sm" variant="outline" disabled={!controlled} onClick={() => shortcut("\t")}>Tab</Button>
      <Button size="icon-sm" variant="outline" disabled={!controlled} aria-label="上箭头" onClick={() => shortcut("\x1b[A")}><ArrowUp /></Button>
      <Button size="icon-sm" variant="outline" disabled={!controlled} aria-label="下箭头" onClick={() => shortcut("\x1b[B")}><ArrowDown /></Button>
      <Button size="icon-sm" variant="outline" disabled={!controlled} aria-label="左箭头" onClick={() => shortcut("\x1b[D")}><ArrowLeft /></Button>
      <Button size="icon-sm" variant="outline" disabled={!controlled} aria-label="右箭头" onClick={() => shortcut("\x1b[C")}><ArrowRight /></Button>
      {([ ["^C", "\x03"], ["^L", "\x0c"], ["^S", "\x13"], ["^Z", "\x1a"], ["/", "/"] ] as const).map(([label, value]) => <Button key={label} size="sm" variant="outline" disabled={!controlled} onClick={() => shortcut(value)}>{label}</Button>)}
    </div>
    <Dialog open={confirm || pending !== null} onOpenChange={open => { if (!open && !pending) setConfirm(false); }}>
      <DialogContent showCloseButton={false}>
        <DialogHeader><DialogTitle>{pending ? "终止倒计时" : "终止终端？"}</DialogTitle><DialogDescription>
          {pending ? `将在 ${seconds} 秒后终止此终端及其中运行的程序。任一查看端可取消。` : "关闭这个终端会终止其中正在运行的程序。"}
        </DialogDescription></DialogHeader>
        <DialogFooter>{pending ? <Button variant="outline" onClick={cancelTermination}>取消终止</Button> : <>
          <Button variant="outline" onClick={() => setConfirm(false)} autoFocus>取消</Button>
          <Button variant="destructive" disabled={!controlled} onClick={requestTermination}>开始倒计时</Button>
        </>}</DialogFooter>
      </DialogContent>
    </Dialog>
    <Dialog open={externalLink !== null} onOpenChange={open => { if (!open) setExternalLink(null); }}>
      <DialogContent><DialogHeader><DialogTitle>打开外部链接？</DialogTitle><DialogDescription className="break-all">{externalLink}</DialogDescription></DialogHeader>
        <DialogFooter><Button variant="outline" onClick={() => setExternalLink(null)}>取消</Button><Button onClick={() => { if (externalLink) openURL(externalLink); setExternalLink(null); }}>打开</Button></DialogFooter>
      </DialogContent>
    </Dialog>
  </div>;
}

function connectionLabel(state: "connecting" | "connected" | "reconnecting" | "disconnected"): string {
  switch (state) {
    case "connecting": return "连接中";
    case "connected": return "已连接";
    case "reconnecting": return "重连中";
    case "disconnected": return "已断开";
  }
}
