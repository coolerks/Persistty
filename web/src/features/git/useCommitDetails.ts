import { useCallback, useEffect, useRef, useState } from "react";
import type { Project } from "@/lib/api/decoder";
import type { CommitDetail } from "@/lib/api/search-git-decoder";
import { errorMessage } from "@/lib/api/client";
import { searchGitAPI } from "@/lib/api/search-git-client";

export function useCommitDetails(project: Project, repo: string, visible: boolean, revision: number, busy: boolean) {
  const scope = JSON.stringify([project.id, project.version, repo, revision]);
  const cache = useRef(new Map<string, { detail: CommitDetail; bytes: number }>());
  const [active, setActive] = useState<{ scope: string; commit: string; parent: string } | null>(null);
  const [result, setResult] = useState<{ key: string; detail: CommitDetail | null; error: string | null } | null>(null);
  const previewController = useRef<AbortController | null>(null);
  const cancelPreview = useCallback(() => { previewController.current?.abort(); }, []);
  const read = useCallback(async (commit: string, parent: string, signal: AbortSignal) => {
    const key = JSON.stringify([scope, commit, parent]), values = cache.current;
    const hit = values.get(key);
    if (hit) { values.delete(key); values.set(key, hit); return hit.detail; }
    const detail = await searchGitAPI.detail(project.id, project.version, repo, commit, parent, signal);
    if (!signal.aborted) {
      const bytes = JSON.stringify(detail).length * 2;
      if (bytes <= 2 << 20) {
        values.set(key, { detail, bytes });
        while (values.size > 8 || [...values.values()].reduce((sum, entry) => sum + entry.bytes, 0) > 2 << 20) values.delete(values.keys().next().value!);
      }
    }
    return detail;
  }, [project.id, project.version, repo, scope]);
  useEffect(() => { cache.current.clear(); setActive(null); setResult(null); return cancelPreview; }, [scope, visible, cancelPreview]);
  useEffect(() => {
    if (!visible || busy || !active || active.scope !== scope) return;
    const key = JSON.stringify([active.commit, active.parent]);
    if (result?.key === key) return;
    const current = new AbortController(); previewController.current = current;
    void read(active.commit, active.parent, current.signal).then(detail => {
      if (!current.signal.aborted) setResult({ key, detail, error: null });
    }).catch(cause => { if (!current.signal.aborted) setResult({ key, detail: null, error: errorMessage(cause) }); });
    return () => current.abort();
  }, [active, busy, visible, scope, read, result]);
  const show = useCallback((commit: string, parent: string, open: boolean) => {
    if (open) { setResult(null); setActive({ scope, commit, parent }); }
    else setActive(previous => previous?.scope === scope && previous.commit === commit && previous.parent === parent ? null : previous);
  }, [scope]);
  const key = active && active.scope === scope ? JSON.stringify([active.commit, active.parent]) : "";
  return { read, show, cancelPreview, preview: active?.scope === scope ? { commit: active.commit, parent: active.parent, detail: result?.key === key ? result.detail : null, error: result?.key === key ? result.error : null } : null };
}
