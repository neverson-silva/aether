import {
  Button,
  Field,
  Input,
  Modal,
  Select,
  Textarea,
  Wizard,
  type WizardStep,
} from '@aether/elisyum-ds'
import { GitBranch, GithubLogo, Stack } from '@phosphor-icons/react'
import { useEffect, useState } from 'react'
import { apiPut } from '../api/client'
import { useCreateCompose } from '../hooks/use-create-compose'
import { useEnvironments } from '../hooks/use-environments'
import { useProjects } from '../hooks/use-projects'
import {
  selectGitHubConnection,
  useSourceControlBranches,
  useSourceControlConnections,
  useSourceControlFile,
  useSourceControlRepositories,
  useStartGitHubManifest,
} from '../hooks/use-source-control'
import { SourceCombobox, type SourceComboboxOption } from './source-combobox'
import {
  type EnvironmentVariableDraft,
  EnvironmentVariableEditor,
} from './environment-variable-editor'

type ComposeSourceMode = 'inline' | 'github'

export function ComposeStackDialog({
  environmentId,
  onCreated,
  onClose,
  open,
  projectId,
}: {
  environmentId?: string
  onCreated: (serviceId: string, projectId: string) => void
  onClose: () => void
  open: boolean
  projectId: string
}) {
  const [step, setStep] = useState(0)
  const [name, setName] = useState('')
  const [selectedProjectId, setSelectedProjectId] = useState(projectId)
  const [selectedEnvironmentId, setSelectedEnvironmentId] = useState(environmentId ?? '')
  const [sourceMode, setSourceMode] = useState<ComposeSourceMode>('inline')
  const [compose, setCompose] = useState('')
  const [repositoryId, setRepositoryId] = useState('')
  const [branch, setBranch] = useState('')
  const [rootDirectory, setRootDirectory] = useState('')
  const [composeFile, setComposeFile] = useState('compose.yaml')
  const [watchDirectory, setWatchDirectory] = useState('')
  const [autoDeploy, setAutoDeploy] = useState(true)
  const [variables, setVariables] = useState<EnvironmentVariableDraft[]>([])
  const [createdServiceId, setCreatedServiceId] = useState('')
  const [error, setError] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)
  const create = useCreateCompose()
  const projects = useProjects()
  const environments = useEnvironments(selectedProjectId)
  const availableProjects = projects.data ?? []
  const availableEnvironments = environments.data ?? []
  const selectedProject = availableProjects.find((project) => project.id === selectedProjectId)
  const selectedEnvironment = availableEnvironments.find(
    (environment) => environment.id === selectedEnvironmentId,
  )
  const connections = useSourceControlConnections(sourceMode === 'github')
  const githubConnection = selectGitHubConnection(connections.data)
  const repositories = useSourceControlRepositories(githubConnection?.installation_id)
  const selectedRepository = repositories.data?.find((repository) => repository.id === repositoryId)
  const branches = useSourceControlBranches(
    selectedRepository?.id,
    githubConnection?.installation_id,
  )
  const branchOptions = (branches.data ?? []).map<SourceComboboxOption>((item) => ({
    value: item.name,
    label: item.name,
    searchText: item.name,
    icon: <GitBranch aria-hidden="true" size={16} />,
  }))
  const repositoryOptions = (repositories.data ?? []).map<SourceComboboxOption>((item) => ({
    value: item.id,
    label: item.name,
    description: item.owner,
    searchText: item.full_name,
    icon: <GithubLogo aria-hidden="true" size={17} />,
  }))
  const normalizedRoot = normalizeRepositoryPath(rootDirectory)
  const normalizedComposeFile = normalizeRepositoryPath(composeFile)
  const composePath = joinRepositoryPath(normalizedRoot, normalizedComposeFile)
  const fileQuery = useSourceControlFile(
    selectedRepository?.id,
    githubConnection?.installation_id,
    composePath,
    branch,
    open && step >= 2 && sourceMode === 'github' && isSafeRepositoryPath(composePath),
  )
  const githubCompose = fileQuery.data?.content ?? ''
  const definition = sourceMode === 'inline' ? compose : githubCompose
  const variablesValid = variables.every(
    (variable) =>
      /^[A-Za-z_][A-Za-z0-9_]*$/.test(variable.name.trim()) &&
      variables.filter((candidate) => candidate.name.trim() === variable.name.trim())
        .length === 1,
  )
  const repositoryValid = Boolean(
    githubConnection &&
      selectedRepository &&
      branch &&
      branches.data?.some((item) => item.name === branch),
  )
  const pathsValid =
    isSafeRepositoryPath(normalizedRoot) &&
    isSafeRepositoryPath(normalizedComposeFile) &&
    parsePaths(watchDirectory).every((path) => isSafeRepositoryPath(path))
  const sourceValid =
    sourceMode === 'inline'
      ? Boolean(compose.trim())
      : repositoryValid &&
        pathsValid &&
        !fileQuery.isLoading &&
        !fileQuery.isError &&
        Boolean(githubCompose.trim())
  const identityValid = Boolean(
    name.trim().length >= 2 && selectedProject,
  )
  const canComplete = identityValid && sourceValid && variablesValid && !isSubmitting
  const startGitHubManifest = useStartGitHubManifest()

  useEffect(() => {
    if (open) return
    setSelectedProjectId(projectId)
    setSelectedEnvironmentId(environmentId ?? '')
  }, [environmentId, open, projectId])

  useEffect(() => {
    if (environments.isFetching || !availableEnvironments.length) return
    if (availableEnvironments.some((environment) => environment.id === selectedEnvironmentId))
      return
    setSelectedEnvironmentId(
      (availableEnvironments.find((environment) => environment.is_default) ??
        availableEnvironments[0]).id,
    )
  }, [availableEnvironments, environments.isFetching, selectedEnvironmentId])

  useEffect(() => {
    if (!selectedRepository || branches.isFetching || branches.isError || !branches.data) return
    const branchNames = branches.data.map((item) => item.name)
    const defaultBranch = selectedRepository.default_branch
    const preferredBranch = branchNames.includes(defaultBranch)
      ? defaultBranch
      : (branchNames[0] ?? '')
    setBranch((current) => (branchNames.includes(current) ? current : preferredBranch))
  }, [selectedRepository, branches.data, branches.isFetching, branches.isError])

  const reset = () => {
    setStep(0)
    setName('')
    setSelectedProjectId(projectId)
    setSelectedEnvironmentId(environmentId ?? '')
    setSourceMode('inline')
    setCompose('')
    setRepositoryId('')
    setBranch('')
    setRootDirectory('')
    setComposeFile('compose.yaml')
    setWatchDirectory('')
    setAutoDeploy(true)
    setVariables([])
    setCreatedServiceId('')
    setError('')
    setIsSubmitting(false)
  }

  const close = () => {
    reset()
    onClose()
  }

  const connectGitHub = async () => {
    setError('')
    try {
      const manifest = await startGitHubManifest.mutateAsync({
        return_url: `${window.location.pathname}${window.location.search}`,
      })
      const form = document.createElement('form')
      form.method = 'POST'
      form.action = `${manifest.url}?state=${encodeURIComponent(manifest.state)}`
      const input = document.createElement('input')
      input.type = 'hidden'
      input.name = 'manifest'
      input.value = manifest.manifest
      form.appendChild(input)
      document.body.appendChild(form)
      form.submit()
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'GitHub connection could not be started.')
    }
  }

  const submit = async () => {
    if (!canComplete) return
    setError('')
    setIsSubmitting(true)
    try {
      let serviceId = createdServiceId
      if (!serviceId) {
        const result = await create.mutateAsync({
          project_id: selectedProjectId,
          environment_id: selectedEnvironmentId || undefined,
          name: name.trim(),
          compose: definition.trim(),
        })
        serviceId = result.service_id || result.id
        setCreatedServiceId(serviceId)
      }
      if (sourceMode === 'github' && githubConnection && selectedRepository) {
        const watchPaths = parsePaths(watchDirectory).map((path) =>
          joinRepositoryPath(normalizedRoot, path),
        )
        await apiPut(`/api/v1/services/${serviceId}/source`, {
          connection_id: githubConnection.id,
          repository_id: selectedRepository.id,
          repository_owner: selectedRepository.owner,
          repository_name: selectedRepository.name,
          repository_full_name: selectedRepository.full_name,
          default_branch: selectedRepository.default_branch,
          branch,
          auto_deploy: autoDeploy,
          root_directory: normalizedRoot,
          environment_template_path: '.env.example',
          watch_paths: watchPaths,
          ignore_paths: [],
          watch_root_files: false,
          compose_file: normalizedComposeFile,
        })
      }
      for (const variable of variables) {
        await apiPut(`/api/v1/services/${serviceId}/environment`, {
          name: variable.name.trim(),
          value: variable.value,
          secret: variable.secret,
        })
      }
      close()
      onCreated(serviceId, selectedProjectId)
    } catch (reason) {
      setError(
        reason instanceof Error
          ? reason.message
          : 'The compose stack could not be created. Verify the definition and try again.',
      )
    } finally {
      setIsSubmitting(false)
    }
  }

  const steps: WizardStep[] = [
    {
      id: 'identity',
      label: 'Identity',
      description: 'Name the stack and choose its project and environment.',
      summary: name || 'Awaiting stack name',
      status: identityValid ? 'complete' : 'default',
      canContinue: identityValid,
      content: (
        <div className="grid max-w-2xl gap-5">
          <div className="flex items-center gap-3 rounded-xl border border-action/30 bg-action-soft p-4">
            <Stack aria-hidden="true" className="text-action-strong" size={22} />
            <span className="grid gap-1">
              <span className="text-label text-text-primary">Compose stack</span>
              <span className="text-supporting text-text-tertiary">
                All services in this Compose definition will share the selected project and environment.
              </span>
            </span>
          </div>
          <div className="grid min-w-0 gap-5 sm:grid-cols-2 [&>*]:min-w-0">
            <Field
              id="compose-project"
              label="Project"
              required
            >
              <Select
                className="min-w-0"
                disabled={
                  Boolean(createdServiceId) || projects.isLoading || projects.isError
                }
                value={selectedProject ? selectedProjectId : ''}
                onChange={(event) => {
                  setSelectedProjectId(event.target.value)
                  setSelectedEnvironmentId('')
                }}
              >
                <option value="">
                  {projects.isLoading ? 'Loading projects…' : 'Choose a project'}
                </option>
                {availableProjects.map((project) => (
                  <option
                    key={project.id}
                    value={project.id}
                  >
                    {project.name || project.slug || 'Unnamed project'}
                  </option>
                ))}
              </Select>
            </Field>
            <Field
              id="compose-environment"
              label="Environment"
            >
              <Select
                className="min-w-0"
                disabled={
                  Boolean(createdServiceId) ||
                  !selectedProjectId ||
                  environments.isLoading ||
                  environments.isError
                }
                value={selectedEnvironment ? selectedEnvironmentId : ''}
                onChange={(event) => setSelectedEnvironmentId(event.target.value)}
              >
                <option value="">
                  {environments.isLoading ? 'Loading environments…' : 'Project default'}
                </option>
                {availableEnvironments.map((environment) => (
                  <option
                    key={environment.id}
                    value={environment.id}
                  >
                    {environment.name || environment.slug || 'Unnamed environment'}
                  </option>
                ))}
              </Select>
            </Field>
          </div>
          <Field
            id="compose-name"
            label="Stack name"
            required
            description="Use the name operators will see in the runtime inventory."
          >
            <Input
              autoFocus
              value={name}
              onChange={(event) => setName(event.target.value)}
              placeholder="payments-stack"
            />
          </Field>
        </div>
      ),
    },
    {
      id: 'source',
      label: 'Source',
      description: 'Choose how Aether receives the Compose definition.',
      summary:
        sourceMode === 'inline'
          ? compose.trim()
            ? 'Pasted Compose YAML'
            : 'Choose a source'
          : selectedRepository
            ? `${selectedRepository.full_name}${branch ? ` · ${branch}` : ''}`
            : 'GitHub repository',
      status: sourceMode === 'inline' ? (compose.trim() ? 'complete' : 'default') : repositoryValid ? 'complete' : 'default',
      canContinue:
        sourceMode === 'inline'
          ? Boolean(compose.trim())
          : repositoryValid,
      content: (
        <div className="grid max-w-3xl gap-5">
          <div className="grid gap-2 sm:grid-cols-2">
            <button
              aria-pressed={sourceMode === 'inline'}
              className={`flex items-center gap-3 rounded-xl border p-4 text-left transition-colors ${sourceMode === 'inline' ? 'border-action bg-action-soft' : 'border-border-subtle bg-surface-1 hover:bg-surface-2'}`}
              onClick={() => setSourceMode('inline')}
              type="button"
            >
              <Stack aria-hidden="true" size={20} />
              <span className="grid gap-1">
                <span className="text-label">Paste Compose</span>
                <span className="text-supporting text-text-tertiary">
                  Keep the definition directly in Aether.
                </span>
              </span>
            </button>
            <button
              aria-pressed={sourceMode === 'github'}
              className={`flex items-center gap-3 rounded-xl border p-4 text-left transition-colors ${sourceMode === 'github' ? 'border-action bg-action-soft' : 'border-border-subtle bg-surface-1 hover:bg-surface-2'}`}
              onClick={() => setSourceMode('github')}
              type="button"
            >
              <GithubLogo aria-hidden="true" size={20} />
              <span className="grid gap-1">
                <span className="text-label">GitHub provider</span>
                <span className="text-supporting text-text-tertiary">
                  Keep the stack connected to a repository and branch.
                </span>
              </span>
            </button>
          </div>
          {sourceMode === 'inline' ? (
            <Field
              id="compose-definition"
              label="Compose definition"
              required
              description="Paste the Docker Compose YAML that defines the stack."
            >
              <Textarea
                className="min-h-64 font-technical text-code"
                value={compose}
                onChange={(event) => setCompose(event.target.value)}
                placeholder={'services:\n  web:\n    image: nginx:stable'}
              />
            </Field>
          ) : connections.isLoading ? (
            <p className="text-supporting text-text-tertiary" role="status">
              Checking the GitHub connection…
            </p>
          ) : connections.isError ? (
            <div className="grid gap-3">
              <p className="text-supporting text-danger" role="alert">
                Could not check the GitHub connection.
              </p>
              <Button tone="neutral" onClick={() => void connections.refetch()}>
                Retry
              </Button>
            </div>
          ) : !githubConnection ? (
            <div className="grid max-w-xl gap-3 rounded-xl border border-border-subtle bg-surface-1 p-4">
              <p className="text-supporting text-text-tertiary">
                Connect GitHub to choose a repository and branch.
              </p>
              <Button
                disabled={startGitHubManifest.isPending}
                onClick={() => void connectGitHub()}
              >
                {startGitHubManifest.isPending ? 'Connecting…' : 'Connect GitHub'}
              </Button>
            </div>
          ) : (
            <div className="grid max-w-2xl gap-5">
              <Field
                id="compose-repository"
                label="Repository"
                required
                description="Search repositories available to this GitHub connection."
              >
                <SourceCombobox
                  aria-label="Repository"
                  emptyDescription="Review the repositories available to this GitHub connection."
                  emptyLabel="No repositories found"
                  error={repositories.isError}
                  errorLabel="Unable to load repositories."
                  id="compose-repository"
                  leadingIcon={<GithubLogo aria-hidden="true" size={17} />}
                  loading={repositories.isLoading}
                  loadingLabel="Loading repositories…"
                  onRetry={repositories.refetch}
                  onValueChange={(value) => {
                    const nextRepository = (repositories.data ?? []).find((item) => item.id === value)
                    setRepositoryId(nextRepository?.id ?? '')
                    setBranch('')
                  }}
                  options={repositoryOptions}
                  placeholder="Search repositories…"
                  selectedLabel={selectedRepository?.full_name ?? ''}
                  title={selectedRepository?.full_name}
                  value={repositoryId}
                />
              </Field>
              <Field
                id="compose-branch"
                label="Branch"
                required
                description="The selected branch supplies the Compose file and future deployments."
              >
                <SourceCombobox
                  aria-label="Branch"
                  disabled={!selectedRepository}
                  emptyDescription="This repository does not have any branches available."
                  emptyLabel="No branches found"
                  error={branches.isError}
                  errorLabel="Unable to load branches."
                  id="compose-branch"
                  leadingIcon={<GitBranch aria-hidden="true" size={17} />}
                  loading={branches.isLoading}
                  loadingLabel="Loading branches…"
                  onRetry={branches.refetch}
                  onValueChange={setBranch}
                  options={branchOptions}
                  placeholder="Choose a branch…"
                  selectedLabel={branch}
                  value={branch}
                />
              </Field>
            </div>
          )}
        </div>
      ),
    },
    {
      id: 'configuration',
      label: 'Configuration',
      description: 'Set repository paths and runtime variables before creating the stack.',
      summary:
        sourceMode === 'github'
          ? `${composePath || 'Compose file'}${watchDirectory.trim() ? ` · watching ${watchDirectory.trim()}` : ''}`
          : `${variables.length} environment variables`,
      status: sourceMode === 'inline' || (sourceValid && variablesValid) ? 'complete' : 'default',
      canContinue: sourceMode === 'inline' ? variablesValid : sourceValid && variablesValid,
      content: (
        <div className="grid max-w-3xl gap-6">
          {sourceMode === 'github' ? (
            <div className="grid gap-5">
              <div className="grid gap-5 sm:grid-cols-2">
                <Field
                  id="compose-root-directory"
                  label="Repository root"
                  description="Directory used as the stack’s project root. Leave blank for the repository root."
                >
                  <Input
                    value={rootDirectory}
                    onChange={(event) => setRootDirectory(event.target.value)}
                    placeholder="apps/payments"
                  />
                </Field>
                <Field
                  id="compose-file"
                  label="Compose file"
                  required
                  description="Path relative to the repository root."
                >
                  <Input
                    value={composeFile}
                    onChange={(event) => setComposeFile(event.target.value)}
                    placeholder="compose.yaml"
                  />
                </Field>
              </div>
              <Field
                id="compose-watch-directory"
                label="Directory to monitor"
                description="Optional repository-relative folder or glob. The Compose file is always monitored. Leave blank to monitor the repository root."
              >
                <Input
                  value={watchDirectory}
                  onChange={(event) => setWatchDirectory(event.target.value)}
                  placeholder="services/payments/**"
                />
              </Field>
              <label className="flex cursor-pointer items-start gap-3 rounded-xl border border-border-subtle bg-surface-1 p-4">
                <input
                  checked={autoDeploy}
                  className="mt-1 accent-action"
                  onChange={(event) => setAutoDeploy(event.target.checked)}
                  type="checkbox"
                />
                <span className="grid gap-1">
                  <span className="text-label text-text-primary">Deploy monitored changes automatically</span>
                  <span className="text-supporting text-text-tertiary">
                    Pushes to the selected branch trigger a stack deployment when monitored paths change.
                  </span>
                </span>
              </label>
              {!pathsValid ? (
                <p className="text-supporting text-danger" role="alert">
                  Repository paths must stay inside the checkout and cannot be absolute.
                </p>
              ) : null}
              {fileQuery.isLoading ? (
                <p className="text-supporting text-text-tertiary" role="status">
                  Loading {composePath} from GitHub…
                </p>
              ) : fileQuery.isError ? (
                <div className="grid gap-3">
                  <p className="text-supporting text-danger" role="alert">
                    Could not read {composePath}. Check the path, branch, and repository permissions.
                  </p>
                  <Button tone="neutral" onClick={() => void fileQuery.refetch()}>
                    Retry file lookup
                  </Button>
                </div>
              ) : githubCompose ? (
                <p className="text-supporting text-success-strong" role="status">
                  Loaded {composePath} from {selectedRepository?.full_name}.
                </p>
              ) : null}
            </div>
          ) : null}
          <EnvironmentVariableEditor variables={variables} onChange={setVariables} />
          {!variablesValid ? (
            <p className="text-supporting text-danger" role="alert">
              Fix invalid or duplicate environment variable names to continue.
            </p>
          ) : null}
        </div>
      ),
    },
    {
      id: 'review',
      label: 'Review',
      description: 'Confirm the stack source and deployment scope.',
      summary: name || 'Ready to create',
      status: canComplete ? 'complete' : error ? 'error' : 'default',
      canContinue: canComplete,
      content: (
        <div className="grid max-w-3xl gap-5">
          <div className="grid gap-4 rounded-xl border border-border-subtle bg-surface-1 p-4 sm:grid-cols-2">
            <ReviewValue label="Stack name" value={name || '—'} />
            <ReviewValue label="Project" value={selectedProject?.name || '—'} />
            <ReviewValue
              label="Environment"
              value={selectedEnvironment?.name || 'Project default'}
            />
            <ReviewValue
              label="Source"
              value={sourceMode === 'inline' ? 'Pasted Compose definition' : 'GitHub repository'}
            />
            {sourceMode === 'github' ? (
              <>
                <ReviewValue label="Repository" value={selectedRepository?.full_name || '—'} />
                <ReviewValue label="Branch" value={branch || '—'} />
                <ReviewValue label="Repository root" value={normalizedRoot || '/'} />
                <ReviewValue label="Compose file" value={normalizedComposeFile || '—'} />
                <ReviewValue label="Monitored path" value={watchDirectory.trim() || 'Repository root'} />
                <ReviewValue label="Automatic deploys" value={autoDeploy ? 'Enabled' : 'Disabled'} />
              </>
            ) : null}
            <ReviewValue label="Environment variables" value={String(variables.length)} />
          </div>
          <div className="grid gap-2">
            <span className="text-label-caps text-text-tertiary">Compose preview</span>
            <pre className="max-h-56 overflow-auto whitespace-pre-wrap break-words rounded-lg bg-surface-dim p-4 font-technical text-code text-text-primary">
              {definition || 'The Compose definition will appear here.'}
            </pre>
          </div>
          {error ? (
            <p className="text-supporting text-danger" role="alert">
              {error}
            </p>
          ) : null}
        </div>
      ),
    },
  ]

  return (
    <Modal
      open={open}
      onOpenChange={(nextOpen) => {
        if (!nextOpen) close()
      }}
      trigger={<span>Open compose creation</span>}
      triggerClassName="sr-only"
      hideCancel
      popupClassName="!grid-rows-[minmax(0,1fr)_auto] !h-[min(50rem,calc(100dvh-2rem))] !max-h-[calc(100dvh-2rem)] !w-[min(64rem,calc(100vw-2rem))] !max-w-[64rem] !gap-0 !p-0 [&>div:first-of-type]:!h-full [&>div:first-of-type]:!min-h-0"
    >
      <div className="grid h-full min-h-0 grid-rows-[minmax(0,1fr)_auto]">
        <Wizard
          activeStep={step}
          backLabel="Back"
          completeLabel={
            isSubmitting
              ? 'Creating…'
              : createdServiceId
                ? 'Retry setup'
                : 'Create compose stack'
          }
          continueLabel="Continue"
          className="!h-full !min-h-0 !grid-rows-[auto_minmax(0,1fr)] rounded-none border-0 [&>div]:!h-full [&>div]:!min-h-0 [&>div>nav>ol]:!grid [&>div>nav>ol]:!grid-cols-1 [&>div>nav>ol]:!overflow-visible [&>div>nav>ol>li]:!min-w-0 [&>div>div:last-child]:!h-full [&>div>div:last-child]:!min-h-0 [&>div>div:last-child]:!grid-rows-[minmax(0,1fr)_auto] [&>div>div:last-child>div:first-child]:!min-h-0 [&>div>div:last-child>div:first-child]:!overflow-y-auto"
          onComplete={() => void submit()}
          onStepChange={setStep}
          steps={steps}
        />
      </div>
    </Modal>
  )
}

function ReviewValue({ label, value }: { label: string; value: string }) {
  return (
    <div className="grid min-w-0 gap-1">
      <span className="text-label-caps text-text-tertiary">{label}</span>
      <span className="truncate font-technical text-code text-text-primary" title={value}>
        {value}
      </span>
    </div>
  )
}

function normalizeRepositoryPath(value: string) {
  const normalized = value.trim().replaceAll('\\', '/').replace(/^\.\//, '').replace(/\/+$/, '')
  return normalized === '.' ? '' : normalized
}

function joinRepositoryPath(root: string, file: string) {
  return [root, normalizeRepositoryPath(file)].filter(Boolean).join('/')
}

function isSafeRepositoryPath(value: string) {
  const path = normalizeRepositoryPath(value)
  return !path.startsWith('/') && !path.split('/').includes('..')
}

function parsePaths(value: string) {
  return value
    .split(/[\n,]/)
    .map(normalizeRepositoryPath)
    .filter(Boolean)
}
