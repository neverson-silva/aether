import { ArrowSquareOut, ArrowsClockwise, Play, RocketLaunch, Stop, TerminalWindow } from "@phosphor-icons/react";
import type { Icon as DesignIcon } from "@aether/design-system";
import { Button, Switch } from "@aether/design-system";

const designIcon = (icon: typeof RocketLaunch) => icon as unknown as DesignIcon;

export function ServiceControlPanel({
  running,
  autodeploy,
  autodeployDisabled,
  restarting,
  stopping,
  starting,
  onOpenTerminal,
  onVisit,
  onDeploy,
  onRestart,
  onStop,
  onStart,
  onAutodeployChange,
  canManageSource,
}: {
  running: boolean;
  autodeploy: boolean;
  autodeployDisabled: boolean;
  restarting: boolean;
  stopping: boolean;
  starting: boolean;
  onOpenTerminal: () => void;
  onVisit: () => void;
  onDeploy: () => void;
  onRestart: () => void;
  onStop: () => void;
  onStart: () => void;
  onAutodeployChange: (checked: boolean) => void;
  canManageSource: boolean;
}) {
  return (
    <section className="mb-8 rounded-xl border border-outline-variant bg-surface-container-lowest p-6">
      <div className="mb-6 flex items-center justify-between gap-4">
        <div><h3 className="font-headline-sm text-headline-sm text-on-surface">Deploy Settings</h3><p className="text-body-sm text-on-surface-variant">Deploy, rebuild or control this service</p></div>
        <div className="flex flex-wrap items-center justify-end gap-3">
          <Button variant="ghost" icon={designIcon(TerminalWindow)} onClick={onOpenTerminal}>Open Terminal</Button>
          <Button variant="ghost" icon={designIcon(ArrowSquareOut)} onClick={onVisit}>Visit URL</Button>
        </div>
      </div>
      <div className="flex flex-wrap items-center gap-3">
        <Button icon={designIcon(RocketLaunch)} onClick={onDeploy}>Deploy</Button>
        <Button variant="secondary" icon={designIcon(ArrowsClockwise)} loading={restarting} onClick={onRestart}>Restart</Button>
        {running ? <Button variant="danger" icon={designIcon(Stop)} loading={stopping} onClick={onStop}>Stop</Button> : <Button variant="success" icon={designIcon(Play)} loading={starting} onClick={onStart}>Start</Button>}
        {canManageSource ? <div className="ml-auto flex items-center gap-3"><span className="text-body-sm text-on-surface-variant">Automatic deploys</span><Switch ariaLabel="Automatic deploys" checked={autodeploy} disabled={autodeployDisabled} loading={autodeployDisabled} onCheckedChange={onAutodeployChange} /></div> : null}
      </div>
    </section>
  );
}
