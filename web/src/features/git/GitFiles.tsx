import { useMemo, useState } from "react";
import { ChevronRight, List, ListTree } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { FileTypeIcon } from "@/features/workspaces/FileTypeIcon";
import { gitFileTree, type FileNode } from "./file-tree";
import { FileHoverCard } from "./GitHoverCards";
import type { GitFileStat } from "@/lib/api/search-git-decoder";
export type GitFile = { path: string; status?: string; oldPath?: string; stat?: GitFileStat };
export type FileView = "list" | "tree";
export function GitFileViewToggle({ label, value, onChange }: { label: string; value: FileView; onChange(value: FileView): void }) {
  return <Tabs value={value} onValueChange={next => { if (next === "list" || next === "tree") onChange(next); }}>
    <TabsList className="git-view-tabs" aria-label={label}>
      <TabsTrigger value="list" aria-label={`${label}：列表`} title="列表"><List /></TabsTrigger>
      <TabsTrigger value="tree" aria-label={`${label}：文件树`} title="文件树"><ListTree /></TabsTrigger>
    </TabsList>
  </Tabs>;
}
function FileRow({ file, list, disabled, onSelect }: { file: GitFile; list: boolean; disabled: boolean; onSelect(path: string): void }) {
  const slash = file.path.lastIndexOf("/");
  const button = <Button variant="ghost" className="git-file-row" disabled={disabled} aria-label={file.path} title={file.stat ? undefined : file.oldPath ? `${file.oldPath} → ${file.path}` : file.path} onClick={() => onSelect(file.path)}>
    <FileTypeIcon path={file.path} /><span className="truncate">{file.path.slice(slash + 1)}</span>
    {list && slash >= 0 && <span className="git-file-parent truncate">{file.path.slice(0, slash)}</span>}
    {file.status && <span className="git-file-status" data-status={file.status} aria-label={`状态 ${file.status}`}>{file.status}</span>}
  </Button>;
  return file.stat ? <FileHoverCard stat={file.stat}>{button}</FileHoverCard> : button;
}
function Directory({ node, disabled, onSelect }: { node: FileNode; disabled: boolean; onSelect(path: string): void }) {
  const [open, setOpen] = useState(true);
  return <Collapsible open={open} onOpenChange={setOpen}>
    <CollapsibleTrigger aria-label={node.path} render={<Button variant="ghost" className="git-file-row git-directory" />}><ChevronRight data-icon="inline-start" className={open ? "rotate-90" : undefined} /><FileTypeIcon path={node.path} kind="directory" expanded={open} /><span className="truncate">{node.name}</span></CollapsibleTrigger>
    <CollapsibleContent className="git-tree-children"><Tree nodes={node.children} disabled={disabled} onSelect={onSelect} /></CollapsibleContent>
  </Collapsible>;
}
function Tree({ nodes, disabled, onSelect }: { nodes: FileNode[]; disabled: boolean; onSelect(path: string): void }) {
  return nodes.map(node => node.file ? <FileRow key={node.path} file={node.file} list={false} disabled={disabled} onSelect={onSelect} /> : <Directory key={node.path} node={node} disabled={disabled} onSelect={onSelect} />);
}
export function GitFiles({ files, view, disabled = false, onSelect }: { files: GitFile[]; view: FileView; disabled?: boolean; onSelect(path: string): void }) {
  const tree = useMemo(() => gitFileTree(files), [files]);
  return <div className="git-files" data-view={view}>{view === "list" ? files.map(file => <FileRow key={file.path} file={file} list disabled={disabled} onSelect={onSelect} />) : <Tree nodes={tree} disabled={disabled} onSelect={onSelect} />}</div>;
}
