import type { GitFile } from "./GitFiles";
export type FileNode = { name: string; path: string; file?: GitFile; children: FileNode[] };
export function gitFileTree(files: GitFile[]): FileNode[] {
  const root: FileNode = { name: "", path: "", children: [] };
  for (const file of files) {
    const parts = file.path.split("/"); let node = root;
    parts.forEach((name, index) => {
      const path = parts.slice(0, index + 1).join("/");
      let child = node.children.find(item => item.path === path);
      if (!child) { child = { name, path, children: [] }; node.children.push(child); }
      if (index === parts.length - 1) child.file = file;
      node = child;
    });
  }
  function sort(nodes: FileNode[]): FileNode[] { return nodes.sort((a, b) => Number(!!a.file) - Number(!!b.file) || a.name.localeCompare(b.name)).map(node => ({ ...node, children: sort(node.children) })); }
  return sort(root.children);
}
