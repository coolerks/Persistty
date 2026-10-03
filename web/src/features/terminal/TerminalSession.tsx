import { useCallback, useEffect, useRef, useState } from "react";
import { Terminal as XTerm } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { WebLinksAddon } from "@xterm/addon-web-links";
import { ArrowDown, ArrowLeft, ArrowRight, ArrowUp, History, X } from "lucide-react";
import "@xterm/xterm/css/xterm.css";
import { Button } from "@/components/ui/button";
import { Toggle } from "@/components/ui/toggle";
import { Alert, AlertAction, AlertDescription } from "@/components/ui/alert";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { api, ApiError, errorMessage } from "@/lib/api/client";
import type { Terminal } from "@/lib/api/decoder";
import { useAuth } from "@/features/auth/auth-context";
import { type TerminalEvent, TerminalSocket, type TerminalRole } from "@/lib/ws/terminal";
import { useTerminalRuntime, type RuntimeState } from "./runtime-context";
import { isVerticalWheel, wheelPixels } from "./terminal-scrolling";

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

export function TerminalSessionView({ terminal, onStateChange }: {
  terminal: Terminal; onStateChange(): void;
}) {
  const { scope } = useTerminalRuntime(terminal.id);
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
  const runtimeRef = useRef<RuntimeState>({ connection: "connecting", role: "observer", viewerId: "", generation: 0, history: false });
  const takeoverRef = useRef<{ resolve(): void; reject(reason: Error): void } | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [historyVisible, setHistoryVisible] = useState(false);
  const [historyBytes, setHistoryBytes] = useState<Uint8Array | null>(null);
  const [renderedHistory, setRenderedHistory] = useState<Uint8Array | null>(null);
  const [historyLines, setHistoryLines] = useState(0);
  const [historyError, setHistoryError] = useState<string | null>(null);
  const [historyLoading, setHistoryLoading] = useState(false);
  const [historyEmpty, setHistoryEmpty] = useState(false);
  const [historyRevision, setHistoryRevision] = useState(0);
  const historyWheelRef = useRef(0);
  const historyModeRef = useRef(false);
  const historyScrollRef = useRef<((pixels: number) => void) | null>(null);
  const setHistoryMode = useCallback((visible: boolean) => {
    historyModeRef.current = visible;
    setHistoryVisible(visible);
  }, []);
  const openHistory = useCallback((pixels = 0) => {
    if (!historyModeRef.current) {
      historyWheelRef.current = pixels;
      setHistoryEmpty(false);
      setHistoryBytes(null);
      setHistoryMode(true);
    } else if (historyScrollRef.current) historyScrollRef.current(pixels);
    else historyWheelRef.current += pixels;
  }, [setHistoryMode]);
  const [externalLink, setExternalLink] = useState<string | null>(null);
  const [ctrl, setCtrl] = useState(false);
  const [alt, setAlt] = useState(false);
  const ctrlRef = useRef(false);
  const altRef = useRef(false);

  useEffect(() => {
    const state = { ...runtimeRef.current, connection, role, history: historyVisible };
    runtimeRef.current = state;
    scope.report(terminal.id, state);
  }, [connection, role, historyVisible, scope, terminal.id]);
  useEffect(() => scope.register(terminal.id, {
    takeover: () => {
      if (socketRef.current?.canControl()) return Promise.resolve();
      if (takeoverRef.current) return Promise.reject(new Error("已有接管请求，请等待。"));
      return new Promise<void>((resolve, reject) => {
        const timer = window.setTimeout(() => { takeoverRef.current = null; reject(new Error("接管未获服务端确认。")); }, 5000);
        takeoverRef.current = { resolve() { window.clearTimeout(timer); takeoverRef.current = null; resolve(); }, reject(reason) { window.clearTimeout(timer); takeoverRef.current = null; reject(reason); } };
        if (!socketRef.current?.sendControl("takeover")) takeoverRef.current.reject(new Error("连接未就绪，无法接管。"));
      });
    },
    target() {
      const state = runtimeRef.current;
      if (!socketRef.current?.canControl()) throw new Error("控制权已变化，未启动倒计时。");
      return { terminal_id: terminal.id, viewer_id: state.viewerId, generation: state.generation };
    },
    cancel: id => socketRef.current?.sendControl("cancel_termination", { request_id: id }) ?? false,
    retry: () => retryRef.current?.(),
    history: () => { if (historyModeRef.current) setHistoryMode(false); else openHistory(); },
    refreshHistory: () => setHistoryRevision(value => value + 1),
    focus: () => xtermRef.current?.focus(),
  }), [scope, terminal.id, openHistory, setHistoryMode]);

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
    let viewportHeight = host.clientHeight;
    // Queries originate from this viewer's read-only attach, not the input owner.
    // Keep automatic DA replies out of onData and the shell keyboard channel.
    const deviceAttributes = (["primary", "secondary"] as const).map(kind =>
      xterm.parser.registerCsiHandler({ ...(kind === "secondary" ? { prefix: ">" } : {}), final: "c" }, params => {
        if (params[0] === 0) socketRef.current?.sendDeviceAttributes(kind);
        return true;
      }));
    const wheel = (event: WheelEvent) => {
      if (event.ctrlKey) { event.stopImmediatePropagation(); return; }
      if (!isVerticalWheel(event.deltaX, event.deltaY)) { event.preventDefault(); event.stopImmediatePropagation(); return; }
      // tmux's outer alternate buffer is not evidence of a TUI in the pane.
      // Preserve explicit application mouse tracking only for the controller.
      if (!historyModeRef.current && !event.shiftKey && socketRef.current?.canControl() && xterm.modes.mouseTrackingMode !== "none") return;
      event.preventDefault(); event.stopImmediatePropagation();
      viewportHeight = host.clientHeight || viewportHeight;
      const screenHeight = Number.parseFloat(host.querySelector<HTMLElement>(".xterm-screen")?.style.height ?? "") || xterm.rows * (xterm.options.fontSize ?? 13);
      if (event.deltaY < 0 || historyModeRef.current) openHistory(wheelPixels(event.deltaY, event.deltaMode, screenHeight / xterm.rows, viewportHeight || screenHeight));
    };
    host.addEventListener("wheel", wheel, { capture: true, passive: false });
    xterm.attachCustomKeyEventHandler(event => !(event.type === "keydown" && (event.ctrlKey || event.metaKey) && event.key.toLowerCase() === "c" && xterm.hasSelection()));
    const inputIntent = (event: Event) => {
      if (historyModeRef.current) return;
      if (socketRef.current?.canControl() || runtimeRef.current.connection !== "connected") return;
      if (event instanceof KeyboardEvent) {
        if (event.type !== "keydown" || ["Shift", "Control", "Alt", "Meta", "CapsLock"].includes(event.key)) return;
        if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === "c" && xterm.hasSelection()) return;
        if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === "v") return;
        if (event.shiftKey && ["PageUp", "PageDown", "Insert"].includes(event.key)) return;
      }
      event.preventDefault(); event.stopPropagation();
      scope.inputIntent(terminal.id);
    };
    for (const type of ["keydown", "beforeinput", "paste", "compositionstart"]) host.addEventListener(type, inputIntent, true);
    xtermRef.current = xterm;
    fitRef.current = fit;
    const resize = () => {
      if (host.clientWidth < 20 || host.clientHeight < 20) return;
      viewportHeight = host.clientHeight;
      if (!socketRef.current?.canControl()) return;
      fit.fit();
      socketRef.current?.sendControl("resize", { cols: xterm.cols, rows: xterm.rows });
    };
    const observer = new ResizeObserver(resize);
    observer.observe(host);
    void document.fonts.load('13px "Persistty Nerd Mono"').then(() => { if (!active) return; xterm.options.fontFamily = "Persistty Nerd Mono, ui-monospace, SFMono-Regular, Menlo, Consolas, monospace"; requestAnimationFrame(() => { if (active) resize(); }); }).catch(() => { /* Keep the available fallback font. */ });
    const themeObserver = new MutationObserver(() => { xterm.options.theme = colors(host); });
    themeObserver.observe(document.documentElement, { attributes: true, attributeFilter: ["class"] });
    const applyInput = (value: string) => {
      if (historyModeRef.current) return;
      if (!socketRef.current?.canControl()) {
        ctrlRef.current = false; altRef.current = false; setCtrl(false); setAlt(false);
        return;
      }
      setHistoryEmpty(false);
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
      if (historyModeRef.current) return;
      if (!socketRef.current?.canControl()) return;
      const bytes = Uint8Array.from(value, char => char.charCodeAt(0) & 255);
      if (!socketRef.current?.sendInput(bytes)) setError("当前端没有控制权，或连接暂时不可用。")
    });
    const connect = () => {
      if (!active) return;
      setConnection(attempts === 0 ? "connecting" : "reconnecting");
      const socket = new TerminalSocket(terminal.id, {
        event(event: TerminalEvent) {
          if (!active) return;
          scope.event(terminal.id, event);
          if (event.type === "ready") {
            streamVersion++;
            queuedBytes = 0;
            xterm.reset();
            xterm.options.disableStdin = event.role !== "controller";
            runtimeRef.current = { ...runtimeRef.current, connection: "connected", role: event.role, viewerId: event.viewer_id, generation: event.generation };
            scope.report(terminal.id, runtimeRef.current);
            setRole(event.role);
            setConnection("connected");
            if (event.role === "observer") xterm.resize(event.cols, event.rows);
            setError(null);
            attempts = 0;
            window.setTimeout(resize, 0);
          } else if (event.type === "control") {
            xterm.options.disableStdin = event.role !== "controller";
            runtimeRef.current = { ...runtimeRef.current, role: event.role, generation: event.generation };
            scope.report(terminal.id, runtimeRef.current);
            setRole(event.role);
            if (event.role === "controller") takeoverRef.current?.resolve();
            if (event.role !== "controller") { ctrlRef.current = false; altRef.current = false; setCtrl(false); setAlt(false); }
            else resize();
          } else if (event.type === "resized") {
            if (!socketRef.current?.canControl()) xterm.resize(event.cols, event.rows);
          } else if (event.type === "error") { setError(event.message); takeoverRef.current?.reject(new Error(event.message)); }
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
          runtimeRef.current = { ...runtimeRef.current, connection: "disconnected", role: "observer" };
          scope.report(terminal.id, runtimeRef.current);
          takeoverRef.current?.reject(new Error("接管期间连接中断。"));
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
      for (const handler of deviceAttributes) handler.dispose();
      input.dispose(); binary.dispose(); links.dispose(); fit.dispose(); xterm.dispose();
      for (const type of ["keydown", "beforeinput", "paste", "compositionstart"]) host.removeEventListener(type, inputIntent, true);
      host.removeEventListener("wheel", wheel, true);
      xtermRef.current = null; fitRef.current = null;
      retryRef.current = null;
      takeoverRef.current?.reject(new Error("终端运行时已关闭。"));
    };
  }, [terminal.id, expire, scope, openHistory]);

  useEffect(() => {
    if (!historyVisible) return;
    const controller = new AbortController();
    setHistoryLoading(true);
    setHistoryError(null);
    api.terminalHistory(terminal.id, controller.signal).then(snapshot => {
      if (controller.signal.aborted) return;
      const binary = atob(snapshot.content_base64);
      // capture-pane can return a blank line when tmux has no scrollback.
      // An empty byte array is still truthy; do not replace the live screen with it.
      if (snapshot.history_size === 0 || snapshot.returned_lines === 0 || binary.trim().length === 0) {
        historyWheelRef.current = 0;
        setHistoryBytes(null); setHistoryEmpty(true); setHistoryLoading(false); setHistoryMode(false);
        return;
      }
      setHistoryBytes(Uint8Array.from(binary, char => char.charCodeAt(0)));
      setHistoryLines(snapshot.returned_lines);
      setHistoryLoading(false);
    }).catch((reason: unknown) => {
      if (!controller.signal.aborted) { setHistoryError(errorMessage(reason)); setHistoryLoading(false); }
    });
    return () => controller.abort();
  }, [terminal.id, historyVisible, historyRevision, setHistoryMode]);

  useEffect(() => {
    if (!historyVisible || !historyBytes || !historyHostRef.current) return;
    const host = historyHostRef.current;
    const terminal = new XTerm({ scrollback: historyLines, convertEol: true, fontSize: 13,
      fontFamily: "ui-monospace, SFMono-Regular, Menlo, Consolas, monospace", theme: colors(host), disableStdin: true });
    const fit = new FitAddon();
    terminal.loadAddon(fit);
    terminal.open(host);
    const observer = new ResizeObserver(() => { if (host.clientWidth > 20 && host.clientHeight > 20) fit.fit(); });
    observer.observe(host);
    fit.fit();
    let active = true;
    void document.fonts.load('13px "Persistty Nerd Mono"').then(() => { if (!active) return; terminal.options.fontFamily = "Persistty Nerd Mono, ui-monospace, SFMono-Regular, Menlo, Consolas, monospace"; requestAnimationFrame(() => { if (active && host.clientWidth > 20 && host.clientHeight > 20) fit.fit(); }); }).catch(() => { /* Keep the available fallback font. */ });
    let parsed = false;
    let bottomPixels = 0;
    let scrollFrame = 0;
    // Re-target momentum latched to live into this independent native viewport.
    const scroll = (pixels: number) => {
      if (!pixels) return;
      const event = new WheelEvent("wheel", { deltaY: pixels, bubbles: true, cancelable: true });
      // Chromium gives synthetic events zero legacy wheelDelta fields. xterm
      // prefers those over deltaY; remove them on this forwarded event only.
      Object.defineProperties(event, { wheelDeltaY: { value: undefined }, wheelDeltaX: { value: undefined }, wheelDelta: { value: undefined } });
      terminal.element?.querySelector(".xterm-scrollable-element")?.dispatchEvent(event);
    };
    historyScrollRef.current = scroll;
    terminal.write(historyBytes, () => {
      if (!active) return;
      terminal.scrollToBottom();
      // write parsing finishes before Viewport's queued render/dimension sync.
      // Apply the initial gesture only after that sync, otherwise it is reset.
      scrollFrame = requestAnimationFrame(() => {
        scrollFrame = requestAnimationFrame(() => {
          if (!active) return;
          terminal.options.smoothScrollDuration = 120;
          parsed = true;
          setRenderedHistory(historyBytes);
          const pixels = historyWheelRef.current;
          historyWheelRef.current = 0;
          scroll(pixels);
        });
      });
    });
    const wheel = (event: WheelEvent) => {
      if (event.ctrlKey) { event.stopImmediatePropagation(); return; }
      if (!isVerticalWheel(event.deltaX, event.deltaY)) { event.preventDefault(); event.stopImmediatePropagation(); return; }
      const lineHeight = Number.parseFloat(host.querySelector<HTMLElement>(".xterm-screen")?.style.height ?? "") / terminal.rows || (terminal.options.fontSize ?? 13);
      const pixels = wheelPixels(event.deltaY, event.deltaMode, lineHeight, host.clientHeight);
      if (!parsed) { historyWheelRef.current += pixels; event.preventDefault(); event.stopImmediatePropagation(); return; }
      if (pixels > 0 && terminal.buffer.active.viewportY === terminal.buffer.active.baseY) {
        bottomPixels += pixels;
        event.preventDefault(); event.stopImmediatePropagation();
        if (bottomPixels >= lineHeight) setHistoryMode(false);
      } else bottomPixels = 0;
      // Native xterm viewport retains fractional position and trackpad inertia.
    };
    host.addEventListener("wheel", wheel, { capture: true, passive: false });
    const themeObserver = new MutationObserver(() => { terminal.options.theme = colors(host); });
    themeObserver.observe(document.documentElement, { attributes: true, attributeFilter: ["class"] });
    return () => { active = false; cancelAnimationFrame(scrollFrame); historyScrollRef.current = null; observer.disconnect(); themeObserver.disconnect(); host.removeEventListener("wheel", wheel, true); fit.dispose(); terminal.dispose(); };
  }, [historyVisible, historyBytes, historyLines, setHistoryMode]);

  const controlled = connection === "connected" && role === "controller";
  function shortcut(value: string) {
    if (historyModeRef.current) return;
    if (!controlled) { if (connection === "connected") scope.inputIntent(terminal.id); else setError("请等待连接恢复。"); }
    else if (!socketRef.current?.sendInput(encoder.encode(value))) setError("输入未发送，请检查控制权和连接。")
    xtermRef.current?.focus();
  }
  function toggleCtrl(pressed: boolean) { if (historyModeRef.current) return; if (!controlled) { scope.inputIntent(terminal.id); return; } ctrlRef.current = pressed; setCtrl(pressed); xtermRef.current?.focus(); }
  function toggleAlt(pressed: boolean) { if (historyModeRef.current) return; if (!controlled) { scope.inputIntent(terminal.id); return; } altRef.current = pressed; setAlt(pressed); xtermRef.current?.focus(); }

  return <div className="terminal-runtime" data-terminal-id={terminal.id} data-role={role} data-connection={connection}>
    {error && <Alert variant="destructive" className="shrink-0"><AlertDescription>{error}</AlertDescription><AlertAction><Button size="icon-xs" variant="ghost" aria-label="关闭错误" onClick={() => setError(null)}><X /></Button></AlertAction></Alert>}
    {historyEmpty && !historyVisible && <div className="terminal-history-heading terminal-history-loading terminal-history-empty" role="status"><History /><span>暂无历史输出</span></div>}
    {historyVisible && !historyBytes && <div className="terminal-history-heading terminal-history-loading" role="status"><History />普通历史快照{historyLoading && <span>读取中…</span>}{historyError && <span role="alert">{historyError}</span>}</div>}
    <div className="terminal-screen-stack">
    <div className="terminal-live" ref={hostRef} style={{ visibility: historyVisible && historyBytes === renderedHistory && historyBytes ? "hidden" : undefined }} aria-label="实时终端" />
    {historyVisible && historyBytes && <div className="terminal-history" style={{ visibility: historyBytes === renderedHistory ? undefined : "hidden" }}><div className="terminal-history-heading"><History />普通历史快照{historyLines === 0 && <span>暂无历史输出</span>}{historyLoading && <span>读取中…</span>}{historyError && <span role="alert">{historyError}</span>}</div><div className="terminal-history-surface" ref={historyHostRef} aria-label="终端历史" /></div>}
    </div>
    <div className="terminal-shortcuts" aria-label="手机终端快捷键">
      <Toggle size="sm" variant="outline" pressed={ctrl} onPressedChange={toggleCtrl}>Ctrl</Toggle>
      <Toggle size="sm" variant="outline" pressed={alt} onPressedChange={toggleAlt}>Alt</Toggle>
      <Button size="sm" variant="outline" onClick={() => shortcut("\x1b")}>Esc</Button>
      <Button size="sm" variant="outline" onClick={() => shortcut("\t")}>Tab</Button>
      <Button size="icon-sm" variant="outline" aria-label="上箭头" onClick={() => shortcut("\x1b[A")}><ArrowUp /></Button>
      <Button size="icon-sm" variant="outline" aria-label="下箭头" onClick={() => shortcut("\x1b[B")}><ArrowDown /></Button>
      <Button size="icon-sm" variant="outline" aria-label="左箭头" onClick={() => shortcut("\x1b[D")}><ArrowLeft /></Button>
      <Button size="icon-sm" variant="outline" aria-label="右箭头" onClick={() => shortcut("\x1b[C")}><ArrowRight /></Button>
      {([ ["^C", "\x03"], ["^L", "\x0c"], ["^S", "\x13"], ["^Z", "\x1a"], ["/", "/"] ] as const).map(([label, value]) => <Button key={label} size="sm" variant="outline" onClick={() => shortcut(value)}>{label}</Button>)}
    </div>
    <Dialog open={externalLink !== null} onOpenChange={open => { if (!open) setExternalLink(null); }}>
      <DialogContent><DialogHeader><DialogTitle>打开外部链接？</DialogTitle><DialogDescription className="break-all">{externalLink}</DialogDescription></DialogHeader>
        <DialogFooter><Button variant="outline" onClick={() => setExternalLink(null)}>取消</Button><Button onClick={() => { if (externalLink) openURL(externalLink); setExternalLink(null); }}>打开</Button></DialogFooter>
      </DialogContent>
    </Dialog>
  </div>;
}
