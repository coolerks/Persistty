import generated from "@/generated/material-icons.json";
import { languageForFile } from "./file-language";

type ThemeMapping = {
  file?: string; folder?: string; folderExpanded?: string; rootFolder?: string; rootFolderExpanded?: string;
  fileNames?: Record<string, string>; fileExtensions?: Record<string, string>; languageIds?: Record<string, string>;
  folderNames?: Record<string, string>; folderNamesExpanded?: Record<string, string>;
  rootFolderNames?: Record<string, string>; rootFolderNamesExpanded?: Record<string, string>;
};
function normalize(mapping: ThemeMapping): ThemeMapping {
  const result = { ...mapping };
  for (const key of ["fileNames", "fileExtensions", "languageIds", "folderNames", "folderNamesExpanded", "rootFolderNames", "rootFolderNamesExpanded"] as const) {
    const entries = mapping[key];
    if (entries) result[key] = Object.fromEntries(Object.entries(entries).map(([name, id]) => [name.toLowerCase(), id]));
  }
  return result;
}
const base = normalize(generated);
const light = normalize(generated.light);
export type FileIconOptions = { path: string; kind?: "file" | "directory" | "root"; expanded?: boolean; theme?: "light" | "dark" };

function lookup(mapping: ThemeMapping, override: ThemeMapping, key: keyof ThemeMapping, name: string): string | undefined {
  const values = mapping[key]; const overrides = override[key];
  return (typeof overrides === "object" && Object.hasOwn(overrides, name) ? overrides[name] : undefined) ?? (typeof values === "object" && Object.hasOwn(values, name) ? values[name] : undefined);
}
function fallback(mapping: ThemeMapping, override: ThemeMapping, key: "file" | "folder" | "folderExpanded" | "rootFolder" | "rootFolderExpanded"): string {
  return override[key] ?? mapping[key] ?? generated.file;
}

export function defaultFileIcon({ kind = "file", expanded = false, theme = "light" }: FileIconOptions): string {
  const override = theme === "light" ? light : {};
  return fallback(base, override, kind === "file" ? "file" : kind === "root" ? expanded ? "rootFolderExpanded" : "rootFolder" : expanded ? "folderExpanded" : "folder");
}

export function fileIconFor(options: FileIconOptions): string {
  const { path, kind = "file", expanded = false, theme = "light" } = options;
  const name = path.split("/").at(-1)?.toLowerCase() ?? "";
  const override = theme === "light" ? light : {};
  if (kind !== "file") {
    const key = kind === "root" ? expanded ? "rootFolderNamesExpanded" : "rootFolderNames" : expanded ? "folderNamesExpanded" : "folderNames";
    return lookup(base, override, key, name) ?? defaultFileIcon(options);
  }
  const exact = lookup(base, override, "fileNames", name);
  if (exact) return exact;
  // Leftmost dot first gives the longest compound suffix; includes dotfiles such as .env.
  for (let dot = name.indexOf("."); dot >= 0; dot = name.indexOf(".", dot + 1)) {
    const matched = lookup(base, override, "fileExtensions", name.slice(dot + 1));
    if (matched) return matched;
  }
  return lookup(base, override, "languageIds", languageForFile(path)) ?? defaultFileIcon(options);
}

const definitions: Record<string, { iconPath: string }> = generated.iconDefinitions;
export function fileIconURL(id: string): string {
  const definition = definitions[id] ?? definitions[generated.file];
  return `${import.meta.env.BASE_URL}${definition?.iconPath ?? "material-icons/file.svg"}`;
}
