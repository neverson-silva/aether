import {
  Button,
  EmptyState,
  Field,
  Input,
  Select,
  Typography,
  Wizard,
} from '@aether/elisyum-ds'
import {
  ArrowLeft,
  GitBranch,
  Package,
  RocketLaunch,
  SelectionAll,
} from '@phosphor-icons/react'
import { useNavigate } from '@tanstack/react-router'
import { useState } from 'react'
import { useCreateApp } from '../hooks/use-create-app'
import { useProjects } from '../hooks/use-projects'

type SourceMode = 'image' | 'git'

export function ServiceComposer() {
  const navigate = useNavigate()
  const projects = useProjects()
  const createApp = useCreateApp()
  const [step, setStep] = useState(0)
  const [projectId, setProjectId] = useState('')
  const [name, setName] = useState('')
  const [source, setSource] = useState<SourceMode>('image')
  const [image, setImage] = useState('')
  const [gitUrl, setGitUrl] = useState('')
  const [gitBranch, setGitBranch] = useState('main')
  const validScope = Boolean(projectId && name.trim().length >= 2)
  const validSource =
    source === 'image' ? image.trim().length > 0 : gitUrl.trim().length > 0
  const complete = async () => {
    const created = await createApp.mutateAsync({
      projectID: projectId,
      payload: {
        name: name.trim(),
        source_type: source,
        image: image.trim(),
        git_url: gitUrl.trim(),
        git_branch: gitBranch.trim() || 'main',
        dockerfile: 'Dockerfile',
        build_type: 'dockerfile',
        port: 80,
        resources: { cpus: '', mem_mb: 0 },
        health_check: {
          enabled: false,
          path: '/',
          interval_ms: 5000,
          timeout_ms: 2000,
          retries: 3,
        },
      },
    })
    await navigate({ to: `/services/${created.service_id ?? created.id}` })
  }
  if (projects.isLoading)
    return (
      <div className="grid gap-4">
        <div className="h-16 animate-pulse rounded-xl bg-surface-1" />
        <div className="h-96 animate-pulse rounded-xl bg-surface-1" />
      </div>
    )
  if (projects.isError)
    return (
      <EmptyState
        title="Project context unavailable"
        description="A service must belong to a project before it can be created."
        action={
          <Button
            tone="neutral"
            onClick={() => projects.refetch()}
          >
            Retry request
          </Button>
        }
      />
    )
  return (
    <div className="grid gap-6">
      <header className="grid gap-4">
        <button
          className="flex w-fit items-center gap-2 text-label text-text-tertiary transition-colors hover:text-text-primary"
          onClick={() => navigate({ to: '/services' })}
          type="button"
        >
          <ArrowLeft size={16} />
          Service field
        </button>
        <div className="flex items-center gap-3 text-label tracking-[0.16em] text-text-subtle">
          <RocketLaunch size={17} />
          SERVICE PROVISIONING
        </div>
        <Typography
          as="h1"
          role="page-title"
        >
          Bring a service into scope.
        </Typography>
        <Typography
          className="max-w-2xl"
          role="supporting"
        >
          Choose the ownership boundary and source artifact before the runtime is
          created.
        </Typography>
      </header>
      <Wizard
        activeStep={step}
        onStepChange={setStep}
        onComplete={() => {
          void complete()
        }}
        completeLabel={createApp.isPending ? 'Creating…' : 'Create service'}
        steps={[
          {
            id: 'scope',
            label: 'Scope',
            description: 'Attach the service to an operational project.',
            summary:
              projects.data?.find((item) => item.id === projectId)?.name ||
              'Awaiting project',
            canContinue: validScope,
            content: (
              <div className="grid max-w-2xl gap-6">
                <Field
                  id="service-project"
                  label="Project"
                  description="The project owns access, environments and delivery context."
                  required
                >
                  <Select
                    onChange={(event) => setProjectId(event.target.value)}
                    value={projectId}
                  >
                    <option value="">Choose a project</option>
                    {(projects.data ?? []).map((project) => (
                      <option
                        key={project.id}
                        value={project.id}
                      >
                        {project.name}
                      </option>
                    ))}
                  </Select>
                </Field>
                <Field
                  id="service-name"
                  label="Service name"
                  description="Use the stable runtime name operators will use in alerts and delivery history."
                  required
                >
                  <Input
                    onChange={(event) => setName(event.target.value)}
                    placeholder="checkout-api"
                    value={name}
                  />
                </Field>
                <div className="grid grid-cols-[auto_minmax(0,1fr)] gap-4 rounded-xl bg-surface-2 p-4">
                  <SelectionAll
                    className="mt-1 text-action-strong"
                    size={20}
                  />
                  <span className="text-supporting text-text-tertiary">
                    Ownership is explicit before infrastructure is provisioned.
                  </span>
                </div>
              </div>
            ),
          },
          {
            id: 'source',
            label: 'Source',
            description: 'Declare the artifact that will become the service runtime.',
            summary:
              source === 'image'
                ? image || 'Awaiting image'
                : gitUrl || 'Awaiting repository',
            canContinue: validSource,
            content: (
              <div className="grid max-w-2xl gap-6">
                <div className="grid gap-2 sm:grid-cols-2">
                  <button
                    className={`flex items-center gap-3 rounded-xl border p-4 text-left transition-colors ${source === 'image' ? 'border-action bg-action-soft' : 'border-border-subtle bg-surface-1 hover:bg-surface-2'}`}
                    onClick={() => setSource('image')}
                    type="button"
                  >
                    <Package size={20} />
                    <span className="grid gap-1">
                      <span className="text-label">Container image</span>
                      <span className="text-supporting text-text-tertiary">
                        Deploy an existing OCI artifact.
                      </span>
                    </span>
                  </button>
                  <button
                    className={`flex items-center gap-3 rounded-xl border p-4 text-left transition-colors ${source === 'git' ? 'border-action bg-action-soft' : 'border-border-subtle bg-surface-1 hover:bg-surface-2'}`}
                    onClick={() => setSource('git')}
                    type="button"
                  >
                    <GitBranch size={20} />
                    <span className="grid gap-1">
                      <span className="text-label">Git repository</span>
                      <span className="text-supporting text-text-tertiary">
                        Build from a controlled source.
                      </span>
                    </span>
                  </button>
                </div>
                {source === 'image' ? (
                  <Field
                    id="service-image"
                    label="Image reference"
                    description="Include the tag or digest to make the deployment reproducible."
                    required
                  >
                    <Input
                      onChange={(event) => setImage(event.target.value)}
                      placeholder="registry.example.com/checkout:stable"
                      value={image}
                    />
                  </Field>
                ) : (
                  <div className="grid gap-5 sm:grid-cols-2">
                    <Field
                      id="service-git"
                      label="Repository URL"
                      required
                    >
                      <Input
                        onChange={(event) => setGitUrl(event.target.value)}
                        placeholder="https://github.com/acme/checkout"
                        value={gitUrl}
                      />
                    </Field>
                    <Field
                      id="service-branch"
                      label="Branch"
                    >
                      <Input
                        onChange={(event) => setGitBranch(event.target.value)}
                        value={gitBranch}
                      />
                    </Field>
                  </div>
                )}
              </div>
            ),
          },
          {
            id: 'review',
            label: 'Review',
            description: 'Confirm the service contract before creation.',
            summary: validSource ? `${name} · ready` : 'Source required',
            canContinue: validScope && validSource,
            content: (
              <div className="grid max-w-2xl gap-6">
                <div className="grid gap-4 rounded-xl border border-border-subtle bg-surface-2 p-5">
                  <div className="grid gap-1">
                    <span className="text-label">SERVICE CONTRACT</span>
                    <span className="font-technical text-log text-text-subtle">
                      The following identity will enter the control plane.
                    </span>
                  </div>
                  <div className="grid gap-2 sm:grid-cols-2">
                    <div className="rounded-lg bg-surface-1 p-3">
                      <Fact
                        label="SERVICE"
                        value={name || '—'}
                      />
                    </div>
                    <div className="rounded-lg bg-surface-1 p-3">
                      <Fact
                        label="SOURCE"
                        value={source === 'image' ? image || '—' : gitUrl || '—'}
                      />
                    </div>
                    <div className="rounded-lg bg-surface-1 p-3">
                      <Fact
                        label="BRANCH"
                        value={source === 'git' ? gitBranch || 'main' : 'N/A'}
                      />
                    </div>
                    <div className="rounded-lg bg-surface-1 p-3">
                      <Fact
                        label="PORT"
                        value="80"
                      />
                    </div>
                  </div>
                </div>
                <Typography role="supporting">
                  Creation establishes the service record. Deployment remains an
                  explicit operator action.
                </Typography>
              </div>
            ),
          },
        ]}
      />
    </div>
  )
}

function Fact({ label, value }: { label: string; value: string }) {
  return (
    <div className="grid gap-1">
      <span className="text-label tracking-[0.1em] text-text-subtle">{label}</span>
      <span className="break-words font-technical text-code text-text-secondary">
        {value}
      </span>
    </div>
  )
}
