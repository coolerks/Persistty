import { useRef, useState, type ReactNode } from "react";
import { ChevronRight } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import { ResizableHandle, ResizablePanel, ResizablePanelGroup } from "@/components/ui/resizable";
import { cn } from "@/lib/utils";
function Pane({ title, open, onOpenChange, actions, children }: { title: string; open: boolean; onOpenChange(open: boolean): void; actions: ReactNode; children: ReactNode }) {
  return <Collapsible open={open} onOpenChange={onOpenChange} className={cn("git-section", open && "git-section-open")}>
    <div className="git-section-heading"><CollapsibleTrigger render={<Button variant="ghost" size="sm" className="git-section-trigger" />} aria-label={title}><ChevronRight data-icon="inline-start" className={open ? "rotate-90" : undefined} />{title}</CollapsibleTrigger>{actions}</div>
    <CollapsibleContent className="git-section-content" role="region" aria-label={`${title}内容`}>{children}</CollapsibleContent>
  </Collapsible>;
}
export function GitSections({ changes, history, changesActions, historyActions }: { changes: ReactNode; history: ReactNode; changesActions: ReactNode; historyActions: ReactNode }) {
  const [changesOpen, setChangesOpen] = useState(true), [historyOpen, setHistoryOpen] = useState(true);
  const layout = useRef<Record<string, number>>({ changes: 50, history: 50 });
  const top = <Pane title="变更" open={changesOpen} onOpenChange={setChangesOpen} actions={changesActions}>{changes}</Pane>;
  const bottom = <Pane title="历史" open={historyOpen} onOpenChange={setHistoryOpen} actions={historyActions}>{history}</Pane>;
  return <div className="git-sections">{changesOpen && historyOpen ? <ResizablePanelGroup orientation="vertical" defaultLayout={layout.current} onLayoutChanged={value => { layout.current = value; }}>
    <ResizablePanel id="changes" minSize="80px">{top}</ResizablePanel><ResizableHandle aria-label="调整变更与历史高度" /><ResizablePanel id="history" minSize="80px">{bottom}</ResizablePanel>
  </ResizablePanelGroup> : <>{top}{bottom}</>}</div>;
}
