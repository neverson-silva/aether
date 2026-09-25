import {
  AlertDialog,
  BulkActionBar,
  Button,
  Checkbox,
  DropdownMenu,
  EmptyState,
  Field,
  InlineError,
  Input,
  Modal,
  RuntimeStatus,
  Skeleton,
  Textarea,
  Typography,
  VariableEditor,
  WorkspaceSwitcher,
} from '@aether/elisyum-ds'
import {
  ArrowLeft,
  ArrowRight,
  BracketsCurly,
  CaretDown,
  CaretUp,
  DotsThree,
  Globe,
  Key,
  PencilSimple,
  Play,
  Plus,
  RocketLaunch,
  Stack,
  Stop,
  TrashSimple,
} from '@phosphor-icons/react'
import { useNavigate, useParams } from '@tanstack/react-router'
import { useEffect, useState } from 'react'
import { useForm } from 'react-hook-form'
import type { ServiceSummary } from '../api/types'
import { CreateServiceFlow } from '../components/create-service-flow'
import {
  DatabaseTechnologyIcon,
  TemplateBrandIcon,
} from '../components/template-brand-icon'
import type { TemplateItem } from '../hooks/types'
import { useCreateEnvironment } from '../hooks/use-create-environment'
import { useDeleteEnvironment } from '../hooks/use-delete-environment'
import { useEnvVars } from '../hooks/use-env-vars'
import { useEnvironments } from '../hooks/use-environments'
import { useProjectVars } from '../hooks/use-project-vars'
import { useProjects } from '../hooks/use-projects'
import { useReplaceEnvVars } from '../hooks/use-replace-env-vars'
import { useReplaceProjectVars } from '../hooks/use-replace-project-vars'
import { useServiceAction } from '../hooks/use-service-action'
import { useServices } from '../hooks/use-service-details'
import { useTemplates } from '../hooks/use-templates'
import { useUpdateEnvironment } from '../hooks/use-update-environment'

export function ProjectDetail() {
  const { projectId } = useParams({ from: '/_shell/projects/$projectId' })
  const navigate = useNavigate()
  const projectsQuery = useProjects()
  const environmentsQuery = useEnvironments(projectId)
  const varsQuery = useProjectVars(projectId)
  const project = projectsQuery.data?.find((item) => item.id === projectId)
  const environments = environmentsQuery.data ?? []
  const [selectedEnvironmentId, setSelectedEnvironmentId] = useState('')
  const [createOpen, setCreateOpen] = useState(false)
  const [environmentName, setEnvironmentName] = useState('')
  const [environmentDescription, setEnvironmentDescription] = useState('')
  const [editingEnvironmentId, setEditingEnvironmentId] = useState<string | null>(null)
  const [variablesOpen, setVariablesOpen] = useState(false)
  const [variableScope, setVariableScope] = useState<'project' | 'environment' | null>(
    null,
  )
  const [variableDrafts, setVariableDrafts] = useState<
    Array<{ key: string; value: string; is_secret: boolean }>
  >([])
  const [deleteServicesOpen, setDeleteServicesOpen] = useState(false)
  const [actionsOpen, setActionsOpen] = useState(false)
  const [selectedServiceIds, setSelectedServiceIds] = useState<Set<string>>(new Set())
  const [actionError, setActionError] = useState('')
  const [environmentError, setEnvironmentError] = useState('')
  const createEnvironment = useCreateEnvironment(projectId)
  const updateEnvironment = useUpdateEnvironment(projectId)
  const deleteEnvironment = useDeleteEnvironment(projectId)
  const environmentVarsQuery = useEnvVars(projectId, selectedEnvironmentId || null)
  const replaceProjectVars = useReplaceProjectVars(projectId)
  const replaceEnvironmentVars = useReplaceEnvVars(projectId, selectedEnvironmentId)
  const variableForm = useForm<Record<string, string>>({ defaultValues: {} })
  const servicesQuery = useServices(projectId, selectedEnvironmentId || undefined)
  const templatesQuery = useTemplates()
  const start = useServiceAction('start')
  const stop = useServiceAction('stop')
  const restart = useServiceAction('restart')
  const remove = useServiceAction('delete')

  useEffect(() => {
    if (!environments.length) {
      setSelectedEnvironmentId('')
      return
    }
    setSelectedEnvironmentId((current) =>
      environments.some((environment) => environment.id === current)
        ? current
        : (
            environments.find((environment) => environment.is_default) ??
            environments[0]
          ).id,
    )
  }, [environments])

  useEffect(() => {
    setSelectedServiceIds(new Set())
  }, [selectedEnvironmentId])

  const selectedEnvironment = environments.find(
    (environment) => environment.id === selectedEnvironmentId,
  )
  const services = servicesQuery.data ?? []
  useEffect(() => {
    setSelectedServiceIds((current) => {
      const eligible = new Set(
        services
          .filter(
            (service) => service.status !== 'starting' && service.status !== 'stopping',
          )
          .map((service) => service.id),
      )
      return new Set(Array.from(current).filter((serviceId) => eligible.has(serviceId)))
    })
  }, [services])
  const variableData =
    variableScope === 'environment'
      ? (environmentVarsQuery.data ?? [])
      : (varsQuery.data ?? [])
  const variableDefinitions = variableDrafts.map((variable) => ({
    name: variable.key,
    secret: variable.is_secret,
  }))
  const selectedCount = selectedServiceIds.size

  const toggleService = (serviceId: string) => {
    setSelectedServiceIds((current) => {
      const next = new Set(current)
      if (next.has(serviceId)) next.delete(serviceId)
      else next.add(serviceId)
      return next
    })
  }

  const runBulkAction = async (action: 'start' | 'stop' | 'restart' | 'delete') => {
    const mutation =
      action === 'start'
        ? start
        : action === 'stop'
          ? stop
          : action === 'restart'
            ? restart
            : remove
    setActionError('')
    try {
      await Promise.all(
        Array.from(selectedServiceIds).map((serviceId) =>
          mutation.mutateAsync(serviceId),
        ),
      )
      setSelectedServiceIds(new Set())
      setDeleteServicesOpen(false)
    } catch {
      const verb =
        action === 'restart'
          ? 'restarted'
          : action === 'delete'
            ? 'deleted'
            : `${action}ed`
      setActionError(
        `The selected services could not be ${verb}. Review their current state and retry.`,
      )
    }
  }

  const submitEnvironment = async () => {
    const name = environmentName.trim()
    if (!name) return
    setEnvironmentError('')
    try {
      if (editingEnvironmentId) {
        await updateEnvironment.mutateAsync({
          environmentID: editingEnvironmentId,
          name,
          description: environmentDescription.trim() || undefined,
        })
        setSelectedEnvironmentId(editingEnvironmentId)
      } else {
        const createdEnvironment = await createEnvironment.mutateAsync({
          name,
          description: environmentDescription.trim() || undefined,
        })
        setSelectedEnvironmentId(createdEnvironment.id)
      }
      setEnvironmentName('')
      setEnvironmentDescription('')
      setEditingEnvironmentId(null)
      setCreateOpen(false)
    } catch {
      setEnvironmentError(
        'The environment could not be created. Verify the name and try again.',
      )
    }
  }

  const openCreateEnvironment = () => {
    setEnvironmentError('')
    setEditingEnvironmentId(null)
    setEnvironmentName('')
    setEnvironmentDescription('')
    setCreateOpen(true)
  }

  const openEditEnvironment = (environmentId: string) => {
    const environment = environments.find((item) => item.id === environmentId)
    if (!environment) return
    setEnvironmentError('')
    setEditingEnvironmentId(environment.id)
    setEnvironmentName(environment.name)
    setEnvironmentDescription(environment.description || '')
    setCreateOpen(true)
  }

  useEffect(() => {
    if (!variablesOpen) return
    const drafts = variableData.map((variable) => ({
      key: variable.key,
      value: variable.value,
      is_secret: variable.is_secret,
    }))
    setVariableDrafts(drafts)
    variableForm.reset(
      Object.fromEntries(drafts.map((variable) => [variable.key, variable.value])),
    )
  }, [variableData, variableForm, variablesOpen])

  const openVariables = (scope: 'project' | 'environment') => {
    if (scope === 'environment' && !selectedEnvironmentId) return
    setVariableScope(scope)
    setVariablesOpen(true)
  }

  const addVariable = (key: string) => {
    setVariableDrafts((current) => [...current, { key, value: '', is_secret: false }])
    variableForm.setValue(key, '')
  }

  const removeVariable = (key: string) => {
    setVariableDrafts((current) => current.filter((variable) => variable.key !== key))
    variableForm.unregister(key)
  }

  const saveVariables = async () => {
    if (!variableScope) return
    const values = variableForm.getValues()
    const entries = variableDrafts.reduce<
      Record<string, { value: string; secret: boolean }>
    >((current, variable) => {
      current[variable.key] = {
        value: values[variable.key] ?? variable.value,
        secret: variable.is_secret,
      }
      return current
    }, {})
    if (variableScope === 'environment')
      await replaceEnvironmentVars.mutateAsync(entries)
    else await replaceProjectVars.mutateAsync(entries)
    setVariablesOpen(false)
    setVariableScope(null)
  }

  if (projectsQuery.isLoading)
    return (
      <div className="grid gap-4">
        <Skeleton className="h-48" />
        <Skeleton className="h-64" />
      </div>
    )
  if (!project || projectsQuery.isError)
    return (
      <EmptyState
        title="Project unavailable"
        description="The control plane did not return this project record."
        action={
          <Button
            tone="neutral"
            onClick={() => projectsQuery.refetch()}
          >
            Retry request
          </Button>
        }
      />
    )

  return (
    <div className="grid w-full max-w-[88rem] gap-4">
      <button
        className="flex w-fit items-center gap-2 text-supporting text-text-tertiary transition-colors hover:text-text-primary"
        onClick={() => navigate({ to: '/projects' })}
        type="button"
      >
        <ArrowLeft size={16} />
        Project field
      </button>
      <div className="grid gap-6 rounded-2xl border border-border-subtle bg-surface-1 p-4 sm:p-6 lg:p-7">
        <header className="grid gap-5">
          <div className="flex flex-wrap items-center justify-between gap-6">
            <div className="grid min-w-0 gap-2">
              <span className="text-label tracking-[0.12em] text-text-subtle">
                PROJECT / {selectedEnvironment?.name?.toUpperCase() ?? 'SCOPE'}
              </span>
              <Typography
                as="h1"
                role="page-title"
              >
                {project.name}
              </Typography>
              <span className="text-supporting text-text-tertiary">
                {services.length} {services.length === 1 ? 'service' : 'services'} ·{' '}
                {selectedEnvironment?.name ?? 'Project'} environment
              </span>
            </div>
            <div className="flex flex-wrap items-center gap-2">
              <WorkspaceSwitcher
                emptyLabel="Choose environment"
                footer={
                  <span className="flex items-center gap-2">
                    <Plus size={17} />
                    Create environment
                  </span>
                }
                onFooterSelect={openCreateEnvironment}
                icon={
                  <Globe
                    size={14}
                    weight="bold"
                  />
                }
                optionActions={(option) => (
                  <span
                    className="flex shrink-0"
                    onClick={(event) => event.stopPropagation()}
                    onPointerDown={(event) => event.stopPropagation()}
                  >
                    <DropdownMenu
                      trigger={
                        <DotsThree
                          aria-label={`Actions for ${String(option.name)}`}
                          size={18}
                          weight="bold"
                        />
                      }
                      triggerClassName="grid size-8 cursor-pointer place-items-center rounded-md text-text-tertiary transition-colors hover:bg-surface-3 hover:text-text-primary focus-visible:outline-2 focus-visible:outline-focus"
                      className="min-w-36"
                      items={[
                        {
                          label: (
                            <span className="flex items-center gap-2">
                              <PencilSimple size={15} />
                              Edit
                            </span>
                          ),
                          onSelect: () => openEditEnvironment(option.id),
                        },
                        {
                          label: (
                            <span
                              onClick={(event) => event.stopPropagation()}
                              onPointerDown={(event) => event.stopPropagation()}
                            >
                              <AlertDialog
                                trigger={
                                  <span className="flex items-center gap-2">
                                    <TrashSimple size={15} />
                                    Delete
                                  </span>
                                }
                                triggerClassName="flex min-h-8 w-full cursor-pointer items-center gap-2 rounded-md px-0 text-left text-supporting text-danger-strong outline-none focus-visible:outline-2 focus-visible:outline-focus"
                                title={`Delete ${String(option.name)}?`}
                                description="This removes the environment and its operational boundary."
                                confirmLabel={
                                  deleteEnvironment.isPending
                                    ? 'Deleting...'
                                    : 'Delete environment'
                                }
                                onConfirm={() =>
                                  deleteEnvironment.mutate(option.id, {
                                    onSuccess: () => {
                                      if (selectedEnvironmentId === option.id)
                                        setSelectedEnvironmentId('')
                                    },
                                  })
                                }
                              />
                            </span>
                          ),
                          danger: true,
                          disabled:
                            environments.length <= 1 || option.detail === 'Default',
                        },
                      ]}
                    />
                  </span>
                )}
                options={environments.map((environment) => ({
                  id: environment.id,
                  name: environment.name,
                  detail: environment.is_default ? 'Default' : undefined,
                }))}
                title="Environments"
                value={selectedEnvironmentId}
                onValueChange={setSelectedEnvironmentId}
              />
              <DropdownMenu
                onOpenChange={setActionsOpen}
                open={actionsOpen}
                trigger={
                  <span className="flex min-w-0 items-center gap-2.5">
                    <span className="grid size-6 shrink-0 place-items-center rounded-md bg-action-soft text-action-strong">
                      <Plus
                        size={14}
                        weight="bold"
                      />
                    </span>
                    <span className="truncate text-supporting font-medium">
                      Actions
                    </span>
                    {actionsOpen ? (
                      <CaretUp
                        aria-hidden="true"
                        className="shrink-0 text-text-tertiary"
                        size={16}
                        weight="bold"
                      />
                    ) : (
                      <CaretDown
                        aria-hidden="true"
                        className="shrink-0 text-text-tertiary"
                        size={16}
                        weight="bold"
                      />
                    )}
                  </span>
                }
                triggerClassName="inline-flex min-h-10 cursor-pointer items-center rounded-xl px-2.5 text-text-primary transition-[background-color,color,transform] duration-[var(--ely-duration-fast)] ease-ely-out hover:bg-surface-2 active:bg-surface-3 motion-safe:active:scale-[var(--ely-motion-press-scale)] focus-visible:outline-2 focus-visible:outline-focus"
                className="min-w-64 !rounded-xl !border-border-subtle !bg-surface-1/90 p-2 backdrop-blur-xl"
                items={[
                  {
                    label: (
                      <span className="flex items-center gap-2">
                        <BracketsCurly size={15} />
                        Project variables
                      </span>
                    ),
                    onSelect: () => openVariables('project'),
                  },
                  {
                    label: (
                      <span className="flex items-center gap-2">
                        <Key size={15} />
                        Environment variables
                      </span>
                    ),
                    disabled: !selectedEnvironment,
                    onSelect: () => openVariables('environment'),
                  },
                ]}
              />
              <Modal
                open={createOpen}
                onOpenChange={(open) => {
                  setCreateOpen(open)
                  if (!open) setEditingEnvironmentId(null)
                }}
                trigger={<span>Open environment creation</span>}
                triggerClassName="sr-only"
                popupClassName="max-w-xl"
                title={editingEnvironmentId ? 'Edit environment' : 'Create environment'}
                description={
                  editingEnvironmentId
                    ? 'Update the identity and operating context for this environment.'
                    : 'Establish a new operational boundary inside this project.'
                }
                footer={
                  <Button
                    disabled={
                      !environmentName.trim() ||
                      createEnvironment.isPending ||
                      updateEnvironment.isPending
                    }
                    onClick={() => void submitEnvironment()}
                  >
                    {createEnvironment.isPending || updateEnvironment.isPending
                      ? 'Saving...'
                      : editingEnvironmentId
                        ? 'Save environment'
                        : 'Create environment'}
                  </Button>
                }
              >
                <div className="grid gap-4">
                  <Field
                    id="environment-name"
                    label="Name"
                    required
                    description="Use a clear lifecycle or delivery boundary."
                  >
                    <Input
                      value={environmentName}
                      onChange={(event) => setEnvironmentName(event.target.value)}
                      placeholder="Production"
                    />
                  </Field>
                  <Field
                    id="environment-description"
                    label="Description"
                  >
                    <Textarea
                      value={environmentDescription}
                      onChange={(event) =>
                        setEnvironmentDescription(event.target.value)
                      }
                      placeholder="Operational context for this environment"
                    />
                  </Field>
                  {environmentError ? (
                    <InlineError>{environmentError}</InlineError>
                  ) : null}
                </div>
              </Modal>
              <Modal
                open={variablesOpen}
                onOpenChange={(open) => {
                  setVariablesOpen(open)
                  if (!open) {
                    setVariableScope(null)
                    setVariableDrafts([])
                  }
                }}
                trigger={<span>Open variable editor</span>}
                triggerClassName="sr-only"
                popupClassName="max-w-3xl"
                title={
                  variableScope === 'environment'
                    ? `${selectedEnvironment?.name ?? 'Environment'} variables`
                    : 'Project variables'
                }
                description={
                  variableScope === 'environment'
                    ? 'Environment-scoped values applied to services in the selected environment.'
                    : 'Project-scoped values available across the project boundary.'
                }
                footer={
                  <Button
                    disabled={
                      variableForm.formState.isSubmitting ||
                      replaceProjectVars.isPending ||
                      replaceEnvironmentVars.isPending
                    }
                    onClick={() => void variableForm.handleSubmit(saveVariables)()}
                  >
                    {replaceProjectVars.isPending || replaceEnvironmentVars.isPending
                      ? 'Saving variables...'
                      : 'Save variables'}
                  </Button>
                }
              >
                {(variableScope === 'environment' && environmentVarsQuery.isLoading) ||
                (variableScope === 'project' && varsQuery.isLoading) ? (
                  <Skeleton className="h-48" />
                ) : (variableScope === 'environment' && environmentVarsQuery.isError) ||
                  (variableScope === 'project' && varsQuery.isError) ? (
                  <InlineError>
                    Variables could not be loaded from the control plane.
                  </InlineError>
                ) : (
                  <VariableEditor
                    variables={variableDefinitions}
                    register={variableForm.register}
                    onAdd={addVariable}
                    onRemove={removeVariable}
                    className="max-h-[min(34rem,calc(100vh-16rem))] overflow-y-auto"
                  />
                )}
              </Modal>
            </div>
          </div>
        </header>

        <section className="grid gap-4 border-t border-border-subtle pt-6">
          <div className="mb-2 flex flex-wrap items-center justify-between gap-x-6 gap-y-3">
            <div className="grid gap-1">
              <div className="flex items-center gap-3">
                <Typography
                  as="h2"
                  role="section-title"
                >
                  Services
                </Typography>
                <span className="font-technical text-log text-text-tertiary">
                  {services.length}
                </span>
              </div>
              <span className="text-supporting text-text-tertiary">
                Manage resources in this environment.
              </span>
            </div>
            <CreateServiceFlow
              projectId={projectId}
              environmentId={selectedEnvironmentId || undefined}
            />
          </div>
          {actionError ? <InlineError>{actionError}</InlineError> : null}
          <BulkActionBar
            count={selectedCount}
            actions={[
              {
                id: 'start',
                label: (
                  <>
                    <Play size={15} />
                    Start selected
                  </>
                ),
                tone: 'accent',
                onSelect: () => void runBulkAction('start'),
              },
              {
                id: 'stop',
                label: (
                  <>
                    <Stop size={15} />
                    Stop selected
                  </>
                ),
                tone: 'neutral',
                onSelect: () => void runBulkAction('stop'),
              },
              {
                id: 'delete',
                label: (
                  <>
                    <TrashSimple size={15} />
                    Delete selected
                  </>
                ),
                tone: 'danger',
                onSelect: () => setDeleteServicesOpen(true),
              },
            ]}
          />
          {servicesQuery.isError ? (
            <EmptyState
              title="Service scope unavailable"
              description="The selected environment could not return its service inventory."
              action={
                <Button
                  tone="neutral"
                  onClick={() => servicesQuery.refetch()}
                >
                  Retry services
                </Button>
              }
            />
          ) : servicesQuery.isLoading ? (
            <div className="grid gap-3">
              <Skeleton className="h-36" />
              <Skeleton className="h-36" />
            </div>
          ) : services.length ? (
            <div className="grid gap-3">
              {services.map((service) => (
                <ServiceCard
                  key={service.id}
                  projectId={projectId}
                  service={service}
                  template={findServiceTemplate(service, templatesQuery.data ?? [])}
                  selected={selectedServiceIds.has(service.id)}
                  onSelect={() => toggleService(service.id)}
                  onOpen={() =>
                    navigate({
                      to: `/services/${service.id}`,
                      search: { from: 'project', projectId },
                    })
                  }
                />
              ))}
            </div>
          ) : (
            <EmptyState
              title="No services in this environment"
              description="Create a service or select another environment to inspect its runtime surface."
              action={
                <Button onClick={() => navigate({ to: '/services/new' })}>
                  <Plus size={16} />
                  Create service
                </Button>
              }
            />
          )}
        </section>
      </div>

      <Modal
        open={deleteServicesOpen}
        onOpenChange={setDeleteServicesOpen}
        trigger={<span>Open service deletion confirmation</span>}
        triggerClassName="sr-only"
        popupClassName="max-w-xl"
        title="Delete selected services"
        description="This is a destructive operation for the selected service resources."
        footer={
          <div className="flex w-full justify-end gap-2">
            <Button
              tone="ghost"
              onClick={() => setDeleteServicesOpen(false)}
            >
              Cancel
            </Button>
            <Button
              tone="danger"
              disabled={remove.isPending}
              onClick={() => void runBulkAction('delete')}
            >
              {remove.isPending ? 'Deleting services...' : 'Delete services'}
            </Button>
          </div>
        }
      >
        <div className="grid gap-3 rounded-xl border border-danger/40 bg-danger-soft p-4">
          <div className="flex items-center gap-2 text-label text-danger-strong">
            <TrashSimple size={17} />
            Danger zone
          </div>
          <p className="text-supporting text-text-secondary">
            You are about to permanently delete {selectedCount} selected{' '}
            {selectedCount === 1 ? 'service' : 'services'} from this environment. This
            cannot be undone.
          </p>
          <p className="font-technical text-log text-danger-strong">
            Confirm only when you intend to remove these resources.
          </p>
        </div>
      </Modal>
    </div>
  )
}

function ServiceCard({
  projectId,
  service,
  template,
  selected,
  onSelect,
  onOpen,
}: {
  projectId: string
  service: ServiceSummary
  template?: TemplateItem
  selected: boolean
  onSelect: () => void
  onOpen: () => void
}) {
  const runtimeCount = service.runtime?.containers.length ?? 0
  const transitioning = service.status === 'starting' || service.status === 'stopping'
  const operationalDetail =
    service.status === 'starting'
      ? 'Starting service…'
      : service.status === 'stopping'
        ? 'Stopping service…'
        : service.status === 'pending'
          ? 'Waiting for first deployment'
          : service.status === 'degraded' || service.status === 'failed'
            ? 'Runtime requires attention'
            : service.status === 'stopped'
              ? 'Service is stopped'
              : runtimeCount
                ? `${runtimeCount} runtime ${runtimeCount === 1 ? 'container' : 'containers'}`
                : 'No runtime assigned'
  const engine = service.spec?.engine?.toLowerCase()
  const engineName = engine
    ? ({
        mariadb: 'MariaDB',
        mongodb: 'MongoDB',
        mysql: 'MySQL',
        postgres: 'PostgreSQL',
        postgresql: 'PostgreSQL',
        redis: 'Redis',
      }[engine] ?? `${engine[0].toUpperCase()}${engine.slice(1)}`)
    : undefined
  const technology = engineName
    ? `${engineName}${service.spec?.version ? ` ${service.spec.version}` : ''}`
    : service.spec?.image
      ? 'Container image'
      : service.kind === 'app'
        ? 'Application runtime'
        : 'Managed resource'
  const statusLabel =
    service.status === 'pending'
      ? 'Pending'
      : service.status === 'deploying'
        ? 'Deploying'
        : service.status === 'starting'
          ? 'Starting'
          : service.status === 'stopping'
            ? 'Stopping'
            : service.status === 'running'
              ? 'Running'
              : service.status === 'failed'
                ? 'Failed'
                : service.status === 'stopped'
                  ? 'Stopped'
                  : service.status === 'degraded'
                    ? 'Degraded'
                    : 'Unknown'
  const serviceHref = `/services/${service.id}?from=project&projectId=${encodeURIComponent(projectId)}`
  return (
    <article
      className={`group grid grid-cols-[minmax(0,1fr)_auto] items-stretch gap-2 rounded-xl border p-4 transition-[background-color,border-color] duration-[var(--ely-duration-fast)] motion-reduce:transition-none sm:p-5 ${selected ? 'border-action/50 bg-action-soft/60' : 'border-border-subtle bg-surface-2 hover:border-border-default hover:bg-surface-3'}`}
    >
      <a
        aria-label={`Open ${service.name}`}
        className="grid min-w-0 grid-cols-[3.5rem_minmax(0,1fr)] gap-x-4 gap-y-4 rounded-lg text-left focus-visible:outline-2 focus-visible:outline-focus"
        href={serviceHref}
        onClick={(event) => {
          if (
            event.button !== 0 ||
            event.metaKey ||
            event.ctrlKey ||
            event.shiftKey ||
            event.altKey
          )
            return
          event.preventDefault()
          onOpen()
        }}
      >
        <span
          className={`grid size-14 shrink-0 place-items-center rounded-xl bg-action-soft transition-colors duration-[var(--ely-duration-fast)] group-hover:bg-action/15 motion-reduce:transition-none ${service.kind === 'database' ? 'text-text-primary' : 'text-action-strong'}`}
        >
          {service.kind === 'database' ? (
            <DatabaseTechnologyIcon engine={engine} />
          ) : service.kind === 'compose' && template ? (
            <TemplateBrandIcon
              template={template}
              className="size-8"
            />
          ) : service.kind === 'app' ? (
            <RocketLaunch
              size={30}
              weight="duotone"
            />
          ) : (
            <Stack
              size={30}
              weight="duotone"
            />
          )}
        </span>
        <div className="flex min-w-0 flex-wrap items-start justify-between gap-x-5 gap-y-2">
          <div className="grid min-w-0 gap-1">
            <span className="max-w-full truncate text-base font-semibold text-text-primary">
              {service.name}
            </span>
            <span className="truncate text-supporting text-text-tertiary">
              {technology}
              {technology ? ' · ' : ''}
              {service.kind === 'database'
                ? 'Database'
                : service.kind === 'app'
                  ? 'Application'
                  : 'Compose'}
            </span>
            {service.spec?.port ? (
              <span className="font-technical text-label text-text-tertiary">
                Port {service.spec.port}
              </span>
            ) : null}
          </div>
          <RuntimeStatus
            className="shrink-0 rounded-full bg-surface-3 px-2.5 py-1"
            label={statusLabel}
            status={toRuntimeStatus(service.status)}
          />
        </div>
        <div className="col-span-2 flex min-w-0 flex-wrap items-center justify-between gap-x-4 gap-y-2 border-t border-border-subtle pt-4">
          <span
            className={`truncate text-supporting ${service.status === 'failed' ? 'text-danger-strong' : service.status === 'degraded' ? 'text-warning-strong' : 'text-text-secondary'}`}
          >
            {operationalDetail}
          </span>
          <span className="flex shrink-0 items-center gap-1 text-supporting font-medium text-action">
            <span>
              {service.status === 'degraded' || service.status === 'failed'
                ? 'Inspect service'
                : 'View service'}
            </span>
            <ArrowRight
              aria-hidden="true"
              className="transition-transform duration-[var(--ely-duration-fast)] group-hover:translate-x-0.5 motion-reduce:transition-none"
              size={17}
            />
          </span>
        </div>
      </a>
      <label
        className={`grid size-8 place-items-center self-center rounded-md text-text-tertiary transition-[opacity,color,background-color] focus-within:opacity-100 motion-reduce:transition-none ${transitioning ? 'cursor-not-allowed opacity-50' : 'cursor-pointer hover:bg-surface-2 hover:text-text-primary'} ${selected ? 'opacity-100' : 'opacity-100 md:opacity-0 md:group-hover:opacity-100'}`}
        aria-label={`Select ${service.name}`}
      >
        <Checkbox
          checked={selected}
          disabled={transitioning}
          onChange={onSelect}
        />
      </label>
    </article>
  )
}

function findServiceTemplate(service: ServiceSummary, templates: TemplateItem[]) {
  if (service.kind !== 'compose' || !service.spec?.compose) return
  const serviceName = service.name.toLowerCase().replace(/[^a-z0-9]+/g, '-')
  const namedMatch = templates.find(
    (template) =>
      template.name.toLowerCase().replace(/[^a-z0-9]+/g, '-') === serviceName,
  )
  if (namedMatch) return namedMatch
  const serviceImages = new Set(extractComposeImages(service.spec.compose))
  if (!serviceImages.size) return
  const imageMatches = templates.filter((template) =>
    template.compose_yaml
      ? extractComposeImages(template.compose_yaml).some((image) =>
          serviceImages.has(image),
        )
      : false,
  )
  return imageMatches.length === 1 ? imageMatches[0] : undefined
}

function extractComposeImages(compose: string) {
  return [...compose.matchAll(/^\s*image:\s*["']?([^\s"'#]+)["']?\s*$/gm)].map(
    (match) => match[1],
  )
}

function toRuntimeStatus(
  status?: string,
): 'healthy' | 'deploying' | 'degraded' | 'failed' | 'stopped' | 'unknown' {
  if (status === 'running' || status === 'ready' || status === 'healthy')
    return 'healthy'
  if (
    status === 'deploying' ||
    status === 'building' ||
    status === 'starting' ||
    status === 'stopping'
  )
    return 'deploying'
  if (status === 'degraded') return 'degraded'
  if (status === 'failed' || status === 'error' || status === 'exited') return 'failed'
  if (status === 'stopped') return 'stopped'
  return 'unknown'
}
