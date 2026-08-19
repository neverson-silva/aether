import { useEffect, useMemo, useState } from "react";
import { createFileRoute } from "@tanstack/react-router";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import {
  useApps,
  useCreateS3,
  useCreateSnapshotSchedule,
  useDeleteS3,
  useDeleteSnapshotSchedule,
  useGoogleConnect,
  useGoogleDisconnect,
  useS3Destinations,
  useSnapshotSchedules,
  useTestS3,
  useUpdateS3,
} from "../../../hooks";
import type { DestinationType, S3Destination } from "../../../api/types";
import { Button, Card, Field, Input, Modal, Select, Table, useToast } from "../../../components/ui";
import { AppPage, AppPageHeader, AppCard } from "../../../components/ds";
import { cn } from "../../../components/ui";

type ProviderDef = {
  label: string;
  description: string;
  fields: string[];
  endpointDerived?: boolean;
  endpointHint?: (v: Record<string, string>) => string;
};

const PROVIDERS: Record<DestinationType, ProviderDef> = {
  "aws": {
    label: "Amazon S3",
    description: "AWS S3 with the region-resolved endpoint.",
    fields: ["bucket", "region", "credentials"],
    endpointDerived: true,
    endpointHint: (v) => `https://s3.${v.region || "us-east-1"}.amazonaws.com`,
  },
  "cloudflare-r2": {
    label: "Cloudflare R2",
    description: "R2 with the endpoint derived from your account ID.",
    fields: ["bucket", "account", "credentials"],
    endpointDerived: true,
    endpointHint: (v) => (v.account ? `https://${v.account}.r2.cloudflarestorage.com` : "Enter your account ID to derive the endpoint"),
  },
  "minio": {
    label: "MinIO",
    description: "Self-hosted MinIO — your endpoint is required.",
    fields: ["endpoint", "bucket", "credentials"],
  },
  "custom-s3": {
    label: "Another compatible S3 Destination",
    description: "Any S3-compatible service — your endpoint is required.",
    fields: ["endpoint", "bucket", "credentials"],
  },
  "google-drive": {
    label: "Google Drive",
    description: "Connect your Google account; Drive is exposed through the built-in S3 wrapper.",
    fields: ["bucket", "oauth"],
  },
};

const schema = z.object({
  name: z.string().min(1, "Name is required"),
  type: z.enum(["aws", "cloudflare-r2", "minio", "custom-s3", "google-drive"]),
  endpoint: z.string().optional(),
  bucket: z.string().optional(),
  region: z.string().optional(),
  account_id: z.string().optional(),
  access_key: z.string().optional(),
  secret_key: z.string().optional(),
  google_client_id: z.string().optional(),
  google_client_secret: z.string().optional(),
});

function derivedEndpoint(type: DestinationType, values: Record<string, string>): string {
  if (type === "aws") return `https://s3.${values.region || "us-east-1"}.amazonaws.com`;
  if (type === "cloudflare-r2" && values.account_id) return `https://${values.account_id}.r2.cloudflarestorage.com`;
  return "";
}

export function Storage() {
  const { data: dests } = useS3Destinations();
  const createS3 = useCreateS3();
  const updateS3 = useUpdateS3();
  const deleteS3 = useDeleteS3();
  const testS3 = useTestS3();
  const googleConnect = useGoogleConnect();
  const googleDisconnect = useGoogleDisconnect();
  const { toast } = useToast();
  const [open, setOpen] = useState(false);
  const [editing, setEditing] = useState<S3Destination | null>(null);

  const {
    register,
    handleSubmit,
    reset,
    watch,
    setValue,
    formState: { errors, isSubmitting },
  } = useForm<z.input<typeof schema>, any, z.output<typeof schema>>({
    resolver: zodResolver(schema),
    defaultValues: { type: "custom-s3", region: "us-east-1" },
  });

  const type = watch("type");
  const values = watch();
  const def = PROVIDERS[type];

  useEffect(() => {
    if (!open) return;
    if (editing) {
      reset({
        name: editing.name,
        type: editing.type,
        endpoint: editing.type === "minio" || editing.type === "custom-s3" ? editing.endpoint : "",
        bucket: editing.bucket || "",
        region: editing.region || "us-east-1",
        account_id: editing.account_id || "",
        access_key: "",
        secret_key: "",
        google_client_id: editing.type === "google-drive" ? editing.google_client_id : "",
        google_client_secret: "",
      });
    } else {
      reset({ type: "custom-s3", region: "us-east-1" });
    }
  }, [open, editing, reset]);

  useEffect(() => {
    if (type === "google-drive") {
      setValue("endpoint", "");
      setValue("region", "");
      setValue("account_id", "");
      setValue("access_key", "");
      setValue("secret_key", "");
    }
  }, [type, setValue]);

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    if (params.get("oauth") === "google-drive") {
      const status = params.get("status") || "";
      if (status === "connected") toast("Google Drive connected");
      else if (status.startsWith("error")) toast(`Google Drive connection failed: ${status.replace("error:", "")}`, "error");
      window.history.replaceState({}, "", window.location.pathname);
    }
  }, [toast]);

  const submit = async (v: z.infer<typeof schema>) => {
    try {
      const body: Record<string, string> = { name: v.name, type: v.type };
      if (v.type !== "google-drive") {
        body.bucket = v.bucket || "";
        body.region = v.region || "us-east-1";
        body.account_id = v.account_id || "";
        if (v.type === "minio" || v.type === "custom-s3") {
          body.endpoint = v.endpoint || "";
        }
        if (v.type === "aws" || v.type === "cloudflare-r2") {
          body.endpoint = derivedEndpoint(v.type, { region: v.region || "", account_id: v.account_id || "" });
        }
        if (v.access_key || v.secret_key || !editing) {
          body.access_key = v.access_key || "";
          body.secret_key = v.secret_key || "";
        }
      } else {
        if (!editing || v.google_client_id) body.google_client_id = v.google_client_id || "";
        if (!editing || v.google_client_secret) body.google_client_secret = v.google_client_secret || "";
        body.bucket = v.bucket || "";
      }
      if (editing) {
        await updateS3.mutateAsync({ id: editing.id, body });
        toast("Destination updated");
        setOpen(false);
        setEditing(null);
      } else {
        const created = await createS3.mutateAsync(body);
        toast("Destination saved");
        setOpen(false);
        setEditing(null);
        if (v.type === "google-drive") {
          googleConnect.mutate(created.id);
        }
      }
    } catch (err) {
      toast(err instanceof Error ? err.message : "failed to save destination", "error");
    }
  };

  const showField = (f: string) => def.fields.includes(f);

  return (
    <AppPage>
      <AppPageHeader
        title="S3 Destinations"
        description="Blob storage destinations for backups and artifacts — S3-compatible services and Google Drive via OAuth."
        actions={
          <Button leftIcon="add" onClick={() => { setEditing(null); setOpen(true); }}>
            New destination
          </Button>
        }
      />

      <AppCard>
        <Table headers={["Name", "Type", "Endpoint", "Status", ""]}>
          {(dests ?? []).map((d) => (
            <tr key={d.id} className="hover:bg-surface-container-high transition-colors">
              <td className="px-sm py-2 font-code-md text-code-md text-on-surface">{d.name}</td>
              <td className="px-sm py-2 font-code-md text-code-md text-on-surface-variant">{PROVIDERS[d.type]?.label ?? d.type}</td>
              <td className="px-sm py-2 font-code-md text-code-md text-on-surface-variant truncate max-w-[260px]">
                {d.type === "google-drive" ? "—" : d.endpoint}
              </td>
              <td className="px-sm py-2">
                {d.type === "google-drive" ? (
                  <GoogleDriveStatus dest={d} onConnect={() => googleConnect.mutate(d.id)} onDisconnect={() => googleDisconnect.mutate(d.id)} />
                ) : (
                  <span className="px-2 py-0.5 rounded border border-outline-variant/50 font-label-caps text-label-caps text-on-surface-variant">
                    configured
                  </span>
                )}
              </td>
              <td className="px-sm py-2 text-right whitespace-nowrap">
                <button
                  onClick={() => testS3.mutate(d.id, {
                    onSuccess: () => toast(`${d.name}: connection OK`),
                    onError: (err) => toast(`${d.name}: ${err instanceof Error ? err.message : "connection failed"}`, "error"),
                  })}
                  disabled={testS3.isPending}
                  className="material-symbols-outlined text-[16px] text-on-surface-variant hover:text-primary transition-colors mr-1"
                  title="Test connection"
                >
                  wifi_tethering
                </button>
                <button
                  onClick={() => { setEditing(d); setOpen(true); }}
                  className="material-symbols-outlined text-[16px] text-on-surface-variant hover:text-primary transition-colors mr-1"
                  title="Edit"
                >
                  edit
                </button>
                <button
                  onClick={() => deleteS3.mutate(d.id)}
                  className="material-symbols-outlined text-[16px] text-on-surface-variant hover:text-error transition-colors"
                  title="Delete"
                >
                  delete
                </button>
              </td>
            </tr>
          ))}
        </Table>
        {(dests ?? []).length === 0 && (
          <p className="font-body-sm text-body-sm text-on-surface-variant p-sm">No destinations configured.</p>
        )}
      </AppCard>


      <Modal open={open} onClose={() => { setOpen(false); setEditing(null); }} title={editing ? "Edit Destination" : "New Destination"}>
        <form onSubmit={handleSubmit(submit)} className="space-y-lg" noValidate>
          <Field label="Name" hint={errors.name?.message}>
            <Input icon="label" placeholder="prod-backups" {...register("name")} />
          </Field>

          <Field label="Destination Type" hint={errors.type?.message}>
            <select {...register("type")} className="w-full bg-surface-container border border-outline-variant rounded-lg px-md py-sm font-body-md text-body-md text-on-surface">
              {(Object.keys(PROVIDERS) as DestinationType[]).map((t) => (
                <option key={t} value={t}>{PROVIDERS[t].label}</option>
              ))}
            </select>
          </Field>

          <div className="bg-surface-container-low border border-outline-variant rounded-lg p-md space-y-lg">
            <p className="font-body-sm text-body-sm text-on-surface-variant">{def.description}</p>

            {type === "google-drive" && (
              <div className="space-y-lg">
                <div className="bg-surface-container-low border border-outline-variant rounded-lg p-md">
                  <div className="font-label-caps text-label-caps text-on-surface-variant mb-xs">Redirect URI — register this exact URI in your Google OAuth client</div>
                  <code className="font-code-md text-code-md text-primary break-all">{window.location.origin}/api/v1/s3-destinations/google/callback</code>
                </div>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-lg">
                  <Field label="Google Client ID" hint={errors.google_client_id?.message}>
                    <Input icon="badge" placeholder="xxxx.apps.googleusercontent.com" {...register("google_client_id")} />
                  </Field>
                  <Field label="Client Secret" hint={errors.google_client_secret?.message}>
                    <Input icon="vpn_key" type="password" placeholder={editing ? "•••••••• (leave blank to keep)" : "GOCSPX-..."} {...register("google_client_secret")} />
                  </Field>
                </div>
                {editing && (
                  <div className="text-center">
                    <p className="font-body-md text-body-md text-on-surface mb-md">
                      Connect your Google account to use Google Drive as a destination.
                    </p>
                    <Button type="button" leftIcon="link" disabled={googleConnect.isPending} onClick={() => googleConnect.mutate(editing.id)}>
                      {editing.oauth_status === "reauth_required" ? "Reconnect Google Drive" : "Connect Google Drive"}
                    </Button>
                  </div>
                )}
              </div>
            )}

            {showField("bucket") && (
              <Field
                label={type === "google-drive" ? "Root folder name" : "Bucket"}
                hint={errors.bucket?.message || (type === "google-drive" ? "Folder created at the root of Drive; empty = Drive root" : undefined)}
              >
                <Input icon="storage" placeholder={type === "google-drive" ? "aether-backups" : "aether-backups"} {...register("bucket")} />
              </Field>
            )}

            {showField("region") && (
              <Field label="Region" hint={errors.region?.message}>
                <Input icon="public" placeholder="us-east-1" {...register("region")} />
              </Field>
            )}

            {showField("account") && (
              <Field label="Account ID" hint={errors.account_id?.message}>
                <Input icon="badge" placeholder="your R2 account id" {...register("account_id")} />
              </Field>
            )}

            {showField("endpoint") && (
              <>
                <Field label="Endpoint URL" hint={errors.endpoint?.message}>
                  <Input icon="dns" placeholder="https://storage.example.com" {...register("endpoint")} />
                </Field>
                <p className="font-body-sm text-body-sm text-on-surface-variant/70 -mt-sm">
                  Required for custom and self-hosted destinations — it is the source of truth for the connection.
                </p>
              </>
            )}

            {def.endpointDerived && (
              <div className="font-code-md text-code-md text-on-surface-variant/70 flex items-center gap-2">
                <span className="material-symbols-outlined text-[14px]">auto_awesome</span>
                Endpoint: {def.endpointHint?.(values as Record<string, string>)}
              </div>
            )}

            {showField("credentials") && (
              <div className="grid grid-cols-1 md:grid-cols-2 gap-lg">
                <Field label={editing ? "Access Key (leave blank to keep)" : "Access Key"} hint={errors.access_key?.message}>
                  <Input icon="key" {...register("access_key")} />
                </Field>
                <Field label={editing ? "Secret Key (leave blank to keep)" : "Secret Key"} hint={errors.secret_key?.message}>
                  <Input icon="vpn_key" type="password" {...register("secret_key")} />
                </Field>
              </div>
            )}
          </div>

          <div className="flex justify-end gap-md border-t border-outline-variant pt-lg">
            <Button type="button" variant="ghost" onClick={() => { setOpen(false); setEditing(null); }}>
              Cancel
            </Button>
            <Button type="submit" disabled={isSubmitting}>
              {editing ? "Save changes" : "Create destination"}
            </Button>
          </div>
        </form>
      </Modal>
    </AppPage>
  );
}

function GoogleDriveStatus({ dest, onConnect, onDisconnect }: { dest: S3Destination; onConnect: () => void; onDisconnect: () => void }) {
  const connected = dest.oauth_status === "connected";
  const reauth = dest.oauth_status === "reauth_required";
  return (
    <div className="flex items-center gap-2">
      <span
        className={cn(
          "px-2 py-0.5 rounded border font-label-caps text-label-caps flex items-center gap-1",
          connected ? "bg-[#4ade80]/10 text-[#4ade80] border-[#4ade80]/20" : "bg-outline/10 text-on-surface-variant border-outline-variant/30",
        )}
      >
        <span className="material-symbols-outlined text-[12px]">{connected ? "check_circle" : "link"}</span>
        {connected ? "Connected" : reauth ? "Reauth required" : "Not connected"}
      </span>
      {connected && dest.oauth_email && <span className="font-code-md text-code-md text-on-surface-variant/70">{dest.oauth_email}</span>}
      {connected || reauth ? (
        <button onClick={onDisconnect} className="font-label-caps text-label-caps text-on-surface-variant hover:text-error transition-colors">
          Disconnect
        </button>
      ) : (
        <button onClick={onConnect} className="font-label-caps text-label-caps text-primary hover:text-primary-fixed transition-colors">
          {reauth ? "Reconnect" : "Connect"}
        </button>
      )}
    </div>
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
            <span className="font-code-md text-code-md text-on-surface-variant/60">{sc.cron}</span>
            <button onClick={() => remove.mutate(sc.id)} className="material-symbols-outlined text-[16px] text-on-surface-variant hover:text-error transition-colors">
              delete
            </button>
          </div>
        ))}
        {(schedules ?? []).length === 0 && (
          <p className="font-body-sm text-body-sm text-on-surface-variant/60">No schedules yet.</p>
        )}
      </div>
    </Card>
  );
}

export const Route = createFileRoute("/_shell/storage/")({
  component: Storage,
});
