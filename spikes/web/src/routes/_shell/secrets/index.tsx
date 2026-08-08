import { AppSecrets } from "./-components/AppSecrets";
import { Row } from "./-components/Row";
import { createFileRoute } from "@tanstack/react-router";
import { useAppDetail, useApps } from "../../../api/hooks";
import { Card, StatusPill, Table } from "../../../components/ui";
import { AppPage, AppPageHeader } from "../../../components/ds";

function Secrets() {
  const { data: apps } = useApps();

  return (
    <AppPage>
      <AppPageHeader
        title="Secrets Vault"
        description="Secret variables encrypted at rest (AES-256-GCM, KEK/DEK). Never stored in plaintext in the database."
      />

      <div className="grid grid-cols-1 xl:grid-cols-2 gap-lg">
        <Card>
          <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase mb-md">
            Secrets per application
          </h2>
          <div className="space-y-sm">
            {(apps ?? []).map((app) => <AppSecrets key={app.id} appId={app.id} name={app.name} />)}
            {!apps?.length && (
              <p className="font-body-sm text-body-sm text-on-surface-variant">No applications.</p>
            )}
          </div>
        </Card>

        <Card>
          <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase mb-md">
            Encryption model
          </h2>
          <div className="space-y-sm">
            <Row label="Algorithm" value="AES-256-GCM" />
            <Row label="Data key (DEK)" value="random, rotatable" />
            <Row label="Master key (KEK)" value="host filesystem 0600" />
            <Row label="Injection" value="env resolvido no schedule" />
            <Row label="Values in logs" value="forbidden" />
          </div>
          <p className="font-body-sm text-body-sm text-on-surface-variant/70 pt-sm">
            DEK rotation re-encrypts values without exposing plaintext. Enterprise: KEK in TPM/HSM.
          </p>
        </Card>
      </div>
    </AppPage>
  );
}



export const Route = createFileRoute('/_shell/secrets/')({
  component: Secrets,
});
