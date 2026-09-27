import { useCallback, useEffect, useRef, useState, type ReactNode } from "react";
import { api, ApiError } from "@/lib/api/client";
import { AuthContext, type AuthState } from "./auth-context";

export function AuthProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<AuthState>({ status: "loading" });
  const [attempt, setAttempt] = useState(0);
  const logoutRequest = useRef<AbortController | null>(null);
  const logoutPending = useRef<Promise<void> | null>(null);
  const expire = useCallback(() => { logoutRequest.current?.abort(); setState({ status: "anonymous" }); }, []);
  useEffect(() => () => logoutRequest.current?.abort(), []);
  useEffect(() => {
    const controller = new AbortController();
    api.session(controller.signal).then(session => {
      if (!controller.signal.aborted) setState({ status: "authenticated", session });
    }).catch((error: unknown) => {
      if (controller.signal.aborted) return;
      setState(error instanceof ApiError && error.status === 401 ? { status: "anonymous" } : { status: "error", error });
    });
    return () => controller.abort();
  }, [attempt]);
  const session = state.status === "authenticated" ? state.session : null;
  useEffect(() => {
    if (!session) return;
    const remaining = Date.parse(session.expires_at) - Date.now();
    const timer = window.setTimeout(() => {
      // Long configured TTLs need another server check, not an early local logout.
      if (remaining > 2_147_483_647) setAttempt(value => value + 1);
      else expire();
    }, Math.max(0, Math.min(remaining, 2_147_483_647)));
    return () => window.clearTimeout(timer);
  }, [session, expire]);
  async function login(password: string, signal: AbortSignal) {
    logoutRequest.current?.abort();
    await logoutPending.current?.catch(() => undefined);
    if (signal.aborted) return;
    const session = await api.login(password, signal);
    if (!signal.aborted) setState({ status: "authenticated", session });
  }
  async function logout(signal: AbortSignal) {
    if (state.status !== "authenticated") return;
    logoutRequest.current?.abort();
    const controller = new AbortController();
    logoutRequest.current = controller;
    const requestSignal = AbortSignal.any([signal, controller.signal]);
    const response = api.logout(state.session.csrf_token, requestSignal);
    logoutPending.current = response;
    try {
      await response;
    } catch (error) {
      if (!requestSignal.aborted && !(error instanceof ApiError && error.status === 401)) throw error;
    } finally {
      if (logoutRequest.current === controller) logoutRequest.current = null;
      if (logoutPending.current === response) logoutPending.current = null;
    }
    if (!requestSignal.aborted) setState(current => current.status === "authenticated" && current.session.csrf_token === state.session.csrf_token ? { status: "anonymous" } : current);
  }
  return <AuthContext.Provider value={{ state, login, logout, expire, retry: () => { setState({ status: "loading" }); setAttempt(value => value + 1); } }}>{children}</AuthContext.Provider>;
}
