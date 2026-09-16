import { CodeBlock, Card, Input, Slider } from "@aether/design-system";
import type { App, ServiceSummary } from "@/api/types";
import { Autopilot } from "./Autopilot";

export function ServiceSettingsTab({
  app,
  service,
  runtimeId,
  onUpdate,
  onOpenWebhook,
}: {
  app: App;
  service: ServiceSummary;
  runtimeId: string;
  onUpdate: (update: Record<string, unknown>) => void;
  onOpenWebhook: () => void;
}) {
  return (
    <div className="grid grid-cols-1 gap-lg xl:grid-cols-3">
      <Card>
        <div className="mb-md flex items-center justify-between"><h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase">Resources</h2></div>
        <p className="mb-sm font-label-caps text-label-caps text-on-surface-variant/60 uppercase">CPU</p>
        <Slider min={0.25} max={8} step={0.25} value={Number.parseFloat(String(app.resources?.cpus ?? "0.5")) || 0.5} onValueChange={(value) => {
          const next = Array.isArray(value) ? value[0] : value;
          if (typeof next === "number") onUpdate({ resources: { cpus: next.toString() } });
        }} />
        <p className="mb-md font-code-md text-code-md text-on-surface-variant">{app.resources?.cpus ?? "0.5"} CPU allocated</p>
        <p className="mb-sm font-label-caps text-label-caps text-on-surface-variant/60 uppercase">Memory</p>
        <Slider min={256} max={8192} step={256} value={Math.min(8192, Math.max(256, app.resources?.mem_mb || 256))} onValueChange={(value) => {
          const next = Array.isArray(value) ? value[0] : value;
          if (typeof next === "number") onUpdate({ resources: { mem_mb: next } });
        }} />
        <p className="mb-md font-code-md text-code-md text-on-surface-variant">{app.resources?.mem_mb && app.resources.mem_mb % 1024 === 0 ? `${app.resources.mem_mb / 1024} GB RAM` : `${app.resources?.mem_mb ?? 256} MB RAM`}</p>
        <p className="mb-sm font-label-caps text-label-caps text-on-surface-variant/60 uppercase">Storage</p>
        <Slider min={0} max={102400} step={1024} value={Math.min(102400, Math.max(0, app.storage_mb ?? 0))} onValueChange={(value) => {
          const next = Array.isArray(value) ? value[0] : value;
          if (typeof next === "number") onUpdate({ resources: { storage_mb: next } });
        }} />
        <p className="mb-md font-code-md text-code-md text-on-surface-variant">{app.storage_mb ? app.storage_mb % 1024 === 0 ? `${app.storage_mb / 1024} GB storage` : `${app.storage_mb} MB storage` : "Unlimited storage"}</p>
        <p className="font-code-md text-code-md text-on-surface-variant/60">Applies on the next deploy. CPU accepts decimals (0.5) or millicores (500m).</p>
      </Card>
      <Card>
        <div className="mb-md flex items-center justify-between"><h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase">Image retention</h2></div>
        <p className="mb-md font-body-sm text-body-sm text-on-surface-variant">Keep the N most recent built images for this app (git builds). Older images are deleted from the internal registry and local storage. 0 = use global policy (default 5).</p>
        <Input type="number" min={0} placeholder="5" defaultValue={app.image_retention || 0} onChange={(event) => {
          const value = Number.parseInt(event.target.value, 10);
          onUpdate({ image_retention: Number.isFinite(value) ? value : 0 });
        }} />
        <p className="mt-xs font-code-md text-code-md text-on-surface-variant/60">Global policy: AETHER_IMAGE_RETENTION (default 5, 0 = disabled)</p>
      </Card>
      {service.capabilities.can_build ? <div className="space-y-lg">
        <Autopilot appID={service.kind === "app" ? runtimeId : app.id} />
        <Card>
          <div className="mb-md flex items-center justify-between"><h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase">Webhook GitHub</h2><button type="button" onClick={onOpenWebhook} className="font-body-sm text-body-sm text-primary transition-colors hover:text-primary-fixed-dim">Configure</button></div>
          <p className="mb-sm font-body-sm text-body-sm text-on-surface-variant">{app.source_type === "git" ? "POST /api/v1/webhooks/github/{serviceID} with the X-Hub-Signature-256 header." : "Available for applications with a git source."}</p>
          {app.source_type === "git" ? <CodeBlock code={`POST /api/v1/webhooks/github/${app.id}\nX-Hub-Signature-256: sha256=<hmac>`} /> : null}
        </Card>
      </div> : null}
    </div>
  );
}
