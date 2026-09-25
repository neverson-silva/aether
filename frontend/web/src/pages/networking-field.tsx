import {
  Badge,
  Button,
  Card,
  Checkbox,
  EmptyState,
  Field,
  Input,
  Marker,
  Modal,
  RuntimeStatus,
  Skeleton,
  Typography,
} from '@aether/elisyum-ds'
import {
  GlobeHemisphereWest,
  MagicWand,
  PencilSimple,
  Plus,
  TrashSimple,
} from '@phosphor-icons/react'
import { useEffect, useState } from 'react'
import type { Domain } from '../api/types'
import { useAddDomain } from '../hooks/use-add-domain'
import { useDomains } from '../hooks/use-domains'
import { useGenerateFreeDomain } from '../hooks/use-generate-free-domain'
import { useHostInfo } from '../hooks/use-host-info'
import { useRemoveDomain } from '../hooks/use-remove-domain'
import { useSaveServerDomains } from '../hooks/use-save-server-domains'
import { useServerDomains } from '../hooks/use-server-domains'
import { useServices } from '../hooks/use-service-details'
import { useUpdateDomain } from '../hooks/use-update-domain'

export function NetworkingField() {
  const services = useServices()
  const hostInfo = useHostInfo()
  const serverDomains = useServerDomains()
  const saveServerDomains = useSaveServerDomains()
  const [serviceId, setServiceId] = useState('')
  const selectedId = serviceId || services.data?.[0]?.id || ''
  const selected = services.data?.find((service) => service.id === selectedId)
  const domains = useDomains('services', selectedId)
  const addDomain = useAddDomain('services', selectedId)
  const updateDomain = useUpdateDomain('services', selectedId)
  const removeDomain = useRemoveDomain('services', selectedId)
  const generateFreeDomain = useGenerateFreeDomain('services', selectedId)
  const [editorOpen, setEditorOpen] = useState(false)
  const [editingDomain, setEditingDomain] = useState<Domain | null>(null)
  const [deletingDomain, setDeletingDomain] = useState<Domain | null>(null)
  const [host, setHost] = useState('')
  const [port, setPort] = useState('80')
  const [https, setHttps] = useState(true)
  const [frontendDomain, setFrontendDomain] = useState('')
  const [apiDomain, setApiDomain] = useState('')
  const [serverHTTPS, setServerHTTPS] = useState(true)

  useEffect(() => {
    if (!serverDomains.data) return
    setFrontendDomain(serverDomains.data.web_domain)
    setApiDomain(serverDomains.data.api_domain)
    setServerHTTPS(serverDomains.data.https)
  }, [serverDomains.data])

  if (services.isLoading)
    return (
      <div className="grid gap-4">
        <Skeleton className="h-48 rounded-2xl" />
        <Skeleton className="h-72 rounded-2xl" />
      </div>
    )
  if (services.isError)
    return (
      <EmptyState
        title="Networking context unavailable"
        description="The control plane did not return the service scope required to inspect domains."
        action={
          <Button
            tone="neutral"
            onClick={() => services.refetch()}
          >
            Retry request
          </Button>
        }
      />
    )

  const openCreate = () => {
    setEditingDomain(null)
    setHost('')
    setPort('80')
    setHttps(true)
    setEditorOpen(true)
  }

  const openEdit = (domain: Domain) => {
    setEditingDomain(domain)
    setHost(domain.host)
    setPort(String(domain.container_port))
    setHttps(domain.https)
    setEditorOpen(true)
  }

  const submitDomain = () => {
    if (!(selectedId && host.trim())) return
    const body = { host: host.trim(), https, container_port: Number(port) || 80 }
    const options = { onSuccess: () => setEditorOpen(false) }
    if (editingDomain)
      updateDomain.mutate({ domainID: editingDomain.id, body }, options)
    else addDomain.mutate(body, options)
  }

  const removeSelectedDomain = () => {
    if (!deletingDomain) return
    removeDomain.mutate(deletingDomain.host, {
      onSuccess: () => setDeletingDomain(null),
    })
  }

  const saveServerDomainSettings = () => {
    saveServerDomains.mutate({
      web_domain: frontendDomain.trim(),
      api_domain: apiDomain.trim(),
      https: serverHTTPS,
    })
  }

  return (
    <div className="grid gap-6">
      <header className="flex flex-wrap items-end justify-between gap-5">
        <div className="grid gap-3">
          <div className="flex items-center gap-3 text-label tracking-[0.16em] text-text-subtle">
            <GlobeHemisphereWest size={17} />
            NETWORK / PUBLIC SURFACES
          </div>
          <Typography
            as="h1"
            role="page-title"
          >
            Networking field
          </Typography>
          <Typography role="supporting">
            Public domains, TLS posture and routing context remain attached to the
            service they expose.
          </Typography>
        </div>
        {selectedId ? (
          <div className="flex flex-wrap items-center gap-2">
            <Button
              tone="ghost"
              disabled={generateFreeDomain.isPending}
              onClick={() => generateFreeDomain.mutate()}
            >
              <MagicWand size={17} />
              {generateFreeDomain.isPending ? 'Generating…' : 'Generate free domain'}
            </Button>
            <Button onClick={openCreate}>
              <Plus size={17} />
              Attach domain
            </Button>
          </div>
        ) : null}
      </header>

      <Card className="grid gap-5 rounded-2xl border-border-subtle bg-surface-1 p-5">
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div className="grid gap-1">
            <span className="text-label tracking-[0.12em] text-text-subtle">
              AETHER CONTROL PLANE
            </span>
            <Typography
              as="h2"
              role="section-title"
            >
              Frontend access
            </Typography>
            <p className="text-supporting text-text-secondary">
              Route the production frontend through Traefik on its internal port 4000.
              The API hostname is optional.
            </p>
          </div>
          <GlobeHemisphereWest
            className="text-action"
            size={22}
          />
        </div>
        {serverDomains.isError ? (
          <div className="rounded-xl border border-warning/35 bg-warning-soft px-4 py-3 text-supporting text-warning-strong">
            Only a platform administrator can manage the Aether frontend and API
            domains.
          </div>
        ) : (
          <>
            <div className="grid gap-4 md:grid-cols-2">
              <Field
                id="frontend-domain"
                label="Frontend domain"
                description="Production web traffic → aether-web:4000"
              >
                <Input
                  disabled={serverDomains.isLoading || saveServerDomains.isPending}
                  onChange={(event) => setFrontendDomain(event.target.value)}
                  placeholder="aether.example.com"
                  value={frontendDomain}
                />
              </Field>
              <Field
                id="api-domain"
                label="API domain (optional)"
                description="Control-plane API traffic → aether-api"
              >
                <Input
                  disabled={serverDomains.isLoading || saveServerDomains.isPending}
                  onChange={(event) => setApiDomain(event.target.value)}
                  placeholder="api.aether.example.com"
                  value={apiDomain}
                />
              </Field>
            </div>
            <div className="flex flex-wrap items-center justify-between gap-4 border-t border-border-subtle pt-4">
              <label className="flex cursor-pointer items-center gap-3 text-supporting text-text-secondary">
                <Checkbox
                  checked={serverHTTPS}
                  disabled={saveServerDomains.isPending}
                  onChange={(event) => setServerHTTPS(event.target.checked)}
                />
                <span>
                  <span className="block text-label text-text-primary">
                    HTTPS with Let’s Encrypt
                  </span>
                  <span className="block text-log text-text-tertiary">
                    Provision certificates for configured hostnames.
                  </span>
                </span>
              </label>
              <Button
                disabled={!frontendDomain.trim() || saveServerDomains.isPending}
                onClick={saveServerDomainSettings}
              >
                {saveServerDomains.isPending
                  ? 'Saving…'
                  : saveServerDomains.isSuccess
                    ? 'Saved'
                    : 'Save control-plane domains'}
              </Button>
            </div>
          </>
        )}
      </Card>

      <div className="grid items-stretch gap-4 xl:grid-cols-[minmax(18rem,0.38fr)_minmax(0,1fr)]">
        <Card className="grid content-start gap-3 rounded-2xl border-border-subtle bg-surface-1 p-3">
          <div className="px-2 pb-2">
            <span className="text-label tracking-[0.12em] text-text-subtle">
              SERVICE SCOPE
            </span>
          </div>
          {services.data?.length ? (
            services.data.map((service) => (
              <button
                aria-pressed={service.id === selectedId}
                className={`relative grid gap-1 rounded-xl border p-4 text-left transition-colors ${service.id === selectedId ? 'border-action/65 bg-surface-2 before:absolute before:inset-y-3 before:left-0 before:w-0.5 before:rounded-full before:bg-action' : 'border-transparent bg-surface-2 hover:border-border-default hover:bg-surface-3'}`}
                key={service.id}
                onClick={() => setServiceId(service.id)}
                type="button"
              >
                <span className="text-supporting text-text-primary">
                  {service.name}
                </span>
                <span className="font-technical text-log text-text-subtle">
                  {service.kind} · {service.id}
                </span>
              </button>
            ))
          ) : (
            <EmptyState
              title="No services in scope"
              description="Create a service before attaching a public domain."
            />
          )}
        </Card>
        <Card className="grid content-start gap-5 rounded-2xl border-border-subtle bg-surface-1 p-5">
          <div className="flex items-center justify-between gap-4">
            <div className="grid gap-1">
              <span className="text-label tracking-[0.12em] text-text-subtle">
                ROUTING INSPECTOR
              </span>
              <Typography
                as="h2"
                role="section-title"
              >
                {selected?.name ?? 'Awaiting service'}
              </Typography>
            </div>
            <span className="font-technical text-log text-text-subtle">
              {domains.data?.length ?? 0} domains
            </span>
          </div>
          {domains.isLoading ? (
            <Skeleton className="h-48 rounded-xl" />
          ) : domains.data?.length ? (
            <div className="grid gap-2">
              {domains.data.map((domain) => (
                <DomainRow
                  domain={domain}
                  key={domain.id}
                  onEdit={() => openEdit(domain)}
                  onDelete={() => setDeletingDomain(domain)}
                />
              ))}
            </div>
          ) : (
            <EmptyState
              title="No public routes"
              description="This service has no domain attached. The private runtime remains unaffected."
              icon={<GlobeHemisphereWest size={28} />}
              action={
                <Button
                  tone="neutral"
                  onClick={openCreate}
                >
                  <Plus size={16} />
                  Attach first domain
                </Button>
              }
            />
          )}
          {hostInfo.data?.free_domain_base ? (
            <div className="grid gap-2 rounded-xl border border-border-subtle bg-surface-2 p-4">
              <span className="text-label tracking-[0.12em] text-text-subtle">
                FREE DNS SURFACE
              </span>
              <span className="font-technical text-log text-text-primary">
                *.{hostInfo.data.free_domain_base}
              </span>
              <span className="text-supporting text-text-secondary">
                Automatic public routing is available for this host. Generated domains
                inherit the selected service port and TLS posture.
              </span>
            </div>
          ) : null}
        </Card>
      </div>

      <Modal
        trigger={<span className="hidden" />}
        open={editorOpen}
        onOpenChange={setEditorOpen}
        title={editingDomain ? 'Edit public domain' : 'Attach public domain'}
        description="Declare the public route and transport posture for the selected service."
        footer={
          <Button
            disabled={addDomain.isPending || updateDomain.isPending}
            onClick={submitDomain}
          >
            {addDomain.isPending || updateDomain.isPending
              ? 'Saving…'
              : editingDomain
                ? 'Save changes'
                : 'Attach domain'}
          </Button>
        }
      >
        <div className="grid gap-4">
          <Field
            id="domain-host"
            label="Hostname"
            required
          >
            <Input
              onChange={(event) => setHost(event.target.value)}
              placeholder="api.company.com"
              value={host}
            />
          </Field>
          <Field
            id="domain-port"
            label="Container port"
            required
          >
            <Input
              inputMode="numeric"
              onChange={(event) => setPort(event.target.value)}
              type="number"
              value={port}
            />
          </Field>
          <label className="flex items-center gap-3 text-supporting text-text-secondary">
            <Checkbox
              checked={https}
              onChange={(event) => setHttps(event.target.checked)}
            />
            Provision HTTPS for this route
          </label>
          {addDomain.isError || updateDomain.isError ? (
            <p className="text-supporting text-danger">
              The domain could not be saved. Verify the hostname and port.
            </p>
          ) : null}
        </div>
      </Modal>
      <Modal
        trigger={<span className="hidden" />}
        open={Boolean(deletingDomain)}
        onOpenChange={(open) => !open && setDeletingDomain(null)}
        title="Remove public domain"
        description="This removes the route from the selected service."
        footer={
          <Button
            tone="danger"
            disabled={removeDomain.isPending}
            onClick={removeSelectedDomain}
          >
            {removeDomain.isPending ? 'Removing…' : 'Remove domain'}
          </Button>
        }
      >
        <p className="text-body text-text-secondary">
          Remove{' '}
          <span className="font-technical text-text-primary">
            {deletingDomain?.host}
          </span>{' '}
          from this service?
        </p>
      </Modal>
    </div>
  )
}

function DomainRow({
  domain,
  onEdit,
  onDelete,
}: {
  domain: Domain
  onEdit: () => void
  onDelete: () => void
}) {
  const normalizedStatus = domain.status.toLowerCase()
  const tone =
    normalizedStatus === 'active' || normalizedStatus === 'ready'
      ? 'success'
      : normalizedStatus === 'error' || normalizedStatus === 'failed'
        ? 'danger'
        : 'warning'
  const runtimeStatus =
    tone === 'success' ? 'healthy' : tone === 'danger' ? 'failed' : 'deploying'
  return (
    <div className="grid gap-4 rounded-xl bg-surface-2 p-4 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-center">
      <button
        className="grid min-w-0 gap-2 text-left"
        onClick={onEdit}
        type="button"
      >
        <span className="truncate text-supporting text-text-primary">
          {domain.host}
        </span>
        <span className="font-technical text-log text-text-subtle">
          PORT {domain.container_port} · CERTIFICATE{' '}
          {domain.cert_status || 'not requested'}
        </span>
        <span>
          <Badge tone={tone}>{domain.status}</Badge>
        </span>
      </button>
      <div className="flex items-center justify-end gap-4">
        <div className="flex items-center gap-3">
          <RuntimeStatus
            status={runtimeStatus}
            label={domain.status}
          />
          <span className="flex items-center gap-1 font-technical text-log text-text-secondary">
            <Marker tone={domain.https ? 'success' : 'warning'} />
            {domain.https ? 'HTTPS' : 'HTTP'}
          </span>
        </div>
        <Button
          aria-label={`Remove ${domain.host}`}
          onClick={onDelete}
          tone="ghost"
        >
          <TrashSimple size={17} />
        </Button>
        <Button
          aria-label={`Edit ${domain.host}`}
          onClick={onEdit}
          tone="ghost"
        >
          <PencilSimple size={17} />
        </Button>
      </div>
    </div>
  )
}
