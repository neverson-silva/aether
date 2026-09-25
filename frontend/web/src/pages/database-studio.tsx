import {
  AlertDialog,
  Button,
  Card,
  CodeEditorLite,
  EmptyState,
  Field,
  Input,
  Modal,
  Select,
  Skeleton,
  showToast,
  Typography,
} from '@aether/elisyum-ds'
import {
  ArrowLeft,
  ArrowsClockwise,
  BracketsCurly,
  CaretDown,
  CaretRight,
  CaretUp,
  Check,
  Database,
  List,
  MagnifyingGlass,
  PencilSimple,
  Play,
  Plus,
  Table as TableIcon,
  Trash,
  X,
} from '@phosphor-icons/react'
import { useQueryClient } from '@tanstack/react-query'
import { useEffect, useRef, useState } from 'react'
import type {
  StudioExecResult,
  StudioObject,
  StudioQueryResult,
  StudioTableDetail,
} from '../api/types'
import { useDatabaseDetail } from '../hooks/use-database-detail'
import type { CreateTableColumn } from '../hooks/use-studio-create-table'
import { useStudioCreateTable } from '../hooks/use-studio-create-table'
import { useStudioExec } from '../hooks/use-studio-exec'
import { useStudioMeta } from '../hooks/use-studio-meta'
import { useStudioObjects } from '../hooks/use-studio-objects'
import { useStudioQuery } from '../hooks/use-studio-query'
import { useStudioSchemas } from '../hooks/use-studio-schemas'
import { useStudioTable } from '../hooks/use-studio-table'
import {
  useStudioAlterTable,
  useStudioDropTable,
  useStudioRenameTable,
} from '../hooks/use-studio-table-ops'

type QueryTab = {
  id: string
  title: string
  sql: string
  active: boolean
  result?: StudioQueryResult
  execution?: StudioExecResult
  output: 'results' | 'messages' | 'plan'
}
type Selection = { schema: string; table: string; type: string }
const writeStatement =
  /^\s*(insert|update|delete|drop|create|alter|truncate|replace|grant|revoke|call|exec|execute|commit|rollback|set|begin)\b/i
const persistenceKey = (id: string) => `aether_studio_tabs_${id}`
const splitPersistenceKey = (id: string) => `aether_studio_split_${id}`
const defaultEditorRatio = 0.42

function clampEditorRatio(ratio: number, height: number) {
  const usableHeight = height - 8
  const minimum = Math.max(0.2, 180 / usableHeight)
  const maximum = Math.min(0.8, 1 - 200 / usableHeight)
  return minimum <= maximum
    ? Math.min(maximum, Math.max(minimum, ratio))
    : defaultEditorRatio
}

function newQueryTab(index: number): QueryTab {
  return {
    id: `query-${Date.now()}-${index}`,
    title: `query_${String(index).padStart(2, '0')}.sql`,
    sql: '',
    active: false,
    output: 'results',
  }
}

function getStatementAtCursor(sql: string, offset: number) {
  let start = 0
  let quote = ''
  for (let index = 0; index < sql.length; index += 1) {
    const character = sql[index]
    if (quote) {
      if (character === quote && sql[index + 1] === quote && quote === "'") index += 1
      else if (character === quote) quote = ''
    } else if (character === "'" || character === '"' || character === '`')
      quote = character
    else if (character === ';') {
      const end = index + 1
      if (offset >= start && offset <= end) return sql.slice(start, end).trim()
      start = end
    }
  }
  return sql.slice(start).trim() || sql.trim()
}

function quoteIdentifier(engine: string, value: string) {
  if (engine === 'mysql' || engine === 'mariadb')
    return `\`${value.replace(/`/g, '``')}\``
  if (engine === 'mssql') return `[${value.replace(/]/g, ']]')}]`
  return `"${value.replace(/"/g, '""')}"`
}

function selectStatement(engine: string, selection: Selection) {
  if (engine === 'mongodb') return `db.${selection.table}.find().limit(200).toArray();`
  if (engine === 'redis') return 'SCAN 0 COUNT 200'
  const table = quoteIdentifier(engine, selection.table)
  if (engine === 'mysql' || engine === 'mariadb')
    return `SELECT * FROM ${table} LIMIT 200;`
  return `SELECT * FROM ${quoteIdentifier(engine, selection.schema)}.${table} LIMIT 200;`
}

export function DatabaseStudio({
  databaseId,
  returnTo,
}: {
  databaseId: string
  returnTo: string
}) {
  const queryClient = useQueryClient()
  const detail = useDatabaseDetail(databaseId)
  const database = detail.data?.database
  const schemas = useStudioSchemas(databaseId)
  const meta = useStudioMeta(databaseId)
  const queryMutation = useStudioQuery(databaseId)
  const execMutation = useStudioExec(databaseId)
  const createTable = useStudioCreateTable(databaseId)
  const renameTable = useStudioRenameTable(databaseId)
  const dropTable = useStudioDropTable(databaseId)
  const alterTable = useStudioAlterTable(databaseId)
  const [tabs, setTabs] = useState<QueryTab[]>(() => {
    try {
      const saved = localStorage.getItem(persistenceKey(databaseId))
      if (saved) {
        const restored = JSON.parse(saved) as QueryTab[]
        if (restored.length)
          return restored.map((tab, index) => ({
            ...tab,
            id: tab.id || `query-restored-${index}`,
            active: index === restored.length - 1,
            result: undefined,
            execution: undefined,
            output: 'results',
          }))
      }
    } catch {}
    return [{ ...newQueryTab(1), active: true }]
  })
  const [counter, setCounter] = useState(tabs.length + 1)
  const [selection, setSelection] = useState<Selection | null>(null)
  const [search, setSearch] = useState('')
  const [running, setRunning] = useState(false)
  const [createOpen, setCreateOpen] = useState(false)
  const [editSelection, setEditSelection] = useState<Selection | null>(null)
  const [renameSelection, setRenameSelection] = useState<Selection | null>(null)
  const [dropSelection, setDropSelection] = useState<Selection | null>(null)
  const [mobileExplorerOpen, setMobileExplorerOpen] = useState(false)
  const [renameValue, setRenameValue] = useState('')
  const [busy, setBusy] = useState(false)
  const [expanded, setExpanded] = useState<Record<string, boolean>>({})
  const [editorRatio, setEditorRatio] = useState(() => {
    try {
      const saved = Number(localStorage.getItem(splitPersistenceKey(databaseId)))
      return Number.isFinite(saved) && saved > 0 ? saved : defaultEditorRatio
    } catch {
      return defaultEditorRatio
    }
  })
  const [resultsOpen, setResultsOpen] = useState(true)
  const [isResizing, setIsResizing] = useState(false)
  const editorRef = useRef<HTMLTextAreaElement>(null)
  const workspaceRef = useRef<HTMLDivElement>(null)
  const splitDragRef = useRef<{
    pointerId: number
    startY: number
    startRatio: number
    height: number
  } | null>(null)
  const activeTab = tabs.find((tab) => tab.active) ?? tabs[0]
  const activeDetail = useStudioTable(
    databaseId,
    selection?.schema ?? '',
    selection?.table ?? '',
    Boolean(selection),
  )
  const engine = database?.engine ?? meta.data?.engine ?? 'postgres'
  const schemaList = schemas.data ?? []

  useEffect(() => {
    const persisted = tabs.map(({ id, title, sql, active, output }) => ({
      id,
      title,
      sql,
      active,
      output,
    }))
    localStorage.setItem(persistenceKey(databaseId), JSON.stringify(persisted))
  }, [databaseId, tabs])

  useEffect(() => {
    try {
      localStorage.setItem(splitPersistenceKey(databaseId), String(editorRatio))
    } catch {}
  }, [databaseId, editorRatio])

  const invalidateObjects = async () => {
    await queryClient.invalidateQueries({ queryKey: ['studio', databaseId] })
  }

  const addTab = (
    title = `query_${String(counter).padStart(2, '0')}.sql`,
    sql = '',
  ) => {
    const tab = { ...newQueryTab(counter), title, sql, active: true }
    setCounter((value) => value + 1)
    setTabs((current) => [...current.map((item) => ({ ...item, active: false })), tab])
    return tab
  }

  const run = async (sqlOverride?: string, targetTabId = activeTab.id) => {
    const sql =
      sqlOverride ??
      getStatementAtCursor(
        activeTab.sql,
        editorRef.current?.selectionStart ?? activeTab.sql.length,
      )
    if (!sql.trim()) return
    setRunning(true)
    setTabs((current) =>
      current.map((tab) =>
        tab.id === targetTabId
          ? { ...tab, result: undefined, execution: undefined }
          : tab,
      ),
    )
    try {
      if (writeStatement.test(sql)) {
        const result = await execMutation.mutateAsync(sql)
        setTabs((current) =>
          current.map((tab) =>
            tab.id === targetTabId ? { ...tab, execution: result } : tab,
          ),
        )
        setResultsOpen(true)
        if (/^\s*(create|alter|drop|truncate)\b/i.test(sql)) await invalidateObjects()
      } else {
        const result = await queryMutation.mutateAsync(sql)
        setTabs((current) =>
          current.map((tab) => (tab.id === targetTabId ? { ...tab, result } : tab)),
        )
        setResultsOpen(true)
      }
    } catch (error) {
      const message =
        error instanceof Error ? error.message : 'The query could not be executed.'
      setTabs((current) =>
        current.map((tab) =>
          tab.id === targetTabId
            ? { ...tab, execution: { message, command_tag: 'ERROR', duration_ms: 0 } }
            : tab,
        ),
      )
    } finally {
      setRunning(false)
    }
  }

  const selectObject = (schema: string, object: StudioObject) => {
    const next = { schema, table: object.name, type: object.type }
    setSelection(next)
    const sql = selectStatement(engine, next)
    const tab = addTab(`${object.name}.sql`, sql)
    void run(sql, tab.id)
  }

  const updateTab = (patch: Partial<QueryTab>) =>
    setTabs((current) =>
      current.map((tab) => (tab.id === activeTab.id ? { ...tab, ...patch } : tab)),
    )
  const closeTab = (id: string) =>
    setTabs((current) => {
      const index = current.findIndex((tab) => tab.id === id)
      const remaining = current.filter((tab) => tab.id !== id)
      if (!remaining.length) return [{ ...newQueryTab(1), active: true }]
      if (current[index]?.active)
        remaining[Math.min(index, remaining.length - 1)].active = true
      return remaining
    })

  const create = async (schema: string, name: string, columns: CreateTableColumn[]) => {
    try {
      await createTable.mutateAsync({ schema, table: name, columns })
      await invalidateObjects()
      setCreateOpen(false)
      showToast('Table created.', 'success')
    } catch (error) {
      showToast(
        error instanceof Error ? error.message : 'Could not create table.',
        'error',
      )
      throw error
    }
  }

  const performRename = async () => {
    if (!(renameSelection && renameValue.trim())) return
    setBusy(true)
    try {
      await renameTable.mutateAsync({
        schema: renameSelection.schema,
        table: renameSelection.table,
        name: renameValue.trim(),
      })
      await invalidateObjects()
      setRenameSelection(null)
      showToast('Database object renamed.', 'success')
    } catch (error) {
      showToast(
        error instanceof Error ? error.message : 'Could not rename this object.',
        'error',
      )
    } finally {
      setBusy(false)
    }
  }

  const performDrop = async () => {
    if (!dropSelection) return
    setBusy(true)
    try {
      await dropTable.mutateAsync({
        schema: dropSelection.schema,
        table: dropSelection.table,
      })
      await invalidateObjects()
      if (
        selection?.table === dropSelection.table &&
        selection.schema === dropSelection.schema
      )
        setSelection(null)
      setDropSelection(null)
      showToast('Database object removed.', 'success')
    } catch (error) {
      showToast(
        error instanceof Error ? error.message : 'Could not remove this object.',
        'error',
      )
    } finally {
      setBusy(false)
    }
  }

  const saveColumns = async (columns: CreateTableColumn[]) => {
    if (!editSelection) return
    setBusy(true)
    try {
      await alterTable.mutateAsync({
        schema: editSelection.schema,
        table: editSelection.table,
        columns,
      })
      await invalidateObjects()
      setEditSelection(null)
      showToast('Table structure updated.', 'success')
    } catch (error) {
      showToast(
        error instanceof Error
          ? error.message
          : 'Could not update the table structure.',
        'error',
      )
    } finally {
      setBusy(false)
    }
  }

  if (detail.isLoading || schemas.isLoading)
    return (
      <main className="flex h-dvh min-w-0 flex-col bg-canvas text-text-primary">
        <StudioTopBar
          databaseName="Loading database"
          returnTo={returnTo}
        />
        <div className="grid flex-1 content-start gap-4 p-4 sm:p-6">
          <Skeleton className="h-12 rounded-xl" />
          <Skeleton className="h-[38rem] rounded-2xl" />
        </div>
      </main>
    )
  if (detail.isError || schemas.isError || !database)
    return (
      <main className="flex h-dvh min-w-0 flex-col bg-canvas p-4 text-text-primary sm:p-6">
        <StudioTopBar
          databaseName="Database Studio"
          returnTo={returnTo}
        />
        <div className="grid flex-1 place-items-center">
          <EmptyState
            title="Database Studio unavailable"
            description="The database structure could not be loaded. Verify that the database is running and try again."
            action={
              <Button
                tone="neutral"
                onClick={() => {
                  void detail.refetch()
                  void schemas.refetch()
                }}
              >
                Retry
              </Button>
            }
          />
        </div>
      </main>
    )

  const shownSchemas = schemaList.filter((schema) =>
    schema.toLowerCase().includes(search.toLowerCase()),
  )

  return (
    <main className="flex h-dvh min-w-0 flex-col overflow-hidden bg-canvas text-text-primary">
      <StudioTopBar
        databaseName={database.name}
        returnTo={returnTo}
      />
      <section className="grid min-h-0 min-w-0 flex-1 overflow-hidden bg-surface-1 xl:grid-cols-[17rem_minmax(0,1fr)]">
        <aside className="hidden min-w-0 border-r border-border-subtle bg-surface-2 xl:flex xl:flex-col">
          <div className="grid gap-3 border-b border-border-subtle p-4">
            <div className="flex items-center justify-between">
              <span className="flex items-center gap-2 text-label font-medium text-text-primary">
                <Database size={16} />
                Object explorer
              </span>
              <Button
                aria-label="Refresh schema"
                size="sm"
                tone="ghost"
                onClick={() => {
                  void schemas.refetch()
                  void invalidateObjects()
                }}
              >
                <ArrowsClockwise size={15} />
              </Button>
            </div>
            <label className="flex items-center gap-2 rounded-lg border border-border-subtle bg-field px-3">
              <MagnifyingGlass
                className="text-text-tertiary"
                size={15}
              />
              <input
                aria-label="Filter schemas"
                className="min-w-0 flex-1 bg-transparent py-2 text-label text-text-primary outline-none placeholder:text-text-tertiary"
                onChange={(event) => setSearch(event.target.value)}
                placeholder="Filter schemas"
                value={search}
              />
            </label>
          </div>
          <div className="min-h-0 flex-1 overflow-y-auto p-2">
            {shownSchemas.map((schema) => (
              <SchemaTree
                key={schema}
                databaseId={databaseId}
                schema={schema}
                expanded={Boolean(expanded[schema])}
                onToggle={() =>
                  setExpanded((current) => ({ ...current, [schema]: !current[schema] }))
                }
                selected={selection}
                onSelect={selectObject}
                onRename={(object) => {
                  setRenameSelection({ schema, table: object.name, type: object.type })
                  setRenameValue(object.name)
                }}
                onDrop={(object) =>
                  setDropSelection({ schema, table: object.name, type: object.type })
                }
                onEdit={(object) =>
                  setEditSelection({ schema, table: object.name, type: object.type })
                }
              />
            ))}
            {selection ? (
              <StudioObjectDetails
                detail={activeDetail.data}
                loading={activeDetail.isLoading}
                selection={selection}
              />
            ) : null}
          </div>
          <div className="grid gap-2 border-t border-border-subtle p-3">
            <Button
              tone="neutral"
              onClick={() => setCreateOpen(true)}
            >
              <Plus size={16} />
              Create table
            </Button>
            <div className="font-technical text-log text-text-subtle">
              {meta.data
                ? `${meta.data.engine} ${meta.data.version} · ${meta.data.tables} tables`
                : database.name}
            </div>
          </div>
        </aside>
        <div className="flex min-h-0 min-w-0 flex-col overflow-hidden">
          <header className="flex flex-wrap items-center justify-between gap-3 border-b border-border-subtle bg-surface-2 px-4 py-3">
            <div className="min-w-0">
              <div className="flex items-center gap-2 text-label text-text-tertiary">
                <Database size={15} />
                <span className="truncate">{database.name}</span>
                {selection ? (
                  <>
                    <CaretRight size={13} />
                    <span>{selection.schema}</span>
                    <CaretRight size={13} />
                    <span className="truncate text-text-primary">
                      {selection.table}
                    </span>
                  </>
                ) : null}
              </div>
              <div className="mt-1 font-technical text-log text-text-subtle">
                {meta.data
                  ? `${meta.data.engine} ${meta.data.version} · ${meta.data.status}`
                  : 'SQL workspace'}
              </div>
            </div>
            <div className="flex flex-wrap gap-2">
              <Button
                className="xl:hidden"
                tone="neutral"
                onClick={() => setMobileExplorerOpen(true)}
              >
                <List size={15} />
                Objects
              </Button>
              <Button
                className="xl:hidden"
                tone="neutral"
                onClick={() => setCreateOpen(true)}
              >
                <Plus size={15} />
                Create
              </Button>
              <Button
                disabled={running || !activeTab.sql.trim()}
                loading={running}
                onClick={() => void run()}
              >
                <Play size={15} />
                Run query
              </Button>
            </div>
          </header>
          <div
            ref={workspaceRef}
            className="grid min-h-0 flex-1"
            style={{
              gridTemplateRows: resultsOpen
                ? `minmax(11.25rem, ${editorRatio}fr) 8px minmax(12.5rem, ${1 - editorRatio}fr)`
                : 'minmax(0, 1fr) 8px 2.5rem',
            }}
          >
            <div className="flex min-h-0 min-w-0 flex-col overflow-hidden">
              <div
                className="flex h-10 shrink-0 items-stretch overflow-x-auto border-b border-border-subtle bg-surface-2"
                role="tablist"
                aria-label="SQL query tabs"
              >
                {tabs.map((tab) => (
                  <div
                    className={`flex shrink-0 items-center border-r border-border-subtle ${tab.active ? 'bg-surface-1' : ''}`}
                    key={tab.id}
                  >
                    <button
                      aria-selected={tab.active}
                      className={`border-t-2 px-3 font-technical text-log ${tab.active ? 'border-action text-action-strong' : 'border-transparent text-text-tertiary hover:text-text-primary'}`}
                      onClick={() =>
                        setTabs((current) =>
                          current.map((item) => ({
                            ...item,
                            active: item.id === tab.id,
                          })),
                        )
                      }
                      role="tab"
                      type="button"
                    >
                      {tab.title}
                    </button>
                    <button
                      aria-label={`Close ${tab.title}`}
                      className="px-2 text-text-tertiary hover:text-text-primary"
                      onClick={() => closeTab(tab.id)}
                      type="button"
                    >
                      <X size={13} />
                    </button>
                  </div>
                ))}
                <button
                  aria-label="New query tab"
                  className="px-3 text-text-tertiary hover:text-action-strong"
                  onClick={() => addTab()}
                  type="button"
                >
                  <Plus size={15} />
                </button>
              </div>
              <div className="min-h-0 min-w-0 flex-1 overflow-hidden [&>div]:!grid [&>div]:!h-full [&>div]:!min-h-0 [&>div]:!grid-rows-[minmax(0,1fr)] [&>div>div:first-child]:hidden">
                <CodeEditorLite
                  ref={editorRef}
                  aria-label="SQL query editor"
                  className="!h-full !min-h-0 resize-none"
                  language={
                    engine === 'mongodb'
                      ? 'javascript'
                      : engine === 'redis'
                        ? 'shell'
                        : 'sql'
                  }
                  onChange={(event) => updateTab({ sql: event.target.value })}
                  onKeyDown={(event) => {
                    if ((event.metaKey || event.ctrlKey) && event.key === 'Enter') {
                      event.preventDefault()
                      void run()
                    }
                  }}
                  placeholder="Write a query. Use ⌘/Ctrl + Enter to run."
                  value={activeTab.sql}
                />
              </div>
            </div>
            <div
              aria-label="Resize editor and results panels"
              aria-orientation="horizontal"
              aria-valuemax={80}
              aria-valuemin={20}
              aria-valuenow={Math.round(editorRatio * 100)}
              className={`group relative z-10 flex cursor-row-resize touch-none items-center justify-center focus-visible:outline-2 focus-visible:outline-focus ${isResizing ? 'bg-action-soft/25' : 'bg-surface-1 hover:bg-action-soft/20'}`}
              onDoubleClick={() => {
                setEditorRatio(defaultEditorRatio)
                setResultsOpen(true)
              }}
              onKeyDown={(event) => {
                if (!workspaceRef.current) return
                const height = workspaceRef.current.clientHeight
                const step = event.shiftKey ? 0.05 : 0.02
                const nextRatio =
                  event.key === 'ArrowUp'
                    ? editorRatio - step
                    : event.key === 'ArrowDown'
                      ? editorRatio + step
                      : event.key === 'Home'
                        ? 0.2
                        : event.key === 'End'
                          ? 0.8
                          : null
                if (nextRatio !== null) {
                  event.preventDefault()
                  setEditorRatio(clampEditorRatio(nextRatio, height))
                  setResultsOpen(true)
                }
              }}
              onPointerCancel={() => {
                splitDragRef.current = null
                setIsResizing(false)
              }}
              onPointerDown={(event) => {
                if (!(resultsOpen && workspaceRef.current)) return
                splitDragRef.current = {
                  pointerId: event.pointerId,
                  startY: event.clientY,
                  startRatio: editorRatio,
                  height: workspaceRef.current.clientHeight,
                }
                event.currentTarget.setPointerCapture(event.pointerId)
                setIsResizing(true)
              }}
              onPointerMove={(event) => {
                const drag = splitDragRef.current
                if (!drag || drag.pointerId !== event.pointerId) return
                setEditorRatio(
                  clampEditorRatio(
                    drag.startRatio + (event.clientY - drag.startY) / drag.height,
                    drag.height,
                  ),
                )
              }}
              onPointerUp={() => {
                splitDragRef.current = null
                setIsResizing(false)
              }}
              role="separator"
              tabIndex={0}
            >
              <span className="h-0.5 w-10 rounded-full bg-border-default transition-colors group-hover:bg-action group-focus-visible:bg-action" />
            </div>
            <section
              aria-label="Query results"
              className="flex min-h-0 min-w-0 flex-col overflow-hidden"
            >
              <div
                className="flex min-h-10 shrink-0 flex-wrap items-center border-b border-border-subtle bg-surface-2"
                role="tablist"
                aria-label="Query output"
              >
                {(['results', 'messages', 'plan'] as const).map((output) => (
                  <button
                    aria-selected={activeTab.output === output}
                    className={`border-b-2 px-4 py-2 text-label ${activeTab.output === output ? 'border-action text-action-strong' : 'border-transparent text-text-tertiary hover:text-text-primary'}`}
                    key={output}
                    onClick={() => updateTab({ output })}
                    role="tab"
                    type="button"
                  >
                    {output === 'plan'
                      ? 'Execution plan'
                      : output[0].toUpperCase() + output.slice(1)}
                  </button>
                ))}
                {activeTab.result ? (
                  <span className="ml-auto px-3 font-technical text-log text-text-tertiary">
                    {activeTab.result.row_count} rows · {activeTab.result.duration_ms}{' '}
                    ms{activeTab.result.truncated ? ' · truncated' : ''}
                  </span>
                ) : null}
                <button
                  aria-label={resultsOpen ? 'Collapse results' : 'Expand results'}
                  aria-expanded={resultsOpen}
                  className="ml-auto grid size-9 shrink-0 place-items-center text-text-tertiary hover:text-text-primary focus-visible:outline-2 focus-visible:outline-focus"
                  onClick={() => setResultsOpen((open) => !open)}
                  title={resultsOpen ? 'Collapse results' : 'Expand results'}
                  type="button"
                >
                  {resultsOpen ? <CaretDown size={15} /> : <CaretUp size={15} />}
                </button>
              </div>
              {resultsOpen ? (
                <div className="min-h-0 flex-1 overflow-auto">
                  {activeTab.output === 'results' ? (
                    <StudioResults
                      result={activeTab.result}
                      execution={activeTab.execution}
                      error={activeTab.result?.error?.message}
                    />
                  ) : activeTab.output === 'messages' ? (
                    <div className="p-4 font-technical text-log text-text-secondary">
                      {activeTab.execution
                        ? `${activeTab.execution.command_tag || 'OK'} — ${activeTab.execution.message} (${activeTab.execution.duration_ms} ms)`
                        : (activeTab.result?.message ??
                          'Run a query to see execution messages.')}
                    </div>
                  ) : (
                    <div className="p-4 text-supporting text-text-tertiary">
                      Execution plan is not returned by this database runtime. Prefix a
                      supported query with EXPLAIN to inspect its plan in Results.
                    </div>
                  )}
                </div>
              ) : null}
            </section>
          </div>
        </div>
        {mobileExplorerOpen ? (
          <Modal
            open
            trigger={<span />}
            triggerClassName="hidden"
            title="Database objects"
            description="Browse schemas and open tables in the query editor."
            onOpenChange={(open) => !open && setMobileExplorerOpen(false)}
            popupClassName="max-h-[85dvh] overflow-y-auto"
          >
            <div className="grid gap-2">
              {shownSchemas.map((schema) => (
                <SchemaTree
                  key={schema}
                  databaseId={databaseId}
                  schema={schema}
                  expanded={Boolean(expanded[schema])}
                  onToggle={() =>
                    setExpanded((current) => ({
                      ...current,
                      [schema]: !current[schema],
                    }))
                  }
                  selected={selection}
                  onSelect={(name, object) => {
                    selectObject(name, object)
                    setMobileExplorerOpen(false)
                  }}
                  onRename={(object) => {
                    setRenameSelection({
                      schema,
                      table: object.name,
                      type: object.type,
                    })
                    setRenameValue(object.name)
                    setMobileExplorerOpen(false)
                  }}
                  onDrop={(object) => {
                    setDropSelection({ schema, table: object.name, type: object.type })
                    setMobileExplorerOpen(false)
                  }}
                  onEdit={(object) => {
                    setEditSelection({ schema, table: object.name, type: object.type })
                    setMobileExplorerOpen(false)
                  }}
                />
              ))}
            </div>
          </Modal>
        ) : null}
        {createOpen ? (
          <TableEditorDialog
            databaseId={databaseId}
            schemas={schemaList}
            onClose={() => setCreateOpen(false)}
            onSave={create}
          />
        ) : null}
        {editSelection ? (
          <TableEditorDialog
            databaseId={databaseId}
            schemas={schemaList}
            table={editSelection.table}
            schema={editSelection.schema}
            onClose={() => setEditSelection(null)}
            onSave={async (_schema, _name, columns) => saveColumns(columns)}
          />
        ) : null}
        {renameSelection ? (
          <Modal
            open
            trigger={<span />}
            triggerClassName="hidden"
            title="Rename database object"
            description={`Rename ${renameSelection.schema}.${renameSelection.table}.`}
            onOpenChange={(open) => !open && setRenameSelection(null)}
            footer={
              <div className="flex gap-2">
                <Button
                  tone="ghost"
                  onClick={() => setRenameSelection(null)}
                >
                  Cancel
                </Button>
                <Button
                  disabled={
                    !renameValue.trim() || renameValue === renameSelection.table || busy
                  }
                  loading={busy}
                  onClick={() => void performRename()}
                >
                  Rename
                </Button>
              </div>
            }
          >
            <Field
              id="studio-rename"
              label="New name"
              required
            >
              <Input
                autoFocus
                value={renameValue}
                onChange={(event) => setRenameValue(event.target.value)}
              />
            </Field>
          </Modal>
        ) : null}
        {dropSelection ? (
          <AlertDialog
            open
            onOpenChange={(open) => !open && setDropSelection(null)}
            title="Drop database object?"
            description={`This permanently removes ${dropSelection.schema}.${dropSelection.table} and its data. This operation cannot be undone.`}
            confirmLabel="Drop object"
            onConfirm={() => void performDrop()}
          />
        ) : null}
      </section>
    </main>
  )
}

function StudioTopBar({
  databaseName,
  returnTo,
}: {
  databaseName: string
  returnTo: string
}) {
  return (
    <header className="flex h-14 shrink-0 items-center justify-between gap-4 border-b border-border-subtle bg-surface-1 px-3 sm:px-5">
      <Button
        className="shrink-0"
        tone="ghost"
        onClick={() => window.location.assign(returnTo)}
      >
        <ArrowLeft size={17} />
        Back to service
      </Button>
      <div className="flex min-w-0 items-center gap-2 text-supporting">
        <Database
          className="shrink-0 text-action-strong"
          size={18}
        />
        <span className="font-medium text-text-primary">Database Studio</span>
        <span className="text-text-subtle">/</span>
        <span className="truncate text-text-secondary">{databaseName}</span>
      </div>
    </header>
  )
}

function SchemaTree({
  databaseId,
  schema,
  expanded,
  onToggle,
  selected,
  onSelect,
  onRename,
  onDrop,
  onEdit,
}: {
  databaseId: string
  schema: string
  expanded: boolean
  onToggle: () => void
  selected: Selection | null
  onSelect: (schema: string, object: StudioObject) => void
  onRename: (object: StudioObject) => void
  onDrop: (object: StudioObject) => void
  onEdit: (object: StudioObject) => void
}) {
  const objects = useStudioObjects(databaseId, schema)
  const filtered = objects.data ?? []
  return (
    <div className="mb-1">
      <button
        className="flex w-full items-center gap-2 rounded-lg px-2 py-2 text-left text-label text-text-secondary hover:bg-surface-3 hover:text-text-primary"
        onClick={onToggle}
        type="button"
      >
        {expanded ? <CaretDown size={14} /> : <CaretRight size={14} />}
        <BracketsCurly
          size={15}
          className="text-action-strong"
        />
        {schema}
        <span className="ml-auto font-technical text-log text-text-subtle">
          {filtered.length || ''}
        </span>
      </button>
      {expanded ? (
        <div className="ml-4 border-l border-border-subtle pl-2">
          {objects.isLoading ? (
            <Skeleton className="my-2 h-8 rounded-lg" />
          ) : objects.isError ? (
            <button
              className="px-2 py-2 text-label text-danger-strong"
              onClick={() => void objects.refetch()}
              type="button"
            >
              Could not load objects · Retry
            </button>
          ) : filtered.length ? (
            filtered.map((object) => (
              <div
                className={`group flex items-center gap-1 rounded-lg ${selected?.schema === schema && selected.table === object.name ? 'bg-action-soft text-action-strong' : 'text-text-secondary hover:bg-surface-3'}`}
                key={`${object.type}:${object.name}`}
              >
                <button
                  className="flex min-w-0 flex-1 items-center gap-2 px-2 py-2 text-left text-label"
                  onClick={() => onSelect(schema, object)}
                  type="button"
                >
                  <TableIcon
                    className="shrink-0"
                    size={14}
                  />
                  <span className="truncate">{object.name}</span>
                  <span className="ml-auto shrink-0 text-[0.65rem] text-text-subtle">
                    {object.type}
                  </span>
                </button>
                {object.type === 'table' ? (
                  <button
                    aria-label={`Edit ${object.name}`}
                    className="p-1 text-text-tertiary opacity-0 hover:text-action-strong focus:opacity-100 group-hover:opacity-100"
                    onClick={() => onEdit(object)}
                    type="button"
                  >
                    <PencilSimple size={13} />
                  </button>
                ) : null}
                <button
                  aria-label={`Rename ${object.name}`}
                  className="p-1 text-text-tertiary opacity-0 hover:text-action-strong focus:opacity-100 group-hover:opacity-100"
                  onClick={() => onRename(object)}
                  type="button"
                >
                  <PencilSimple size={13} />
                </button>
                <button
                  aria-label={`Drop ${object.name}`}
                  className="mr-1 p-1 text-text-tertiary opacity-0 hover:text-danger-strong focus:opacity-100 group-hover:opacity-100"
                  onClick={() => onDrop(object)}
                  type="button"
                >
                  <Trash size={13} />
                </button>
              </div>
            ))
          ) : (
            <p className="px-2 py-2 text-label text-text-tertiary">No objects</p>
          )}
        </div>
      ) : null}
    </div>
  )
}

function StudioObjectDetails({
  detail,
  loading,
  selection,
}: {
  detail?: StudioTableDetail
  loading: boolean
  selection: Selection
}) {
  if (loading)
    return (
      <div className="grid gap-2 border-t border-border-subtle p-3">
        <Skeleton className="h-5 rounded" />
        <Skeleton className="h-8 rounded" />
      </div>
    )
  if (!detail) return null
  const columns = detail.columns ?? []
  const indexes = detail.indexes ?? []
  const constraints = detail.constraints ?? []
  const foreignKeys = detail.foreign_keys ?? []
  const triggers = detail.triggers ?? []
  const properties = [
    {
      label: 'Columns',
      rows: columns.map((column) => ({
        name: column.name,
        value: `${column.type}${column.primary_key ? ' · primary key' : ''}${column.nullable ? ' · nullable' : ''}${column.default ? ` · default ${column.default}` : ''}`,
      })),
    },
    {
      label: 'Indexes',
      rows: indexes.map((index) => ({
        name: index.name,
        value: `${index.unique ? 'Unique · ' : ''}${index.method} (${(index.columns ?? []).join(', ')})`,
      })),
    },
    {
      label: 'Constraints',
      rows: constraints.map((constraint) => ({
        name: constraint.name,
        value: constraint.definition || constraint.type,
      })),
    },
    {
      label: 'Foreign keys',
      rows: foreignKeys.map((key) => ({
        name: key.name,
        value: `${(key.columns ?? []).join(', ')} → ${key.ref_table} (${(key.ref_columns ?? []).join(', ')})`,
      })),
    },
    {
      label: 'Triggers',
      rows: triggers.map((trigger) => ({
        name: trigger.name,
        value: `${trigger.timing} ${trigger.event} · ${trigger.function}`,
      })),
    },
  ].filter((group) => group.rows.length)
  return (
    <section
      className="grid gap-2 border-t border-border-subtle px-2 py-3"
      aria-label={`${selection.table} structure`}
    >
      <div className="flex items-center gap-2 px-2 text-label font-medium text-text-primary">
        <TableIcon size={14} />
        Structure
      </div>
      <div className="px-2 font-technical text-log text-text-subtle">
        {detail.owner || selection.schema} · {detail.type}
      </div>
      {properties.map((group) => (
        <details
          className="group rounded-lg"
          key={group.label}
        >
          <summary className="flex cursor-pointer list-none items-center justify-between rounded-lg px-2 py-2 text-label text-text-secondary hover:bg-surface-3">
            <span>{group.label}</span>
            <span className="font-technical text-log text-text-subtle">
              {group.rows.length}
            </span>
          </summary>
          <div className="grid gap-1 px-2 pb-2">
            {group.rows.map((row) => (
              <div
                className="grid gap-0.5 rounded-md bg-surface-1 px-2 py-1.5"
                key={row.name}
              >
                <span className="truncate text-label text-text-primary">
                  {row.name}
                </span>
                <span className="break-words font-technical text-[0.68rem] text-text-tertiary">
                  {row.value}
                </span>
              </div>
            ))}
          </div>
        </details>
      ))}
    </section>
  )
}

function StudioResults({
  result,
  execution,
  error,
}: {
  result?: StudioQueryResult
  execution?: StudioExecResult
  error?: string
}) {
  if (error || execution?.command_tag === 'ERROR')
    return (
      <div className="m-4 rounded-xl border border-danger/35 bg-danger-soft/20 p-4 font-technical text-log text-danger-strong">
        {error ?? execution?.message}
      </div>
    )
  if (execution)
    return (
      <div className="m-4 flex items-center gap-2 rounded-xl border border-success/30 bg-success-soft/20 p-4 font-technical text-log text-success-strong">
        <Check size={16} />
        {execution.command_tag || 'OK'} — {execution.message} · {execution.duration_ms}{' '}
        ms
      </div>
    )
  if (!result)
    return (
      <div className="grid min-h-40 place-items-center px-5 text-center text-supporting text-text-tertiary">
        Run a query or select a table from the explorer to inspect its rows.
      </div>
    )
  const columns = Array.isArray(result.columns) ? result.columns : []
  const rawRows = Array.isArray(result.rows) ? result.rows : []
  const rows = rawRows.map((row) =>
    Array.isArray(row)
      ? row
      : columns.map((column) =>
          row && typeof row === 'object'
            ? (row as Record<string, unknown>)[column]
            : row,
        ),
  )
  if (!columns.length)
    return (
      <div className="p-4 text-supporting text-text-secondary">
        {result.message ?? 'Query completed without returning rows.'}
      </div>
    )
  return (
    <div className="h-full min-h-0 overflow-auto">
      <table className="w-full border-collapse text-left text-label">
        <thead className="sticky top-0 bg-surface-2 text-text-secondary">
          <tr>
            <th className="border-b border-border-subtle px-3 py-2 font-normal">#</th>
            {columns.map((column, index) => (
              <th
                className="whitespace-nowrap border-b border-border-subtle px-3 py-2 font-normal"
                key={`${column}-${index}`}
              >
                {column}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((row, rowIndex) => (
            <tr
              className="hover:bg-surface-2"
              key={rowIndex}
            >
              <td className="border-b border-border-subtle px-3 py-2 font-technical text-text-subtle">
                {rowIndex + 1}
              </td>
              {columns.map((_, columnIndex) => {
                const value = row[columnIndex]
                return (
                  <td
                    className={`max-w-96 border-b border-border-subtle px-3 py-2 font-technical ${value == null ? 'italic text-text-subtle' : 'text-text-primary'}`}
                    key={columnIndex}
                  >
                    {value == null
                      ? 'NULL'
                      : typeof value === 'object'
                        ? JSON.stringify(value)
                        : String(value)}
                  </td>
                )
              })}
            </tr>
          ))}
        </tbody>
      </table>
      {rows.length === 0 ? (
        <p className="p-4 text-label text-text-tertiary">No rows returned.</p>
      ) : null}
    </div>
  )
}

function TableEditorDialog({
  databaseId,
  schemas,
  schema: initialSchema,
  table: initialTable,
  onClose,
  onSave,
}: {
  databaseId: string
  schemas: string[]
  schema?: string
  table?: string
  onClose: () => void
  onSave: (schema: string, table: string, columns: CreateTableColumn[]) => Promise<void>
}) {
  const isEdit = Boolean(initialTable && initialSchema)
  const detail = useStudioTable(
    databaseId,
    initialSchema ?? '',
    initialTable ?? '',
    isEdit,
  )
  const [schema, setSchema] = useState(initialSchema ?? schemas[0] ?? 'public')
  const [table, setTable] = useState(initialTable ?? '')
  const [columns, setColumns] = useState<CreateTableColumn[]>([
    { name: '', type: 'TEXT', nullable: true, primary: false },
  ])
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)
  useEffect(() => {
    if (detail.data?.columns)
      setColumns(
        detail.data.columns.map((column) => ({
          name: column.name,
          type: column.type,
          nullable: column.nullable,
          primary: column.primary_key,
          default: column.default ?? undefined,
        })),
      )
  }, [detail.data])
  const submit = async () => {
    setError('')
    if (
      !(table.trim() && columns.length) ||
      columns.some((column) => !(column.name.trim() && column.type.trim()))
    ) {
      setError('Enter a table name and complete every column.')
      return
    }
    setSaving(true)
    try {
      await onSave(schema, table.trim(), columns)
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Could not save this table.')
    } finally {
      setSaving(false)
    }
  }
  const updateColumn = (index: number, patch: Partial<CreateTableColumn>) =>
    setColumns((current) =>
      current.map((column, item) =>
        item === index ? { ...column, ...patch } : column,
      ),
    )
  return (
    <Modal
      open
      trigger={<span />}
      triggerClassName="hidden"
      title={isEdit ? 'Edit table structure' : 'Create database table'}
      description={
        isEdit
          ? `Update the columns in ${initialSchema}.${initialTable}.`
          : 'Define the table and its initial columns.'
      }
      onOpenChange={(open) => !open && onClose()}
      popupClassName="w-[min(56rem,calc(100vw-2rem))] max-h-[90dvh] overflow-y-auto"
      footer={
        <div className="flex gap-2">
          <Button
            tone="ghost"
            onClick={onClose}
          >
            Cancel
          </Button>
          <Button
            disabled={saving || !table.trim()}
            loading={saving}
            onClick={() => void submit()}
          >
            {isEdit ? 'Save structure' : 'Create table'}
          </Button>
        </div>
      }
    >
      {isEdit && detail.isLoading ? (
        <Skeleton className="h-48 rounded-xl" />
      ) : (
        <div className="grid gap-4">
          {error ? (
            <p
              className="text-supporting text-danger-strong"
              role="alert"
            >
              {error}
            </p>
          ) : null}
          <div className="grid gap-4 sm:grid-cols-[1fr_2fr]">
            {!isEdit ? (
              <Field
                id="studio-table-schema"
                label="Schema"
              >
                <Select
                  value={schema}
                  onChange={(event) => setSchema(event.target.value)}
                >
                  {schemas.map((item) => (
                    <option
                      key={item}
                      value={item}
                    >
                      {item}
                    </option>
                  ))}
                </Select>
              </Field>
            ) : (
              <Field
                id="studio-table-schema"
                label="Schema"
              >
                <Input
                  disabled
                  value={schema}
                />
              </Field>
            )}
            <Field
              id="studio-table-name"
              label="Table name"
              required
            >
              <Input
                disabled={isEdit}
                value={table}
                onChange={(event) => setTable(event.target.value)}
                placeholder="orders"
              />
            </Field>
          </div>
          <div className="grid gap-2">
            {columns.map((column, index) => (
              <div
                className="grid gap-2 rounded-xl border border-border-subtle bg-surface-2 p-3 sm:grid-cols-[minmax(8rem,1.2fr)_minmax(8rem,1fr)_auto_auto_auto] sm:items-end"
                key={`${column.name}-${index}`}
              >
                <Field
                  id={`studio-column-name-${index}`}
                  label={index === 0 ? 'Column' : `Column ${index + 1}`}
                >
                  <Input
                    value={column.name}
                    onChange={(event) =>
                      updateColumn(index, { name: event.target.value })
                    }
                    placeholder="column_name"
                  />
                </Field>
                <Field
                  id={`studio-column-type-${index}`}
                  label="Type"
                >
                  <Input
                    value={column.type}
                    onChange={(event) =>
                      updateColumn(index, { type: event.target.value })
                    }
                    placeholder="TEXT"
                  />
                </Field>
                <label className="flex items-center gap-2 pb-2 text-label text-text-secondary">
                  <input
                    checked={column.nullable}
                    onChange={(event) =>
                      updateColumn(index, { nullable: event.target.checked })
                    }
                    type="checkbox"
                  />
                  Nullable
                </label>
                <label className="flex items-center gap-2 pb-2 text-label text-text-secondary">
                  <input
                    checked={column.primary}
                    onChange={(event) =>
                      updateColumn(index, { primary: event.target.checked })
                    }
                    type="checkbox"
                  />
                  Primary
                </label>
                <Button
                  aria-label={`Remove column ${index + 1}`}
                  disabled={columns.length === 1}
                  size="sm"
                  tone="ghost"
                  onClick={() =>
                    setColumns((current) => current.filter((_, item) => item !== index))
                  }
                >
                  <Trash size={15} />
                </Button>
              </div>
            ))}
          </div>
          <Button
            className="w-fit"
            tone="neutral"
            onClick={() =>
              setColumns((current) => [
                ...current,
                { name: '', type: 'TEXT', nullable: true, primary: false },
              ])
            }
          >
            <Plus size={15} />
            Add column
          </Button>
        </div>
      )}
    </Modal>
  )
}
