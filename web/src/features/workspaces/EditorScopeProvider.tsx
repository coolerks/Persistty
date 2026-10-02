import { useEffect, useRef, useState, type ReactNode } from "react";
import { useNavigate } from "react-router";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { useAuth } from "@/features/auth/auth-context";
import type { Project } from "@/lib/api/decoder";
import { EditorContext } from "./editor-context";
import { EditorScope, editorScopes } from "./editor-session";
import { useWorkspaceView } from "./workspace-view";
import { PrivilegedSaveDialog } from "./PrivilegedSaveDialog";

export function EditorScopeProvider({ project, children }: { project: Project; children: ReactNode }) {
  const auth = useAuth();
  const navigate = useNavigate();
  const [leaveError, setLeaveError] = useState<string | null>(null);
  const [scope] = useState(() => new EditorScope(project));
  const cleanup = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  const csrf = auth.state.status === "authenticated" ? auth.state.session.csrf_token : "";
  useEffect(() => { void scope.configure(project, csrf); }, [scope, project, csrf]);
  useEffect(() => {
    clearTimeout(cleanup.current);
    editorScopes.set(project.id, scope);
    const unsubscribe = useWorkspaceView.subscribe(() => scope.closeInactive());
    const leave = (event: MouseEvent) => {
      if (event.button !== 0 || event.ctrlKey || event.metaKey || event.shiftKey || event.altKey || !(event.target instanceof Element)) return;
      const link = event.target.closest<HTMLAnchorElement>("a[href]"); if (!link || link.target === "_blank" || link.hasAttribute("download")) return;
      const target = new URL(link.href); if (target.origin !== location.origin || target.pathname.startsWith("/api/") || target.pathname === location.pathname) return;
      event.preventDefault(); event.stopPropagation();
      void scope.protect().then(ok => { if (ok) navigate(target.pathname + target.search + target.hash); else setLeaveError("草稿保护失败，仍保留当前页面和输入。请导出内容后再离开。"); });
    };
    document.addEventListener("click", leave, true);
    const unload = (event: BeforeUnloadEvent) => {
      if ([...new Set(scope.buffers.values())].some(buffer => buffer.dirty)) { event.preventDefault(); }
    };
    const refresh = () => { void scope.refresh(); };
    const poll = setInterval(refresh, 15000);
    window.addEventListener("beforeunload", unload); window.addEventListener("online", refresh); window.addEventListener("focus", refresh);
    return () => { document.removeEventListener("click", leave, true); unsubscribe(); clearInterval(poll); window.removeEventListener("beforeunload", unload); window.removeEventListener("online", refresh); window.removeEventListener("focus", refresh); cleanup.current = setTimeout(() => { scope.dispose(); if (editorScopes.get(project.id) === scope) editorScopes.delete(project.id); }, 0); };
  }, [scope, project.id, navigate]);
  return <EditorContext.Provider value={scope}>{leaveError && <Alert variant="destructive"><AlertDescription>{leaveError}</AlertDescription></Alert>}{children}<PrivilegedSaveDialog /></EditorContext.Provider>;
}
