import {
  Button,
  Field,
  Input,
  Modal,
  Select,
  Skeleton,
  Typography,
} from '@aether/elisyum-ds'
import { MagnifyingGlass } from '@phosphor-icons/react'
import { useEffect, useState } from 'react'
import type { TemplateItem } from '../hooks/types'
import { useInstallTemplate } from '../hooks/use-install-template'
import { useTemplateCategories } from '../hooks/use-template-categories'
import { useTemplatesFiltered } from '../hooks/use-templates-filtered'
import { TemplateBrandIcon } from './template-brand-icon'

export function TemplateBrowserDialog({
  onClose,
  open,
  projectId,
}: {
  onClose: () => void
  open: boolean
  projectId: string
}) {
  const [query, setQuery] = useState('')
  const [category, setCategory] = useState('')
  const [selected, setSelected] = useState<TemplateItem | null>(null)
  const [name, setName] = useState('')
  const [overrides, setOverrides] = useState<Record<string, string>>({})
  const [initialOverrides, setInitialOverrides] = useState<Record<string, string>>({})
  const templates = useTemplatesFiltered({
    category: category || undefined,
    q: query || undefined,
  })
  const categories = useTemplateCategories()
  const install = useInstallTemplate()

  useEffect(() => {
    if (!open) return
    setSelected(null)
    setName('')
    setOverrides({})
    setInitialOverrides({})
  }, [open])

  const pick = (template: TemplateItem) => {
    setSelected(template)
    setName(
      template.name
        .toLowerCase()
        .replace(/[^a-z0-9]+/g, '-')
        .replace(/^-|-$/g, ''),
    )
    const defaults = resolveTemplateEnvironment(template)
    setOverrides(defaults)
    setInitialOverrides(defaults)
  }

  const submit = async () => {
    if (!(selected && name.trim())) return
    const changedOverrides = Object.fromEntries(
      Object.entries(overrides).filter(
        ([key, value]) => value !== initialOverrides[key],
      ),
    )
    await install.mutateAsync({
      id: selected.id,
      project_id: projectId,
      name: name.trim(),
      overrides: changedOverrides,
    })
    onClose()
  }

  const variableEntries = selected ? Object.entries(overrides) : []
  return (
    <Modal
      open={open}
      onOpenChange={(nextOpen) => {
        if (!nextOpen) onClose()
      }}
      trigger={<span>Open template browser</span>}
      triggerClassName="sr-only"
      title={selected ? `${selected.name} · Configure` : 'Browse templates'}
      description={
        selected
          ? 'Review the template identity and environment defaults before installation.'
          : 'Choose a ready-made service template from the live catalog.'
      }
      popupClassName="!w-[min(72rem,calc(100vw-32px))] !max-w-[72rem]"
      footer={
        selected ? (
          <div className="flex w-full justify-between gap-2">
            <Button
              tone="ghost"
              onClick={() => setSelected(null)}
            >
              Back to templates
            </Button>
            <Button
              disabled={!name.trim() || install.isPending}
              onClick={() => void submit()}
            >
              {install.isPending ? 'Installing...' : 'Install template'}
            </Button>
          </div>
        ) : null
      }
    >
      {selected ? (
        <div className="grid gap-5">
          <div className="grid gap-2 rounded-xl border border-action/35 bg-action-soft p-4">
            <Typography
              as="h2"
              role="section-title"
            >
              {selected.name}
            </Typography>
            <p className="text-supporting text-text-secondary">
              {selected.description || 'Ready-made service template'}
            </p>
            <span className="font-technical text-log text-text-tertiary">
              {selected.category || 'General'} · {selected.installs} installs
            </span>
          </div>
          <Field
            id="template-name"
            label="Service name"
            required
          >
            <Input
              value={name}
              onChange={(event) => setName(event.target.value)}
            />
          </Field>
          {variableEntries.length ? (
            <div className="grid gap-3">
              <span className="text-label-caps text-text-subtle">
                Environment variables
              </span>
              {variableEntries.map(([key, value]) => (
                <Field
                  id={`template-${key}`}
                  key={key}
                  label={key}
                >
                  <Input
                    value={value}
                    onChange={(event) =>
                      setOverrides((current) => ({
                        ...current,
                        [key]: event.target.value,
                      }))
                    }
                  />
                </Field>
              ))}
            </div>
          ) : null}
          {install.isError ? (
            <p
              className="text-supporting text-danger"
              role="alert"
            >
              The template could not be installed. Review the selected configuration and
              try again.
            </p>
          ) : null}
        </div>
      ) : (
        <div className="grid min-h-0 gap-4">
          <div className="grid gap-3 sm:grid-cols-[minmax(0,1fr)_14rem]">
            <label className="flex min-h-10 items-center gap-2 rounded-lg border border-border-subtle bg-surface-2 px-3">
              <MagnifyingGlass
                aria-hidden="true"
                className="text-text-tertiary"
                size={17}
              />
              <input
                aria-label="Search templates"
                className="min-w-0 flex-1 bg-transparent text-supporting text-text-primary outline-none placeholder:text-text-tertiary"
                onChange={(event) => setQuery(event.target.value)}
                placeholder="Search templates..."
                value={query}
              />
            </label>
            <Select
              aria-label="Template category"
              value={category}
              onChange={(event) => setCategory(event.target.value)}
            >
              <option value="">All categories</option>
              {(categories.data ?? []).map((item) => (
                <option
                  key={item}
                  value={item}
                >
                  {item}
                </option>
              ))}
            </Select>
          </div>
          {templates.isLoading ? (
            <div className="grid gap-2 sm:grid-cols-2 xl:grid-cols-3">
              <Skeleton className="h-24" />
              <Skeleton className="h-24" />
              <Skeleton className="h-24" />
              <Skeleton className="h-24" />
            </div>
          ) : templates.isError ? (
            <p
              className="text-supporting text-danger"
              role="alert"
            >
              The template catalog could not be loaded.
            </p>
          ) : (
            <div className="grid max-h-[min(38rem,calc(100vh-18rem))] gap-3 overflow-y-auto pr-1 sm:grid-cols-2 xl:grid-cols-3">
              {(templates.data ?? []).map((template) => (
                <button
                  className="group grid gap-2 rounded-xl border border-border-subtle bg-surface-1 p-4 text-left transition-[background-color,border-color,transform] hover:-translate-y-px hover:border-action/50 hover:bg-surface-2 focus-visible:outline-2 focus-visible:outline-focus"
                  key={template.id}
                  onClick={() => pick(template)}
                  type="button"
                >
                  <span className="flex items-start justify-between gap-3">
                    <span className="grid size-10 place-items-center rounded-lg bg-surface-2 text-action">
                      <TemplateBrandIcon template={template} />
                    </span>
                    <span className="font-technical text-log text-text-tertiary">
                      {template.category || 'General'}
                    </span>
                  </span>
                  <span className="text-supporting font-medium text-text-primary">
                    {template.name}
                  </span>
                  <span className="line-clamp-2 text-label text-text-tertiary">
                    {template.description || 'Ready-made service template'}
                  </span>
                </button>
              ))}
              {!templates.data?.length ? (
                <p className="col-span-full py-8 text-center text-supporting text-text-tertiary">
                  No templates match your search.
                </p>
              ) : null}
            </div>
          )}
        </div>
      )}
    </Modal>
  )
}

function generateSecret(length = 24) {
  const alphabet = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789'
  const bytes = crypto.getRandomValues(new Uint8Array(length))
  return Array.from(bytes, (byte) => alphabet[byte % alphabet.length]).join('')
}

function resolveTemplateValue(
  value: string,
  variables: Record<string, string>,
  cache: Map<string, string>,
  seed: string,
  stack = new Set<string>(),
): string {
  return value.replace(/\$\{([^}]+)\}/g, (_, rawToken: string) => {
    const token = rawToken.trim()
    const [rawName, rawLength] = token.split(':')
    const name = rawName.toLowerCase()
    if (token in variables && !stack.has(token)) {
      const cached = cache.get(token)
      if (cached !== undefined) return cached
      const next = new Set(stack)
      next.add(token)
      const resolved: string = resolveTemplateValue(
        variables[token],
        variables,
        cache,
        seed,
        next,
      )
      cache.set(token, resolved)
      return resolved
    }
    const length = Number(rawLength) > 0 ? Number(rawLength) : 32
    if (name === 'password' || name === 'secret' || name === 'hash' || name === 'jwt')
      return generateSecret(length)
    if (name === 'uuid') return crypto.randomUUID()
    if (name === 'randomport') return String(1024 + Math.floor(Math.random() * 64512))
    if (name === 'timestamp' || name === 'timestampms') return String(Date.now())
    if (name === 'timestamps') return String(Math.round(Date.now() / 1000))
    if (name === 'domain')
      return `${seed.toLowerCase().replace(/[^a-z0-9-]+/g, '-')}.localhost`
    if (name === 'username') return `user${generateSecret(8).toLowerCase()}`
    if (name === 'email') return `user-${generateSecret(8).toLowerCase()}@example.com`
    return `\${${rawToken}}`
  })
}

function resolveTemplateEnvironment(template: TemplateItem) {
  const variables = Object.fromEntries(
    (template.variables ?? []).map((entry) => [entry.name, entry.value]),
  )
  for (const entry of template.environment ?? [])
    if (!(entry.name in variables)) variables[entry.name] = entry.value
  const cache = new Map<string, string>()
  return Object.fromEntries(
    (template.environment ?? template.variables ?? []).map((entry) => [
      entry.name,
      entry.value
        ? resolveTemplateValue(entry.value, variables, cache, template.name)
        : /password|passwd|secret|token|api[_-]?key|access[_-]?key|private[_-]?key|credential/i.test(
              entry.name,
            )
          ? generateSecret()
          : '',
    ]),
  )
}
