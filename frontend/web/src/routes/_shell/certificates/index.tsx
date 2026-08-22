import { createFileRoute } from "@tanstack/react-router";
import { Certificate } from "@phosphor-icons/react";
import type { Icon as DesignIcon } from "@aether/design-system";
import { Badge, Card, EmptyState, InlineError, Skeleton } from "@aether/design-system";
import { useCertificates } from "../../../hooks";

function statusTone(status: string) { return status === "issued" || status === "valid" ? "success" as const : status === "pending" || status === "validating" ? "warning" as const : status === "failed" ? "danger" as const : "neutral" as const; }

function Certificates() {
  const query = useCertificates();
  return <main className="mx-auto flex w-full max-w-screen-2xl flex-col gap-8 p-6 lg:p-8"><header className="space-y-2"><p className="text-label-caps text-primary">Security</p><h1 className="text-display font-semibold text-foreground">Certificates</h1><p className="text-body-md text-muted-foreground">TLS certificates issued via ACME for linked domains.</p></header>{query.error ? <InlineError title="Could not load certificates" message="Try again to refresh certificate status." onRetry={() => query.refetch()} /> : null}<Card padding="none"><div className="overflow-x-auto"><table className="w-full min-w-[720px] text-left"><thead className="border-b border-border text-label-caps text-muted-foreground"><tr>{["App", "Domain", "HTTPS", "Status", "Linked at"].map((header) => <th key={header} className="px-4 py-3 font-semibold">{header}</th>)}</tr></thead><tbody className="divide-y divide-border">{query.isLoading ? <tr><td colSpan={5} className="p-6"><Skeleton variant="table" /></td></tr> : (query.data ?? []).map((certificate) => <tr key={certificate.id} className="transition-colors hover:bg-surface-container"><td className="px-4 py-3 text-body-md text-foreground">{certificate.app_name}</td><td className="px-4 py-3 font-mono text-code-md text-primary">{certificate.host}</td><td className="px-4 py-3"><Badge tone={certificate.https ? "success" : "neutral"} dot>{certificate.https ? "Enabled" : "Disabled"}</Badge></td><td className="px-4 py-3"><Badge tone={statusTone(certificate.cert_status)}>{certificate.cert_status}</Badge></td><td className="px-4 py-3 font-mono text-code-md text-muted-foreground">{certificate.created_at ? new Date(certificate.created_at).toLocaleDateString("en") : "—"}</td></tr>)}</tbody></table></div>{!query.isLoading && !query.data?.length ? <EmptyState icon={Certificate as unknown as DesignIcon} title="No certificates yet" description="Add a domain to an application to request a certificate." /> : null}</Card></main>;
}

export const Route = createFileRoute("/_shell/certificates/")({ component: Certificates });
