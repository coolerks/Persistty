import { useEffect, useState } from "react";
import { searchGitAPI } from "@/lib/api/search-git-client";
import type { Project } from "@/lib/api/decoder";
import type { OpenFile } from "@/features/workspaces/workspace-view";
import { subscribeBaseline } from "./baseline-requests";
export function useGitBaseline(project: Project, file: OpenFile, identity: string, enabled: boolean) {
  const key = JSON.stringify([project.id, project.version, file.folderId, file.path, identity]);
  const [result, setResult] = useState<{ key: string; value: Awaited<ReturnType<typeof searchGitAPI.baseline>> | null } | null>(null);
  useEffect(() => {
    if (!enabled) return;
    return subscribeBaseline(key, project.id,
      signal => searchGitAPI.baseline(project.id, project.version, file.folderId, file.path, signal),
      value => setResult({ key, value }));
  }, [key, enabled, project.id, project.version, file.folderId, file.path]);
  return enabled && result?.key === key ? result.value : null;
}
