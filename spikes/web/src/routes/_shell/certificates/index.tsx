import { createFileRoute } from "@tanstack/react-router";
import { useCertificates } from "../../../api/hooks";
import { StatusPill, Table } from "../../../components/ui";
import { AppPage, AppPageHeader, AppCard } from "../../../components/ds";

function tone(s: string): string {
  return s === "issued" || s === "valid" ? "active" : s === "pending" || s === "validating" ? "pending" : s === "failed" ? "failed" : "disabled";
}

function Certificates() {
  const { data: certs, isLoading } = useCertificates();
  return (
    <AppPage>
      <AppPageHeader
        title="Certificates"
        description="TLS certificates issued via ACME for linked domains."
      />
      <AppCard>
        <Table headers={["App", "Domain", "HTTPS", "Status", "Linked at"]}>
          {isLoading && (
            <tr>
              <td colSpan={5} className="px-sm py-lg text-center font-body-sm text-body-sm text-on-surface-variant">Loading…</td>
            </tr>
          )}
          {(certs ?? []).map((c) => (
            <tr key={c.id} className="hover:bg-surface-container-high transition-colors">
              <td className="px-sm py-2 font-body-md text-body-md text-on-surface">{c.app_name}</td>
              <td className="px-sm py-2 font-code-md text-code-md text-primary">{c.host}</td>
              <td className="px-sm py-2"><StatusPill status={c.https ? "active" : "disabled"} pulse={c.https} /></td>
              <td className="px-sm py-2"><StatusPill status={tone(c.cert_status)} /></td>
              <td className="px-sm py-2 font-code-md text-code-md text-on-surface-variant/60">
                {c.created_at ? new Date(c.created_at).toLocaleDateString() : "—"}
              </td>
            </tr>
          ))}
        </Table>
        {(certs ?? []).length === 0 && !isLoading && (
          <p className="font-body-sm text-body-sm text-on-surface-variant p-sm">
            No domains linked yet. Add a domain to an application to request a certificate.
          </p>
        )}
      </AppCard>
    </AppPage>
  );
}

export const Route = createFileRoute("/_shell/certificates/")({
  component: Certificates,
});
