export const qk = {
  me: ["me"] as const,
  members: ["members"] as const,
  keys: ["api-keys"] as const,
  composes: ["composes"] as const,
  projects: ["projects"] as const,
  apps: ["apps"] as const,
  app: (id: string) => ["app", id] as const,
  deployments: (id: string) => ["deployments", id] as const,
  domains: (kind: string, id: string) => ["domains", kind, id] as const,
  timeline: (id: string) => ["timeline", id] as const,
  backups: ["backups"] as const,
  stats: (id: string) => ["stats", id] as const,
};
