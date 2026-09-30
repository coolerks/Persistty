import { useState } from "react";
import { useEffectiveTheme } from "@/features/settings/use-effective-theme";
import { defaultFileIcon, fileIconFor, fileIconURL, type FileIconOptions } from "./file-icons";

export function FileTypeIcon({ path, kind = "file", expanded = false }: Omit<FileIconOptions, "theme">) {
  const theme = useEffectiveTheme();
  const options = { path, kind, expanded, theme };
  const id = fileIconFor(options);
  const [failed, setFailed] = useState<string | null>(null);
  const actual = failed === id ? defaultFileIcon(options) : id;
  return <img className="size-4 shrink-0" data-file-icon={actual} data-icon="inline-start" src={fileIconURL(actual)} alt="" aria-hidden="true" draggable={false} onError={actual === id && failed !== id ? () => setFailed(id) : undefined} />;
}
