import metadata from "@/generated/languages.json";

export type LanguageDefinition = { id: string; aliases?: string[]; extensions?: string[]; filenames?: string[]; firstLine?: string };
export const fileLanguages: readonly LanguageDefinition[] = metadata.languages;
const ids = new Set(fileLanguages.map(language => language.id));
const filenames = new Map<string, string>();
const extensions = new Map<string, string>();
for (const language of fileLanguages) {
  for (const name of language.filenames ?? []) if (!filenames.has(name.toLowerCase())) filenames.set(name.toLowerCase(), language.id);
  for (const suffix of language.extensions ?? []) if (!extensions.has(suffix.toLowerCase())) extensions.set(suffix.toLowerCase(), language.id);
}
const suffixes = [...extensions.keys()].sort((a, b) => b.length - a.length);
const firstLines = fileLanguages.flatMap(language => language.firstLine ? [{ id: language.id, pattern: new RegExp(language.firstLine) }] : []);

export function isFileLanguage(value: unknown): value is string { return typeof value === "string" && ids.has(value); }
export function languageLabel(id: string): string { return fileLanguages.find(language => language.id === id)?.aliases?.[0] ?? id; }

export function languageForFile(path: string, content = ""): string {
  const name = path.split("/").at(-1)?.toLowerCase() ?? "";
  const exact = filenames.get(name);
  if (exact) return exact;
  for (const suffix of suffixes) if (name.endsWith(suffix)) return extensions.get(suffix) ?? "plaintext";
  const firstLine = content.slice(0, 1024).split(/\r?\n/, 1)[0] ?? "";
  return firstLines.find(language => language.pattern.test(firstLine))?.id ?? "plaintext";
}
