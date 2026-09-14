import { ChartLine, Code, Database, Gauge, Gear, Globe, ListChecks, RocketLaunch, TerminalWindow } from "@phosphor-icons/react";

export const SERVICE_TABS = [
  "overview",
  "deployments",
  "variables",
  "compose",
  "domains",
  "logs",
  "metrics",
  "settings",
  "cron",
  "terminal",
  "backup",
] as const;

export type ServiceTab = (typeof SERVICE_TABS)[number];

export const SERVICE_TAB_ICONS: Record<ServiceTab, typeof Code> = {
  overview: Gauge,
  deployments: RocketLaunch,
  variables: ListChecks,
  compose: Code,
  domains: Globe,
  logs: TerminalWindow,
  metrics: ChartLine,
  settings: Gear,
  cron: ListChecks,
  terminal: TerminalWindow,
  backup: Database,
};
