import { useEffect, useId, useRef } from "react";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Field, FieldLabel } from "@/components/ui/field";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import { errorMessage } from "@/lib/api/client";
import { useEditorScope } from "./editor-context";

const messages: Record<string, string> = { authorization_failed: "系统授权未成功，请检查密码和服务器授权配置后重新申请。", conflict: "文件已变化，请返回编辑器比较后重新申请。", expired: "本次授权已过期，请重新申请。", cancelled: "本次保存已取消，输入已保留。", forbidden: "服务器拒绝此文件的提权保存。", elevation_unavailable: "授权执行服务不可用，输入已保留。", invalid_request: "本次授权与保存内容不一致，请重新申请。", outcome_unknown: "结果尚未确认，请查询结果或重新读取服务器内容并比较。", unauthenticated: "登录已失效，本次保存未接受。请重新登录。" };
export function PrivilegedSaveDialog() {
  const scope = useEditorScope(), attempt = scope.elevation;
  const password = useRef<HTMLInputElement>(null), cancel = useRef<HTMLButtonElement>(null), label = useId();
  useEffect(() => {
    const input = password.current;
    if (!attempt || attempt.phase !== "prepared" || !attempt.request) return () => { if (input) input.value = ""; };
    const timer = setTimeout(() => attempt.buffer.expireElevation(), Math.max(0, Date.parse(attempt.request.expires_at) - Date.now()));
    return () => { clearTimeout(timer); if (input) input.value = ""; };
  }, [attempt, attempt?.phase, attempt?.request]);
  const close = () => { if (password.current) password.current.value = ""; attempt?.buffer.dismissElevation(); };
  return <Dialog open={attempt !== null} onOpenChange={open => { if (!open) close(); }}><DialogContent showCloseButton={false} initialFocus={cancel} className="max-h-[calc(100dvh-2rem)] overflow-y-auto sm:max-w-lg"><DialogHeader><DialogTitle>单文件提权保存</DialogTitle><DialogDescription>输入当前系统用户密码，仅保存本次确认的内容。系统授权不会用于后续自动保存。</DialogDescription></DialogHeader>
    {attempt && <>
      {attempt.phase === "preparing" && <p role="status">正在检查允许范围和文件版本…</p>}
      {attempt.request && <p className="min-w-0 break-all">目标：{attempt.request.target_path}</p>}
      <Collapsible><CollapsibleTrigger render={<Button variant="outline" size="sm" />}>查看本次保存内容</CollapsibleTrigger><CollapsibleContent><pre aria-label="本次保存内容" tabIndex={0} className="mt-2 max-h-48 overflow-auto whitespace-pre-wrap break-all rounded-md border p-2 font-mono text-sm">{attempt.content}</pre></CollapsibleContent></Collapsible>
      {attempt.phase === "prepared" && <><p className="text-muted-foreground">有效至 {attempt.request ? new Date(attempt.request.expires_at).toLocaleTimeString() : ""}；此后输入的编辑内容仍会保留。</p><Field><FieldLabel htmlFor={label}>系统用户密码</FieldLabel><Input ref={password} id={label} type="password" autoComplete="off" spellCheck={false} maxLength={1024} /></Field></>}
      {attempt.phase === "executing" && <p role="status">正在授权并保存…关闭后将取消尚未接受的提交；已接受的保存需确认实际结果。</p>}
      {attempt.result?.state === "applied" && <p role="status">本次内容已保存。{attempt.buffer.dirty ? "之后的输入仍未保存，自动保存已暂停。" : ""}</p>}
      {attempt.result?.code && <p role="alert">{messages[attempt.result.code] ?? "本次保存未完成，输入已保留。"}</p>}
      {attempt.error !== null && <p role="alert">{errorMessage(attempt.error)}</p>}
      {attempt.phase === "unknown" && <p>只查询本次结果，不会重发密码或再次写入。</p>}
    </>}
    <DialogFooter><Button ref={cancel} variant="outline" onClick={close}>{attempt?.phase === "result" ? "关闭" : "取消"}</Button>{attempt?.phase === "prepared" && <Button onClick={() => { const value = password.current?.value ?? ""; if (password.current) password.current.value = ""; if (!value) { password.current?.focus(); return; } void attempt.buffer.executeElevation(value); }}>确认一次保存</Button>}{attempt?.phase === "unknown" && <Button variant="outline" disabled={attempt.querying} onClick={() => void attempt.buffer.queryElevation()}>查询本次结果</Button>}</DialogFooter>
  </DialogContent></Dialog>;
}
