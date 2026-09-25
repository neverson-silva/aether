import {
  Button,
  Card,
  EmptyState,
  Field,
  Input,
  Marker,
  Modal,
  RuntimeStatus,
  Select,
  Skeleton,
  Typography,
} from '@aether/elisyum-ds'
import { Database, Plus } from '@phosphor-icons/react'
import { useNavigate } from '@tanstack/react-router'
import { useEffect, useState } from 'react'
import { useCreateDatabase } from '../hooks/use-create-database'
import { useEnvironments } from '../hooks/use-environments'
import { useProjects } from '../hooks/use-projects'
import { useServices } from '../hooks/use-service-details'

export function DatabaseField() {
  const navigate = useNavigate()
  const query = useServices()
  const rows = (query.data ?? []).filter((service) => service.kind === 'database')
  return (
    <div className="grid gap-7">
      <header className="flex flex-wrap items-end justify-between gap-5">
        <div className="grid gap-3">
          <div className="flex items-center gap-3 text-label tracking-[0.16em] text-text-subtle">
            <Database size={17} />
            RUNTIME / DATA
          </div>
          <Typography
            as="h1"
            role="page-title"
          >
            Database field
          </Typography>
          <Typography role="supporting">
            Managed data runtimes with engine, lifecycle and recovery context held in
            one operational surface.
          </Typography>
        </div>
        <CreateDatabaseDialog />
      </header>
      <section className="grid gap-4">
        <div className="flex items-center justify-between gap-4 px-1">
          <div className="flex items-center gap-3 text-label tracking-[0.12em] text-text-subtle">
            <Marker tone="accent" />
            {rows.length} DATABASES IN SCOPE
          </div>
          <span className="font-technical text-log text-text-tertiary">
            MANAGED RUNTIMES
          </span>
        </div>
        {query.isError ? (
          <EmptyState
            title="Database field unavailable"
            description="The control plane did not return the managed database collection."
            action={
              <Button
                tone="neutral"
                onClick={() => query.refetch()}
              >
                Retry request
              </Button>
            }
          />
        ) : query.isLoading ? (
          <div className="grid gap-2">
            <Skeleton className="h-20 rounded-xl" />
            <Skeleton className="h-20 rounded-xl" />
          </div>
        ) : rows.length ? (
          <Card className="grid gap-2 rounded-2xl border-border-subtle bg-surface-1 p-3">
            {rows.map((service) => (
              <button
                className="grid min-h-[5rem] grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-4 rounded-xl border border-transparent bg-surface-2 px-4 py-3 text-left transition-colors hover:border-border-default hover:bg-surface-3"
                key={service.id}
                onClick={() => {
                  void navigate({ to: `/databases/${service.id}` })
                }}
                type="button"
              >
                <span className="grid size-10 place-items-center rounded-xl bg-surface-1 text-text-tertiary">
                  <Database size={19} />
                </span>
                <span className="grid min-w-0 gap-1">
                  <span className="truncate text-supporting text-text-primary">
                    {service.name}
                  </span>
                  <span className="truncate font-technical text-log text-text-subtle">
                    {service.spec?.engine ?? 'managed'} ·{' '}
                    {service.spec?.version ?? 'default'} · {service.id}
                  </span>
                </span>
                <RuntimeStatus
                  label={service.status}
                  status={toRuntimeStatus(service.status)}
                />
              </button>
            ))}
          </Card>
        ) : (
          <EmptyState
            title="No database services"
            description="Provision a managed database to establish the first data runtime in this organization."
            icon={<Database size={28} />}
            action={<CreateDatabaseDialog />}
          />
        )}
      </section>
    </div>
  )
}

function CreateDatabaseDialog() {
  const projects = useProjects()
  const create = useCreateDatabase()
  const [open, setOpen] = useState(false)
  const [projectId, setProjectId] = useState('')
  const [environmentId, setEnvironmentId] = useState('')
  const [name, setName] = useState('')
  const [engine, setEngine] = useState('postgres')
  const [version, setVersion] = useState('')
  const [user, setUser] = useState('')
  const [password, setPassword] = useState('')
  const [memory, setMemory] = useState('512')
  const [storage, setStorage] = useState('0')
  const [cpus, setCPUs] = useState('0.5')
  const [error, setError] = useState('')
  const environments = useEnvironments(projectId)
  useEffect(() => {
    const options = environments.data ?? []
    setEnvironmentId((current) =>
      options.some((item) => item.id === current)
        ? current
        : ((options.find((item) => item.is_default) ?? options[0])?.id ?? ''),
    )
  }, [environments.data])
  const submit = async () => {
    if (!(projectId && name.trim())) return
    setError('')
    try {
      await create.mutateAsync({
        project_id: projectId,
        environment_id: environmentId || undefined,
        name: name.trim(),
        engine,
        version: version || undefined,
        user: user || undefined,
        password: password || undefined,
        cpus,
        mem_mb: Number(memory) || 512,
        storage_mb: Number(storage) || 0,
      })
      setOpen(false)
      setName('')
      setPassword('')
    } catch (value) {
      setError(value instanceof Error ? value.message : 'Database creation failed')
    }
  }
  return (
    <Modal
      open={open}
      onOpenChange={setOpen}
      trigger={
        <>
          <Plus size={17} />
          New database
        </>
      }
      title="Create database"
      description="Provision a managed data runtime inside an existing project boundary."
      popupClassName="max-w-2xl"
      footer={
        <Button
          disabled={!(projectId && name.trim()) || create.isPending}
          onClick={() => void submit()}
        >
          {create.isPending ? 'Provisioning…' : 'Create database'}
        </Button>
      }
    >
      <div className="grid gap-4">
        <div className="grid gap-4 sm:grid-cols-2">
          <Field
            id="database-project"
            label="Project"
            required
          >
            <Select
              value={projectId}
              onChange={(event) => setProjectId(event.target.value)}
            >
              <option value="">Choose project</option>
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
            id="database-environment"
            label="Environment"
          >
            <Select
              disabled={!projectId || environments.isLoading}
              value={environmentId}
              onChange={(event) => setEnvironmentId(event.target.value)}
            >
              <option value="">Project default</option>
              {(environments.data ?? []).map((environment) => (
                <option
                  key={environment.id}
                  value={environment.id}
                >
                  {environment.name}
                </option>
              ))}
            </Select>
          </Field>
        </div>
        <div className="grid gap-4 sm:grid-cols-2">
          <Field
            id="database-name"
            label="Database name"
            required
          >
            <Input
              value={name}
              onChange={(event) => setName(event.target.value)}
              placeholder="orders-db"
            />
          </Field>
          <Field
            id="database-engine"
            label="Engine"
            required
          >
            <Select
              value={engine}
              onChange={(event) => setEngine(event.target.value)}
            >
              <option value="postgres">PostgreSQL</option>
              <option value="mysql">MySQL</option>
              <option value="mariadb">MariaDB</option>
              <option value="mongodb">MongoDB</option>
              <option value="redis">Redis</option>
              <option value="mssql">SQL Server</option>
              <option value="oracle">Oracle</option>
            </Select>
          </Field>
        </div>
        <div className="grid gap-4 sm:grid-cols-2">
          <Field
            id="database-version"
            label="Version"
          >
            <Input
              value={version}
              onChange={(event) => setVersion(event.target.value)}
              placeholder="latest"
            />
          </Field>
          <Field
            id="database-user"
            label="User"
          >
            <Input
              value={user}
              onChange={(event) => setUser(event.target.value)}
              placeholder="operator"
            />
          </Field>
        </div>
        <div className="grid gap-4 sm:grid-cols-2">
          <Field
            id="database-password"
            label="Password"
          >
            <Input
              type="password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              placeholder="Optional managed credential"
            />
          </Field>
          <Field
            id="database-cpus"
            label="CPU limit (vCPU)"
          >
            <Select
              value={cpus}
              onChange={(event) => setCPUs(event.target.value)}
            >
              <option value="0.25">0.25 vCPU</option>
              <option value="0.5">0.5 vCPU</option>
              <option value="0.75">0.75 vCPU</option>
              <option value="1">1 vCPU</option>
              <option value="1.25">1.25 vCPU</option>
              <option value="1.5">1.5 vCPU</option>
              <option value="1.75">1.75 vCPU</option>
              <option value="2">2 vCPU</option>
            </Select>
          </Field>
        </div>
        <div className="grid gap-4 sm:grid-cols-2">
          <Field
            id="database-memory"
            label="Memory limit (MB)"
          >
            <Input
              inputMode="numeric"
              max={2048}
              min={256}
              step={256}
              type="number"
              value={memory}
              onChange={(event) => setMemory(event.target.value)}
            />
          </Field>
          <Field
            id="database-storage"
            label="SSD limit (MB)"
            description="Use 0 for the 100 GiB platform default."
          >
            <Input
              inputMode="numeric"
              max={102400}
              min={0}
              step={1024}
              type="number"
              value={storage}
              onChange={(event) => setStorage(event.target.value)}
            />
          </Field>
        </div>
        {error ? (
          <p
            className="text-supporting text-danger"
            role="alert"
          >
            {error}
          </p>
        ) : null}
      </div>
    </Modal>
  )
}

function toRuntimeStatus(
  status?: string,
): 'healthy' | 'deploying' | 'degraded' | 'failed' | 'stopped' | 'unknown' {
  if (status === 'running' || status === 'ready') return 'healthy'
  if (status === 'building' || status === 'deploying' || status === 'starting')
    return 'deploying'
  if (status === 'degraded') return 'degraded'
  if (status === 'failed' || status === 'error' || status === 'exited') return 'failed'
  if (status === 'stopped') return 'stopped'
  return 'unknown'
}
