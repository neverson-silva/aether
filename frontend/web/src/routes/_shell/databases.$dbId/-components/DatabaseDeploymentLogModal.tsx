import { Dialog, LogViewer } from "@aether/design-system";
import { useDatabaseDeploymentLog } from "../../../../hooks";
import type { DatabaseDeployment } from "../../../../hooks/use-database-deployments";
import { toDeploymentLogLines } from "../../../../lib/deployment-log-lines";

export function DatabaseDeploymentLogModal({ dbId, deployment, onClose }: { dbId: string; deployment: DatabaseDeployment | null; onClose: () => void }) {
  const log = useDatabaseDeploymentLog(dbId, deployment?.id ?? null);
  const lines = log.data ? toDeploymentLogLines(log.data.content) : [];

  return (
    <Dialog open={!!deployment} trigger={<span />} onOpenChange={(open) => { if (!open) onClose(); }} title={`Deployment #${deployment?.number ?? ""} log`} size="lg" overflow="hidden">
      <div className="min-h-0 overflow-hidden pt-1">
        <div className="mt-1">
          <LogViewer lines={lines} loading={log.isLoading} />
        </div>
      </div>
    </Dialog>
  );
}
