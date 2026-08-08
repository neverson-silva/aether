import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import {
  useCreateSSO,
  useDeleteSSO,
  useSSO,
  useSSOAuthURL,
} from "../../../api/hooks";
import {
  Button,
  Card,
  Field,
  Input,
  Modal,
  StatusPill,
  useToast,
} from "../../../components/ui";
import { AppPageHeader } from "../../../components/ds";

const schema = z.object({
  name: z.string().min(1, "Name is required"),
  issuer: z.string().min(1, "Issuer URL is required").url("Invalid URL"),
  client_id: z.string().min(1, "Client ID is required"),
  client_secret: z.string().optional(),
  scopes: z.string().default("openid email profile"),
});

function Sso() {
  const { data: providers, isLoading } = useSSO();
  const create = useCreateSSO();
  const remove = useDeleteSSO();
  const authURL = useSSOAuthURL();
  const { toast } = useToast();
  const [open, setOpen] = useState(false);
  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<z.input<typeof schema>, any, z.output<typeof schema>>({
    resolver: zodResolver(schema),
    defaultValues: { scopes: "openid email profile" },
  });

  const submit = async (values: z.infer<typeof schema>) => {
    try {
      await create.mutateAsync(values);
      toast("Provider created");
      setOpen(false);
      reset();
    } catch (err) {
      toast(err instanceof Error ? err.message : "failed", "error");
    }
  };

  const connect = (id: string) => {
    authURL.mutate(id, {
      onSuccess: (res) => {
        window.open(res.url, "_blank", "width=700,height=600");
      },
      onError: (err) => toast(err.message, "error"),
    });
  };

  return (
    <div className="space-y-lg">
      <AppPageHeader
        title="SSO / Identity"
        description="OIDC providers with discovery, authorization-code flow and automatic user provisioning."
      />

      <div className="flex items-center justify-between">
        <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase">Providers</h2>
        <Button onClick={() => setOpen(true)}>
          <span className="material-symbols-outlined text-[16px]">add</span>
          New provider
        </Button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-md">
        {isLoading && <p className="font-body-sm text-body-sm text-on-surface-variant">Loading…</p>}
        {(providers ?? []).map((p) => (
          <Card key={p.id} className="flex flex-col gap-sm">
            <div className="flex items-center justify-between">
              <span className="material-symbols-outlined text-[28px] text-primary">badge</span>
              <StatusPill status={p.enabled ? "active" : "disabled"} pulse={p.enabled} />
            </div>
            <h3 className="font-headline-sm text-headline-sm text-on-surface">{p.name}</h3>
            <p className="font-body-sm text-body-sm text-on-surface-variant font-code-md text-code-md">{p.issuer}</p>
            <div className="flex items-center gap-sm">
              <span className="font-code-md text-code-md text-on-surface-variant/60">{p.client_id}</span>
            </div>
            <div className="flex justify-between items-center mt-auto pt-md border-t border-outline-variant">
              <Button variant="ghost" onClick={() => connect(p.id)}>
                <span className="material-symbols-outlined text-[16px]">login</span>
                Connect
              </Button>
              <button
                onClick={() => remove.mutate(p.id)}
                className="material-symbols-outlined text-[16px] text-on-surface-variant hover:text-error transition-colors"
              >
                delete
              </button>
            </div>
          </Card>
        ))}
        {(providers ?? []).length === 0 && !isLoading && (
          <Card className="md:col-span-2 xl:col-span-3">
            <p className="font-body-sm text-body-sm text-on-surface-variant">
              No providers yet. Add an OIDC issuer (Google, GitHub, Keycloak, Authentik…) to enable SSO.
            </p>
          </Card>
        )}
      </div>

      <Modal open={open} onClose={() => setOpen(false)} title="New OIDC provider">
        <form onSubmit={handleSubmit(submit)} className="space-y-lg" noValidate>
          <Field label="Name" hint={errors.name?.message}>
            <Input icon="label" placeholder="google" {...register("name")} />
          </Field>
          <Field label="Issuer URL" hint={errors.issuer?.message || "e.g. https://accounts.google.com"}>
            <Input icon="dns" placeholder="https://accounts.google.com" {...register("issuer")} />
          </Field>
          <Field label="Client ID" hint={errors.client_id?.message}>
            <Input icon="key" {...register("client_id")} />
          </Field>
          <Field label="Client Secret" hint={errors.client_secret?.message || "encrypted at rest"}>
            <Input icon="vpn_key" type="password" {...register("client_secret")} />
          </Field>
          <Field label="Scopes" hint={errors.scopes?.message}>
            <Input icon="checklist" {...register("scopes")} />
          </Field>
          <div className="flex justify-end gap-md border-t border-outline-variant pt-lg">
            <Button type="button" variant="ghost" onClick={() => setOpen(false)}>Cancel</Button>
            <Button type="submit">Create provider</Button>
          </div>
        </form>
      </Modal>
    </div>
  );
}

export const Route = createFileRoute("/_shell/sso/")({
  component: Sso,
});
