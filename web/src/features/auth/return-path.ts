export function safeReturnPath(value: string | null): string {
  if (value === "/projects" || value === "/terminals") return value;
  if (value && /^\/(projects|terminals)\/[A-Za-z0-9_-]{1,128}$/.test(value)) return value;
  return "/projects";
}
