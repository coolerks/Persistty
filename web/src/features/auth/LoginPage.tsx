import { useEffect, useRef, useState, type FormEvent } from "react";
import { Navigate, useSearchParams } from "react-router";
import { ArrowRight, Files, LockKeyhole, TerminalSquare } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Field, FieldLabel, FieldError, FieldGroup } from "@/components/ui/field";
import { ApiError, errorMessage } from "@/lib/api/client";
import { useAuth } from "./auth-context";
import { safeReturnPath } from "./return-path";

export function LoginPage() {
  const auth = useAuth();
  const [search] = useSearchParams();
  const [password, setPassword] = useState("");
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [blockedUntil, setBlockedUntil] = useState(0);
  const request = useRef<AbortController | null>(null);
  useEffect(() => () => request.current?.abort(), []);
  useEffect(() => {
    if (!blockedUntil) return;
    const timer = window.setTimeout(() => setBlockedUntil(0), Math.max(0, blockedUntil - Date.now()));
    return () => window.clearTimeout(timer);
  }, [blockedUntil]);
  if (auth.state.status === "authenticated") return <Navigate to={safeReturnPath(search.get("return"))} replace />;
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (request.current || !password || blockedUntil > Date.now()) return;
    const controller = new AbortController();
    request.current = controller;
    setPending(true); setError(null);
    const submitted = password;
    setPassword("");
    try { await auth.login(submitted, controller.signal); }
    catch (error: unknown) {
      if (!controller.signal.aborted) {
        setError(errorMessage(error));
        if (error instanceof ApiError && error.status === 429 && error.retryAfter !== null)
          setBlockedUntil(Date.now() + Math.min(error.retryAfter, 86400) * 1000);
      }
    } finally {
      request.current = null;
      if (!controller.signal.aborted) setPending(false);
    }
  }
  return <main className="login-main"><div className="login-frame">
    <div className="login-intro">
      <span className="login-eyebrow">你的开发工作台</span>
      <h2>回到你的开发现场</h2>
      <p>打开项目，接着上次的进度继续。</p>
      <div className="login-capabilities">
        <div><TerminalSquare aria-hidden="true" /><span>持久终端<small>离开页面，任务仍在运行</small></span></div>
        <div><Files aria-hidden="true" /><span>文件与代码<small>浏览、编辑与搜索，一处完成</small></span></div>
      </div>
    </div>
    <section className="login-form" aria-labelledby="login-title">
      <div className="login-form-heading"><span className="login-lock"><LockKeyhole aria-hidden="true" /></span><h1 id="login-title">登录工作台</h1><p>使用访问密码进入 Persistty。</p></div>
      <form onSubmit={event => { void submit(event); }}>
        <FieldGroup>
          <Field data-invalid={Boolean(error)} data-disabled={pending || blockedUntil > 0}>
            <FieldLabel htmlFor="password">访问密码</FieldLabel>
            <Input id="password" type="password" autoComplete="current-password" placeholder="输入访问密码" autoFocus required maxLength={1024}
              value={password} onChange={event => setPassword(event.target.value)} disabled={pending || blockedUntil > 0}
              aria-invalid={Boolean(error)} aria-describedby={error ? "login-error" : undefined} />
            {error && <FieldError id="login-error">{error}</FieldError>}
          </Field>
          <Button type="submit" disabled={pending || blockedUntil > 0 || !password}>
            {pending ? "正在登录" : "登录"}<ArrowRight data-icon="inline-end" />
          </Button>
        </FieldGroup>
      </form>
    </section></div>
  </main>;
}
