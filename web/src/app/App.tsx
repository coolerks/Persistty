import { useEffect, useRef, useState } from "react";
import { Link, Navigate, NavLink, Outlet, Route, Routes, useLocation, useOutletContext } from "react-router";
import { TerminalSquare, Folder, LogOut } from "lucide-react";
import { Button, buttonVariants } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { AuthProvider } from "@/features/auth/AuthProvider";
import { useAuth } from "@/features/auth/auth-context";
import { LoginPage } from "@/features/auth/LoginPage";
import { ThemeSelect } from "@/features/settings/ThemeSelect";
import { ProjectsPage, ProjectRoute } from "@/features/workspaces/ProjectsPage";
import { TerminalsPage } from "@/features/terminal/TerminalsPage";
import { errorMessage } from "@/lib/api/client";
import { Failure, Loading } from "@/components/Feedback";

function Authenticated() {
  const { state } = useAuth();
  const location = useLocation();
  const shell = useOutletContext<unknown>();
  if (state.status !== "authenticated") return <Navigate to={`/login?return=${encodeURIComponent(location.pathname)}`} replace />;
  return <Outlet context={shell} />;
}

function Shell() {
  const auth = useAuth();
  const location = useLocation();
  const workspaceRoute = /^\/projects\/[^/]+\/?$/.test(location.pathname);
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const request = useRef<AbortController | null>(null);
  useEffect(() => () => request.current?.abort(), []);
  async function logout() {
    if (request.current) return;
    const controller = new AbortController(); request.current = controller;
    setPending(true); setError(null);
    try { await auth.logout(controller.signal); }
    catch (error: unknown) { if (!controller.signal.aborted) setError(errorMessage(error)); }
    finally { request.current = null; if (!controller.signal.aborted) setPending(false); }
  }
  return <div className={`app-shell ${workspaceRoute ? "app-shell-workspace" : ""}`}>
    {!workspaceRoute && <header className="app-header">
      <Link to="/projects" className="brand"><TerminalSquare aria-hidden="true" />Persistty</Link>
      <div className="flex items-center gap-2"><ThemeSelect />
        {auth.state.status === "authenticated" && <Tooltip><TooltipTrigger render={<Button variant="ghost" size="icon" aria-label="退出登录" disabled={pending} onClick={() => { void logout(); }} />}><LogOut /></TooltipTrigger><TooltipContent>退出登录</TooltipContent></Tooltip>}
      </div>
    </header>}
    {error && <Alert variant="destructive"><AlertDescription>{error}</AlertDescription></Alert>}
    {auth.state.status === "authenticated" && !workspaceRoute && <nav aria-label="主导航" className="main-nav">
      <NavLink to="/projects"><Folder aria-hidden="true" />项目</NavLink><NavLink to="/terminals"><TerminalSquare aria-hidden="true" />终端</NavLink>
    </nav>}
    {auth.state.status === "loading" ? <main className="page-main"><Loading /></main> : auth.state.status === "error" ? <main className="page-main"><Failure error={auth.state.error} retry={auth.retry} /></main> : <Outlet context={{ headerActions: <>
      <Link className={buttonVariants({ variant: "ghost", size: "icon-sm" })} to="/projects" aria-label="Persistty 项目面板" title="Persistty 项目面板"><Folder /></Link><ThemeSelect />
      <Button variant="ghost" size="icon-sm" aria-label="退出登录" title="退出登录" disabled={pending} onClick={() => void logout()}><LogOut /></Button>
    </> }} />}
    {!workspaceRoute && <footer className="app-footer">{auth.state.status === "authenticated" ? "已登录" : "Persistty"}</footer>}
  </div>;
}

export function App() {
  return <TooltipProvider delay={300}><AuthProvider><Routes><Route element={<Shell />}>
    <Route path="/login" element={<LoginPage />} />
    <Route element={<Authenticated />}>
      <Route path="/projects" element={<ProjectsPage />} />
      <Route path="/projects/:projectId" element={<ProjectRoute />} />
      <Route path="/terminals" element={<TerminalsPage />} />
      <Route path="/terminals/:terminalId" element={<TerminalsPage />} />
    </Route>
    <Route path="/" element={<Navigate to="/projects" replace />} />
    <Route path="*" element={<main className="page-main"><h1>页面不存在</h1><Link className={buttonVariants({ variant: "outline" })} to="/projects">项目面板</Link></main>} />
  </Route></Routes></AuthProvider></TooltipProvider>;
}
