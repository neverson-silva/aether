import {
  ArrowSquareOut,
  ArrowsClockwise,
  Play,
  RocketLaunch,
  Stop,
  TerminalWindow,
} from "@phosphor-icons/react";
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
    <section className="mb-lg overflow-hidden rounded-2xl border border-outline-variant bg-surface-container shadow-md">
      <div className="flex flex-col justify-between gap-md border-b border-outline-variant bg-surface-container-low/70 p-lg sm:flex-row sm:items-center">
        <div>
          <p className="font-label-caps text-label-caps text-primary">
            Operations
          </p>
          <h3 className="mt-xs font-headline-sm text-headline-sm text-on-surface">
            Deploy settings
          </h3>
          <p className="mt-xs text-body-sm text-on-surface-variant">
            Ship a new version or control the running service.
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-sm sm:justify-end">
          <Button
            variant="ghost"
            icon={designIcon(TerminalWindow)}
            onClick={onOpenTerminal}
          >
            Open Terminal
          </Button>
          <Button
            variant="ghost"
            icon={designIcon(ArrowSquareOut)}
            onClick={onVisit}
          >
            Visit URL
          </Button>
        </div>
      </div>
      <div className="flex flex-col gap-md p-lg sm:flex-row sm:items-center">
        <div className="flex flex-wrap items-center gap-sm">
          <Button icon={designIcon(RocketLaunch)} onClick={onDeploy}>
            Deploy
          </Button>
          <Button
            variant="secondary"
            icon={designIcon(ArrowsClockwise)}
            loading={restarting}
            onClick={onRestart}
          >
            Restart
          </Button>
          {running ? (
            <Button
              variant="danger"
              icon={designIcon(Stop)}
              loading={stopping}
              onClick={onStop}
            >
              Stop
            </Button>
          ) : (
            <Button
              variant="success"
              icon={designIcon(Play)}
              loading={starting}
              onClick={onStart}
            >
              Start
            </Button>
          )}
        </div>
        {canManageSource ? (
          <div className="flex items-center justify-between gap-md rounded-xl border border-outline-variant bg-surface-container-low px-md py-sm sm:ml-auto">
            <span>
              <span className="block text-body-sm font-semibold text-on-surface">
                Automatic deploys
              </span>
              <span className="block text-label-caps text-on-surface-variant">
                Deploy on every push
              </span>
            </span>
            <Switch
              ariaLabel="Automatic deploys"
              checked={autodeploy}
              disabled={autodeployDisabled}
              loading={autodeployDisabled}
              onCheckedChange={onAutodeployChange}
            />
          </div>
        ) : null}
      </div>
    </section>
  );
}
