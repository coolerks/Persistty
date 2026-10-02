import { useEffect, useState } from "react";
import { Monitor, Sun, Moon } from "lucide-react";
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { applyTheme, isThemeMode, readTheme, themeKey } from "./theme";

export function ThemeSelect() {
  const [mode, setMode] = useState(readTheme);
  const [storageError, setStorageError] = useState(false);
  useEffect(() => {
    applyTheme(mode);
    const media = window.matchMedia("(prefers-color-scheme: dark)");
    const update = () => applyTheme(mode);
    media.addEventListener("change", update);
    return () => media.removeEventListener("change", update);
  }, [mode]);
  return <div className="flex flex-col gap-1">
    <Select value={mode} onValueChange={value => {
      if (!isThemeMode(value)) return;
      setMode(value);
      try { localStorage.setItem(themeKey, value); setStorageError(false); }
      catch { setStorageError(true); }
    }}>
      <SelectTrigger aria-label="主题" title={`主题：${mode === "light" ? "浅色" : mode === "dark" ? "深色" : "跟随系统"}`} className="theme-icon-trigger"><SelectValue>{value => <>{value === "light" ? <Sun /> : value === "dark" ? <Moon /> : <Monitor />}<span className="sr-only">{value === "light" ? "浅色" : value === "dark" ? "深色" : "跟随系统"}</span></>}</SelectValue></SelectTrigger>
      <SelectContent align="end" alignItemWithTrigger={false}><SelectGroup>
        <SelectItem value="system"><Monitor />跟随系统</SelectItem>
        <SelectItem value="light"><Sun />浅色</SelectItem>
        <SelectItem value="dark"><Moon />深色</SelectItem>
      </SelectGroup></SelectContent>
    </Select>
    {storageError && <span role="status" className="text-xs text-destructive">主题偏好未保存</span>}
  </div>;
}
