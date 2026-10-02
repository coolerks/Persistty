import type { ReactElement } from "react";
import { ExternalLink, GitCommitHorizontal } from "lucide-react";
import { HoverCard, HoverCardContent, HoverCardTrigger } from "@/components/ui/hover-card";
import { Button } from "@/components/ui/button";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Loading } from "@/components/Feedback";
import type { CommitDetail, GitFileStat } from "@/lib/api/search-git-decoder";
import type { searchGitAPI } from "@/lib/api/search-git-client";
type Commit = Awaited<ReturnType<typeof searchGitAPI.log>>["items"][number];
const statusNames = { A: "新增", M: "修改", D: "删除", T: "类型变更", R: "重命名", C: "复制" };
function Counts({ added, removed }: { added: number | null; removed: number | null }) {
  return added === null || removed === null ? <span>二进制文件，不统计文本行数</span> : <><span className="git-added">{added} 行插入 (+)</span><span> · </span><span className="git-removed">{removed} 行删除 (-)</span></>;
}
export function CommitHoverCard({ commit, detail, error, onOpenChange, children }: { commit: Commit; detail: CommitDetail | null; error: string | null; onOpenChange(open: boolean): void; children: ReactElement }) {
  const counts = detail?.stats.reduce((total, stat) => ({ added: total.added + (stat.additions ?? 0), removed: total.removed + (stat.deletions ?? 0), binary: total.binary + (stat.additions === null ? 1 : 0) }), { added: 0, removed: 0, binary: 0 });
  return <HoverCard onOpenChange={onOpenChange}><HoverCardTrigger render={children} delay={450} closeDelay={200} /><HoverCardContent className="git-hover-card" side="right" align="start" role="region" aria-label="提交信息">
    <p><strong>{commit.author}</strong> · <time dateTime={commit.date}>{new Date(commit.date).toLocaleString("zh-CN")}</time></p>
    <div className="git-hover-message">{detail?.message || commit.subject || "无提交消息"}</div>
    {error ? <Alert variant="destructive"><AlertDescription>{error}</AlertDescription></Alert> : counts && detail ? <p>已更改 {detail.stats.length} 个文件，<Counts added={counts.added} removed={counts.removed} />{counts.binary > 0 && `；另有 ${counts.binary} 个二进制文件`}</p> : <Loading />}
    {detail && <p className="git-hover-baseline">比较基线：{detail.parent_id ? detail.parent_id.slice(0, 12) : "空树（根提交）"}</p>}
    <div className="git-hover-footer"><GitCommitHorizontal aria-hidden="true" /><code>{commit.id}</code>{detail?.github_url && <Button variant="link" size="sm" nativeButton={false} role="link" render={<a href={detail.github_url} target="_blank" rel="noopener noreferrer" />}><ExternalLink data-icon="inline-start" />在 GitHub 上打开</Button>}</div>
  </HoverCardContent></HoverCard>;
}
export function FileHoverCard({ stat, children }: { stat: GitFileStat; children: ReactElement }) {
  return <HoverCard><HoverCardTrigger render={children} delay={450} closeDelay={200} /><HoverCardContent className="git-hover-card" side="right" align="start" role="region" aria-label="文件变更信息">
    <p className="break-all"><strong>{stat.path}</strong></p>{stat.old_path && <p className="break-all">原路径：{stat.old_path}</p>}
    <p>{statusNames[stat.status]} · <Counts added={stat.additions} removed={stat.deletions} /></p>
  </HoverCardContent></HoverCard>;
}
