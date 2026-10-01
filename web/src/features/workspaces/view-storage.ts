import { useSyncExternalStore } from "react";
let warning: string | null = null;
const listeners = new Set<() => void>();
const fallback = new Map<string, string>();
function report(message: string): void { if (warning === message) return; warning = message; queueMicrotask(() => { for (const listener of listeners) listener(); }); }
const subscribe = (listener: () => void) => { listeners.add(listener); return () => listeners.delete(listener); };
export function useViewStorageWarning(): string | null { return useSyncExternalStore(subscribe, () => warning); }
export const viewStorage = {
  getItem(key: string): string | null { if (fallback.has(key)) return fallback.get(key) ?? null; try { const raw = localStorage.getItem(key); if (raw && raw.length > 2 * 1024 * 1024) throw new Error(); return raw; } catch { report("工作台记录无法读取，当前视图仍可使用；草稿由独立存储保护。"); return null; } },
  setItem(key: string, value: string): void { try { if (value.length > 2 * 1024 * 1024) throw new Error(); localStorage.setItem(key, value); fallback.delete(key); } catch { fallback.set(key, value); report("工作台视图未能持久保存，下次打开可能恢复默认布局。请保留页面；文件草稿状态单独显示。"); } },
  removeItem(key: string): void { fallback.delete(key); try { localStorage.removeItem(key); } catch { report("无法清理工作台视图记录。"); } },
};
export const panelStorage = {
  getItem(key: string): string | null {
    const raw = viewStorage.getItem(key); if (!raw) return null;
    try { const value: unknown = JSON.parse(raw); if (!value || typeof value !== "object" || Array.isArray(value)) throw new Error(); const sizes = Object.values(value); if (sizes.length < 1 || sizes.length > 4 || !sizes.every(size => typeof size === "number" && Number.isFinite(size) && size >= 0 && size <= 100) || Math.abs(sizes.reduce((sum, size) => sum + Number(size), 0) - 100) > 0.1) throw new Error(); return raw; }
    catch { report("面板尺寸记录无效，已恢复默认尺寸。文件与终端标签仍保留。"); return null; }
  },
  setItem: viewStorage.setItem,
};
