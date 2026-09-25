import {
  Button,
  EmptyState,
  Field,
  Input,
  Textarea,
  Typography,
  Wizard,
} from '@aether/elisyum-ds'
import {
  ArrowLeft,
  CheckCircle,
  FolderSimple,
  IdentificationCard,
  TextAa,
} from '@phosphor-icons/react'
import { useNavigate } from '@tanstack/react-router'
import { useState } from 'react'
import { useCreateProject } from '../hooks/use-create-project'

export function ProjectComposer() {
  const navigate = useNavigate()
  const createProject = useCreateProject()
  const [step, setStep] = useState(0)
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const valid = name.trim().length >= 2
  const complete = async () => {
    const project = await createProject.mutateAsync(name.trim())
    await navigate({ to: `/projects/${project.id}` })
  }
  if (createProject.isError)
    return (
      <EmptyState
        title="Project creation failed"
        description="The control plane could not establish this project boundary."
        action={
          <Button
            tone="neutral"
            onClick={() => createProject.reset()}
          >
            Return to workflow
          </Button>
        }
      />
    )
  return (
    <div className="grid gap-6">
      <header className="grid gap-4">
        <button
          className="flex w-fit items-center gap-2 text-label text-text-tertiary transition-colors hover:text-text-primary"
          onClick={() => navigate({ to: '/projects' })}
          type="button"
        >
          <ArrowLeft size={16} />
          Project field
        </button>
        <div className="flex items-center gap-3 text-label tracking-[0.16em] text-text-subtle">
          <FolderSimple size={17} />
          PROJECT PROVISIONING
        </div>
        <Typography
          as="h1"
          role="page-title"
        >
          Establish a project boundary.
        </Typography>
        <Typography
          className="max-w-2xl"
          role="supporting"
        >
          Define the operational scope first. Services, environments and access context
          attach to this boundary after creation.
        </Typography>
      </header>
      <Wizard
        activeStep={step}
        onStepChange={setStep}
        onComplete={() => {
          void complete()
        }}
        completeLabel={createProject.isPending ? 'Establishing…' : 'Establish project'}
        steps={[
          {
            id: 'identity',
            label: 'Identity',
            description: 'Name the boundary that operators will recognize.',
            summary: name || 'Awaiting project name',
            canContinue: valid,
            content: (
              <div className="grid max-w-2xl gap-6">
                <Field
                  id="project-name"
                  label="Project name"
                  description="Use the name operators will see in inventory, delivery and audit surfaces."
                  required
                >
                  <Input
                    autoFocus
                    onChange={(event) => setName(event.target.value)}
                    placeholder="Payments platform"
                    value={name}
                  />
                </Field>
                <div className="grid grid-cols-[auto_minmax(0,1fr)] gap-4 rounded-xl bg-surface-2 p-4">
                  <IdentificationCard
                    className="mt-1 text-action-strong"
                    size={20}
                  />
                  <div className="grid gap-1">
                    <span className="text-label text-text-primary">
                      Identity is durable
                    </span>
                    <span className="text-supporting text-text-tertiary">
                      The project identifier remains stable even as services and
                      environments change.
                    </span>
                  </div>
                </div>
              </div>
            ),
          },
          {
            id: 'context',
            label: 'Context',
            description: 'Leave an operational note for the team that owns this scope.',
            summary: description || 'No description',
            content: (
              <div className="grid max-w-2xl gap-6">
                <Field
                  id="project-description"
                  label="Operational description"
                  description="Optional context shown to operators inspecting this project."
                >
                  <Textarea
                    onChange={(event) => setDescription(event.target.value)}
                    placeholder="Customer-facing payment workloads and their recovery boundary."
                    value={description}
                  />
                </Field>
                <div className="grid grid-cols-[auto_minmax(0,1fr)] gap-4 rounded-xl bg-surface-2 p-4">
                  <TextAa
                    className="mt-1 text-action-strong"
                    size={20}
                  />
                  <div className="grid gap-1">
                    <span className="text-label text-text-primary">
                      Context stays close to the resource
                    </span>
                    <span className="text-supporting text-text-tertiary">
                      Descriptions are visible in the project inspector without
                      replacing technical metadata.
                    </span>
                  </div>
                </div>
              </div>
            ),
          },
          {
            id: 'review',
            label: 'Review',
            description:
              'Confirm the boundary before it becomes available to the organization.',
            summary: valid ? `${name} · ready` : 'Needs project identity',
            canContinue: valid,
            content: (
              <div className="grid max-w-2xl gap-6">
                <div className="grid gap-4 rounded-xl border border-border-subtle bg-surface-2 p-5">
                  <div className="flex items-center gap-3">
                    <CheckCircle
                      className="text-success"
                      size={20}
                    />
                    <span className="text-label text-text-primary">
                      Project summary
                    </span>
                  </div>
                  <div className="grid gap-2 sm:grid-cols-2">
                    <div className="rounded-lg bg-surface-1 p-3">
                      <Fact
                        label="NAME"
                        value={name || '—'}
                      />
                    </div>
                    <div className="rounded-lg bg-surface-1 p-3">
                      <Fact
                        label="DESCRIPTION"
                        value={description || 'No description'}
                      />
                    </div>
                  </div>
                </div>
                <Typography role="supporting">
                  Creating the project does not deploy or mutate any service. It only
                  establishes the scope for the next operation.
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
