import { useEffect, useRef, useState } from "react";

export function useFileEvents(projectId: string, folderId: string, version: number, path: string, onRescan: () => void) {
  const callback = useRef(onRescan);
  callback.current = onRescan;
  const [mode, setMode] = useState<"watching" | "polling">("polling");
  useEffect(() => {
    let disposed = false;
    let socket: WebSocket | null = null;
    let retry: number | null = null;
    const poll = window.setInterval(() => callback.current(), 15000);
    function connect() {
      if (disposed) return;
      const scheme = window.location.protocol === "https:" ? "wss:" : "ws:";
      const query = new URLSearchParams({ project_id: projectId, folder_id: folderId, project_version: String(version), path });
      try { socket = new WebSocket(`${scheme}//${window.location.host}/api/v1/events?${query}`); }
      catch { setMode("polling"); retry = window.setTimeout(connect, 2000); return; }
      socket.onopen = () => callback.current();
      socket.onmessage = event => {
        try {
          const value: unknown = JSON.parse(String(event.data));
          if (typeof value !== "object" || value === null || !("project_id" in value) || !("folder_id" in value) || !("rescan" in value) || !("mode" in value)) return;
          if (value.project_id !== projectId || value.folder_id !== folderId || value.rescan !== true) return;
          setMode(value.mode === "watching" ? "watching" : "polling");
          callback.current();
        } catch { callback.current(); }
      };
      socket.onerror = () => socket?.close();
      socket.onclose = () => { setMode("polling"); if (!disposed) retry = window.setTimeout(connect, 2000); };
    }
    connect();
    return () => { disposed = true; window.clearInterval(poll); if (retry !== null) window.clearTimeout(retry); socket?.close(); };
  }, [projectId, folderId, version, path]);
  return mode;
}
