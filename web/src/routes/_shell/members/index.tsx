import { createFileRoute } from "@tanstack/react-router";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { useAddMember, useMembers, useUpdateMember } from "../../../hooks";
import {
  Button,
  Card,
  Field,
  Input,
  Modal,
  Select,
  StatusPill,
  Table,
  useToast,
} from "../../../components/ui";
import { AppPage, AppPageHeader, AppCard } from "../../../components/ds";
import { useState } from "react";

const addSchema = z.object({
  email: z.string().email("Invalid email"),
  name: z.string().optional(),
  password: z.string().min(8, "Password must be at least 8 characters"),
  role: z.enum(["owner", "admin", "developer", "viewer"]),
});

const ROLE_DESC: Record<string, string> = {
  owner: "Full control of the organization",
  admin: "Apps, members, backups and certificates",
  developer: "Create, edit and deploy apps",
  viewer: "Read-only access",
};

function Members() {
  const { data: members } = useMembers();
  const addMember = useAddMember();
  const updateMember = useUpdateMember();
  const { toast } = useToast();
  const [open, setOpen] = useState(false);

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors, isSubmitting },
  } = useForm<z.infer<typeof addSchema>>({
    resolver: zodResolver(addSchema),
    defaultValues: { email: "", name: "", password: "", role: "developer" },
  });

  const submit = async (values: z.infer<typeof addSchema>) => {
    try {
      await addMember.mutateAsync({ ...values, name: values.name || values.email });
      toast("Member added");
      setOpen(false);
      reset();
    } catch (err) {
      toast(err instanceof Error ? err.message : "error adding member", "error");
    }
  };

  return (
    <AppPage>
      <AppPageHeader
        title="Members"
        description="Organization access control by roles."
        actions={
          <Button leftIcon="person_add" onClick={() => setOpen(true)}>
            Add user
          </Button>
        }
      />

      <AppCard>
        <Table headers={["User", "Email", "Role", "Permissions"]}>
          {(members ?? []).map((m) => (
            <tr key={m.user_id} className="hover:bg-surface-container-high transition-colors">
              <td className="px-sm py-2 font-body-md text-body-md text-on-surface">{m.name}</td>
              <td className="px-sm py-2 font-code-md text-code-md text-on-surface-variant">{m.email}</td>
              <td className="px-sm py-2">
                <div className="flex items-center gap-sm">
                  <StatusPill status={m.role} />
                  <Select
                    defaultValue={m.role}
                    className="!w-auto !py-1 !text-body-sm"
                    onChange={(e) =>
                      updateMember.mutate(
                        { userID: m.user_id, role: e.target.value },
                        { onSuccess: () => toast("Role atualizado"), onError: (err) => toast(err.message, "error") }
                      )
                    }
                  >
                    <option value="owner">owner</option>
                    <option value="admin">admin</option>
                    <option value="developer">developer</option>
                    <option value="viewer">viewer</option>
                  </Select>
                </div>
              </td>
              <td className="px-sm py-2 font-body-sm text-body-sm text-on-surface-variant">{ROLE_DESC[m.role]}</td>
            </tr>
          ))}
        </Table>
      </AppCard>

      <Modal open={open} onClose={() => setOpen(false)} title="Add user">
        <form onSubmit={handleSubmit(submit)} className="space-y-lg" noValidate>
          <Field label="Name" hint={errors.name?.message}>
            <Input icon="badge" placeholder="Jane Doe" {...register("name")} />
          </Field>
          <Field label="Email" hint={errors.email?.message}>
            <Input icon="mail" type="email" placeholder="dev@organization.tld" {...register("email")} />
          </Field>
          <Field label="Password" hint={errors.password?.message}>
            <Input icon="key" type="password" placeholder="Minimum 8 characters" {...register("password")} />
          </Field>
          <Field label="Role" hint={errors.role?.message}>
            <Select {...register("role")}>
              <option value="owner">owner</option>
              <option value="admin">admin</option>
              <option value="developer">developer</option>
              <option value="viewer">viewer</option>
            </Select>
          </Field>
          <div className="flex justify-end gap-md">
            <Button type="button" variant="ghost" onClick={() => setOpen(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={isSubmitting}>
              Add
            </Button>
          </div>
        </form>
      </Modal>
    </AppPage>
  );
}

export const Route = createFileRoute('/_shell/members/')({
  component: Members,
});
