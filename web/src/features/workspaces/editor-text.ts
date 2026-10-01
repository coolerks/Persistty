// The view uses LF; disk text keeps BOM and unchanged original separators.
export function editorText(raw: string): string { return raw.replace(/^\uFEFF/, "").replace(/\r\n|\r/g, "\n"); }
export function applyEditorText(raw: string, next: string, insertedSeparator?: string, stripBOM = true): string {
  const old = (stripBOM ? raw.replace(/^\uFEFF/, "") : raw).replace(/\r\n|\r/g, "\n"); next = next.replace(/\r\n|\r/g, "\n");
  if (old === next) return raw;
  let start = 0; while (start < old.length && start < next.length && old[start] === next[start]) start++;
  let end = 0; while (end < old.length - start && end < next.length - start && old[old.length - end - 1] === next[next.length - end - 1]) end++;
  function offset(index: number): number {
    let i = stripBOM && raw.startsWith("\uFEFF") ? 1 : 0;
    for (let n = 0; n < index; n++, i++) if (raw[i] === "\r" && raw[i + 1] === "\n") i++;
    return i;
  }
  const crlf = raw.match(/\r\n/g)?.length ?? 0;
  const lf = (raw.match(/\n/g)?.length ?? 0) - crlf;
  const cr = (raw.match(/\r/g)?.length ?? 0) - crlf;
  const separator = insertedSeparator ?? (crlf > lf && crlf >= cr ? "\r\n" : cr > lf ? "\r" : "\n");
  return raw.slice(0, offset(start)) + next.slice(start, next.length - end).replace(/\n/g, separator) + raw.slice(offset(old.length - end));
}

export type TextChange = { rangeOffset: number; rangeLength: number; text: string };
// Monaco gives simultaneous edits in offsets of the previous LF view. Map every
// boundary once, then apply from right to left so untouched separators survive.
export function applyEditorChanges(raw: string, changes: readonly TextChange[]): string {
  const sorted = [...changes].sort((a, b) => b.rangeOffset - a.rangeOffset);
  const points = [...new Set(sorted.flatMap(change => [change.rangeOffset, change.rangeOffset + change.rangeLength]))].sort((a, b) => a - b);
  const offsets = new Map<number, number>(); let index = raw.startsWith("\uFEFF") ? 1 : 0, normalized = 0;
  for (const point of points) { while (normalized < point && index < raw.length) { if (raw[index] === "\r" && raw[index + 1] === "\n") index++; index++; normalized++; } offsets.set(point, index); }
  const crlf = raw.match(/\r\n/g)?.length ?? 0, lf = (raw.match(/\n/g)?.length ?? 0) - crlf, cr = (raw.match(/\r/g)?.length ?? 0) - crlf;
  const separator = crlf > lf && crlf >= cr ? "\r\n" : cr > lf ? "\r" : "\n";
  for (const change of sorted) { const start = offsets.get(change.rangeOffset) ?? 0, end = offsets.get(change.rangeOffset + change.rangeLength) ?? raw.length; raw = raw.slice(0, start) + applyEditorText(raw.slice(start, end), change.text, separator, false) + raw.slice(end); }
  return raw;
}
