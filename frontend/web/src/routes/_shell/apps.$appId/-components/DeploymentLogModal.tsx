import { Dialog, LogViewer } from "@aether/design-system";
import { useDeploymentLog, useServiceDeploymentLog } from "../../../../hooks";
import { toDeploymentLogLines } from "../../../../lib/deployment-log-lines";

export function DeploymentLogModal({ appId, serviceId, deploymentId, onClose }: { appId: string; serviceId?: string; deploymentId: string | null; onClose: () => void }) {
  const legacyLog = useDeploymentLog(appId, serviceId ? null : deploymentId);
  const serviceLog = useServiceDeploymentLog(serviceId ?? "", serviceId ? deploymentId : null);
  const log = serviceId ? serviceLog : legacyLog;
  const lines = log.data ? toDeploymentLogLines(log.data.content, log.data.error) : [];

  return (
    <Dialog open={!!deploymentId} onOpenChange={(open) => { if (!open) onClose(); }} title={`Deployment #${log.data?.number ?? ""} log`} size="lg" overflow="hidden" trigger={<button type="button" className="hidden" aria-hidden="true" tabIndex={-1} />}>
      <LogViewer lines={lines} loading={log.isLoading} fullHeight />
    </Dialog>
  );
}
