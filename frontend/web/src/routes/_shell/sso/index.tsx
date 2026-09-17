import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { IdentificationCard, SignIn, Trash } from "@phosphor-icons/react";
import type { Icon as DesignIcon } from "@aether/design-system";
import {
  Badge,
  Button,
  Card,
  Dialog,
  EmptyState,
  Field,
  Input,
  Skeleton,
  useToast,
} from "@aether/design-system";
import {
  useCreateSSO,
  useDeleteSSO,
  useSSO,
  useSSOAuthURL,
} from "../../../hooks";

const schema = z.object({
  name: z.string().min(1, "Name is required"),
  issuer: z.string().url("Invalid URL"),
  client_id: z.string().min(1, "Client ID is required"),
  client_secret: z.string().optional(),
  scopes: z.string().default("openid email profile"),
});
function Sso() {
  const query = useSSO();
  const create = useCreateSSO();
  const remove = useDeleteSSO();
  const authURL = useSSOAuthURL();
  const { add } = useToast();
  const [open, setOpen] = useState(false);
  const form = useForm<z.input<typeof schema>>({
    resolver: zodResolver(schema),
    defaultValues: { scopes: "openid email profile" },
  });
  const submit = async (values: z.input<typeof schema>) => {
    try {
      await create.mutateAsync(values);
      setOpen(false);
      form.reset();
      add({ title: "SSO provider created", tone: "success" });
    } catch (error) {
      add({
        title: "Could not create provider",
        description:
          error instanceof Error ? error.message : "Try again later.",
        tone: "error",
      });
    }
  };
  const connect = (id: string) =>
    authURL.mutate(id, {
      onSuccess: (response) =>
        window.open(
          response.url,
          "_blank",
          "width=700,height=600,noopener,noreferrer",
        ),
      onError: (error) =>
        add({
          title: "Could not connect provider",
          description: error.message,
          tone: "error",
        }),
    });
  return (
    <main className="space-y-lg">
      <header className="relative isolate overflow-hidden rounded-2xl border border-border bg-gradient-to-br from-primary/10 via-surface-card to-secondary/5 px-lg py-xl shadow-sm sm:flex sm:items-end sm:justify-between">
        <div>
          <p className="font-label-caps text-label-caps uppercase text-primary">Access</p>
          <h1 className="mt-sm font-display-lg text-display-lg tracking-[-0.04em] text-on-surface">
            SSO / Identity
          </h1>
          <p className="mt-sm text-body-md text-on-surface-variant">
            OIDC providers with discovery and automatic user provisioning.
          </p>
        </div>
        <Dialog
          open={open}
          onOpenChange={setOpen}
          title="Create OIDC provider"
          description="Connect an identity provider for organization members."
          trigger={
            <Button icon={IdentificationCard as unknown as DesignIcon}>
              New provider
            </Button>
          }
        >
          <form onSubmit={form.handleSubmit(submit)} className="space-y-5">
            <Field label="Name" error={form.formState.errors.name?.message}>
              <Input placeholder="Google" {...form.register("name")} />
            </Field>
            <Field
              label="Issuer URL"
              error={form.formState.errors.issuer?.message}
            >
              <Input
                placeholder="https://accounts.google.com"
                {...form.register("issuer")}
              />
            </Field>
            <Field
              label="Client ID"
              error={form.formState.errors.client_id?.message}
            >
              <Input {...form.register("client_id")} />
            </Field>
            <Field label="Client secret" description="Encrypted at rest.">
              <Input type="password" {...form.register("client_secret")} />
            </Field>
            <Field label="Scopes">
              <Input {...form.register("scopes")} />
            </Field>
            <div className="flex justify-end gap-2 border-t border-border pt-4">
              <Button
                type="button"
                variant="ghost"
                onClick={() => setOpen(false)}
              >
                Cancel
              </Button>
              <Button type="submit" loading={create.isPending}>
                Create provider
              </Button>
            </div>
          </form>
        </Dialog>
      </header>
      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
        {query.isLoading ? (
          <Skeleton variant="card" />
        ) : (
          (query.data ?? []).map((provider) => (
            <Card key={provider.id} variant="elevated" className="flex flex-col gap-4">
              <div className="flex items-center justify-between">
                <IdentificationCard size={28} className="text-primary" />
                <Badge tone={provider.enabled ? "success" : "neutral"} dot>
                  {provider.enabled ? "Active" : "Disabled"}
                </Badge>
              </div>
              <h2 className="text-headline-sm font-semibold text-foreground">
                {provider.name}
              </h2>
              <p className="break-all font-mono text-code-md text-muted-foreground">
                {provider.issuer}
              </p>
              <p className="font-mono text-code-md text-muted-foreground">
                {provider.client_id}
              </p>
              <div className="mt-auto flex justify-between border-t border-border pt-4">
                <Button
                  variant="ghost"
                  icon={SignIn as unknown as DesignIcon}
                  loading={authURL.isPending}
                  onClick={() => connect(provider.id)}
                >
                  Connect
                </Button>
                <Button
                  variant="quiet"
                  icon={Trash as unknown as DesignIcon}
                  aria-label={`Delete ${provider.name}`}
                  onClick={() => remove.mutate(provider.id)}
                />
              </div>
            </Card>
          ))
        )}
      </div>
      {!query.isLoading && !query.data?.length ? (
        <EmptyState
          icon={IdentificationCard as unknown as DesignIcon}
          title="No SSO providers"
          description="Add an OIDC issuer to enable SSO."
        />
      ) : null}
    </main>
  );
}
export const Route = createFileRoute("/_shell/sso/")({ component: Sso });
