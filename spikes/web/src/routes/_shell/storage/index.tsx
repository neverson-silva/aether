import { useState } from "react";
import { useSnapshots, useCreateSnapshot, useRestoreSnapshot, useDeleteSnapshot, useSnapshotSchedules, useCreateSnapshotSchedule, useDeleteSnapshotSchedule, useApps } from "../../../api/hooks";
import { fmtDate } from "../../../components/ui";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { createFileRoute } from "@tanstack/react-router";
import {
  useCreateS3,
  useDeleteS3,
  useS3Destinations,
} from "../../../api/hooks";
import {
  Button,
  Field,
  Input,
  Modal,
  Table,
  useToast,
  Card,
  Select,
} from "../../../components/ui";
import { AppPage, AppPageHeader, AppCard } from "../../../components/ds";

const schema = z.object({
  name: z.string().min(1, "Name is required"),
  endpoint: z.string().min(1, "Endpoint is required"),
  bucket: z.string().min(1, "Bucket is required"),
  region: z.string().default("us-east-1"),
  access_key: z.string().min(1, "Access key is required"),
  secret_key: z.string().min(1, "Secret key is required"),
  provider: z.string().optional(),
});

function useS3() {
  return useS3Destinations();
}

export function Storage() {
  const { data: dests } = useS3();
  const createS3 = useCreateS3();
  const deleteS3 = useDeleteS3();
  const { toast } = useToast();
  const [open, setOpen] = useState(false);
  const {
    register,
    handleSubmit,
    reset,
    formState: { errors, isSubmitting },
  } = useForm<z.input<typeof schema>, any, z.output<typeof schema>>({
    resolver: zodResolver(schema),
    defaultValues: { provider: "aws", region: "us-east-1" },
  });

  const submit = async (values: z.infer<typeof schema>) => {
    try {
      await createS3.mutateAsync({
        name: values.name,
        endpoint: values.endpoint,
        bucket: values.bucket,
        region: values.region,
        access_key: values.access_key,
        secret_key: values.secret_key,
      });
      toast("Destination saved");
      setOpen(false);
      reset();
    } catch (err) {
      toast(err instanceof Error ? err.message : "failed to create destination", "error");
    }
  };

  return (
    <AppPage>
      <AppPageHeader
        title="S3 Destinations"
        description="Blob storage destinations for backups, images and artifacts (S3-compatible)."
        actions={
          <Button leftIcon="add" onClick={() => setOpen(true)}>
            New destination
          </Button>
        }
      />

      <AppCard>
        <Table headers={["Name", "Endpoint", "Bucket", "Region", ""]}>
          {(dests ?? []).map((d) => (
            <tr key={d.id} className="hover:bg-surface-container-high transition-colors">
              <td className="px-sm py-2 font-code-md text-code-md text-on-surface">{d.name}</td>
              <td className="px-sm py-2 font-code-md text-code-md text-on-surface-variant">{d.endpoint}</td>
              <td className="px-sm py-2 font-code-md text-code-md text-on-surface-variant">{d.bucket}</td>
              <td className="px-sm py-2 font-code-md text-code-md text-on-surface-variant">{d.region}</td>
              <td className="px-sm py-2 text-right">
                <button
                  onClick={() => deleteS3.mutate(d.id, { onSuccess: () => toast("Destination removed") })}
                  className="material-symbols-outlined text-[16px] text-on-surface-variant hover:text-error transition-colors"
                >
                  delete
                </button>
              </td>
            </tr>
          ))}
        </Table>
        {(dests ?? []).length === 0 && (
          <p className="font-body-sm text-body-sm text-on-surface-variant p-sm">
            No destinations configured.
          </p>
        )}
      </AppCard>

      <Modal open={open} onClose={() => setOpen(false)} title="New S3 Destination">
        <form onSubmit={handleSubmit(submit)} className="space-y-lg" noValidate>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-lg">
            <Field label="Name" hint={errors.name?.message}>
              <Input icon="label" placeholder="prod-backups" {...register("name")} />
            </Field>
            <Field label="Endpoint" hint={errors.endpoint?.message}>
              <Input icon="dns" placeholder="https://s3.us-east-1.amazonaws.com" {...register("endpoint")} />
            </Field>
            <Field label="Bucket" hint={errors.bucket?.message}>
              <Input icon="storage" placeholder="aether-backups" {...register("bucket")} />
            </Field>
            <Field label="Region" hint={errors.region?.message}>
              <Input icon="public" placeholder="us-east-1" {...register("region")} />
            </Field>
            <Field label="Access Key" hint={errors.access_key?.message}>
              <Input icon="key" {...register("access_key")} />
            </Field>
            <Field label="Secret Key" hint={errors.secret_key?.message}>
              <Input icon="vpn_key" type="password" {...register("secret_key")} />
            </Field>
          </div>
          <div className="flex justify-end gap-md border-t border-outline-variant pt-lg">
            <Button type="button" variant="ghost" onClick={() => setOpen(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={isSubmitting}>
              Save destination
            </Button>
          </div>
        </form>
      </Modal>
    </AppPage>
  );
}


export function SnapshotSchedulesSection() {
  const { data: schedules } = useSnapshotSchedules();
  const { data: apps } = useApps();
  const create = useCreateSnapshotSchedule();
  const remove = useDeleteSnapshotSchedule();
  const { toast } = useToast();
  const [appID, setAppID] = useState("");
  const [volume, setVolume] = useState("");
  const [cron, setCron] = useState("@daily");
  const [retention, setRetention] = useState("7");

  const add = async () => {
    if (!appID || !volume.trim()) {
      toast("Select an app and enter a volume", "error");
      return;
    }
    try {
      await create.mutateAsync({ app_id: appID, volume: volume.trim(), name_prefix: "scheduled", cron, retention: parseInt(retention) || 7, enabled: true });
      toast(`Snapshot schedule created (${cron})`);
      setVolume("");
    } catch (err) {
      toast(err instanceof Error ? err.message : "failed to create schedule", "error");
    }
  };

  return (
    <Card>
      <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase mb-md">Snapshot schedules</h2>
      <div className="grid grid-cols-1 md:grid-cols-5 gap-sm mb-sm">
        <Select value={appID} onChange={(e) => setAppID(e.target.value)}>
          <option value="">App...</option>
          {(apps ?? []).map((a) => (
            <option key={a.id} value={a.id}>{a.name}</option>
          ))}
        </Select>
        <Input icon="folder" placeholder="volume path" value={volume} onChange={(e) => setVolume(e.target.value)} />
        <Select value={cron} onChange={(e) => setCron(e.target.value)}>
          <option value="@daily">Daily</option>
          <option value="@weekly">Weekly</option>
          <option value="@hourly">Hourly</option>
          <option value="0 3 * * *">Daily 03:00</option>
          <option value="30 22 * * 5">Weekly Fri 22:30</option>
        </Select>
        <Input icon="history" placeholder="retention" value={retention} onChange={(e) => setRetention(e.target.value)} type="number" />
        <Button onClick={add}>
          <span className="material-symbols-outlined text-[16px]">add</span>
          Schedule
        </Button>
      </div>
      <div className="space-y-sm">
        {(schedules ?? []).map((sc) => (
          <div key={sc.id} className="flex items-center gap-sm p-sm rounded border border-outline-variant/60">
            <span className={`material-symbols-outlined text-[16px] ${sc.enabled ? "text-[#4ade80]" : "text-on-surface-variant/40"}`}>schedule</span>
            <span className="font-body-md text-body-md text-on-surface flex-1">{sc.volume}</span>
            <span className="font-code-md text-code-md text-on-surface-variant">{sc.cron}</span>
            <span className="font-code-md text-code-md text-on-surface-variant/60">keep {sc.retention}</span>
            {sc.next_run && <span className="font-code-md text-code-md text-on-surface-variant/60">next {new Date(sc.next_run).toLocaleString()}</span>}
            <Button variant="ghost" onClick={() => remove.mutateAsync(sc.id)}>
              <span className="material-symbols-outlined text-[16px]">delete</span>
            </Button>
          </div>
        ))}
        {(schedules ?? []).length === 0 && <p className="font-body-sm text-body-sm text-on-surface-variant">No schedules yet. Automate daily/weekly snapshots with retention.</p>}
      </div>
    </Card>
  );
}

export function SnapshotsSection() {
  const { data: snaps } = useSnapshots();
  const create = useCreateSnapshot();
  const restore = useRestoreSnapshot();
  const remove = useDeleteSnapshot();
  const { toast } = useToast();
  const [open, setOpen] = useState(false);
  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<z.input<typeof snapSchema>, any, z.output<typeof snapSchema>>({
    resolver: zodResolver(snapSchema),
  });

  const submit = async (values: z.infer<typeof snapSchema>) => {
    try {
      await create.mutateAsync({ app_id: values.app_id || "", volume: values.volume, name: values.name });
      toast("Snapshot created");
      setOpen(false);
      reset();
    } catch (err) {
      toast(err instanceof Error ? err.message : "failed", "error");
    }
  };

  return (
    <div className="mt-lg">
      <div className="flex items-center justify-between">
        <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase">Volume snapshots (zstd + dedup)</h2>
        <Button onClick={() => setOpen(true)}>
          <span className="material-symbols-outlined text-[16px]">add</span>
          New snapshot
        </Button>
      </div>
      <div className="bg-surface-container-low border border-outline-variant rounded-lg mt-md">
        <Table headers={["Name", "Volume", "Size", "Chunks", "Dedup saved", "Created", ""]}>
          {(snaps ?? []).map((sn) => (
            <tr key={sn.id} className="hover:bg-surface-container-high transition-colors">
              <td className="px-sm py-2 font-body-md text-body-md text-on-surface">{sn.name}</td>
              <td className="px-sm py-2 font-code-md text-code-md text-on-surface-variant">{sn.volume}</td>
              <td className="px-sm py-2 font-code-md text-code-md text-on-surface-variant">{fmtBytesUI(sn.size)}</td>
              <td className="px-sm py-2 font-code-md text-code-md text-on-surface-variant">{sn.chunks}</td>
              <td className="px-sm py-2 font-code-md text-code-md text-[#4ade80]">{fmtBytesUI(sn.dedup_saved)}</td>
              <td className="px-sm py-2 font-code-md text-code-md text-on-surface-variant/60">{fmtDate(sn.created_at)}</td>
              <td className="px-sm py-2 text-right space-x-sm">
                <button
                  onClick={() => restore.mutate({ id: sn.id }, { onSuccess: () => toast("Restore started") })}
                  className="material-symbols-outlined text-[16px] text-on-surface-variant hover:text-primary transition-colors"
                  title="Restore"
                >
                  history
                </button>
                <button
                  onClick={() => remove.mutate(sn.id)}
                  className="material-symbols-outlined text-[16px] text-on-surface-variant hover:text-error transition-colors"
                >
                  delete
                </button>
              </td>
            </tr>
          ))}
        </Table>
        {(snaps ?? []).length === 0 && (
          <p className="font-body-sm text-body-sm text-on-surface-variant p-sm">
            No snapshots yet. Chunks are content-addressed (sha256) — identical blocks are stored once.
          </p>
        )}
      </div>
      <Modal open={open} onClose={() => setOpen(false)} title="New volume snapshot">
        <form onSubmit={handleSubmit(submit)} className="space-y-lg" noValidate>
          <Field label="Name" hint={errors.name?.message}>
            <Input icon="label" placeholder="pre-deploy-1" {...register("name")} />
          </Field>
          <Field label="Volume" hint={errors.volume?.message || "Docker/Podman volume name (e.g. aether-web-data)"}>
            <Input icon="storage" placeholder="aether-app-data" {...register("volume")} />
          </Field>
          <Field label="App ID (optional)" hint={errors.app_id?.message}>
            <Input icon="apps" placeholder="app id to link the snapshot" {...register("app_id")} />
          </Field>
          <div className="flex justify-end gap-md border-t border-outline-variant pt-lg">
            <Button type="button" variant="ghost" onClick={() => setOpen(false)}>Cancel</Button>
            <Button type="submit">Create</Button>
          </div>
        </form>
      </Modal>
    </div>
  );
}

const snapSchema = z.object({
  name: z.string().min(1, "Name is required"),
  volume: z.string().min(1, "Volume is required"),
  app_id: z.string().optional(),
});

function fmtBytesUI(n: number): string {
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KiB`;
  return `${(n / (1024 * 1024)).toFixed(1)} MiB`;
}

export function StoragePage() {
  return (
    <>
      <Storage />
      <SnapshotSchedulesSection />
      <SnapshotsSection />
    </>
  );
}

export const Route = createFileRoute('/_shell/storage/')({
  component: StoragePage,
});
