import { MagnifyingGlass } from '@phosphor-icons/react'
import { useEffect, useMemo, useRef, useState, type KeyboardEvent as ReactKeyboardEvent, type ReactNode } from 'react'
import { Dialog as BaseDialog } from '@base-ui/react/dialog'
import { Kbd } from '../kbd'

export interface CommandPaletteItem {
  id: string
  label: string
  icon?: ReactNode
  keywords?: string[]
  group?: string
  shortcut?: string
  onSelect?: () => void
}

export interface CommandPaletteProps {
  items: CommandPaletteItem[]
  trigger?: ReactNode
  title?: ReactNode
  triggerClassName?: string
}

function CommandResult({ active, item, index, onChoose, onHover }: { active: boolean; item: CommandPaletteItem; index: number; onChoose: (item: CommandPaletteItem) => void; onHover: (index: number) => void }) {
  return <button aria-current={active ? 'true' : undefined} aria-selected={active} className={`flex min-h-12 w-full cursor-pointer items-center gap-3 rounded-[11px] border border-transparent px-2.5 text-left text-supporting outline-none transition-[background-color,border-color,color] duration-[var(--ely-duration-fast)] hover:bg-surface-2 hover:text-text-primary focus-visible:outline-2 focus-visible:outline-focus motion-reduce:transition-none ${active ? 'bg-action-soft text-action-strong' : 'text-text-secondary'}`} id={`command-${item.id}`} onClick={() => onChoose(item)} onMouseEnter={() => onHover(index)} role="option" type="button"><span className={`h-5 w-0.5 shrink-0 rounded-full ${active ? 'bg-action' : 'bg-transparent'}`} /><span className={`grid size-8 shrink-0 place-items-center rounded-lg transition-colors ${active ? 'bg-action/15 text-action-strong' : 'bg-surface-2 text-text-tertiary'}`}>{item.icon}</span><span className="min-w-0 flex-1 truncate">{item.label}</span>{item.shortcut ? <kbd className="shrink-0 font-technical text-log text-text-tertiary">{item.shortcut}</kbd> : null}</button>
}

export function CommandPalette({ items, trigger = <span>Search commands</span>, triggerClassName }: CommandPaletteProps) {
  const [open, setOpen] = useState(false)
  const [query, setQuery] = useState('')
  const [activeIndex, setActiveIndex] = useState(0)
  const inputRef = useRef<HTMLInputElement>(null)
  const filteredItems = useMemo(() => {
    const normalized = query.trim().toLowerCase()
    return normalized ? items.filter((item) => [item.label, ...(item.keywords ?? [])].join(' ').toLowerCase().includes(normalized)) : items
  }, [items, query])
  const groupedItems = useMemo(() => {
    const groups = new Map<string, CommandPaletteItem[]>()
    for (const item of filteredItems) {
      const group = item.group ?? ''
      groups.set(group, [...(groups.get(group) ?? []), item])
    }
    return [...groups.entries()]
  }, [filteredItems])

  useEffect(() => {
    const handleKeyDown = (event: globalThis.KeyboardEvent) => {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k') {
        event.preventDefault()
        setOpen((current) => !current)
      }
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [])

  useEffect(() => setActiveIndex((index) => Math.min(index, Math.max(filteredItems.length - 1, 0))), [filteredItems.length])

  useEffect(() => {
    if (open) inputRef.current?.focus()
  }, [open])

  const choose = (item: CommandPaletteItem) => {
    setOpen(false)
    setQuery('')
    item.onSelect?.()
  }

  const handleSearchKeyDown = (event: ReactKeyboardEvent<HTMLInputElement>) => {
    if (!filteredItems.length) return
    if (event.key === 'ArrowDown') {
      event.preventDefault()
      setActiveIndex((index) => (index + 1) % filteredItems.length)
    }
    if (event.key === 'ArrowUp') {
      event.preventDefault()
      setActiveIndex((index) => (index - 1 + filteredItems.length) % filteredItems.length)
    }
    if (event.key === 'Enter') {
      event.preventDefault()
      if (filteredItems[activeIndex]) choose(filteredItems[activeIndex])
    }
  }

  return <BaseDialog.Root onOpenChange={(nextOpen) => { setOpen(nextOpen); if (!nextOpen) { setQuery(''); setActiveIndex(0) } }} open={open}><BaseDialog.Trigger aria-label="Search commands" className={triggerClassName ?? 'inline-flex min-h-9 cursor-pointer items-center justify-center rounded-lg px-3 text-label text-action-strong transition-colors hover:bg-action-soft focus-visible:outline-2 focus-visible:outline-focus'}>{trigger}</BaseDialog.Trigger><BaseDialog.Portal><BaseDialog.Backdrop className="ely-modal-backdrop" /><BaseDialog.Viewport className="ely-modal-viewport"><BaseDialog.Popup aria-label="Search commands" className="ely-modal-surface ely-modal-surface--command !flex !max-h-[min(70dvh,680px)] !w-[min(680px,calc(100vw-32px))] !max-w-[680px] !flex-col !gap-0 !overflow-hidden !rounded-[20px] !p-0"><div className="flex min-h-0 flex-col"><div className="flex h-14 shrink-0 items-center gap-3 px-4"><MagnifyingGlass aria-hidden="true" className="shrink-0 text-text-tertiary" size={19} /><input ref={inputRef} aria-activedescendant={filteredItems[activeIndex] ? `command-${filteredItems[activeIndex].id}` : undefined} aria-controls="command-palette-results" aria-label="Search commands" aria-autocomplete="list" aria-expanded="true" className="min-w-0 flex-1 bg-transparent text-body text-text-primary outline-none placeholder:text-text-tertiary" onChange={(event) => { setQuery(event.target.value); setActiveIndex(0) }} onKeyDown={handleSearchKeyDown} placeholder="Search commands" role="combobox" value={query} /><span className="pointer-events-none"><Kbd>⌘ K</Kbd></span></div><div className="min-h-0 overflow-x-hidden overflow-y-auto px-3.5 pb-3.5 pt-3" id="command-palette-results" role="listbox" aria-label="Command results">{filteredItems.length ? <div className="grid gap-3">{groupedItems.map(([group, groupItems]) => <section className="grid gap-1" key={group || 'ungrouped'}>{group ? <h2 className="px-2 pb-1 text-label-caps text-text-tertiary">{group}</h2> : null}<div className="grid gap-0.5">{groupItems.map((item) => <CommandResult active={filteredItems.indexOf(item) === activeIndex} index={filteredItems.indexOf(item)} item={item} key={item.id} onChoose={choose} onHover={setActiveIndex} />)}</div></section>)}</div> : <div className="px-2 py-5 text-center"><span className="block text-supporting text-text-primary">No commands found</span><span className="mt-1 block text-label text-text-tertiary">Try another search or action.</span></div>}</div></div></BaseDialog.Popup></BaseDialog.Viewport></BaseDialog.Portal></BaseDialog.Root>
}
