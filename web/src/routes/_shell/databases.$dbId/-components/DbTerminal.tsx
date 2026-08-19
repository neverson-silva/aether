import { TerminalShell } from "../../../../components/TerminalShell";

export function DbTerminal({ dbId }: { dbId: string }) {
  return <TerminalShell wsUrl={`/api/v1/ws/db-terminal/${dbId}`} />;
}
