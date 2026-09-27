import { Link, useParams } from "react-router";
import { TerminalSquare, RefreshCw } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Empty, EmptyHeader, EmptyTitle, EmptyMedia, EmptyDescription } from "@/components/ui/empty";
import { api } from "@/lib/api/client";
import { useResource } from "@/lib/api/use-resource";
import { Failure, Loading } from "@/components/Feedback";

export function TerminalsPage() {
  const { terminalId } = useParams();
  return <TerminalContent key={terminalId ?? "list"} terminalId={terminalId ?? null} />;
}
function TerminalContent({ terminalId }: { terminalId: string | null }) {
  const { resource, refresh } = useResource(api.terminals);
  const terminal = resource.status === "ready" && terminalId ? resource.data.find(item => item.id === terminalId) : undefined;
  return <main className="page-main">
    <div className="page-heading"><h1>{terminal?.display_name ?? "终端"}</h1><Button variant="ghost" size="icon" aria-label="刷新终端" title="刷新终端" disabled={resource.status === "loading"} onClick={refresh}><RefreshCw /></Button></div>
    {resource.status === "loading" && <Loading />}
    {resource.status === "error" && <Failure error={resource.error} retry={refresh} />}
    {resource.status === "ready" && (terminalId ? terminal ?
      <div className="flex flex-col items-start gap-4"><Badge variant="secondary">不可用</Badge><p className="break-all font-mono text-sm">{terminal.working_directory}</p><Button asChild variant="outline"><Link to="/terminals">全部终端</Link></Button></div> :
      <Empty><EmptyHeader><EmptyMedia variant="icon"><TerminalSquare /></EmptyMedia><EmptyTitle>终端不存在</EmptyTitle></EmptyHeader><Button asChild variant="outline"><Link to="/terminals">全部终端</Link></Button></Empty> :
      resource.data.length === 0 ? <Empty><EmptyHeader><EmptyMedia variant="icon"><TerminalSquare /></EmptyMedia><EmptyTitle>暂无终端</EmptyTitle></EmptyHeader></Empty> :
      <ul className="resource-list">{resource.data.map(item => <li key={item.id}><Button asChild variant="ghost" className="resource-row"><Link to={`/terminals/${encodeURIComponent(item.id)}`}><TerminalSquare data-icon="inline-start" /><span className="resource-text"><span>{item.display_name}</span><span className="resource-path">{item.working_directory}</span></span><Badge variant="secondary">不可用</Badge></Link></Button></li>)}</ul>)}
    {terminal && !terminal.project_id && <EmptyDescription>未关联项目</EmptyDescription>}
  </main>;
}
