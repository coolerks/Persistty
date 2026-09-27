import { AlertCircle, RefreshCw } from "lucide-react";
import { Alert, AlertTitle, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { errorMessage } from "@/lib/api/client";

export function Loading() {
  return <div role="status" aria-label="正在加载" className="flex flex-col gap-3 py-8"><Skeleton className="h-7 w-36" /><Skeleton className="h-12 w-full" /><Skeleton className="h-12 w-full" /></div>;
}
export function Failure({ error, retry }: { error: unknown; retry(): void }) {
  return <Alert variant="destructive"><AlertCircle /><AlertTitle>读取失败</AlertTitle><AlertDescription>
    <p>{errorMessage(error)}</p><Button variant="outline" size="sm" onClick={retry}><RefreshCw data-icon="inline-start" />重试</Button>
  </AlertDescription></Alert>;
}
