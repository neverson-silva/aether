import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { apiGet, apiPost, apiPut } from "../api/client";

export interface SourceControlConnection {
  id: string;
  provider: string;
  external_account_id: string;
  external_account_name: string;
  installation_id: string;
  status: string;
  created_at: string;
  updated_at: string;
}

export interface SourceControlRepository {
  id: string;
  owner: string;
  name: string;
  full_name: string;
  default_branch: string;
}

export interface SourceControlBranch {
  name: string;
}

export interface ServiceSource {
  id: string;
  service_id: string;
  connection_id: string;
  organization_id: string;
  repository_id: string;
  repository_owner: string;
  repository_name: string;
  repository_full_name: string;
  default_branch: string;
  branch: string;
  auto_deploy: boolean;
  root_directory: string;
  watch_paths: string[];
  ignore_paths: string[];
  watch_root_files: boolean;
  created_at: string;
  updated_at: string;
}

export interface ServiceSourceInput {
  connection_id: string;
  repository_id: string;
  repository_owner: string;
  repository_name: string;
  repository_full_name: string;
  default_branch: string;
  branch: string;
  auto_deploy: boolean;
  root_directory: string;
  watch_paths: string[];
  ignore_paths: string[];
  watch_root_files: boolean;
}

type ServiceSourceResponse = Partial<ServiceSource> & {
  ID?: string;
  ServiceID?: string;
  ConnectionID?: string;
  OrganizationID?: string;
  RepositoryID?: string;
  RepositoryOwner?: string;
  RepositoryName?: string;
  RepositoryFullName?: string;
  DefaultBranch?: string;
  Branch?: string;
  AutoDeploy?: boolean;
  RootDirectory?: string;
  WatchPaths?: string[];
  IgnorePaths?: string[];
  WatchRootFiles?: boolean;
  CreatedAt?: string;
  UpdatedAt?: string;
};

function normalizeServiceSource(raw: ServiceSourceResponse): ServiceSource {
  return {
    id: raw.id ?? raw.ID ?? "",
    service_id: raw.service_id ?? raw.ServiceID ?? "",
    connection_id: raw.connection_id ?? raw.ConnectionID ?? "",
    organization_id: raw.organization_id ?? raw.OrganizationID ?? "",
    repository_id: raw.repository_id ?? raw.RepositoryID ?? "",
    repository_owner: raw.repository_owner ?? raw.RepositoryOwner ?? "",
    repository_name: raw.repository_name ?? raw.RepositoryName ?? "",
    repository_full_name: raw.repository_full_name ?? raw.RepositoryFullName ?? "",
    default_branch: raw.default_branch ?? raw.DefaultBranch ?? "",
    branch: raw.branch ?? raw.Branch ?? "",
    auto_deploy: raw.auto_deploy ?? raw.AutoDeploy ?? false,
    root_directory: raw.root_directory ?? raw.RootDirectory ?? "",
    watch_paths: raw.watch_paths ?? raw.WatchPaths ?? [],
    ignore_paths: raw.ignore_paths ?? raw.IgnorePaths ?? [],
    watch_root_files: raw.watch_root_files ?? raw.WatchRootFiles ?? false,
    created_at: raw.created_at ?? raw.CreatedAt ?? "",
    updated_at: raw.updated_at ?? raw.UpdatedAt ?? "",
  };
}

export function useSourceControlConnections() {
  return useQuery({
    queryKey: ["source-control", "connections"],
    queryFn: () => apiGet<SourceControlConnection[]>("/api/v1/source-control/github/connections"),
  });
}

export function useStartGitHubManifest() {
  return useMutation({
    mutationFn: (body: { return_url: string }) => apiPost<{ url: string; manifest: string; state: string }>("/api/v1/source-control/github/manifest/start", body),
  });
}

export function useSourceControlRepositories(installationID: string | undefined) {
  return useQuery({
    queryKey: ["source-control", "repositories", installationID],
    queryFn: () => apiGet<SourceControlRepository[]>(`/api/v1/source-control/github/repositories?installation_id=${encodeURIComponent(installationID ?? "")}`),
    enabled: !!installationID,
    refetchOnMount: "always",
  });
}

export function useSourceControlBranches(repositoryID: string | undefined, installationID: string | undefined) {
  return useQuery({
    queryKey: ["source-control", "branches", repositoryID, installationID],
    queryFn: () => apiGet<SourceControlBranch[]>(`/api/v1/source-control/github/repositories/${encodeURIComponent(repositoryID ?? "")}/branches?installation_id=${encodeURIComponent(installationID ?? "")}`),
    enabled: !!repositoryID && !!installationID,
  });
}

export function useSourceControlFile(repositoryID: string | undefined, installationID: string | undefined, path: string, ref: string) {
  return useQuery({
    queryKey: ["source-control", "file", repositoryID, installationID, path, ref],
    queryFn: () => apiGet<{ path: string; ref: string; content: string }>(`/api/v1/source-control/github/repositories/${encodeURIComponent(repositoryID ?? "")}/file?installation_id=${encodeURIComponent(installationID ?? "")}&path=${encodeURIComponent(path)}&ref=${encodeURIComponent(ref)}`),
    enabled: !!repositoryID && !!installationID && !!path && !!ref,
  });
}

export function useServiceSource(appID: string) {
  return useQuery({
    queryKey: ["service-source", appID],
    queryFn: async () => {
      const source = normalizeServiceSource(await apiGet<ServiceSourceResponse>(`/api/v1/apps/${appID}/source`));
      return {
        ...source,
        watch_paths: source.watch_paths ?? [],
        ignore_paths: source.ignore_paths ?? [],
      };
    },
    enabled: !!appID,
    retry: (count, error) => (error instanceof Error && "status" in error && (error as { status?: number }).status === 404 ? false : count < 2),
  });
}

export function useSaveServiceSource(appID: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: ServiceSourceInput) => apiPut<ServiceSource>(`/api/v1/apps/${appID}/source`, body),
    onSuccess: (source) => {
      queryClient.setQueryData(["service-source", appID], source);
    },
  });
}
