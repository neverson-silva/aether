import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { UserPlus } from "@phosphor-icons/react";
import type { Icon as DesignIcon } from "@aether/design-system";
import {
  Badge,
  Button,
  Card,
  Dialog,
  EmptyState,
  Field,
  Input,
  Select,
  Skeleton,
  useToast,
} from "@aether/design-system";
import { useAddMember, useMembers, useUpdateMember } from "../../../hooks";

const schema = z.object({
  email: z.string().email("Invalid email"),
  name: z.string().optional(),
  password: z.string().min(8, "Password must be at least 8 characters"),
  role: z.enum(["owner", "admin", "developer", "viewer"]),
});
const roleDescription: Record<string, string> = {
  owner: "Full organization control",
  admin: "Apps, members, backups and certificates",
  developer: "Create, edit and deploy apps",
  viewer: "Read-only access",
};
function Members() {
  const query = useMembers();
  const addMember = useAddMember();
  const update = useUpdateMember();
  const { add } = useToast();
  const [open, setOpen] = useState(false);
  const form = useForm<z.infer<typeof schema>>({
    resolver: zodResolver(schema),
    defaultValues: { email: "", name: "", password: "", role: "developer" },
  });
  const submit = async (values: z.infer<typeof schema>) => {
    try {
      await addMember.mutateAsync({
        ...values,
        name: values.name || values.email,
      });
      setOpen(false);
      form.reset();
      add({ title: "Member added", tone: "success" });
    } catch (error) {
      add({
        title: "Could not add member",
        description:
          error instanceof Error ? error.message : "Try again later.",
        tone: "error",
      });
    }
  };
  return (
    <main className="mx-auto flex w-full max-w-screen-2xl flex-col gap-lg p-6 lg:p-8">
      <header className="relative isolate overflow-hidden rounded-2xl border border-border bg-gradient-to-br from-primary/10 via-surface-card to-secondary/5 px-lg py-xl shadow-sm sm:flex sm:items-end sm:justify-between">
        <div>
          <p className="font-label-caps text-label-caps uppercase text-primary">
            Organization
          </p>
          <h1 className="mt-sm font-display-lg text-display-lg tracking-[-0.04em] text-on-surface">
            Members
          </h1>
          <p className="mt-sm text-body-md text-on-surface-variant">
            Manage organization access by role.
          </p>
        </div>
        <Dialog
          open={open}
          onOpenChange={setOpen}
          title="Add member"
          trigger={
            <Button icon={UserPlus as unknown as DesignIcon}>Add member</Button>
          }
        >
          <form onSubmit={form.handleSubmit(submit)} className="space-y-5">
            <Field label="Name" error={form.formState.errors.name?.message}>
              <Input placeholder="Jane Doe" {...form.register("name")} />
            </Field>
            <Field label="Email" error={form.formState.errors.email?.message}>
              <Input
                type="email"
                placeholder="dev@example.com"
                {...form.register("email")}
              />
            </Field>
            <Field
              label="Password"
              error={form.formState.errors.password?.message}
            >
              <Input
                type="password"
                placeholder="Minimum 8 characters"
                {...form.register("password")}
              />
            </Field>
            <Select
              label="Role"
              {...form.register("role")}
              options={["owner", "admin", "developer", "viewer"].map(
                (role) => ({ label: role, value: role }),
              )}
            />
            <div className="flex justify-end gap-2 border-t border-border pt-4">
              <Button
                type="button"
                variant="ghost"
                onClick={() => setOpen(false)}
              >
                Cancel
              </Button>
              <Button type="submit" loading={addMember.isPending}>
                Add member
              </Button>
            </div>
          </form>
        </Dialog>
      </header>
      <Card variant="elevated" padding="none">
        <div className="overflow-x-auto">
          <table className="w-full min-w-[760px] text-left">
            <thead className="border-b border-border text-label-caps text-muted-foreground">
              <tr>
                {["User", "Email", "Role", "Permissions"].map((header) => (
                  <th key={header} className="px-3 py-3">
                    {header}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {query.isLoading ? (
                <tr>
                  <td colSpan={4} className="p-6">
                    <Skeleton variant="table" />
                  </td>
                </tr>
              ) : (
                (query.data ?? []).map((member) => (
                  <tr
                    key={member.user_id ?? member.email}
                    className="hover:bg-surface-container"
                  >
                    <td className="px-3 py-3 text-foreground">{member.name}</td>
                    <td className="px-3 py-3 font-mono text-code-md text-muted-foreground">
                      {member.email}
                    </td>
                    <td className="px-3 py-3">
                      <div className="flex items-center gap-2">
                        <Badge
                          tone={
                            member.role === "owner" || member.role === "admin"
                              ? "accent"
                              : "neutral"
                          }
                        >
                          {member.role}
                        </Badge>
                        <Select
                          aria-label={`Change role for ${member.name}`}
                          value={member.role}
                          onValueChange={(value) =>
                            value &&
                            update.mutate(
                              {
                                userID: member.user_id,
                                role: value,
                              },
                              {
                                onSuccess: () =>
                                  add({
                                    title: "Role updated",
                                    tone: "success",
                                  }),
                                onError: (error) =>
                                  add({
                                    title: "Could not update role",
                                    description: error.message,
                                    tone: "error",
                                  }),
                              },
                            )
                          }
                          options={[
                            "owner",
                            "admin",
                            "developer",
                            "viewer",
                          ].map((role) => ({ label: role, value: role }))}
                        />
                      </div>
                    </td>
                    <td className="px-3 py-3 text-body-sm text-muted-foreground">
                      {roleDescription[member.role]}
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
        {!query.isLoading && !query.data?.length ? (
          <EmptyState
            icon={UserPlus as unknown as DesignIcon}
            title="No members yet"
            description="Add a member to collaborate on this organization."
          />
        ) : null}
      </Card>
    </main>
  );
}
export const Route = createFileRoute("/_shell/members/")({
  component: Members,
});
