import { useEffect, useRef, useState } from "react";
import { Search, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Dialog, DialogContent, DialogClose, DialogDescription, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { errorMessage } from "@/lib/api/client";
import { searchGitAPI } from "@/lib/api/search-git-client";
import type { FileNames } from "@/lib/api/search-git-decoder";
import type { Project } from "@/lib/api/decoder";
import type { OpenFile } from "./workspace-view";
import { FileTypeIcon } from "./FileTypeIcon";

export function FileQuickOpen({ project, onOpen }: { project: Project; onOpen(file: OpenFile): void }) {
  const [open, setOpen] = useState(false), [query, setQuery] = useState("");
  const [response, setResponse] = useState<{ key: string; data?: FileNames; error?: string } | null>(null);
  const [selected, setSelected] = useState(0);
  const results = useRef<HTMLDivElement>(null);
  const keyword = query.trim(), key = `${project.id}:${project.version}:${keyword}`;
  // Project polling creates fresh arrays without changing folder identities.
  // That refresh must not cancel a slow search with the same project version.
  const folderIDs = project.folders.map(folder => folder.id).join("\0");
  const current = response?.key === key ? response : null;
  const items = current?.data?.items ?? [];
  const valid = new TextEncoder().encode(keyword).length <= 256;
  useEffect(() => {
    const keyboard = (event: KeyboardEvent) => {
      if (!(event.ctrlKey || event.metaKey) || event.shiftKey || event.altKey || event.isComposing || event.key.toLowerCase() !== "p" || document.querySelector('[data-slot="dialog-content"]')) return;
      event.preventDefault(); setQuery(""); setResponse(null); setSelected(0); setOpen(true);
    };
    window.addEventListener("keydown", keyboard);
    return () => window.removeEventListener("keydown", keyboard);
  }, []);
  useEffect(() => {
    if (!open || !keyword || !valid) return;
    const controller = new AbortController();
    const folders = new Set(folderIDs.split("\0"));
    const timer = setTimeout(() => {
      void searchGitAPI.fileNames(project.id, project.version, keyword, controller.signal).then(data => {
        if (controller.signal.aborted) return;
        if (data.project_version !== project.version || data.items.some(item => !folders.has(item.folder_id))) throw new Error("项目配置已变化，请刷新项目后重试。");
        setResponse({ key, data });
      }).catch(cause => { if (!controller.signal.aborted) setResponse({ key, error: errorMessage(cause) }); });
    }, 250);
    return () => { clearTimeout(timer); controller.abort(); };
  }, [open, keyword, valid, key, project.id, project.version, folderIDs]);
  function choose(index: number) {
    const item = items[index]; if (!item) return;
    onOpen({ folderId: item.folder_id, path: item.path }); setOpen(false);
  }
  function select(index: number) {
    setSelected(index); results.current?.querySelectorAll("button")[index]?.scrollIntoView({ block: "nearest" });
  }
  return <Dialog open={open} onOpenChange={value => { setOpen(value); if (value) { setQuery(""); setResponse(null); setSelected(0); } }}>
    <DialogTrigger render={<Button variant="outline" className="file-quick-trigger" aria-label="按名称搜索文件" />}><Search /><span>搜索文件</span><kbd>⌘ / Ctrl P</kbd></DialogTrigger>
    <DialogContent className="file-quick-dialog" initialFocus={true} showCloseButton={false}>
      <DialogHeader className="sr-only"><DialogTitle>按名称搜索文件</DialogTitle><DialogDescription>搜索项目中的文件名，方向键选择，Enter 打开文件。</DialogDescription></DialogHeader>
      <div className="file-quick-input-row"><Input className="pr-10" aria-label="文件名关键词" placeholder="输入文件名关键词…" value={query} maxLength={256} onChange={event => { setQuery(event.target.value); setResponse(null); setSelected(0); }} onKeyDown={event => {
        if (event.nativeEvent.isComposing || !items.length) return;
        if (event.key === "ArrowDown" || event.key === "ArrowUp") { event.preventDefault(); select((selected + (event.key === "ArrowDown" ? 1 : items.length - 1)) % items.length); }
        if (event.key === "Enter") { event.preventDefault(); choose(selected); }
      }} /><DialogClose render={<Button variant="ghost" size="icon-sm" aria-label="关闭搜索" title="关闭搜索" />}><X /></DialogClose></div>
      <div ref={results} className="file-quick-results" aria-label="文件搜索结果" aria-busy={!!keyword && valid && !current}>
        {items.map((item, index) => <Button key={`${item.folder_id}:${item.path}`} variant="ghost" className="file-quick-result" aria-label={`打开 ${item.path}`} aria-current={index === selected} onFocus={() => setSelected(index)} onClick={() => choose(index)}>
          <FileTypeIcon path={item.path} /><span><span>{item.path.split("/").at(-1)}</span><span className="file-quick-path">{project.folders.find(folder => folder.id === item.folder_id)?.path}/{item.path}</span></span>
        </Button>)}
        {!keyword ? <p className="file-quick-status">输入关键词，在当前项目中按文件名搜索。</p> : !valid ? <p role="alert" className="file-quick-status">关键词过长，请缩短后重试。</p> : current?.error ? <p role="alert" className="file-quick-status">{current.error}</p> : !current ? <p role="status" className="file-quick-status">正在搜索…</p> : !items.length ? <p role="status" className="file-quick-status">没有匹配的文件</p> : null}
      </div>
    </DialogContent>
  </Dialog>;
}
