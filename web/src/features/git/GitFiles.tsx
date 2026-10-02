import { useMemo, useState } from "react";
import { ChevronRight, List, ListTree } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { FileTypeIcon } from "@/features/workspaces/FileTypeIcon";
import { gitFileTree, type FileNode } from "./file-tree";
export type GitFile = { path: string; status?: string; oldPath?: string };
export type FileView = "list" | "tree";
export function GitFileViewToggle({ label, value, onChange }: { label: string; value: FileView; onChange(value: FileView): void }) {
  return <ToggleGroup aria-label={label} size="sm" spacing={0} value={[value]} onValueChange={values => { const next = values[0]; if (next === "list" || next === "tree") onChange(next); }}>
    <ToggleGroupItem value="list" aria-label={`${label}：列表`} title="列表"><List /></ToggleGroupItem>
    <ToggleGroupItem value="tree" aria-label={`${label}：文件树`} title="文件树"><ListTree /></ToggleGroupItem>
  </ToggleGroup>;
}
function FileRow({ file, list, disabled, onSelect }: { file: GitFile; list: boolean; disabled: boolean; onSelect(path: string): void }) {
  const slash = file.path.lastIndexOf("/");
  return <Button variant="ghost" className="git-file-row" disabled={disabled} aria-label={file.path} title={file.oldPath ? `${file.oldPath} → ${file.path}` : file.path} onClick={() => onSelect(file.path)}>
    <FileTypeIcon path={file.path} /><span className="truncate">{file.path.slice(slash + 1)}</span>
    {list && slash >= 0 && <span className="git-file-parent truncate">{file.path.slice(0, slash)}</span>}
    {file.status && <span className="git-file-status" data-status={file.status} aria-label={`状态 ${file.status}`}>{file.status}</span>}
  </Button>;
}
function Directory({ node, disabled, onSelect }: { node: FileNode; disabled: boolean; onSelect(path: string): void }) {
  const [open, setOpen] = useState(true);
  return <Collapsible open={open} onOpenChange={setOpen}>
    <CollapsibleTrigger render={<Button variant="ghost" className="git-file-row git-directory" />}><ChevronRight data-icon="inline-start" className={open ? "rotate-90" : undefined} /><FileTypeIcon path={node.path} kind="directory" expanded={open} /><span className="truncate">{node.name}</span></CollapsibleTrigger>
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
