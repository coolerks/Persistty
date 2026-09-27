import { decodeEnvelope, decodeError, decodeList, decodeProject, decodeSession, decodeTerminal, ProtocolError } from "./decoder";

export class ApiError extends Error {
  constructor(public readonly status: number, public readonly code: string, message: string,
    public readonly requestId: string, public readonly retryAfter: number | null) {
    super(message); this.name = "ApiError";
  }
}

async function request<T>(path: string, decode: (value: unknown) => T, options: RequestInit = {}): Promise<T> {
  const response = await fetch(`/api/v1${path}`, { ...options, credentials: "same-origin", cache: "no-store", redirect: "error" });
  if (!response.ok) {
    const value: unknown = await response.json().catch(() => { throw new ProtocolError(); });
    const error = decodeError(value);
    const retry = response.headers.get("Retry-After");
    const seconds = retry === null ? NaN : /^\d+$/.test(retry) ? Number(retry) : Math.max(0, (Date.parse(retry) - Date.now()) / 1000);
    const retryAfter = Number.isFinite(seconds) && seconds >= 0 ? seconds : null;
    throw new ApiError(response.status, error.code, error.message, error.requestId, retryAfter);
  }
  if (response.status === 204) return decode(undefined);
  const value: unknown = await response.json().catch(() => { throw new ProtocolError(); });
  return decodeEnvelope(value, decode);
}

export const api = {
  session: (signal: AbortSignal) => request("/auth/session", decodeSession, { signal }),
  login: (password: string, signal: AbortSignal) => request("/auth/login", decodeSession,
    { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ password }), signal }),
  logout: (csrf: string, signal: AbortSignal) => request("/auth/logout", () => undefined,
    { method: "POST", headers: { "X-CSRF-Token": csrf }, signal }),
  projects: (signal: AbortSignal) => request("/projects", decodeList(decodeProject), { signal }),
  project: (projectId: string, signal: AbortSignal) => request(`/projects/${encodeURIComponent(projectId)}`, decodeProject, { signal }),
  terminals: (signal: AbortSignal) => request("/terminals", decodeList(decodeTerminal), { signal }),
};

export function errorMessage(error: unknown): string {
  if (error instanceof ApiError && error.status === 429) return `请求过于频繁。${error.retryAfter !== null ? `请在 ${Math.ceil(error.retryAfter)} 秒后重试。` : "请稍后重试。"}`;
  if (error instanceof Error && error.name !== "TypeError") return error.message;
  return "无法连接服务器，请检查网络后重试。";
}
