import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { useCreatePreview, useDeletePreview, usePreviews } from "../../../../hooks";
import { GitBranch, Rocket, Trash } from "@phosphor-icons/react";
import type { Icon as DesignIcon } from "@aether/design-system";
import { Badge, Button, Card, Field, Input, useToast } from "@aether/design-system";
const schema = z.object({
  branch: z.string().min(1, "Branch is required"),
});

export function Previews({ appID }: { appID: string }) {
  const { data: previews } = usePreviews(appID);
  const createPreview = useCreatePreview(appID);
  const deletePreview = useDeletePreview(appID);
  const { add } = useToast();
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<z.infer<typeof schema>>({ resolver: zodResolver(schema), defaultValues: { branch: "" } });

  const submit = async (values: z.infer<typeof schema>) => {
    try {
      await createPreview.mutateAsync(values.branch);
    } catch (err) {
      add({ title: "Could not create preview", description: err instanceof Error ? err.message : "Try again later.", tone: "error" });
    }
  };

  return (
    <Card className="mt-lg">
      <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase mb-md">
        Preview Deployments
      </h2>
      <form onSubmit={handleSubmit(submit)} className="flex items-end gap-md mb-md" noValidate>
        <div className="grow">
          <Field label="Branch" error={errors.branch?.message}>
            <Input leadingIcon={GitBranch as unknown as DesignIcon} placeholder="feature-xyz" {...register("branch")} />
          </Field>
        </div>
        <Button type="submit" disabled={isSubmitting}>
          <Rocket size={16} />Create preview
        </Button>
      </form>
      <div className="overflow-x-auto"><table className="w-full min-w-[600px] text-left"><thead><tr className="border-b border-border text-label-caps text-muted-foreground"><th className="px-3 py-3">Branch</th><th className="px-3 py-3">Domain</th><th className="px-3 py-3">Status</th><th className="px-3 py-3" /></tr></thead><tbody className="divide-y divide-border">{(previews ?? []).map((p) => <tr key={p.id} className="transition-colors hover:bg-surface-container"><td className="px-3 py-3 font-mono text-code-md text-foreground">{p.branch}</td><td className="px-3 py-3 font-mono text-code-md text-primary">{p.domain}</td><td className="px-3 py-3"><Badge tone={p.status === "active" ? "success" : "warning"}>{p.status}</Badge></td><td className="px-3 py-3 text-right"><Button variant="quiet" size="sm" aria-label={`Delete ${p.branch}`} onClick={() => deletePreview.mutate(p.id)}><Trash size={16} /></Button></td></tr>)}</tbody></table></div>
      {(previews ?? []).length === 0 && (
        <p className="font-body-sm text-body-sm text-on-surface-variant mt-sm">No preview deployments.</p>
      )}
    </Card>
  );
}
