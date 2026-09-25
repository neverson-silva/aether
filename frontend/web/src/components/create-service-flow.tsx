import {
  Button,
  CommandPalette,
  type CommandPaletteItem,
  Field,
  InlineError,
  Input,
  Modal,
  Select,
  Wizard,
  type WizardStep,
} from '@aether/elisyum-ds'
import {
  BracketsCurly,
  Database,
  GitBranch,
  GithubLogo,
  Package,
  RocketLaunch,
  Stack,
} from '@phosphor-icons/react'
import { useNavigate } from '@tanstack/react-router'
import { useEffect, useMemo, useRef, useState } from 'react'
import { apiGet, apiPut } from '../api/client'
import { useCreateApp } from '../hooks/use-create-app'
import { useCreateDatabase } from '../hooks/use-create-database'
import { useEnvironments } from '../hooks/use-environments'
import { useProjects } from '../hooks/use-projects'
import {
  selectGitHubConnection,
  useSourceControlBranches,
  useSourceControlConnections,
  useSourceControlRepositories,
  useStartGitHubManifest,
} from '../hooks/use-source-control'
import { ComposeStackDialog } from './compose-stack-dialog'
import {
  type EnvironmentVariableDraft,
  EnvironmentVariableEditor,
} from './environment-variable-editor'
import { SourceCombobox, type SourceComboboxOption } from './source-combobox'
import { TemplateBrowserDialog } from './template-browser-dialog'

type ServiceKind = 'web' | 'api' | 'database'
type SourceMode = 'image' | 'git'
type BuildType = 'buildpacks' | 'dockerfile' | 'custom' | 'compose'
const databaseEngines = {
  postgres: { label: 'PostgreSQL', versions: ['17', '16', '15', '14', 'latest'] },
  mysql: { label: 'MySQL', versions: ['8.4', '8.0', '8.4.3', 'latest'] },
  mariadb: { label: 'MariaDB', versions: ['11.4', '11.6', '10.11', 'latest'] },
  mongodb: { label: 'MongoDB', versions: ['7.0', '6.0', '7.0.16', 'latest'] },
  redis: { label: 'Redis', versions: ['7.2', '7.4', '6.2', 'latest'] },
  mssql: {
    label: 'SQL Server',
    versions: ['2022', '2022-CU15', '2022-CU14', 'latest'],
  },
  oracle: { label: 'Oracle', versions: ['23', '23.3', '23.2', 'latest'] },
} as const

export function CreateServiceFlow({
  projectId,
  environmentId,
}: {
  projectId: string
  environmentId?: string
}) {
  const navigate = useNavigate()
  const [kind, setKind] = useState<ServiceKind | null>(null)
  const [step, setStep] = useState(0)
  const [selectedProjectId, setSelectedProjectId] = useState(projectId)
  const [selectedEnvironmentId, setSelectedEnvironmentId] = useState(
    environmentId ?? '',
  )
  const [name, setName] = useState('')
  const [source, setSource] = useState<SourceMode>('image')
  const [image, setImage] = useState('')
  const [buildType, setBuildType] = useState<BuildType>('buildpacks')
  const [dockerfilePath, setDockerfilePath] = useState('Dockerfile')
  const [composeFile, setComposeFile] = useState('compose.yaml')
  const [rootFolder, setRootFolder] = useState('')
  const [watchPaths, setWatchPaths] = useState('')
  const [environmentTemplatePath, setEnvironmentTemplatePath] = useState('.env.example')
  const [appPort, setAppPort] = useState(3000)
  const [installCommand, setInstallCommand] = useState('')
  const [buildCommand, setBuildCommand] = useState('')
  const [startCommand, setStartCommand] = useState('')
  const [environmentVariables, setEnvironmentVariables] = useState<
    EnvironmentVariableDraft[]
  >([])
  const [repositoryId, setRepositoryId] = useState('')
  const [gitBranch, setGitBranch] = useState('')
  const [engine, setEngine] = useState('postgres')
  const [version, setVersion] = useState<string>(databaseEngines.postgres.versions[0])
  const [databaseCPUs, setDatabaseCPUs] = useState(0.5)
  const [databaseMemoryMB, setDatabaseMemoryMB] = useState(512)
  const [databaseStorageGB, setDatabaseStorageGB] = useState(0)
  const [appCPUs, setAppCPUs] = useState(0.5)
  const [appMemoryMB, setAppMemoryMB] = useState(512)
  const [appStorageGB, setAppStorageGB] = useState(0)
  const [error, setError] = useState('')
  const [environmentTemplateMessage, setEnvironmentTemplateMessage] = useState('')
  const environmentTemplateLoadedKey = useRef('')
  const [composeOpen, setComposeOpen] = useState(false)
  const [templatesOpen, setTemplatesOpen] = useState(false)
  const createApp = useCreateApp()
  const createDatabase = useCreateDatabase()
  const projects = useProjects()
  const environments = useEnvironments(selectedProjectId)
  const sourceConnections = useSourceControlConnections()
  const githubConnection = selectGitHubConnection(sourceConnections.data)
  const githubManifest = useStartGitHubManifest()
  const repositories = useSourceControlRepositories(githubConnection?.installation_id)
  const selectedRepository = repositories.data?.find((item) => item.id === repositoryId)
  const selectedRepositoryId = selectedRepository?.id
  const githubInstallationId = githubConnection?.installation_id
  const branches = useSourceControlBranches(
    selectedRepository?.id,
    githubConnection?.installation_id,
  )
  const repositoryOptions = (repositories.data ?? []).map<SourceComboboxOption>(
    (repository) => ({
      value: repository.id,
      label: repository.name,
      description: repository.owner,
      searchText: repository.full_name,
      icon: (
        <GithubLogo
          aria-hidden="true"
          size={17}
        />
      ),
    }),
  )
  const branchOptions = (branches.data ?? []).map<SourceComboboxOption>((branch) => ({
    value: branch.name,
    label: branch.name,
    searchText: branch.name,
    icon: (
      <GitBranch
        aria-hidden="true"
        size={16}
      />
    ),
  }))
  const isDatabase = kind === 'database'
  const isCreating = createApp.isPending || createDatabase.isPending
  const isBranchValid = Boolean(
    gitBranch && branches.data?.some((branch) => branch.name === gitBranch),
  )
  const sourceValid =
    source === 'image'
      ? image.trim().length > 0
      : Boolean(
          githubConnection && selectedRepository && branches.isSuccess && isBranchValid,
        )
  const buildConfigurationValid =
    source !== 'git' || buildType !== 'compose' || Boolean(composeFile.trim())
  const identityValid = Boolean(selectedProjectId) && name.trim().length >= 2
  const environmentVariablesValid = environmentVariables.every(
    (variable) =>
      /^[A-Za-z_][A-Za-z0-9_]*$/.test(variable.name.trim()) &&
      environmentVariables.filter(
        (candidate) => candidate.name.trim() === variable.name.trim(),
      ).length === 1,
  )

  useEffect(() => {
    if (
      !selectedRepository ||
      branches.isFetching ||
      branches.isError ||
      !branches.data
    )
      return
    const branchNames = branches.data.map((branch) => branch.name)
    const defaultBranch = selectedRepository.default_branch
    const preferredBranch = branchNames.includes(defaultBranch)
      ? defaultBranch
      : (branchNames[0] ?? '')
    setGitBranch((current) =>
      branchNames.includes(current) ? current : preferredBranch,
    )
  }, [selectedRepository, branches.data, branches.isFetching, branches.isError])

  useEffect(() => {
    const options = environments.data ?? []
    if (
      environments.isFetching ||
      !options.length ||
      options.some((item) => item.id === selectedEnvironmentId)
    )
      return
    setSelectedEnvironmentId((options.find((item) => item.is_default) ?? options[0]).id)
  }, [environments.data, environments.isFetching, selectedEnvironmentId])

  const close = () => {
    setKind(null)
    setStep(0)
    setError('')
    setEnvironmentTemplateMessage('')
  }

  const choose = (nextKind: ServiceKind) => {
    setKind(nextKind)
    setStep(0)
    setError('')
    setName('')
    setImage('')
    setBuildType('buildpacks')
    setDockerfilePath('Dockerfile')
    setComposeFile('compose.yaml')
    setRootFolder('')
    setWatchPaths('')
    setEnvironmentTemplatePath('.env.example')
    setAppPort(nextKind === 'api' ? 8080 : 3000)
    setInstallCommand('')
    setBuildCommand('')
    setStartCommand('')
    setEnvironmentVariables([])
    setEnvironmentTemplateMessage('')
    environmentTemplateLoadedKey.current = ''
    setRepositoryId('')
    setGitBranch('')
    setAppCPUs(0.5)
    setAppMemoryMB(512)
    setAppStorageGB(0)
    setSource(nextKind === 'api' ? 'git' : 'image')
  }

  useEffect(() => {
    if (
      step !== 3 ||
      source !== 'git' ||
      !selectedRepositoryId ||
      !githubInstallationId
    ) {
      if (step === 3 && source !== 'git') {
        setEnvironmentTemplateMessage(
          'Add environment variables manually for this container image.',
        )
      }
      return
    }
    const templatePath = environmentTemplatePath.trim()
    if (!templatePath) {
      setEnvironmentTemplateMessage(
        'Add environment variables manually or set a template file path in step 2.',
      )
      return
    }
    const normalizedRoot = rootFolder.trim().replace(/^\.\//, '').replace(/\/$/, '')
    const path = [normalizedRoot, templatePath].filter(Boolean).join('/')
    const requestKey = [
      selectedRepositoryId,
      githubInstallationId,
      path,
      gitBranch,
    ].join('|')
    if (environmentTemplateLoadedKey.current === requestKey) return
    let active = true
    setEnvironmentTemplateMessage('Loading variables from the example file…')
    apiGet<{
      variables?: Array<{ key: string; value?: string; secret?: boolean }>
    }>(
      `/api/v1/source-control/github/repositories/${encodeURIComponent(selectedRepositoryId)}/file?installation_id=${encodeURIComponent(githubInstallationId)}&path=${encodeURIComponent(path)}&ref=${encodeURIComponent(gitBranch)}`,
    )
      .then((file) => {
        if (!active) return
        const variables = (file.variables ?? []).map((variable, index) => ({
          id: `template-${index}-${variable.key}`,
          name: variable.key,
          value: variable.value ?? '',
          secret: variable.secret ?? false,
          revealed: true,
        }))
        setEnvironmentVariables(variables)
        environmentTemplateLoadedKey.current = requestKey
        setEnvironmentTemplateMessage(
          variables.length
            ? `Loaded ${variables.length} variable names from ${path}. Review values before creating the service.`
            : `No variable names were found in ${path}. You can add them below.`,
        )
      })
      .catch(() => {
        if (active) {
          setEnvironmentTemplateMessage(
            `Could not load ${path}. Add environment variables manually or check the file path and branch in step 2.`,
          )
        }
      })
    return () => {
      active = false
    }
  }, [
    step,
    source,
    selectedRepositoryId,
    githubInstallationId,
    gitBranch,
    rootFolder,
    environmentTemplatePath,
  ])

  const connectGitHub = async () => {
    setError('')
    try {
      const manifest = await githubManifest.mutateAsync({
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
      setError(
        reason instanceof Error
          ? reason.message
          : 'GitHub connection could not be started.',
      )
    }
  }

  const complete = async () => {
    if (
      !(
        kind &&
        identityValid &&
        sourceValid &&
        buildConfigurationValid &&
        environmentVariablesValid
      )
    )
      return
    setError('')
    try {
      const gitUrl = selectedRepository
        ? `https://github.com/${selectedRepository.full_name}.git`
        : ''
      const created = isDatabase
        ? await createDatabase.mutateAsync({
            project_id: selectedProjectId,
            environment_id: selectedEnvironmentId || undefined,
            name: name.trim(),
            engine,
            version,
            cpus: databaseCPUs.toFixed(2).replace(/0+$/, '').replace(/\.$/, ''),
            mem_mb: databaseMemoryMB,
            storage_mb: databaseStorageGB * 1024,
          })
        : await createApp.mutateAsync({
            projectID: selectedProjectId,
            payload: {
              name: name.trim(),
              source_type: source,
              image: source === 'image' ? image.trim() : '',
              git_url: source === 'git' ? gitUrl.trim() : '',
              git_branch: gitBranch.trim(),
              dockerfile:
                source === 'git' && buildType === 'dockerfile'
                  ? dockerfilePath.trim()
                  : '',
              compose_file:
                source === 'git' && buildType === 'compose' ? composeFile.trim() : '',
              build_type: source === 'git' ? buildType : 'buildpacks',
              install_command:
                source === 'git' && buildType === 'custom' ? installCommand.trim() : '',
              build_command:
                source === 'git' && buildType === 'custom' ? buildCommand.trim() : '',
              start_command:
                source === 'git' && buildType === 'custom' ? startCommand.trim() : '',
              root_folder: source === 'git' ? rootFolder.trim() : '',
              dist_folder: '',
              watch_paths: source === 'git' ? watchPaths : '',
              ...(source !== 'git' || buildType !== 'compose' ? { port: appPort } : {}),
              environment_id: selectedEnvironmentId || undefined,
              env: environmentVariables
                .filter((variable) => variable.name.trim())
                .map(({ name: variableName, value, secret }) => ({
                  name: variableName.trim(),
                  value,
                  secret,
                })),
              resources: {
                cpus: formatCPU(appCPUs),
                mem_mb: appMemoryMB,
                storage_mb: appStorageGB * 1024,
              },
              health_check: {
                enabled: false,
                path: '/',
                interval_ms: 5000,
                timeout_ms: 2000,
                retries: 3,
              },
            },
          })
      if (!isDatabase && selectedRepository && githubConnection) {
        await apiPut(`/api/v1/services/${created.service_id ?? created.id}/source`, {
          connection_id: githubConnection.id,
          repository_id: selectedRepository.id,
          repository_owner: selectedRepository.owner,
          repository_name: selectedRepository.name,
          repository_full_name: selectedRepository.full_name,
          default_branch: selectedRepository.default_branch,
          branch: gitBranch,
          auto_deploy: false,
          root_directory: rootFolder.trim(),
          environment_template_path: environmentTemplatePath.trim() || '.env.example',
          watch_paths: parseWatchPaths(watchPaths),
          ignore_paths: [],
          watch_root_files: true,
          compose_file: buildType === 'compose' ? composeFile.trim() : '',
        })
      }
      close()
      await navigate({
        to: `/services/${created.service_id ?? created.id}`,
        search: { from: 'project', projectId: selectedProjectId },
      })
    } catch (reason) {
      setError(
        reason instanceof Error
          ? reason.message
          : 'The service could not be created. Review the configuration and try again.',
      )
    }
  }

  const items = useMemo<CommandPaletteItem[]>(
    () => [
      {
        id: 'web',
        label: 'Web application',
        icon: (
          <RocketLaunch
            aria-hidden="true"
            size={17}
            weight="duotone"
          />
        ),
        keywords: ['frontend', 'react', 'vite', 'website'],
        onSelect: () => choose('web'),
      },
      {
        id: 'api',
        label: 'API service',
        icon: (
          <BracketsCurly
            aria-hidden="true"
            size={17}
            weight="duotone"
          />
        ),
        keywords: ['backend', 'go', 'node', 'python'],
        onSelect: () => choose('api'),
      },
      {
        id: 'database',
        label: 'Database',
        icon: (
          <Database
            aria-hidden="true"
            size={17}
            weight="duotone"
          />
        ),
        keywords: ['managed', 'postgres', 'mysql', 'redis', 'data'],
        onSelect: () => choose('database'),
      },
      {
        id: 'compose',
        label: 'Compose stack',
        icon: (
          <Stack
            aria-hidden="true"
            size={17}
            weight="duotone"
          />
        ),
        keywords: ['docker', 'multi-container', 'yaml'],
        onSelect: () => setComposeOpen(true),
      },
      {
        id: 'templates',
        label: 'Browse templates',
        icon: (
          <Package
            aria-hidden="true"
            size={17}
            weight="duotone"
          />
        ),
        keywords: ['catalog', 'marketplace', 'ready-made'],
        onSelect: () => setTemplatesOpen(true),
      },
    ],
    [],
  )

  const availableProjects = projects.data ?? []
  const availableEnvironments = environments.data ?? []
  const projectFields = (
    <div className="grid min-w-0 grid-cols-1 gap-5 sm:grid-cols-[minmax(0,1fr)_minmax(0,1fr)] [&>*]:min-w-0">
      <Field
        id="service-project"
        label="Project"
        required
      >
        <Select
          className="min-w-0"
          disabled={projects.isLoading || projects.isError}
          value={
            availableProjects.some((project) => project.id === selectedProjectId)
              ? selectedProjectId
              : ''
          }
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
        id="service-environment"
        label="Environment"
      >
        <Select
          className="min-w-0"
          disabled={
            !selectedProjectId || environments.isLoading || environments.isError
          }
          value={
            availableEnvironments.some(
              (environment) => environment.id === selectedEnvironmentId,
            )
              ? selectedEnvironmentId
              : ''
          }
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
  )

  const steps: WizardStep[] = isDatabase
    ? [
        {
          id: 'identity',
          label: 'Identity',
          description: 'Give the managed data service a durable operational identity.',
          summary: name || 'Awaiting service name',
          canContinue: identityValid,
          content: (
            <div className="grid max-w-xl gap-5">
              <div className="flex items-center gap-3 rounded-xl border border-action/30 bg-action-soft p-4">
                <Database
                  size={22}
                  className="text-action-strong"
                />
                <span className="grid gap-1">
                  <span className="text-label text-text-primary">Managed database</span>
                  <span className="text-supporting text-text-tertiary">
                    Provisioned inside the selected project boundary.
                  </span>
                </span>
              </div>
              {projectFields}
              <Field
                id="service-name"
                label="Service name"
                description="Use the stable name operators will see in runtime and delivery surfaces."
                required
              >
                <Input
                  autoFocus
                  value={name}
                  onChange={(event) => setName(event.target.value)}
                  placeholder="primary-db"
                />
              </Field>
            </div>
          ),
        },
        {
          id: 'engine',
          label: 'Engine',
          description: 'Choose the database engine and runtime limits.',
          summary: `${databaseEngines[engine as keyof typeof databaseEngines]?.label ?? engine} ${version}`,
          canContinue: true,
          content: (
            <DatabaseEngineStep
              engine={engine}
              version={version}
              cpus={databaseCPUs}
              memoryMB={databaseMemoryMB}
              storageGB={databaseStorageGB}
              onEngineChange={(value) => {
                setEngine(value)
                setVersion(
                  databaseEngines[value as keyof typeof databaseEngines].versions[0],
                )
              }}
              onVersionChange={setVersion}
              onCPUsChange={setDatabaseCPUs}
              onMemoryChange={setDatabaseMemoryMB}
              onStorageChange={setDatabaseStorageGB}
            />
          ),
        },
      ]
    : [
        {
          id: 'identity',
          label: 'Identity',
          description: 'Establish the service name and its operational role.',
          summary: name || 'Awaiting service name',
          canContinue: identityValid,
          content: (
            <div className="grid max-w-xl gap-5">
              <div className="flex items-center gap-3 rounded-xl border border-action/30 bg-action-soft p-4">
                <RocketLaunch
                  size={22}
                  className="text-action-strong"
                />
                <span className="grid gap-1">
                  <span className="text-label text-text-primary">
                    {kind === 'api' ? 'API service' : 'Web application'}
                  </span>
                  <span className="text-supporting text-text-tertiary">
                    A deployable service attached to this project and environment.
                  </span>
                </span>
              </div>
              {projectFields}
              <Field
                id="service-name"
                label="Service name"
                description="Use the stable runtime name operators will use in alerts and delivery history."
                required
              >
                <Input
                  autoFocus
                  value={name}
                  onChange={(event) => setName(event.target.value)}
                  placeholder={kind === 'api' ? 'billing-api' : 'customer-portal'}
                />
              </Field>
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
              : sourceValid
                ? `${selectedRepository?.full_name} · ${gitBranch}`
                : 'Incomplete · Source required',
          status: sourceValid && buildConfigurationValid ? 'complete' : 'default',
          canContinue: sourceValid && buildConfigurationValid,
          content: (
            <div className="grid max-w-xl gap-5">
              <div className="grid gap-2 sm:grid-cols-2">
                <button
                  aria-pressed={source === 'image'}
                  className={`flex items-center gap-3 rounded-xl border p-4 text-left transition-colors ${source === 'image' ? 'border-action bg-action-soft' : 'border-border-subtle bg-surface-1 hover:bg-surface-2'}`}
                  onClick={() => {
                    setSource('image')
                    setRepositoryId('')
                    setGitBranch('')
                  }}
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
                  aria-pressed={source === 'git'}
                  className={`flex items-center gap-3 rounded-xl border p-4 text-left transition-colors ${source === 'git' ? 'border-action bg-action-soft' : 'border-border-subtle bg-surface-1 hover:bg-surface-2'}`}
                  onClick={() => {
                    setSource('git')
                    setRepositoryId('')
                    setGitBranch('')
                  }}
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
                    value={image}
                    onChange={(event) => setImage(event.target.value)}
                    placeholder="registry.example.com/team/service:stable"
                  />
                </Field>
              ) : sourceConnections.isLoading ? (
                <p
                  className="text-supporting text-text-tertiary"
                  role="status"
                >
                  Checking GitHub connection…
                </p>
              ) : sourceConnections.isError ? (
                <div className="grid gap-3">
                  <InlineError>Could not check the GitHub connection.</InlineError>
                  <Button
                    type="button"
                    tone="neutral"
                    onClick={() => void sourceConnections.refetch()}
                  >
                    Retry
                  </Button>
                </div>
              ) : !githubConnection ? (
                <div className="grid gap-3 rounded-xl border border-border-subtle bg-surface-1 p-4">
                  <p className="text-supporting text-text-tertiary">
                    Connect GitHub to choose a repository and branch.
                  </p>
                  <Button
                    type="button"
                    disabled={githubManifest.isPending}
                    onClick={() => void connectGitHub()}
                  >
                    {githubManifest.isPending ? 'Connecting…' : 'Connect GitHub'}
                  </Button>
                </div>
              ) : (
                <div className="grid max-w-xl gap-5">
                  <Field
                    id="service-repository"
                    label="Repository"
                    required
                    description="Search by repository name or organization."
                  >
                    <SourceCombobox
                      autoComplete="off"
                      aria-label="Repository"
                      emptyActionLabel={
                        repositories.data?.length ? undefined : 'Reconnect GitHub'
                      }
                      emptyDescription={
                        repositories.data?.length
                          ? 'Try another search or check your Git connection.'
                          : 'Review the repositories available to this GitHub connection.'
                      }
                      emptyLabel={
                        repositories.data?.length
                          ? 'No repositories found'
                          : 'No repositories available'
                      }
                      error={repositories.isError}
                      errorLabel="Unable to load repositories."
                      id="service-repository"
                      leadingIcon={
                        <GithubLogo
                          aria-hidden="true"
                          size={17}
                        />
                      }
                      loading={repositories.isLoading}
                      loadingLabel="Loading repositories…"
                      onEmptyAction={connectGitHub}
                      onRetry={repositories.refetch}
                      onValueChange={(value) => {
                        const nextRepository = (repositories.data ?? []).find(
                          (item) => item.id === value,
                        )
                        setRepositoryId(nextRepository?.id ?? '')
                        setGitBranch('')
                      }}
                      options={repositoryOptions}
                      placeholder="Search repositories…"
                      selectedLabel={selectedRepository?.full_name ?? ''}
                      title={selectedRepository?.full_name}
                      value={repositoryId}
                    />
                  </Field>
                  <Field
                    id="service-branch"
                    label="Branch"
                    required
                    description={
                      selectedRepository
                        ? 'Choose a branch from this repository.'
                        : undefined
                    }
                  >
                    <SourceCombobox
                      disabled={!selectedRepository}
                      emptyDescription="This repository does not have any branches available."
                      emptyLabel="No branches found"
                      error={branches.isError}
                      errorLabel="Unable to load branches."
                      id="service-branch"
                      aria-label="Branch"
                      leadingIcon={
                        <GitBranch
                          aria-hidden="true"
                          size={17}
                        />
                      }
                      loading={Boolean(selectedRepository && branches.isLoading)}
                      loadingLabel="Loading branches…"
                      onRetry={branches.refetch}
                      onValueChange={setGitBranch}
                      options={branchOptions}
                      placeholder={
                        selectedRepository
                          ? 'Search branches…'
                          : 'Select a repository first'
                      }
                      selectedLabel={gitBranch}
                      value={gitBranch}
                    />
                  </Field>
                </div>
              )}
              {source === 'git' ? (
                <BuildConfiguration
                  buildType={buildType}
                  dockerfilePath={dockerfilePath}
                  composeFile={composeFile}
                  rootFolder={rootFolder}
                  watchPaths={watchPaths}
                  environmentTemplatePath={environmentTemplatePath}
                  port={appPort}
                  installCommand={installCommand}
                  buildCommand={buildCommand}
                  startCommand={startCommand}
                  onBuildTypeChange={setBuildType}
                  onDockerfilePathChange={setDockerfilePath}
                  onComposeFileChange={setComposeFile}
                  onRootFolderChange={setRootFolder}
                  onWatchPathsChange={setWatchPaths}
                  onEnvironmentTemplatePathChange={setEnvironmentTemplatePath}
                  onPortChange={setAppPort}
                  onInstallCommandChange={setInstallCommand}
                  onBuildCommandChange={setBuildCommand}
                  onStartCommandChange={setStartCommand}
                />
              ) : null}
            </div>
          ),
        },
        {
          id: 'resources',
          label: 'Resources',
          description: 'Set the maximum CPU, memory, and storage allocation.',
          summary: `${formatCPU(appCPUs)} vCPU · ${formatMemory(appMemoryMB)} · ${formatStorage(appStorageGB)}`,
          canContinue: true,
          content: (
            <ResourceAllocationStep
              cpus={appCPUs}
              memoryMB={appMemoryMB}
              storageGB={appStorageGB}
              onCPUsChange={setAppCPUs}
              onMemoryChange={setAppMemoryMB}
              onStorageChange={setAppStorageGB}
            />
          ),
        },
        {
          id: 'environment',
          label: 'Environment',
          description: 'Set the values available to the service at runtime.',
          summary: `${environmentVariables.filter((variable) => variable.name.trim()).length} variables`,
          canContinue: environmentVariablesValid,
          content: (
            <div className="grid max-w-3xl gap-4">
              {environmentTemplateMessage ? (
                <p
                  className="text-supporting text-text-tertiary"
                  role="status"
                >
                  {environmentTemplateMessage}
                </p>
              ) : null}
              <EnvironmentVariableEditor
                variables={environmentVariables}
                onChange={setEnvironmentVariables}
              />
              {!environmentVariablesValid ? (
                <p
                  className="text-supporting text-danger"
                  role="alert"
                >
                  Fix invalid or duplicate environment variable names to continue.
                </p>
              ) : null}
            </div>
          ),
        },
      ]

  return (
    <>
      <CommandPalette
        items={items}
        trigger={
          <span className="flex items-center gap-2">
            <PlusMark />
            Create service
          </span>
        }
        triggerClassName="inline-flex min-h-10 cursor-pointer items-center gap-2 rounded-xl bg-action px-4 text-label font-medium text-on-action shadow-elevation-1 transition-[background-color,box-shadow,transform] duration-[var(--ely-duration-fast)] hover:bg-action-strong hover:shadow-elevation-2 focus-visible:outline-2 focus-visible:outline-focus motion-safe:active:scale-[var(--ely-motion-press-scale)]"
      />
      <Modal
        open={Boolean(kind)}
        onOpenChange={(open) => {
          if (!open) close()
        }}
        trigger={<span>Open service creation</span>}
        triggerClassName="sr-only"
        hideCancel
        popupClassName="!grid-rows-[minmax(0,1fr)_auto] !h-[min(50rem,calc(100dvh-2rem))] !max-h-[calc(100dvh-2rem)] !w-[min(64rem,calc(100vw-2rem))] !max-w-[64rem] !gap-0 !p-0 [&>div:first-of-type]:!h-full [&>div:first-of-type]:!min-h-0"
      >
        <div className="grid h-full min-h-0 grid-rows-[minmax(0,1fr)_auto]">
          <Wizard
            activeStep={step}
            onStepChange={setStep}
            onComplete={() => void complete()}
            completeLabel={isCreating ? 'Provisioning...' : 'Create service'}
            steps={steps}
            className="!h-full !min-h-0 !grid-rows-[auto_minmax(0,1fr)] rounded-none border-0 [&>div]:!h-full [&>div]:!min-h-0 [&>div>div:last-child]:!h-full [&>div>div:last-child]:!min-h-0 [&>div>div:last-child]:!grid-rows-[minmax(0,1fr)_auto] [&>div>div:last-child>div:first-child]:!min-h-0 [&>div>div:last-child>div:first-child]:!overflow-y-auto"
          />
          {error ? (
            <div className="px-5 pb-5 sm:px-7">
              <InlineError>{error}</InlineError>
            </div>
          ) : null}
        </div>
      </Modal>
      <ComposeStackDialog
        environmentId={selectedEnvironmentId || undefined}
        onClose={() => setComposeOpen(false)}
        onCreated={(serviceId) => {
          void navigate({
            to: `/services/${serviceId}`,
            search: { from: 'project', projectId: selectedProjectId },
          })
        }}
        open={composeOpen}
        projectId={selectedProjectId}
      />
      <TemplateBrowserDialog
        onClose={() => setTemplatesOpen(false)}
        open={templatesOpen}
        projectId={selectedProjectId}
      />
    </>
  )
}

function PlusMark() {
  return (
    <span className="grid size-5 place-items-center rounded-md bg-action-strong/30 text-on-action">
      <span
        aria-hidden="true"
        className="text-lg leading-none"
      >
        +
      </span>
    </span>
  )
}

function BuildConfiguration({
  buildType,
  dockerfilePath,
  composeFile,
  rootFolder,
  watchPaths,
  environmentTemplatePath,
  port,
  installCommand,
  buildCommand,
  startCommand,
  onBuildTypeChange,
  onDockerfilePathChange,
  onComposeFileChange,
  onRootFolderChange,
  onWatchPathsChange,
  onEnvironmentTemplatePathChange,
  onPortChange,
  onInstallCommandChange,
  onBuildCommandChange,
  onStartCommandChange,
}: {
  buildType: BuildType
  dockerfilePath: string
  composeFile: string
  rootFolder: string
  watchPaths: string
  environmentTemplatePath: string
  port: number
  installCommand: string
  buildCommand: string
  startCommand: string
  onBuildTypeChange: (value: BuildType) => void
  onDockerfilePathChange: (value: string) => void
  onComposeFileChange: (value: string) => void
  onRootFolderChange: (value: string) => void
  onWatchPathsChange: (value: string) => void
  onEnvironmentTemplatePathChange: (value: string) => void
  onPortChange: (value: number) => void
  onInstallCommandChange: (value: string) => void
  onBuildCommandChange: (value: string) => void
  onStartCommandChange: (value: string) => void
}) {
  return (
    <div className="grid gap-4 border-t border-border-subtle pt-4">
      <Field
        id="service-build-type"
        label="Build strategy"
        description="Choose how Aether turns the repository into a runtime image."
        required
      >
        <Select
          value={buildType}
          onChange={(event) => onBuildTypeChange(event.target.value as BuildType)}
        >
          <option value="buildpacks">SmartBuild (CNB)</option>
          <option value="dockerfile">Dockerfile</option>
          <option value="custom">Custom commands</option>
          <option value="compose">Compose</option>
        </Select>
      </Field>
      <p className="text-supporting text-text-tertiary">
        {buildType === 'dockerfile'
          ? 'Build the application using a Dockerfile from the repository.'
          : buildType === 'buildpacks'
            ? 'Automatically detect and build the application using Cloud Native Buildpacks.'
            : buildType === 'custom'
              ? 'Use custom install, build, and start commands.'
              : 'Deploy using a Compose file from the repository.'}
      </p>
      {buildType === 'dockerfile' ? (
        <Field
          id="service-dockerfile-path"
          label="Dockerfile path"
          description="Path relative to the repository root or configured root directory."
        >
          <Input
            value={dockerfilePath}
            onChange={(event) => onDockerfilePathChange(event.target.value)}
            placeholder="Dockerfile"
          />
        </Field>
      ) : null}
      {buildType === 'compose' ? (
        <Field
          id="service-compose-file"
          label="Compose file"
          description="Path relative to the configured root directory."
          required
        >
          <Input
            aria-invalid={!composeFile.trim()}
            value={composeFile}
            onChange={(event) => onComposeFileChange(event.target.value)}
            placeholder="compose.yaml"
          />
        </Field>
      ) : null}
      {buildType !== 'compose' ? (
        <Field
          id="service-port"
          label="Public port"
          description="The container port exposed by this service."
        >
          <Input
            type="number"
            min={1}
            max={65535}
            value={port}
            onChange={(event) => onPortChange(Number(event.target.value))}
          />
        </Field>
      ) : null}
      {buildType === 'custom' ? (
        <div className="grid gap-4 sm:grid-cols-2">
          <Field
            id="service-install-command"
            label="Install command"
          >
            <Input
              value={installCommand}
              onChange={(event) => onInstallCommandChange(event.target.value)}
              placeholder="npm install"
            />
          </Field>
          <Field
            id="service-build-command"
            label="Build command"
          >
            <Input
              value={buildCommand}
              onChange={(event) => onBuildCommandChange(event.target.value)}
              placeholder="npm run build"
            />
          </Field>
          <Field
            id="service-start-command"
            label="Start command"
          >
            <Input
              value={startCommand}
              onChange={(event) => onStartCommandChange(event.target.value)}
              placeholder="npm start"
            />
          </Field>
        </div>
      ) : null}
      <details className="group rounded-xl border border-border-subtle bg-surface-1 p-4">
        <summary className="cursor-pointer list-none text-label font-semibold text-text-primary focus-visible:outline-2 focus-visible:outline-focus">
          Advanced settings
        </summary>
        <div className="mt-4 grid gap-4 border-t border-border-subtle pt-4">
          <Field
            id="service-root-folder"
            label="Root folder"
            description="Build from this folder, useful for monorepositories."
          >
            <Input
              value={rootFolder}
              onChange={(event) => onRootFolderChange(event.target.value)}
              placeholder="Repository root"
            />
          </Field>
          <Field
            id="service-watch-paths"
            label="Watch paths"
            description="Comma-separated paths that trigger automatic builds."
          >
            <Input
              value={watchPaths}
              onChange={(event) => onWatchPathsChange(event.target.value)}
              placeholder="apps/api/**, packages/shared/**"
            />
          </Field>
          <Field
            id="service-environment-template"
            label="Example environment file"
            description="Variable names are loaded automatically into step 4."
          >
            <Input
              value={environmentTemplatePath}
              onChange={(event) => onEnvironmentTemplatePathChange(event.target.value)}
              placeholder=".env.example"
            />
          </Field>
        </div>
      </details>
    </div>
  )
}

function ResourceAllocationStep({
  cpus,
  memoryMB,
  storageGB,
  onCPUsChange,
  onMemoryChange,
  onStorageChange,
}: {
  cpus: number
  memoryMB: number
  storageGB: number
  onCPUsChange: (value: number) => void
  onMemoryChange: (value: number) => void
  onStorageChange: (value: number) => void
}) {
  return (
    <div className="grid max-w-xl gap-5">
      <div className="grid gap-4 rounded-xl bg-surface-2 p-4">
        <span className="text-label-caps text-text-subtle">
          MAXIMUM RUNTIME ALLOCATION
        </span>
        <ResourceLimit
          id="app-cpu"
          label="CPU"
          value={`${formatCPU(cpus)} vCPU`}
          min={0.25}
          max={2}
          step={0.25}
          current={cpus}
          onChange={onCPUsChange}
        />
        <ResourceLimit
          id="app-memory"
          label="Memory"
          value={formatMemory(memoryMB)}
          min={256}
          max={2048}
          step={256}
          current={memoryMB}
          onChange={onMemoryChange}
        />
        <ResourceLimit
          id="app-storage"
          label="SSD storage"
          value={formatStorage(storageGB)}
          min={0}
          max={100}
          step={1}
          current={storageGB}
          onChange={onStorageChange}
          helper="Choose 0 to use the platform default."
        />
      </div>
    </div>
  )
}

function parseWatchPaths(value: string) {
  return value
    .split(/[\n,]/)
    .map((path) => path.trim())
    .filter(Boolean)
}

function formatCPU(value: number) {
  return value.toFixed(2).replace(/0+$/, '').replace(/\.$/, '')
}

function formatMemory(memoryMB: number) {
  return memoryMB >= 1024
    ? `${(memoryMB / 1024).toFixed(memoryMB % 1024 ? 2 : 0)} GiB`
    : `${memoryMB} MiB`
}

function formatStorage(storageGB: number) {
  return storageGB === 0 ? 'Platform default · 100 GiB' : `${storageGB} GiB`
}

function DatabaseEngineStep({
  engine,
  version,
  cpus,
  memoryMB,
  storageGB,
  onEngineChange,
  onVersionChange,
  onCPUsChange,
  onMemoryChange,
  onStorageChange,
}: {
  engine: string
  version: string
  cpus: number
  memoryMB: number
  storageGB: number
  onEngineChange: (value: string) => void
  onVersionChange: (value: string) => void
  onCPUsChange: (value: number) => void
  onMemoryChange: (value: number) => void
  onStorageChange: (value: number) => void
}) {
  const options =
    databaseEngines[engine as keyof typeof databaseEngines] ?? databaseEngines.postgres
  return (
    <div className="grid max-w-xl gap-5">
      <div className="grid gap-5 sm:grid-cols-2">
        <Field
          id="database-engine"
          label="Database engine"
          required
        >
          <Select
            value={engine}
            onChange={(event) => onEngineChange(event.target.value)}
          >
            {Object.entries(databaseEngines).map(([value, option]) => (
              <option
                key={value}
                value={value}
              >
                {option.label}
              </option>
            ))}
          </Select>
        </Field>
        <Field
          id="database-version"
          label="Version"
          required
        >
          <Select
            value={version}
            onChange={(event) => onVersionChange(event.target.value)}
          >
            {options.versions.map((option) => (
              <option
                key={option}
                value={option}
              >
                {option}
              </option>
            ))}
          </Select>
        </Field>
      </div>
      <div className="grid gap-4 rounded-xl bg-surface-2 p-4">
        <span className="text-label-caps text-text-subtle">
          MAXIMUM RUNTIME ALLOCATION
        </span>
        <ResourceLimit
          id="database-cpu"
          label="CPU"
          value={`${cpus.toFixed(2).replace(/0+$/, '').replace(/\.$/, '')} vCPU`}
          min={0.25}
          max={2}
          step={0.25}
          current={cpus}
          onChange={onCPUsChange}
        />
        <ResourceLimit
          id="database-memory"
          label="Memory"
          value={
            memoryMB >= 1024
              ? `${(memoryMB / 1024).toFixed(memoryMB % 1024 ? 2 : 0)} GiB`
              : `${memoryMB} MiB`
          }
          min={256}
          max={2048}
          step={256}
          current={memoryMB}
          onChange={onMemoryChange}
        />
        <ResourceLimit
          id="database-storage"
          label="SSD storage"
          value={storageGB === 0 ? 'Default · 100 GiB' : `${storageGB} GiB`}
          min={0}
          max={100}
          step={1}
          current={storageGB}
          onChange={onStorageChange}
          helper="Choose 0 to use the platform default."
        />
      </div>
    </div>
  )
}

function ResourceLimit({
  id,
  label,
  value,
  min,
  max,
  step,
  current,
  onChange,
  helper,
}: {
  id: string
  label: string
  value: string
  min: number
  max: number
  step: number
  current: number
  onChange: (value: number) => void
  helper?: string
}) {
  return (
    <Field
      id={id}
      label={label}
      description={helper}
    >
      <div className="grid gap-2">
        <div className="flex items-center justify-between gap-3">
          <span className="text-supporting text-text-secondary">Maximum</span>
          <span className="font-technical text-code text-action-strong">{value}</span>
        </div>
        <input
          aria-label={`${label} maximum`}
          className="w-full accent-action"
          max={max}
          min={min}
          onChange={(event) => onChange(Number(event.target.value))}
          step={step}
          type="range"
          value={current}
        />
        <div className="flex justify-between font-technical text-log text-text-subtle">
          <span>
            {min === 0 ? 'Default' : `${min}${label === 'CPU' ? ' vCPU' : ' MiB'}`}
          </span>
          <span>
            {max === 100 ? `${max} GiB` : `${max}${label === 'CPU' ? ' vCPU' : ' MiB'}`}
          </span>
        </div>
      </div>
    </Field>
  )
}
