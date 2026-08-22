import { AppSecrets } from "./-components/AppSecrets";
import { Row } from "./-components/Row";
import { createFileRoute } from "@tanstack/react-router";
import { useAppDetail, useApps } from "../../../hooks";
import { Card, EmptyState } from "@aether/design-system";
import type { Icon as DesignIcon } from "@aether/design-system";
import { LockKey } from "@phosphor-icons/react";

function Secrets() {
  const { data: apps } = useApps();

  return (
    <main className="mx-auto flex w-full max-w-screen-2xl flex-col gap-8 p-6 lg:p-8">
      <header className="space-y-2"><p className="text-label-caps text-primary">Security</p><h1 className="text-display font-semibold text-foreground">Secrets vault</h1><p className="text-body-md text-muted-foreground">Secret variables encrypted at rest with AES-256-GCM. Values are never stored in plaintext.</p></header>

      <div className="grid grid-cols-1 xl:grid-cols-2 gap-lg">
        <Card>
          <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase mb-md">
            Secrets per application
          </h2>
          <div className="space-y-sm">
            {(apps ?? []).map((app) => <AppSecrets key={app.id} appId={app.id} name={app.name} />)}
            {!apps?.length && <EmptyState icon={LockKey as unknown as DesignIcon} title="No applications" description="Create an application to manage encrypted secrets." />}
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
            <Row label="Injection" value="Environment at schedule time" />
            <Row label="Values in logs" value="forbidden" />
          </div>
          <p className="font-body-sm text-body-sm text-on-surface-variant/70 pt-sm">
            DEK rotation re-encrypts values without exposing plaintext. Enterprise: KEK in TPM/HSM.
          </p>
        </Card>
      </div>
    </main>
  );
}



export const Route = createFileRoute('/_shell/secrets/')({
  component: Secrets,
});
