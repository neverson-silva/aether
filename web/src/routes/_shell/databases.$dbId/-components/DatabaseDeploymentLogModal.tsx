import { Modal } from "../../../../components/ui";
import { useDatabaseDeploymentLog } from "../../../../hooks";
import type { DatabaseDeployment } from "../../../../hooks/use-database-deployments";
import { classify, LogLine } from "../../apps.$appId/-components/LiveLogs";

export function DatabaseDeploymentLogModal({ dbId, deployment, onClose }: { dbId: string; deployment: DatabaseDeployment | null; onClose: () => void }) {
  const log = useDatabaseDeploymentLog(dbId, deployment?.id ?? null);

  return (
    <Modal open={!!deployment} onClose={onClose} title={`Deployment #${deployment?.number ?? ""} log`} wide>
      <div className="bg-[#0a0a0a] border border-[#1a1a1a] rounded-lg p-3 font-code-md text-[12px] text-[#d1d5db] max-h-[55vh] overflow-y-auto sidebar-scroll">
        {log.data?.content ? (
          log.data.content.split("\n").filter((l) => l.trim() !== "").map((l, i) => {
            const row = classify(l);
            return (
              <div key={i} className="whitespace-pre-wrap break-all py-[1px] hover:bg-white/[0.03] rounded px-1">
                <LogLine row={row} />
              </div>
            );
          })
        ) : (
          <span className="text-on-surface-variant/50">{log.isFetching ? "Loading logs..." : "No log captured for this deployment."}</span>
        )}
      </div>
    </Modal>
  );
}