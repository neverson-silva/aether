import { Eye, EyeSlash, Plus, TrashSimple } from '@phosphor-icons/react'
import { useState, type ReactNode } from 'react'
import type { FieldValues, UseFormRegister } from 'react-hook-form'
import { IconButton } from './icon-button'
import { Input } from './input'

export interface VariableDefinition {
  name: string
  description?: ReactNode
  secret?: boolean
}
export interface VariableEditorProps<T extends FieldValues> {
  variables: VariableDefinition[]
  register: UseFormRegister<T>
  className?: string
  onAdd?: (name: string) => void
  onRemove?: (name: string) => void
}

export function VariableEditor<T extends FieldValues>({
  className = '',
  register,
  variables,
  onAdd,
  onRemove,
}: VariableEditorProps<T>) {
  const [adding, setAdding] = useState(false)
  const [newName, setNewName] = useState('')
  const addVariable = () => {
    const normalized = newName.trim()
    if (!normalized || variables.some((variable) => variable.name === normalized)) return
    onAdd?.(normalized)
    setNewName('')
    setAdding(false)
  }
  return (
    <div className={`grid gap-3 ${className}`}>
      <div className="divide-y divide-border-subtle overflow-hidden rounded-xl border border-border-subtle">
        {variables.map((variable) => <VariableField key={variable.name} onRemove={onRemove} register={register} variable={variable} />)}
        {!variables.length ? <div className="px-4 py-6 text-center text-supporting text-text-tertiary">No variables configured yet.</div> : null}
      </div>
      {adding ? <div className="flex items-center gap-2 rounded-xl border border-action/50 bg-action-soft p-3"><Input autoFocus aria-label="New variable name" onChange={(event) => setNewName(event.target.value)} onKeyDown={(event) => { if (event.key === 'Enter') { event.preventDefault(); addVariable() } }} placeholder="VARIABLE_NAME" value={newName} /><button className="grid size-9 shrink-0 place-items-center rounded-lg bg-action text-on-action transition-colors hover:bg-action-strong focus-visible:outline-2 focus-visible:outline-focus" onClick={addVariable} type="button"><Plus size={17} weight="bold" /></button><button className="px-2 text-label text-text-tertiary hover:text-text-primary" onClick={() => { setAdding(false); setNewName('') }} type="button">Cancel</button></div> : <button className="flex w-fit items-center gap-2 rounded-lg px-2 py-1.5 text-label text-action-strong transition-colors hover:bg-action-soft focus-visible:outline-2 focus-visible:outline-focus" onClick={() => setAdding(true)} type="button"><Plus size={16} />Add variable</button>}
    </div>
  )
}
function VariableField<T extends FieldValues>({
  register,
  variable,
  onRemove,
}: {
  register: UseFormRegister<T>
  variable: VariableDefinition
  onRemove?: (name: string) => void
}) {
  const [visible, setVisible] = useState(!variable.secret)
  return (
    <div className="group grid gap-2 border-l-2 border-transparent bg-surface-1 px-3 py-3 transition-[background-color,border-color] duration-[var(--ely-duration-fast)] motion-reduce:transition-none hover:border-border-default hover:bg-surface-2 focus-within:border-action focus-within:bg-surface-2 sm:grid-cols-[minmax(12rem,0.7fr)_minmax(0,1.3fr)] sm:items-center sm:gap-6">
      <div className="min-w-0">
        <div className="flex items-center justify-between gap-3">
          <label
            className="text-label text-text-secondary"
            htmlFor={variable.name}
          >
            {variable.name}
          </label>
          <div className="flex items-center gap-1">
            {variable.secret ? <IconButton label={visible ? `Hide ${variable.name}` : `Show ${variable.name}`} onClick={() => setVisible((current) => !current)} size="sm">{visible ? <EyeSlash aria-hidden="true" size={16} /> : <Eye aria-hidden="true" size={16} />}</IconButton> : null}
            {onRemove ? <IconButton label={`Delete ${variable.name}`} onClick={() => onRemove(variable.name)} size="sm"><TrashSimple aria-hidden="true" size={16} /></IconButton> : null}
          </div>
        </div>
        {variable.description ? (
          <p className="text-log text-text-tertiary">{variable.description}</p>
        ) : null}
      </div>
      <Input
        id={variable.name}
        type={variable.secret && !visible ? 'password' : 'text'}
        {...register(variable.name as never)}
      />
    </div>
  )
}
