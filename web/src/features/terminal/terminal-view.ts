import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";
import type { Terminal } from "@/lib/api/decoder";

type TerminalView = {
  dismissed: Record<string, true>;
  dismiss(terminal: Terminal): void;
};

// This is a browser view preference, never a request to delete server metadata.
export const useTerminalView = create<TerminalView>()(persist(set => ({
  dismissed: {},
  dismiss: terminal => {
    if (terminal.state === "terminated") set(state => ({ dismissed: { ...state.dismissed, [terminal.id]: true } }));
  },
}), { name: "persistty.terminal-view.v1", storage: createJSONStorage(() => localStorage), partialize: state => ({ dismissed: state.dismissed }) }));

export function terminalVisible(terminal: Terminal, dismissed: Record<string, true>): boolean {
  return terminal.state !== "terminated" || !dismissed[terminal.id];
}
