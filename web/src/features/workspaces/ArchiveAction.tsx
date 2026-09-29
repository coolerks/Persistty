import { useEffect, useState } from "react";
import { Download, X } from "lucide-react";
import { Button, buttonVariants } from "@/components/ui/button";
import { api, archiveDownloadURL, errorMessage } from "@/lib/api/client";
import type { ArchiveRecord, Project } from "@/lib/api/decoder";
import { useAuth } from "@/features/auth/auth-context";

export function useArchiveAction(project: Project) {
  const auth = useAuth();
  const [item, setItem] = useState<ArchiveRecord | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState(false);
  const csrf = auth.state.status === "authenticated" ? auth.state.session.csrf_token : "";

  useEffect(() => {
    if (!item || item.status !== "pending") return;
    const controller = new AbortController();
    const timer = window.setInterval(() => {
      void api.archiveStatus(item.id, controller.signal).then(setItem).catch(cause => {
        if (!controller.signal.aborted) setError(errorMessage(cause));
      });
    }, 800);
    return () => { controller.abort(); window.clearInterval(timer); };
  }, [item]);

  async function begin(folderId: string, path: string) {
    if (!csrf || pending || item?.status === "pending") return;
    setPending(true); setError(null); setItem(null);
    try { setItem(await api.createArchive(project.id, folderId, project.version, path, csrf, new AbortController().signal)); }
    catch (cause: unknown) { setError(errorMessage(cause)); }
    finally { setPending(false); }
  }
  async function cancel() {
    if (!item || !csrf || pending) return;
    setPending(true);
    try { await api.cancelArchive(item.id, csrf, new AbortController().signal); setItem(null); setError(null); }
    catch (cause: unknown) { setError(errorMessage(cause)); }
    finally { setPending(false); }
  }
  const status = item?.status === "pending" ? "正在生成 ZIP" : item?.status === "ready" ? "ZIP 已就绪" : item?.status === "failed" ? `ZIP 生成失败：${item.error_code}` : item?.status === "cancelled" ? "ZIP 已取消" : "";
  return { begin, element: (item || error) && <div className="flex items-center gap-2 border-t px-3 py-2 text-sm" role="status"><span className="min-w-0 flex-1 truncate">{error ?? status}</span>{item?.status === "ready" && <a className={buttonVariants({ size: "sm", variant: "outline" })} href={archiveDownloadURL(item.id)}><Download data-icon="inline-start" />下载</a>}{item && <Button size="icon" variant="ghost" aria-label="取消 ZIP 任务" title="取消 ZIP 任务" disabled={pending} onClick={() => void cancel()}><X /></Button>}</div> };
}
