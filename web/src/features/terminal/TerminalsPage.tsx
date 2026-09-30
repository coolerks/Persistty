import { useParams } from "react-router";
import { TerminalWorkspace } from "./TerminalWorkspace";
import { TerminalRuntimeProvider } from "./TerminalRuntime";

export function TerminalsPage() {
  const { terminalId } = useParams();
  return <TerminalRuntimeProvider><main className="page-main terminal-page"><TerminalWorkspace key={terminalId ?? "all"} initialId={terminalId} /></main></TerminalRuntimeProvider>;
}
