import { Dialog, LogViewer } from "@aether/design-system";
import { useDeploymentLog } from "../../../../hooks";
import { toDeploymentLogLines } from "../../../../lib/deployment-log-lines";

export function DeploymentLogModal({ appId, deploymentId, onClose }: { appId: string; deploymentId: string | null; onClose: () => void }) {
  const log = useDeploymentLog(appId, deploymentId);
  const lines = log.data ? toDeploymentLogLines(log.data.content, log.data.error) : [];

  return (
    <Dialog open={!!deploymentId} onOpenChange={(open) => { if (!open) onClose(); }} title={`Deployment #${log.data?.number ?? ""} log`} size="lg" overflow="hidden" trigger={<button type="button" className="hidden" aria-hidden="true" tabIndex={-1} />}>
      <div className="min-h-0 overflow-hidden pt-1">
        <div className="mt-1">
          <LogViewer lines={lines} loading={log.isLoading} />
        </div>
      </div>
    </Dialog>
  );
}
