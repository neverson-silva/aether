import { createFileRoute } from "@tanstack/react-router";
import { Certificate } from "@phosphor-icons/react";
import type { Icon as DesignIcon } from "@aether/design-system";
import {
  Badge,
  Card,
  EmptyState,
  InlineError,
  Skeleton,
} from "@aether/design-system";
import { useCertificates } from "../../../hooks";
import { PageHeader } from "../../../components/PageHeader";

function statusTone(status: string) {
  return status === "issued" || status === "valid"
    ? ("success" as const)
    : status === "pending" || status === "validating"
      ? ("warning" as const)
      : status === "failed"
        ? ("danger" as const)
        : ("neutral" as const);
}

function Certificates() {
  const query = useCertificates();
  return (
    <main className="mx-auto flex w-full max-w-screen-2xl flex-col gap-lg p-6 lg:p-8">
      <PageHeader
        eyebrow="Security"
        title="Certificates"
        description="TLS certificates issued via ACME for linked domains."
      />
      {query.error ? (
        <InlineError
          title="Could not load certificates"
          message="Try again to refresh certificate status."
          onRetry={() => query.refetch()}
        />
      ) : null}
      <Card variant="elevated" padding="none">
        <div className="overflow-hidden rounded-2xl border border-border">
          <div className="overflow-x-auto">
            <table className="w-full min-w-[720px] text-left">
              <thead className="bg-surface-container-low/80 text-label-caps text-muted-foreground">
                <tr>
                  {["App", "Domain", "HTTPS", "Status", "Linked at"].map(
                    (header) => (
                      <th key={header} className="px-4 py-3 font-semibold">
                        {header}
                      </th>
                    ),
                  )}
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {query.isLoading ? (
                  <tr>
                    <td colSpan={5} className="p-6">
                      <Skeleton variant="table" />
                    </td>
                  </tr>
                ) : (
                  (query.data ?? []).map((certificate) => (
                    <tr
                      key={certificate.id}
                      className="transition-[background-color] duration-150 hover:bg-surface-container"
                    >
                      <td className="px-4 py-3 text-body-md text-foreground">
                        {certificate.app_name}
                      </td>
                      <td className="px-4 py-3 font-mono text-code-md text-primary">
                        {certificate.host}
                      </td>
                      <td className="px-4 py-3">
                        <Badge
                          tone={certificate.https ? "success" : "neutral"}
                          dot
                        >
                          {certificate.https ? "Enabled" : "Disabled"}
                        </Badge>
                      </td>
                      <td className="px-4 py-3">
                        <Badge tone={statusTone(certificate.cert_status)}>
                          {certificate.cert_status}
                        </Badge>
                      </td>
                      <td className="px-4 py-3 font-mono text-code-md text-muted-foreground">
                        {certificate.created_at
                          ? new Date(certificate.created_at).toLocaleDateString(
                              "en",
                            )
                          : "—"}
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </div>
        {!query.isLoading && !query.data?.length ? (
          <EmptyState
            icon={Certificate as unknown as DesignIcon}
            title="No certificates yet"
            description="Add a domain to an application to request a certificate."
          />
        ) : null}
      </Card>
    </main>
  );
}

export const Route = createFileRoute("/_shell/certificates/")({
  component: Certificates,
});
