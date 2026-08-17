import { Modal } from "../../../../components/ui";
import { useDeploymentLog } from "../../../../hooks";
import { classify, LogLine } from "./LiveLogs";

export function DeploymentLogModal({ appId, deploymentId, onClose }: { appId: string; deploymentId: string | null; onClose: () => void }) {
  const log = useDeploymentLog(appId, deploymentId);

  return (
    <Modal open={!!deploymentId} onClose={onClose} title={`Deployment #${log.data?.number ?? ""} log`} wide>
      {log.data ? (
        <div>
          {log.data.error && (
            <div className="mb-md px-sm py-2 rounded bg-error/10 border border-error/20 font-code-md text-code-md text-error">
              ✗ {log.data.error}
            </div>
          )}
          <div className="bg-[#0a0a0a] border border-[#1a1a1a] rounded-lg p-3 font-code-md text-[12px] text-[#d1d5db] max-h-[55vh] overflow-y-auto sidebar-scroll">
            {log.data.content ? (
              log.data.content.split("\n").filter((l) => l.trim() !== "").map((l, i) => {
                const row = classify(l);
                return (
                  <div key={i} className="whitespace-pre-wrap break-all py-[1px] hover:bg-white/[0.03] rounded px-1">
                    <LogLine row={row} />
                  </div>
                );
              })
            ) : (
              <span className="text-on-surface-variant/50">No log captured for this deployment.</span>
            )}
          </div>
        </div>
      ) : (
        <p className="font-body-sm text-body-sm text-on-surface-variant">Loading logs...</p>
      )}
    </Modal>
  );
}