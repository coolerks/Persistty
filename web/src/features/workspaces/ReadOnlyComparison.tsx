import { lazy, Suspense } from "react";
import { Loading } from "@/components/Feedback";
import { editorText } from "./editor-text";
const DesktopDiff = lazy(() => import("./DesktopDiff"));
export function ReadOnlyComparison({ original, modified, path, mobile }: { original: string; modified: string; path: string; mobile: boolean }) {
  return mobile ? <div className="mobile-comparison"><section aria-label="原内容"><h3>原内容</h3><pre>{editorText(original)}</pre></section><section aria-label="比较内容"><h3>比较内容</h3><pre>{editorText(modified)}</pre></section></div> : <div className="draft-diff"><Suspense fallback={<Loading />}><DesktopDiff original={original} modified={modified} language={path.endsWith(".json") ? "json" : "plaintext"} /></Suspense></div>;
}
