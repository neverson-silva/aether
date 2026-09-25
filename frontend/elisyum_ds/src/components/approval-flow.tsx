import type { ReactNode } from 'react'
import { Avatar } from './avatar'
import type { BadgeTone } from './badge'
import { Marker } from './marker'
import { Button } from './button'

export interface ApprovalPerson {
  id: string
  name: string
  role?: ReactNode
  decision?: 'pending' | 'approved' | 'rejected'
}
export interface ApprovalFlowProps {
  title?: ReactNode
  people: ApprovalPerson[]
  onApprove?: (id: string) => void
  onReject?: (id: string) => void
  className?: string
}
const decisionTone: Record<NonNullable<ApprovalPerson['decision']>, BadgeTone> = {
  pending: 'warning',
  approved: 'success',
  rejected: 'danger',
}
export function ApprovalFlow({
  className = '',
  onApprove,
  onReject,
  people,
  title = 'Approval flow',
}: ApprovalFlowProps) {
  return (
    <section
      className={`grid gap-4 rounded-xl border border-border-subtle bg-surface-1 p-4 ${className}`}
    >
      <h2 className="text-section-title text-text-primary">{title}</h2>
      <ul className="m-0 grid list-none p-0">
        {people.map((person) => (
          <li
            className="flex flex-wrap items-center gap-3 border-t border-border-subtle py-3 transition-[background-color,border-color] duration-[var(--ely-duration-fast)] motion-reduce:transition-none hover:bg-surface-2"
            key={person.id}
          >
            <Avatar
              name={person.name}
              size="sm"
            />
            <div className="min-w-0 flex-1">
              <p className="text-supporting text-text-primary">{person.name}</p>
              <p className="text-log text-text-tertiary">{person.role}</p>
            </div>
            <span className="inline-flex items-center gap-2 text-log text-text-secondary">
              <Marker
                aria-hidden="true"
                tone={decisionTone[person.decision ?? 'pending']}
              />
              {person.decision ?? 'pending'}
            </span>
            {!person.decision || person.decision === 'pending' ? (
              <div className="flex gap-1">
                <Button
                  onClick={() => onApprove?.(person.id)}
                  size="sm"
                >
                  Approve
                </Button>
                <Button
                  onClick={() => onReject?.(person.id)}
                  size="sm"
                  tone="danger"
                >
                  Reject
                </Button>
              </div>
            ) : null}
          </li>
        ))}
      </ul>
    </section>
  )
}
