export type ThemeMode = "system" | "light" | "dark";
export const themeKey = "persistty.theme.v1";
export function isThemeMode(value: unknown): value is ThemeMode { return value === "system" || value === "light" || value === "dark"; }
export function readTheme(): ThemeMode {
  try { const value = localStorage.getItem(themeKey); return isThemeMode(value) ? value : "system"; }
  catch { return "system"; }
}
export function applyTheme(mode: ThemeMode) {
  const dark = mode === "dark" || (mode === "system" && window.matchMedia("(prefers-color-scheme: dark)").matches);
  document.documentElement.classList.toggle("dark", dark);
  document.documentElement.style.colorScheme = dark ? "dark" : "light";
}
