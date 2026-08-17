import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { useApiKeys, useCreateApiKey, useDeleteApiKey } from "../../../hooks";
import {
  Button,
  Card,
  CodeBlock,
  Field,
  Input,
  Modal,
  Table,
  useToast,
} from "../../../components/ui";
import { AppPage, AppPageHeader, AppCard } from "../../../components/ds";

const createSchema = z.object({
  name: z.string().min(1, "Name is required"),
  scopes: z.string().optional(),
});

function ApiKeys() {
  const { data: keys } = useApiKeys();
  const createKey = useCreateApiKey();
  const deleteKey = useDeleteApiKey();
  const { toast } = useToast();
  const [open, setOpen] = useState(false);
  const [created, setCreated] = useState<string | null>(null);

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors, isSubmitting },
  } = useForm<z.infer<typeof createSchema>>({
    resolver: zodResolver(createSchema),
    defaultValues: { name: "", scopes: "" },
  });

  const submit = async (values: z.infer<typeof createSchema>) => {
    try {
      const scopes = values.scopes
        ? values.scopes.split(",").map((s) => s.trim()).filter(Boolean)
        : [];
      const res = await createKey.mutateAsync({ name: values.name, scopes });
      setCreated(res.key);
      reset();
    } catch (err) {
      toast(err instanceof Error ? err.message : "error creating key", "error");
    }
  };

  return (
    <AppPage>
      <AppPageHeader
        title="API Keys"
        description="Automation keys for the API v1. The value is shown only once."
        actions={ <Button leftIcon="key" onClick={() => setOpen(true)}>
          New key
        </Button> }
      />

      <AppCard>
        <Table headers={["Name", "Scopes", "Created at", ""]}>
          {(keys ?? []).map((k) => (
            <tr key={k.id} className="hover:bg-surface-container-high transition-colors">
              <td className="px-sm py-2 font-body-md text-body-md text-on-surface">{k.name}</td>
              <td className="px-sm py-2 font-code-md text-code-md text-on-surface-variant">
                {k.scopes.length ? k.scopes.join(", ") : "no scopes"}
              </td>
              <td className="px-sm py-2 font-code-md text-code-md text-on-surface-variant/60">
                {new Date(k.created_at).toLocaleDateString("pt-BR")}
              </td>
              <td className="px-sm py-2 text-right">
                <button
                  onClick={() => deleteKey.mutate(k.id, { onSuccess: () => toast("Key revoked") })}
                  className="material-symbols-outlined text-[16px] text-on-surface-variant hover:text-error transition-colors"
                >
                  delete
                </button>
              </td>
            </tr>
          ))}
        </Table>
      </AppCard>

      <Modal open={open} onClose={() => setOpen(false)} title="New API Key">
        {created ? (
          <div className="space-y-lg">
            <p className="font-body-sm text-body-sm text-on-surface-variant">
              Save this key now — it cannot be recovered again.
            </p>
            <CodeBlock>{created}</CodeBlock>
            <div className="flex justify-end">
              <Button onClick={() => { setCreated(null); setOpen(false); }}>Done</Button>
            </div>
          </div>
        ) : (
          <form onSubmit={handleSubmit(submit)} className="space-y-lg" noValidate>
            <Field label="Nome" hint={errors.name?.message}>
              <Input icon="label" placeholder="ci-deploy" {...register("name")} />
            </Field>
            <Field label="Scopes" hint={errors.scopes?.message || "comma-separated (empty = role permissions)"}>
              <Input icon="tune" placeholder="app.deploy, app.read" {...register("scopes")} />
            </Field>
            <div className="flex justify-end gap-md">
              <Button type="button" variant="ghost" onClick={() => setOpen(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={isSubmitting}>
                Create
              </Button>
            </div>
          </form>
        )}
      </Modal>
    </AppPage>
  );
}

export const Route = createFileRoute('/_shell/api-keys/')({
  component: ApiKeys,
});