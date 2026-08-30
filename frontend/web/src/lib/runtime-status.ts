import type { RuntimeStatusValue } from "@aether/design-system";

const deployingStates = ["queued", "building", "starting", "health_checking", "provisioning", "syncing"];
const failedStates = ["failed", "cancelled", "rolled_back", "error", "dead"];
const offlineStates = ["stopped", "exited", "no_container", "offline", "disabled"];

export function mapRuntimeStatus(state?: string, deploymentStatus?: string): RuntimeStatusValue {
  const values = [deploymentStatus, state].filter(Boolean) as string[];
  if (values.some((value) => failedStates.includes(value))) return "failed";
  if (values.some((value) => deployingStates.includes(value))) return "deploying";
  if (values.some((value) => ["degraded", "warning"].includes(value))) return "degraded";
  if (values.some((value) => ["running", "ready", "healthy", "active"].includes(value))) return "healthy";
  if (values.some((value) => offlineStates.includes(value))) return "offline";
  if (values.includes("paused")) return "paused";
  return "validating";
}

export function isRuntimeLive(status: RuntimeStatusValue) {
  return status === "healthy" || status === "deploying";
}
