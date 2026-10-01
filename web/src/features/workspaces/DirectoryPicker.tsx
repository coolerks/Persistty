import { useEffect, useRef, useState } from "react";
import { ArrowUp, House } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from "@/components/ui/dialog";
import { api, errorMessage } from "@/lib/api/client";
import type { DirectoryListing } from "@/lib/api/decoder";
import { FileTypeIcon } from "./FileTypeIcon";

export function DirectoryPicker({ open, onOpenChange, onSelect }: { open: boolean; onOpenChange(open: boolean): void; onSelect(path: string): void }) {
  const [path, setPath] = useState("");
  const [listing, setListing] = useState<DirectoryListing | null>(null);
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const request = useRef<AbortController | null>(null);
  const generation = useRef(0);

  async function browse(target: string) {
    request.current?.abort();
    const controller = new AbortController();
    request.current = controller;
    const serial = ++generation.current;
    setPending(true); setError(null);
    try {
      const result = await api.directory(target, controller.signal);
      if (!controller.signal.aborted && serial === generation.current) { setListing(result); setPath(result.path); }
    } catch (cause: unknown) {
      if (!controller.signal.aborted && serial === generation.current) setError(errorMessage(cause));
    } finally {
      if (serial === generation.current) setPending(false);
    }
  }

  useEffect(() => {
    if (open) { void browse(""); }
    const activeGeneration = generation;
    return () => { activeGeneration.current++; request.current?.abort(); request.current = null; };
  }, [open]);

  return <Dialog open={open} onOpenChange={onOpenChange}><DialogContent className="directory-picker">
    <DialogHeader><DialogTitle>选择服务器文件夹</DialogTitle></DialogHeader>
    <form className="flex min-w-0 gap-2" onSubmit={event => { event.preventDefault(); void browse(path); }}>
      <Input className="min-w-0 flex-1" aria-label="服务器绝对路径" value={path} onChange={event => setPath(event.target.value)} placeholder="/home/user/project" />
      <Button className="shrink-0" type="submit" disabled={pending}>打开</Button>
    </form>
    {error && <p role="alert" className="text-sm text-destructive">{error}</p>}
    <div className="flex min-w-0 items-center gap-2 border-b pb-2 text-sm text-muted-foreground">
      <Button size="icon" variant="ghost" aria-label="上级目录" title="上级目录" disabled={!listing || pending} onClick={() => listing && void browse(listing.parent)}><ArrowUp /></Button>
      <Button size="icon" variant="ghost" aria-label="主目录" title="主目录" disabled={pending} onClick={() => void browse("")}><House /></Button>
      <span className="min-w-0 flex-1 truncate font-mono" title={listing?.path}>{listing?.path ?? ""}</span>
    </div>
    <ul aria-label="服务器目录" className="max-h-72 min-h-36 overflow-y-auto">
      {listing?.items.map(name => <li key={name}><Button variant="ghost" className="flex h-9 w-full justify-start gap-2 truncate font-normal" onClick={() => void browse(listing.path === "/" ? `/${name}` : `${listing.path}/${name}`)}><FileTypeIcon path={name} kind="directory" /><span className="truncate">{name}</span></Button></li>)}
      {listing?.items.length === 0 && <li className="py-4 text-center text-sm text-muted-foreground">此目录下没有子文件夹</li>}
    </ul>
    <DialogFooter><Button variant="outline" onClick={() => onOpenChange(false)}>取消</Button><Button disabled={!listing || pending || path !== listing.path} onClick={() => { if (listing) { onSelect(listing.path); onOpenChange(false); } }}>选择此文件夹</Button></DialogFooter>
  </DialogContent></Dialog>;
}
