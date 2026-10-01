export type LineChange = { line: number; kind: "added" | "modified" | "deleted" };
export function lineChanges(original: string, modified: string): LineChange[] {
  if (original === modified) return [];
  const before = original.split("\n"), after = modified.split("\n");
  let prefix = 0, suffix = 0;
  while (prefix < before.length && prefix < after.length && before[prefix] === after[prefix]) prefix++;
  while (suffix < before.length - prefix && suffix < after.length - prefix && before[before.length - 1 - suffix] === after[after.length - 1 - suffix]) suffix++;
  const a = before.slice(prefix, before.length - suffix), b = after.slice(prefix, after.length - suffix);
  const changes = new Map<number, LineChange["kind"]>();
  const mark = (line: number, kind: LineChange["kind"]) => { line = Math.min(after.length, Math.max(1, line)); const prior = changes.get(line); changes.set(line, prior && prior !== kind ? "modified" : kind); };
  if (a.length * b.length > 250000 || a.length + b.length > 20000) {
    for (let line = 0; line < b.length; line++) mark(prefix + line + 1, a.length ? "modified" : "added");
    if (!b.length) mark(prefix + 1, "deleted");
  } else {
    const width = b.length + 1, table = new Uint32Array((a.length + 1) * width);
    for (let i = a.length - 1; i >= 0; i--) for (let j = b.length - 1; j >= 0; j--) table[i * width + j] = a[i] === b[j] ? table[(i + 1) * width + j + 1]! + 1 : Math.max(table[(i + 1) * width + j]!, table[i * width + j + 1]!);
    let i = 0, j = 0;
    while (i < a.length || j < b.length) {
      if (i < a.length && j < b.length && a[i] === b[j]) { i++; j++; }
      else if (i < a.length && (j === b.length || table[(i + 1) * width + j]! >= table[i * width + j + 1]!)) { mark(prefix + j + 1, "deleted"); i++; }
      else { mark(prefix + j + 1, "added"); j++; }
    }
  }
  return [...changes].map(([line, kind]) => ({ line, kind }));
}
