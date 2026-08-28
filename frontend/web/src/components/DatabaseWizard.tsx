import { useState } from "react";
import { Eye, EyeSlash } from "@phosphor-icons/react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { useCreateDatabase, useProjects } from "../hooks";
import { TechIcon } from "./TechIcon";
import { AdvancedSettings } from "./AdvancedSettings";
import { Button, Dialog, Field, Input, NativeSelect, useToast } from "@aether/design-system";

export const DB_ENGINES: { value: string; label: string; tagline: string; icon: string; versions: string[] }[] = [
  { value: "postgres", label: "PostgreSQL", tagline: "Relational SQL", icon: "postgres", versions: ["17", "16", "15", "14", "latest"] },
  { value: "mysql", label: "MySQL", tagline: "Relational SQL", icon: "mysql", versions: ["8.4", "8.0", "8.4.3", "latest"] },
  { value: "mariadb", label: "MariaDB", tagline: "MySQL compatible", icon: "mariadb", versions: ["11.4", "11.6", "10.11", "latest"] },
  { value: "mongodb", label: "MongoDB", tagline: "NoSQL document", icon: "mongodb", versions: ["7.0", "6.0", "7.0.16", "latest"] },
  { value: "redis", label: "Redis", tagline: "In-memory datastore", icon: "redis", versions: ["7.2", "7.4", "6.2", "latest"] },
  { value: "mssql", label: "SQL Server", tagline: "Enterprise SQL", icon: "mssql", versions: ["2022", "2022-CU15", "2022-CU14", "latest"] },
  { value: "oracle", label: "Oracle", tagline: "Enterprise SQL", icon: "oracle", versions: ["23", "23.3", "23.2", "latest"] },
];

const schema = z.object({
  project_id: z.string().min(1, "Project is required"),
  name: z.string().min(1, "Name is required").regex(/^[a-z0-9-_]+$/, "lowercase only"),
  engine: z.enum(["postgres", "mysql", "mariadb", "mongodb", "redis", "mssql", "oracle"]),
  version: z.string().optional(),
  user: z
    .string()
    .optional()
    .refine((v) => !v || /^[a-zA-Z][a-zA-Z0-9_]{0,62}$/.test(v), "lowercase/numbers/underscore, 63 max"),
  password: z.string().optional().refine((v) => !v || (v.length >= 8 && v.length <= 128), "8-128 characters"),
});

type FormValues = z.infer<typeof schema>;

export function DatabaseWizard({ open, onClose, fixedProjectId, initialEngine }: { open: boolean; onClose: () => void; fixedProjectId?: string; initialEngine?: string }) {
  const { data: projects } = useProjects();
  const createDb = useCreateDatabase();
  const { add } = useToast();
  const [memMB, setMemMB] = useState(512);
  const [storageMB, setStorageMB] = useState(0);
  const [showPassword, setShowPassword] = useState(false);

  const {
    register,
    setValue,
    watch,
    handleSubmit,
    reset,
    formState: { errors, isSubmitting },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { project_id: fixedProjectId ?? "", engine: (DB_ENGINES.find((e) => e.value === initialEngine)?.value ?? "postgres") as FormValues["engine"], version: "" },
  });
  const engineValue = watch("engine");
  const selected = DB_ENGINES.find((e) => e.value === engineValue) ?? DB_ENGINES[0];

  const submit = async (values: FormValues) => {
    try {
      await createDb.mutateAsync({
        project_id: values.project_id,
        name: values.name,
        engine: values.engine,
        version: values.version || undefined,
        user: values.user?.trim() || undefined,
        password: values.password || undefined,
        mem_mb: memMB,
        storage_mb: storageMB,
      });
      add({ title: "Deploy it manually from the service page", tone: "info" });
      onClose();
      reset();
    } catch (err) {
      add({ title: err instanceof Error ? err.message : "Failed to create database", tone: "error" });
    }
  };

  return (
    <Dialog trigger={<span />} open={open} onOpenChange={(value) => { if (!value) onClose(); }} title="Create database">
      <form onSubmit={handleSubmit(submit)} className="space-y-lg" noValidate>
        <div>
          <p className="font-label-caps text-label-caps text-on-surface-variant uppercase mb-sm">Engine</p>
          {errors.engine && <p className="font-body-sm text-body-sm text-error mb-sm">{errors.engine.message}</p>}
          <div className="grid grid-cols-2 sm:grid-cols-3 gap-sm">
            {DB_ENGINES.map((eng) => {
              const active = engineValue === eng.value;
              return (
                <button
                  key={eng.value}
                  type="button"
                  onClick={() => {
                    setValue("engine", eng.value as never, { shouldValidate: true });
                    setValue("version", "", { shouldValidate: true });
                  }}
                  className={`flex flex-col gap-sm p-sm rounded-lg border transition-colors text-left ${active ? "border-primary bg-primary/10" : "border-outline-variant hover:border-primary/40 bg-surface-container-low"}`}
                >
                  <div className="flex items-center justify-between">
                    <TechIcon name={eng.icon} size={22} className={active ? "text-primary" : "text-on-surface-variant"} />
                    <span className={`w-2 h-2 rounded-full ${active ? "bg-green-400" : "bg-outline-variant"}`} />
                  </div>
                  <div>
                    <p className="font-body-md text-body-md text-on-surface">{eng.label}</p>
                    <p className="font-code-md text-code-md text-on-surface-variant/70">{eng.tagline}</p>
                  </div>
                </button>
              );
            })}
          </div>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-lg">
          <Field label="Project" error={errors.project_id?.message}>
            <NativeSelect {...register("project_id")} disabled={!!fixedProjectId} options={[{ label: "Select...", value: "" }, ...(projects ?? []).map((p) => ({ label: p.name, value: p.id }))]} />
          </Field>
          <Field label="Name" error={errors.name?.message}>
            <Input placeholder="main-db" {...register("name")} />
          </Field>
          <Field label="Version" description="Official image tag — empty = default for the engine">
            <NativeSelect {...register("version")} options={[{ label: "Default", value: "" }, ...selected.versions.map((v) => ({ label: v, value: v }))]} />
          </Field>
          <Field label="Database user" error={errors.user?.message} description={!errors.user?.message ? "Empty = aether (auto)" : undefined}>
            <Input placeholder="aether" {...register("user")} />
          </Field>
          <Field label="Password" error={errors.password?.message} description={!errors.password?.message ? "Empty = auto-generated" : undefined}>
            <div className="relative">
              <Input type={showPassword ? "text" : "password"} placeholder="••••••••" {...register("password")} />
              <button
                type="button"
                onClick={() => setShowPassword((v) => !v)}
                aria-label={showPassword ? "Hide password" : "Show password"}
                className="absolute right-sm top-1/2 -translate-y-1/2 text-on-surface-variant hover:text-on-surface"
              >
                {showPassword ? <EyeSlash size={16} /> : <Eye size={16} />}
              </button>
            </div>
          </Field>
        </div>

        <AdvancedSettings
          showHealth={false}
          values={{ cpu: "0.5", memMB, storageMB, healthEnabled: false, healthPath: "/" }}
          onChange={(v) => {
            setMemMB(v.memMB);
            setStorageMB(v.storageMB);
          }}
        />

        <div className="flex justify-end gap-md border-t border-outline-variant pt-lg">
          <Button type="button" variant="ghost" onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit" disabled={isSubmitting}>
            {isSubmitting ? "Provisioning..." : "Create database"}
          </Button>
        </div>
      </form>
    </Dialog>
  );
}
