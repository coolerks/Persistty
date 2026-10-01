import { searchGitAPI } from "@/lib/api/search-git-client";

type Baseline = Awaited<ReturnType<typeof searchGitAPI.baseline>>;
type Entry = {
  project: string; listeners: Set<(value: Baseline) => void>;
  load: (signal: AbortSignal) => Promise<Baseline>;
  controller: AbortController | null; queued: boolean; started: number;
  value: Baseline | null;
};
// Share only active subscriptions, never persistent server truth or credentials.
// Baselines use one background request at a time, leaving a tool slot for input.
const entries = new Map<string, Entry>();
const queue: Entry[] = [];
let running = false;
async function pump() {
  if (running) return;
  const entry = queue.shift();
  if (!entry) return;
  entry.queued = false;
  if (!entry.listeners.size) { void pump(); return; }
  running = true;
  const controller = new AbortController(); entry.controller = controller; entry.started = Date.now();
  try {
    const value = await entry.load(controller.signal);
    if (!controller.signal.aborted) { entry.value = value; for (const listener of entry.listeners) listener(value); }
  } catch {
    if (!controller.signal.aborted) {
      const value: Baseline = { state: "unavailable", repo_id: "", head: "", content: "", version: null };
      entry.value = value; for (const listener of entry.listeners) listener(value);
    }
  } finally { entry.controller = null; running = false; void pump(); }
}
function refresh(entry: Entry, force = false) {
  if (document.visibilityState === "hidden") return;
  if (entry.queued || entry.controller || (!force && Date.now() - entry.started < 30000)) return;
  entry.queued = true; queue.push(entry); queueMicrotask(() => void pump());
}
export function refreshGitBaselines(project: string) {
  for (const entry of entries.values()) if (entry.project === project) refresh(entry, true);
}
export function subscribeBaseline(key: string, project: string, load: Entry["load"], listener: (value: Baseline) => void) {
  let entry = entries.get(key);
  if (!entry) { entry = { project, load, listeners: new Set(), controller: null, queued: false, started: -Infinity, value: null }; entries.set(key, entry); }
  const current = entry;
  current.listeners.add(listener);
  if (current.value) listener(current.value);
  refresh(current);
  const update = () => { if (document.visibilityState !== "hidden") refresh(current); };
  window.addEventListener("focus", update); window.addEventListener("online", update); document.addEventListener("visibilitychange", update);
  return () => {
    window.removeEventListener("focus", update); window.removeEventListener("online", update); document.removeEventListener("visibilitychange", update);
    current.listeners.delete(listener);
    if (!current.listeners.size) {
      current.controller?.abort(); entries.delete(key);
      const index = queue.indexOf(current); if (index >= 0) queue.splice(index, 1);
    }
  };
}
