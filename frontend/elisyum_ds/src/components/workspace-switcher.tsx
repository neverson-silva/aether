import { CaretDown, CaretUp, Pulse } from '@phosphor-icons/react'
import { useState, type ReactNode } from 'react'
import { DropdownMenu, type DropdownMenuItem } from './dropdown-menu'

export interface WorkspaceOption {
  id: string
  name: ReactNode
  detail?: ReactNode
}

export interface WorkspaceSwitcherProps {
  emptyLabel?: ReactNode
  icon?: ReactNode
  options: WorkspaceOption[]
  optionActions?: (option: WorkspaceOption) => ReactNode
  footer?: ReactNode
  onFooterSelect?: () => void
  title?: ReactNode
  value?: string
  onValueChange?: (value: string) => void
}

export function WorkspaceSwitcher({ emptyLabel, footer, icon, onFooterSelect, onValueChange, optionActions, options, title, value }: WorkspaceSwitcherProps) {
  const [open, setOpen] = useState(false)
  const current = options.find((option) => option.id === value)
  const label = current?.name || emptyLabel || 'Choose organization'
  const items: DropdownMenuItem[] = options.map((option) => ({
    label: <span className="flex min-w-52 items-center justify-between gap-4"><span className="flex min-w-0 items-center gap-2"><span className={`size-1.5 shrink-0 rounded-full ${option.id === value ? 'bg-action' : 'bg-transparent'}`} /> {optionActions || footer ? <span className="grid min-w-0 gap-0.5"><span className="truncate">{option.name}</span>{option.detail ? <span className="font-technical text-log text-text-tertiary">{option.detail}</span> : null}</span> : <span className="truncate">{option.name}</span>}</span>{optionActions || footer ? optionActions?.(option) : option.detail ? <span className="font-technical text-log text-text-tertiary">{option.detail}</span> : null}</span>,
    onSelect: () => onValueChange?.(option.id),
  }))
  return <DropdownMenu className="min-w-64 !rounded-xl !border-border-subtle !bg-surface-1/90 p-2 backdrop-blur-xl" footer={footer} onFooterSelect={onFooterSelect} onOpenChange={setOpen} open={open} title={title ?? 'Organizations'} items={items} triggerClassName="inline-flex min-h-10 max-w-72 cursor-pointer items-center rounded-xl px-2.5 text-text-primary transition-[background-color,color,transform] duration-[var(--ely-duration-fast)] ease-ely-out hover:bg-surface-2 active:bg-surface-3 motion-safe:active:scale-[var(--ely-motion-press-scale)] focus-visible:outline-2 focus-visible:outline-focus" trigger={<span className="flex min-w-0 items-center gap-2.5"><span className="grid size-6 shrink-0 place-items-center rounded-md bg-action-soft text-action-strong">{icon ?? <Pulse size={14} weight="bold" />}</span><span className="truncate text-supporting font-medium">{label}</span>{open ? <CaretUp aria-hidden="true" className="shrink-0 text-text-tertiary" size={16} weight="bold" /> : <CaretDown aria-hidden="true" className="shrink-0 text-text-tertiary" size={16} weight="bold" />}</span>} />
}
