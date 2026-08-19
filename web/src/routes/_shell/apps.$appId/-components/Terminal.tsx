import { TerminalShell } from "../../../../components/TerminalShell";

export function Terminal({ appID }: { appID: string }) {
  return <TerminalShell wsUrl={`/api/v1/ws/terminal/${appID}`} />;
}
