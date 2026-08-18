import { useMemo } from "react";
import { useOrg } from "./OrgProvider";
import { useProjects } from "../hooks";
import type { VarGroup, PickedVar } from "./VariablePicker";

interface ScopeInput {
  serviceName?: string;
  serviceVars?: { key: string; value: string; secret: boolean }[];
  projectEnvs?: { name: string; value: string; secret: boolean }[];
}

const SYSTEM_VARS: PickedVar[] = [
  { name: "SERVICE_NAME", description: "Current service name.", scope: "system" },
  { name: "SERVICE_ID", description: "Current service ID.", scope: "system" },
  { name: "DEPLOYMENT_ID", description: "Currently deployed deployment ID.", scope: "system" },
  { name: "COMMIT_SHA", description: "Currently deployed commit hash.", scope: "system" },
  { name: "BRANCH", description: "Repository branch.", scope: "system" },
  { name: "CONTAINER_NAME", description: "Running container name.", scope: "system" },
  { name: "HOSTNAME", description: "Internal container hostname.", scope: "system" },
  { name: "PORT", description: "Public service port.", scope: "system" },
  { name: "INTERNAL_HOST", description: "Hostname on the container-to-container network.", scope: "system" },
  { name: "INTERNAL_NETWORK", description: "Project internal network.", scope: "system" },
];

export function useScopeVariables(input: ScopeInput = {}): VarGroup[] {
  const { currentOrg } = useOrg();
  const { data: projects } = useProjects();

  return useMemo<VarGroup[]>(() => {
    const groups: VarGroup[] = [];

    const svcItems: PickedVar[] = [];
    if (input.serviceName) {
      svcItems.unshift({ name: "SERVICE_NAME", description: "Current service name.", scope: "service", value: input.serviceName });
    }
    for (const v of input.serviceVars ?? []) {
      svcItems.push({ name: v.key, description: v.secret ? "Secret service variable." : undefined, scope: "service", value: v.secret ? undefined : v.value });
    }
    if (svcItems.length) groups.push({ scope: "service", items: svcItems });

    const projItems: PickedVar[] = [];
    const proj = projects?.[0];
    if (proj) {
      projItems.push({ name: "PROJECT_NAME", description: "Current project name.", scope: "project", value: proj.name });
      projItems.push({ name: "PROJECT_SLUG", description: "Current project slug.", scope: "project", value: proj.slug ?? "" });
      projItems.push({ name: "PROJECT_ID", description: "Current project ID.", scope: "project", value: proj.id });
    } else {
      projItems.push({ name: "PROJECT_NAME", description: "Current project name.", scope: "project" });
      projItems.push({ name: "PROJECT_SLUG", description: "Current project slug.", scope: "project" });
      projItems.push({ name: "PROJECT_ID", description: "Current project ID.", scope: "project" });
    }
    groups.push({ scope: "project", items: projItems });

    const orgItems: PickedVar[] = [
      { name: "ORGANIZATION_NAME", description: "Current organization name.", scope: "organization", value: currentOrg?.name },
      { name: "ORGANIZATION_SLUG", description: "Current organization slug.", scope: "organization", value: currentOrg?.slug },
    ];
    groups.push({ scope: "organization", items: orgItems });

    const envItems: PickedVar[] = [];
    for (const v of input.projectEnvs ?? []) {
      envItems.push({ name: v.name, description: v.secret ? "Secret environment variable." : undefined, scope: "environment", value: v.secret ? undefined : v.value });
    }
    if (envItems.length) groups.push({ scope: "environment", items: envItems });

    const secretItems: PickedVar[] = [];
    for (const v of input.serviceVars ?? []) {
      if (v.secret) secretItems.push({ name: v.key, description: "Encrypted secret.", scope: "secrets" });
    }
    for (const v of input.projectEnvs ?? []) {
      if (v.secret) secretItems.push({ name: v.name, description: "Encrypted secret.", scope: "secrets" });
    }
    if (secretItems.length) groups.push({ scope: "secrets", items: secretItems });

    groups.push({ scope: "system", items: SYSTEM_VARS });

    return groups;
  }, [input.serviceName, input.serviceVars, input.projectEnvs, currentOrg, projects]);
}
