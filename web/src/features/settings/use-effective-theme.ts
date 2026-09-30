import { useSyncExternalStore } from "react";

// One observer for all subscribers, including large file trees.
const listeners = new Set<() => void>();
let observer: MutationObserver | undefined;
function subscribe(listener: () => void) {
  listeners.add(listener);
  if (!observer) {
    observer = new MutationObserver(() => { for (const update of listeners) update(); });
    observer.observe(document.documentElement, { attributes: true, attributeFilter: ["class"] });
  }
  return () => {
    listeners.delete(listener);
    if (!listeners.size) { observer?.disconnect(); observer = undefined; }
  };
}
function snapshot(): "light" | "dark" { return document.documentElement.classList.contains("dark") ? "dark" : "light"; }
export function useEffectiveTheme(): "light" | "dark" { return useSyncExternalStore(subscribe, snapshot); }
