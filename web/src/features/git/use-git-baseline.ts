import { useEffect, useState } from "react";
import { searchGitAPI } from "@/lib/api/search-git-client";
import type { Project } from "@/lib/api/decoder";
import type { OpenFile } from "@/features/workspaces/workspace-view";
export function useGitBaseline(project: Project, file: OpenFile, identity: string, enabled: boolean) {
  const key = JSON.stringify([project.id, project.version, file.folderId, file.path, identity]);
  const [result, setResult] = useState<{ key: string; value: Awaited<ReturnType<typeof searchGitAPI.baseline>> | null } | null>(null);
  useEffect(() => {
    if (!enabled) return;
    let controller: AbortController | null = null, busy = false, active = true;
    const refresh = async () => {
      if (busy || document.visibilityState === "hidden") return;
      busy = true; controller = new AbortController();
      try { const value = await searchGitAPI.baseline(project.id, project.version, file.folderId, file.path, controller.signal); if (active && !controller.signal.aborted) setResult({ key, value }); }
      catch { if (active && !controller.signal.aborted) setResult({ key, value: { state: "unavailable", repo_id: "", head: "", content: "", version: null } }); }
      finally { busy = false; }
    };
    void refresh(); const timer = setInterval(() => void refresh(), 15000);
    window.addEventListener("focus", refresh); window.addEventListener("online", refresh); document.addEventListener("visibilitychange", refresh);
    return () => { active = false; controller?.abort(); clearInterval(timer); window.removeEventListener("focus", refresh); window.removeEventListener("online", refresh); document.removeEventListener("visibilitychange", refresh); };
  }, [key, enabled, project.id, project.version, file.folderId, file.path]);
  return enabled && result?.key === key ? result.value : null;
}
