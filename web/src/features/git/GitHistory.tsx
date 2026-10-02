import { useEffect, useMemo, useRef, type ReactElement } from "react";
import { ChevronRight } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import { ContextMenu, ContextMenuContent, ContextMenuGroup, ContextMenuLabel, ContextMenuRadioGroup, ContextMenuRadioItem, ContextMenuTrigger } from "@/components/ui/context-menu";
import { Loading } from "@/components/Feedback";
import type { searchGitAPI } from "@/lib/api/search-git-client";
import { commitGraph, type GraphRow } from "./commit-graph";
import { GitFiles, type FileView } from "./GitFiles";
type History = Awaited<ReturnType<typeof searchGitAPI.log>>;
type Detail = Awaited<ReturnType<typeof searchGitAPI.detail>>;
function CommitParents({ commit, parent, pending, onParent, children }: {
  commit: History["items"][number]; parent: string; pending: boolean;
  onParent(commit: string, parent: string): void; children: ReactElement;
}) {
  if (commit.parents.length < 2) return children;
  return <ContextMenu><ContextMenuTrigger render={children} /><ContextMenuContent><ContextMenuGroup>
    <ContextMenuLabel>比较父提交</ContextMenuLabel>
    <ContextMenuRadioGroup value={parent || commit.parents[0]}>{commit.parents.map((id, index) =>
      <ContextMenuRadioItem key={id} value={id} disabled={pending} closeOnClick title={id} onClick={() => onParent(commit.id, id)}>父提交 {index + 1} · {id.slice(0, 12)}</ContextMenuRadioItem>
    )}</ContextMenuRadioGroup>
  </ContextMenuGroup></ContextMenuContent></ContextMenu>;
}
function Graph({ row, width, expanded }: { row: GraphRow; width: number; expanded: boolean }) {
  const x = (lane: number) => 10 + lane * 14;
  return <div className="git-graph" aria-hidden="true">
    <svg width={width} height="32" data-graph-lane={row.lane}>
      {row.edges.map((edge, i) => <path key={i} data-parent={edge.parent} className={`git-graph-color-${edge.from % 5}`} d={edge.through ? `M${x(edge.from)} 0 C${x(edge.from)} 16 ${x(edge.to)} 16 ${x(edge.to)} 32` : `M${x(edge.from)} 16 C${x(edge.from)} 24 ${x(edge.to)} 24 ${x(edge.to)} 32`} />)}
      <path className={`git-graph-color-${row.lane % 5}`} d={`M${x(row.lane)} 0 V16`} />
      <circle className={`git-graph-color-${row.lane % 5}`} cx={x(row.lane)} cy="16" r="4" data-expanded={expanded} />
    </svg>
    <svg className="git-graph-tail" width={width} viewBox={`0 0 ${width} 1`} preserveAspectRatio="none">{[...new Set(row.edges.map(edge => edge.to))].map(lane => <path key={lane} className={`git-graph-color-${lane % 5}`} d={`M${x(lane)} 0 V1`} />)}</svg>
  </div>;
}
export function GitHistory({ history, detail, expanded, branch, view, pending, onExpand, onParent, onCompare, onMore }: {
  history: History | null; detail: Detail | null; expanded: string; branch: string; view: FileView; pending: boolean;
  onExpand(commit: string, open: boolean): void; onParent(commit: string, parent: string): void; onCompare(path: string): void; onMore(): Promise<boolean>;
}) {
  const container = useRef<HTMLDivElement>(null);
  const loadingPage = useRef(false), failedPage = useRef("");
  useEffect(() => {
    const scroller = container.current?.closest(".git-section-content");
    if (!scroller || !history || history.next_offset < 0) return;
    const page = `${history.head}:${history.next_offset}`;
    const onScroll = () => {
      if (scroller.clientHeight === 0) return;
      const atBottom = scroller.scrollHeight - scroller.scrollTop - scroller.clientHeight <= 24;
      if (!atBottom) { failedPage.current = ""; return; }
      if (pending || loadingPage.current || failedPage.current === page) return;
      loadingPage.current = true;
      void onMore().then(success => { if (!success) failedPage.current = page; }, () => { failedPage.current = page; })
        .finally(() => { loadingPage.current = false; });
    };
    scroller.addEventListener("scroll", onScroll);
    return () => scroller.removeEventListener("scroll", onScroll);
  }, [history, pending, onMore]);
  const rows = useMemo(() => commitGraph(history?.items ?? []), [history]);
  const width = Math.max(24, ...rows.map(row => row.width * 14 + 10));
  return <div ref={container} className="git-history">{history && !history.items.length && <p>尚无提交历史。</p>}
    {history?.items.map((commit, index) => {
      const open = expanded === commit.id, current = detail?.commit.id === commit.id ? detail : null;
      return <Collapsible key={commit.id} open={open} onOpenChange={value => onExpand(commit.id, value)} className="git-commit" style={{ gridTemplateColumns: `${width}px minmax(0, 1fr)` }}>
        <Graph row={rows[index]!} width={width} expanded={open} />
        <CommitParents commit={commit} parent={current?.parent_id ?? ""} pending={pending} onParent={onParent}><div className="git-commit-body"><CollapsibleTrigger render={<Button variant="ghost" className="git-commit-trigger" />} title={`${commit.subject}\n${commit.author} · ${commit.date}\n${commit.id}${commit.parents.length > 1 ? "\n右键选择比较父提交" : ""}`}>
          <ChevronRight data-icon="inline-start" className={open ? "rotate-90" : undefined} /><span className="truncate">{commit.subject || "无标题"}</span>
          {commit.id === history.head && branch ? <Badge variant="secondary">{branch}</Badge> : <span className="git-commit-id">{commit.id.slice(0, 7)}</span>}
        </CollapsibleTrigger>
        <CollapsibleContent className="git-commit-detail">{!current ? pending ? <Loading /> : <p>无法读取提交详情，请收起后重试。</p> : <>
          <p className="git-commit-author">{commit.author} · {new Date(commit.date).toLocaleString()}</p>
          <div className="git-commit-files"><GitFiles files={current.files.map(path => ({ path }))} view={view} disabled={pending} onSelect={onCompare} />{!current.files.length && <p>此父提交下没有文件变更。</p>}</div>
        </>}</CollapsibleContent></div></CommitParents>
      </Collapsible>;
    })}
  </div>;
}
