import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { useAuth } from "@/features/auth/auth-context";
import { draftStorage, downloadDraft, type FileDraft } from "./editor-drafts";
export function DraftRecovery({ projectId }: { projectId: string }) {
  const auth = useAuth();
  const [drafts, setDrafts] = useState<FileDraft[]>([]);
  const [error, setError] = useState<string | null>(null);
  useEffect(() => {
    let active = true;
    if (auth.state.status === "authenticated") void draftStorage.list(projectId).then(items => { if (active) setDrafts(items); }).catch(() => { if (active) setError("无法读取本地草稿，原有内容未清理。"); });
    return () => { active = false; };
  }, [auth.state.status, projectId]);
  if (auth.state.status !== "authenticated") return null;
  return <div className="flex flex-col gap-2">{error && <Alert variant="destructive"><AlertDescription>{error}</AlertDescription></Alert>}{drafts.map(draft => <div key={draft.id} className="flex items-center gap-2"><span className="truncate">{draft.sourceRoot}/{draft.file.path}</span><Button variant="outline" onClick={() => downloadDraft(draft.content, draft.file.path)}>导出草稿</Button></div>)}</div>;
}
