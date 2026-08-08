import { useMemo } from "react";
import { useOrg } from "./OrgProvider";
import { useProjects } from "../api/hooks";
import type { VarGroup, PickedVar } from "./VariablePicker";

interface ScopeInput {
  serviceName?: string;
  serviceVars?: { key: string; value: string; secret: boolean }[];
  projectEnvs?: { name: string; value: string; secret: boolean }[];
}

const SYSTEM_VARS: PickedVar[] = [
  { name: "SERVICE_NAME", description: "Nome do serviço atual.", scope: "system" },
  { name: "SERVICE_ID", description: "ID do serviço atual.", scope: "system" },
  { name: "DEPLOYMENT_ID", description: "ID do deployment atualmente implantado.", scope: "system" },
  { name: "COMMIT_SHA", description: "Hash do commit atualmente implantado.", scope: "system" },
  { name: "BRANCH", description: "Branch do repositório.", scope: "system" },
  { name: "CONTAINER_NAME", description: "Nome do container em execução.", scope: "system" },
  { name: "HOSTNAME", description: "Hostname interno do container.", scope: "system" },
  { name: "PORT", description: "Porta pública do serviço.", scope: "system" },
  { name: "INTERNAL_HOST", description: "Hostname na rede container-to-container.", scope: "system" },
  { name: "INTERNAL_NETWORK", description: "Rede interna do projeto.", scope: "system" },
];

export function useScopeVariables(input: ScopeInput = {}): VarGroup[] {
  const { currentOrg } = useOrg();
  const { data: projects } = useProjects();

  return useMemo<VarGroup[]>(() => {
    const groups: VarGroup[] = [];

    // Service
    const svcItems: PickedVar[] = [];
    if (input.serviceName) {
      svcItems.unshift({ name: "SERVICE_NAME", description: "Nome do serviço atual.", scope: "service", value: input.serviceName });
    }
    for (const v of input.serviceVars ?? []) {
      svcItems.push({ name: v.key, description: v.secret ? "Variável secreta do serviço." : undefined, scope: "service", value: v.secret ? undefined : v.value });
    }
    if (svcItems.length) groups.push({ scope: "service", items: svcItems });

    // Project
    const projItems: PickedVar[] = [];
    const proj = projects?.[0];
    if (proj) {
      projItems.push({ name: "PROJECT_NAME", description: "Nome do projeto atual.", scope: "project", value: proj.name });
      projItems.push({ name: "PROJECT_SLUG", description: "Slug do projeto atual.", scope: "project", value: proj.slug ?? "" });
      projItems.push({ name: "PROJECT_ID", description: "ID do projeto atual.", scope: "project", value: proj.id });
    } else {
      projItems.push({ name: "PROJECT_NAME", description: "Nome do projeto atual.", scope: "project" });
      projItems.push({ name: "PROJECT_SLUG", description: "Slug do projeto atual.", scope: "project" });
      projItems.push({ name: "PROJECT_ID", description: "ID do projeto atual.", scope: "project" });
    }
    groups.push({ scope: "project", items: projItems });

    // Organization
    const orgItems: PickedVar[] = [
      { name: "ORGANIZATION_NAME", description: "Nome da organização atual.", scope: "organization", value: currentOrg?.name },
      { name: "ORGANIZATION_SLUG", description: "Slug da organização atual.", scope: "organization", value: currentOrg?.slug },
    ];
    groups.push({ scope: "organization", items: orgItems });

    // Environment (projeto compartilhado)
    const envItems: PickedVar[] = [];
    for (const v of input.projectEnvs ?? []) {
      envItems.push({ name: v.name, description: v.secret ? "Variável secreta do ambiente." : undefined, scope: "environment", value: v.secret ? undefined : v.value });
    }
    if (envItems.length) groups.push({ scope: "environment", items: envItems });

    // Secrets (separadas)
    const secretItems: PickedVar[] = [];
    for (const v of input.serviceVars ?? []) {
      if (v.secret) secretItems.push({ name: v.key, description: "Secreto criptografado.", scope: "secrets" });
    }
    for (const v of input.projectEnvs ?? []) {
      if (v.secret) secretItems.push({ name: v.name, description: "Secreto criptografado.", scope: "secrets" });
    }
    if (secretItems.length) groups.push({ scope: "secrets", items: secretItems });

    // System
    groups.push({ scope: "system", items: SYSTEM_VARS });

    return groups;
  }, [input.serviceName, input.serviceVars, input.projectEnvs, currentOrg, projects]);
}
