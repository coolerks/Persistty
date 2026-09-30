import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { fileLanguages, isFileLanguage, languageLabel } from "./file-language";

export function LanguageSelect({ mode, detected, onChange }: { mode: string | undefined; detected: string; onChange(mode: string | undefined): void }) {
  return <Select value={mode ?? "auto"} onValueChange={value => {
    if (value === "auto") onChange(undefined);
    else if (isFileLanguage(value)) onChange(value);
  }}>
    <SelectTrigger size="sm" aria-label="语言模式" className="editor-language-select" title={mode ? languageLabel(mode) : `自动识别：${languageLabel(detected)}`}><SelectValue>{value => value === "auto" ? `自动：${languageLabel(detected)}` : languageLabel(String(value))}</SelectValue></SelectTrigger>
    <SelectContent align="end" alignItemWithTrigger={false} className="max-h-80 w-auto max-w-[calc(100vw-2rem)]"><SelectGroup>
      <SelectItem value="auto">自动识别</SelectItem>
      {[...fileLanguages].sort((a, b) => languageLabel(a.id).localeCompare(languageLabel(b.id))).map(language => <SelectItem key={language.id} value={language.id}>{languageLabel(language.id)}</SelectItem>)}
    </SelectGroup></SelectContent>
  </Select>;
}
