import { Eye, History, MoreHorizontal, PanelBottom, PanelTop, Radio, RefreshCw, RotateCw, TerminalSquare, X } from "lucide-react";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { TabsTrigger } from "@/components/ui/tabs";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { ContextMenu, ContextMenuContent, ContextMenuGroup, ContextMenuItem, ContextMenuTrigger } from "@/components/ui/context-menu";
import { DropdownMenu, DropdownMenuContent, DropdownMenuGroup, DropdownMenuItem, DropdownMenuLabel, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import type { Terminal } from "@/lib/api/decoder";
import { useTerminalRuntime } from "./runtime-context";

export function TerminalTab({ terminal, region, value = terminal.id, active, onActivate, onRefresh, onMove, position = "bottom", allowBatch = true }: {
  terminal: Terminal; region: Terminal[]; value?: string; active: boolean; onActivate(): void; onRefresh(): void;
  onMove?: ((terminal: Terminal) => void) | undefined; position?: "top" | "bottom"; allowBatch?: boolean;
}) {
  const { scope, entry } = useTerminalRuntime(terminal.id);
  const item = entry?.terminal ?? terminal;
  const controlled = entry?.state.connection === "connected" && entry.state.role === "controller";
  const running = item.state === "running";
  function close(items: Terminal[]) { scope.close(items.filter(item => (scope.entry(item.id)?.terminal ?? item).state === "running").map(item => scope.entry(item.id)?.terminal ?? item)); }
  const actions = [
    { label: "接管", icon: Eye, disabled: !running || controlled, run: () => scope.takeover(item) },
    { label: "刷新终端", icon: RefreshCw, run: onRefresh },
    { label: "重命名", icon: TerminalSquare, run: () => scope.rename(item) },
    { label: entry?.state.history ? "返回实时终端" : "查看终端历史", icon: entry?.state.history ? Radio : History, disabled: !entry?.actions, run: () => entry?.actions?.history() },
    ...(entry?.state.history ? [{ label: "刷新终端历史", icon: RotateCw, run: () => entry.actions?.refreshHistory() }] : []),
    ...(entry?.state.connection === "disconnected" ? [{ label: "重试连接", icon: RotateCw, run: () => entry.actions?.retry() }] : []),
    ...(onMove ? [{ label: position === "top" ? "移回下方终端面板" : "移到上方标签", icon: position === "top" ? PanelBottom : PanelTop, run: () => onMove(item) }] : []),
    { label: "关闭当前", icon: X, destructive: true, disabled: !running, run: () => close([item]) },
    ...(allowBatch ? [
      { label: "关闭其他", icon: X, destructive: true, disabled: !region.some(other => other.id !== item.id && other.state === "running"), run: () => close(region.filter(other => other.id !== item.id)) },
      { label: "全部关闭", icon: X, destructive: true, disabled: !region.some(other => other.state === "running"), run: () => close(region) },
    ] : []),
  ];
  const connection = entry ? ({ connecting: "连接中", connected: "已连接", reconnecting: "重连中", disconnected: "已断开" })[entry.state.connection] : "未连接";
  const description = `${item.display_name} · ${item.state === "running" ? controlled ? "控制中" : "未接管" : item.state === "terminated" ? "已结束" : "不可用"} · ${connection} · 初始目录：${item.working_directory}`;
  return <ContextMenu><ContextMenuTrigger render={<div className={cn("terminal-tab", active && "active")} />}
    data-terminal-id={item.id}
    draggable={Boolean(onMove)} onDragStart={event => {
      if (onMove) { event.dataTransfer.setData(position === "top" ? "application/x-persistty-upper-terminal" : "application/x-persistty-lower-terminal", JSON.stringify(position === "top" ? { id: item.id } : item)); event.dataTransfer.effectAllowed = "move"; }
    }}>
    {running && !controlled && <Tooltip><TooltipTrigger render={<Button size="icon-xs" variant="ghost" className="terminal-tab-takeover" aria-label={`接管 ${item.display_name}`} onClick={() => scope.takeover(item)} />}><Eye /></TooltipTrigger><TooltipContent>未接管，点击接管</TooltipContent></Tooltip>}
    <Tooltip><TooltipTrigger render={<TabsTrigger value={value} className="terminal-tab-label" onClick={onActivate} onDoubleClick={() => scope.rename(item)} />}>
      <TerminalSquare data-icon="inline-start" /><span className="truncate">{item.display_name}</span>
      <span className={cn("terminal-state-dot", item.state)} aria-label={description} />
    </TooltipTrigger><TooltipContent>{description}</TooltipContent></Tooltip>
    <Button size="icon-xs" variant="ghost" className="terminal-tab-close" aria-label={`关闭终端 ${item.display_name}`} disabled={!running} onClick={() => close([item])}><X /></Button>
    <DropdownMenu><DropdownMenuTrigger render={<Button size="icon-xs" variant="ghost" className="terminal-tab-more" aria-label={`${item.display_name} 更多`} />}><MoreHorizontal /></DropdownMenuTrigger>
      <DropdownMenuContent className="terminal-menu"><DropdownMenuGroup><DropdownMenuLabel className="break-all">初始目录：{item.working_directory}</DropdownMenuLabel>
        {actions.map(action => <DropdownMenuItem key={action.label} disabled={action.disabled} variant={action.destructive ? "destructive" : "default"} onClick={action.run}><action.icon />{action.label}</DropdownMenuItem>)}
      </DropdownMenuGroup></DropdownMenuContent>
    </DropdownMenu>
  </ContextMenuTrigger><ContextMenuContent><ContextMenuGroup>{actions.map(action => <ContextMenuItem key={action.label} disabled={action.disabled} variant={action.destructive ? "destructive" : "default"} onClick={action.run}><action.icon />{action.label}</ContextMenuItem>)}</ContextMenuGroup></ContextMenuContent></ContextMenu>;
}
