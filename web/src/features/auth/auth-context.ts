import { createContext, useContext } from "react";
import type { Session } from "@/lib/api/decoder";

export type AuthState = { status: "loading" } | { status: "anonymous" } | { status: "authenticated"; session: Session } | { status: "error"; error: unknown };
export type AuthContextValue = { state: AuthState; login(password: string, signal: AbortSignal): Promise<void>; logout(signal: AbortSignal): Promise<void>; expire(): void; retry(): void };
export const AuthContext = createContext<AuthContextValue | null>(null);
export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) throw new Error("缺少认证上下文");
  return context;
}
