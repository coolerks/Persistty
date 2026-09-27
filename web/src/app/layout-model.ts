export type ContentTab = { kind: "file"; id: string } | { kind: "preview"; id: string } | { kind: "diff"; id: string } | { kind: "terminal"; id: string };
export type HorizontalGroup<T> = { id: string; tabs: T[]; activeId: string | null };
export type DesktopLayout = { schema: 1; mode: "desktop"; orientation: "horizontal"; editorGroups: HorizontalGroup<ContentTab>[]; terminalGroups: HorizontalGroup<Extract<ContentTab, { kind: "terminal" }>>[] };
export type MobileLayout = { schema: 1; mode: "mobile"; active: ContentTab | null };
export type WorkbenchCommand =
  | { type: "collapse_panel"; panel: "sidebar" | "terminal" }
  | { type: "close_file_view"; fileId: string }
  | { type: "move_terminal_view"; terminalId: string; target: "editor" | "terminal"; groupId: string }
  | { type: "request_terminal_termination"; terminalId: string };
export function tabCloseCommand(tab: ContentTab): WorkbenchCommand {
  return tab.kind === "terminal" ? { type: "request_terminal_termination", terminalId: tab.id } : { type: "close_file_view", fileId: tab.id };
}
export function acceptsTab(area: "editor" | "terminal", tab: ContentTab): boolean {
  return area === "editor" || tab.kind === "terminal";
}
export function layoutStorageKey(projectId: string, mode: "desktop" | "mobile") {
  return `persistty.layout.v1.${mode}.${encodeURIComponent(projectId)}`;
}
