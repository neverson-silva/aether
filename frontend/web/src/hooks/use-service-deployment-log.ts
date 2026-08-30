import { useEffect, useRef } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import { useRealtime, useRealtimeEvent } from "../components/RealtimeProvider";
import type { DeploymentLog } from "./types";

export function useServiceDeploymentLog(serviceId: string, deploymentId: string | null) {
  const qc = useQueryClient();
  const { connected } = useRealtime();
  const key = ["service-deploy-log", serviceId, deploymentId] as const;
  const query = useQuery({
    queryKey: key,
    enabled: !!serviceId && !!deploymentId,
    queryFn: () => apiGet<DeploymentLog>(`/api/v1/services/${serviceId}/deployments/${deploymentId}/log`),
  });

  useRealtimeEvent((ev, replay) => {
    if (!deploymentId || replay) return;
    if (ev.type !== "deploy.build.log" || ev.resource_id !== deploymentId) return;
    const line = (ev.payload as { line?: string } | undefined)?.line;
    if (!line) return;
    qc.setQueryData<DeploymentLog>(key, (old) => {
      const base = old ?? { number: 0, status: "building", error: "", content: "" };
      return { ...base, content: base.content ? base.content + "\n" + line : line };
    });
  });

  const prevConnected = useRef(connected);
  useEffect(() => {
    if (prevConnected.current === false && connected === true && deploymentId) {
      qc.invalidateQueries({ queryKey: key });
    }
    prevConnected.current = connected;
  }, [connected, deploymentId, qc]);

  return query;
}
