import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { useCreatePreview, useDeletePreview, usePreviews } from "../../../../api/hooks";
import {
  Button,
  Card,
  Field,
  Input,
  StatusPill,
  Table,
  useToast,
} from "../../../../components/ui";
const schema = z.object({
  branch: z.string().min(1, "Branch is required"),
});

export function Previews({ appID }: { appID: string }) {
  const { data: previews } = usePreviews(appID);
  const createPreview = useCreatePreview(appID);
  const deletePreview = useDeletePreview(appID);
  const { toast } = useToast();
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<z.infer<typeof schema>>({ resolver: zodResolver(schema), defaultValues: { branch: "" } });

  const submit = async (values: z.infer<typeof schema>) => {
    try {
      const p = await createPreview.mutateAsync(values.branch);
      toast(`Preview created at ${p.domain}`);
    } catch (err) {
      toast(err instanceof Error ? err.message : "failed to create preview", "error");
    }
  };

  return (
    <Card className="mt-lg">
      <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase mb-md">
        Preview Deployments
      </h2>
      <form onSubmit={handleSubmit(submit)} className="flex items-end gap-md mb-md" noValidate>
        <div className="grow">
          <Field label="Branch" hint={errors.branch?.message}>
            <Input icon="fork_right" placeholder="feature-xyz" {...register("branch")} />
          </Field>
        </div>
        <Button type="submit" disabled={isSubmitting}>
          <span className="material-symbols-outlined text-[16px]">rocket_launch</span>
          Create preview
        </Button>
      </form>
      <Table headers={["Branch", "Domain", "Status", ""]}>
        {(previews ?? []).map((p) => (
          <tr key={p.id} className="hover:bg-surface-container-high transition-colors">
            <td className="px-sm py-2 font-code-md text-code-md text-on-surface">{p.branch}</td>
            <td className="px-sm py-2 font-code-md text-code-md text-primary">{p.domain}</td>
            <td className="px-sm py-2">
              <StatusPill status={p.status} pulse={p.status === "active"} />
            </td>
            <td className="px-sm py-2 text-right">
              <button
                onClick={() => deletePreview.mutate(p.id, { onSuccess: () => toast("Preview removed") })}
                className="material-symbols-outlined text-[16px] text-on-surface-variant hover:text-error transition-colors"
              >
                delete
              </button>
            </td>
          </tr>
        ))}
      </Table>
      {(previews ?? []).length === 0 && (
        <p className="font-body-sm text-body-sm text-on-surface-variant mt-sm">No preview deployments.</p>
      )}
    </Card>
  );
}
