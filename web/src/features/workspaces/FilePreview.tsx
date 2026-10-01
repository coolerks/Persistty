import { useEffect, useState } from "react";
import { api, errorMessage } from "@/lib/api/client";
import { Loading } from "@/components/Feedback";
import { Alert, AlertDescription } from "@/components/ui/alert";
import type { Project } from "@/lib/api/decoder";
import type { OpenFile } from "./workspace-view";

// Authenticated native <img> only. SVG never enters the document or an iframe.
export function FilePreview({ project, file }: { project: Project; file: OpenFile }) {
  const [state, setState] = useState<{ url?: string; error?: string; loading?: boolean }>({ loading: true });
  useEffect(() => {
    const controller = new AbortController();
    void api.inspect(project.id, file.folderId, project.version, file.path, controller.signal).then(info => {
      if (controller.signal.aborted) return;
      if (!info.previewable || info.size > 16 * 1024 * 1024 || info.width * info.height > 16_000_000) { setState({ error: "此文件不支持安全预览，或超过 16 MiB / 1600 万像素上限。请下载查看。" }); return; }
      setState({ url: `/api/v1/projects/${encodeURIComponent(project.id)}/folders/${encodeURIComponent(file.folderId)}/preview?project_version=${project.version}&path=${encodeURIComponent(file.path)}` });
    }).catch(reason => { if (!controller.signal.aborted) setState({ error: errorMessage(reason) }); });
    return () => controller.abort();
  }, [project.id, project.version, file.folderId, file.path]);
  return <div className="file-preview">{state.loading ? <Loading /> : state.error ? <Alert><AlertDescription>{state.error}</AlertDescription></Alert> : <img src={state.url} alt={file.path} onError={() => setState({ error: "图片无法解码或已发生变化，请刷新或下载。" })} onLoad={event => { const image = event.currentTarget; if (image.naturalWidth > 8192 || image.naturalHeight > 8192 || image.naturalWidth * image.naturalHeight > 16_000_000) setState({ error: "图片超过像素上限，请下载查看。" }); }} />}</div>;
}
