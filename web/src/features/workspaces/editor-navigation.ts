import { create } from "zustand";
import type { FileVersion } from "@/lib/api/decoder";
import type { OpenFile } from "./workspace-view";
export type EditorLocation = { projectId: string; file: OpenFile; version: FileVersion; line: number; column: number; endColumn: number; id: string };
export const useEditorLocation = create<{ location: EditorLocation | null }>(() => ({ location: null }));
