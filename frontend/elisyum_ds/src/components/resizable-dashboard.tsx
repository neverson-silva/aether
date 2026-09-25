import type { ReactNode } from 'react'
export interface ResizableDashboardPanel {
  id: string
  title: ReactNode
  content: ReactNode
}
export interface ResizableDashboardProps {
  panels: ResizableDashboardPanel[]
  className?: string
}
export function ResizableDashboard({
  className = '',
  panels,
}: ResizableDashboardProps) {
  return (
    <div
      className={`grid auto-rows-[minmax(12rem,auto)] grid-cols-1 divide-y divide-border-subtle border-y border-border-subtle bg-surface-1 md:grid-cols-2 md:divide-x md:divide-y-0 ${className}`}
    >
      {panels.map((panel, index) => (
        <section
          className="p-4 transition-[background-color] duration-[var(--ely-duration-fast)] motion-reduce:transition-none hover:bg-surface-2"
          key={panel.id}
        >
          <div className="mb-4 flex items-center justify-between gap-3">
            <h2 className="text-label text-text-secondary">{panel.title}</h2>
            <span className="font-technical text-log text-text-subtle">
              {String(index + 1).padStart(2, '0')}
            </span>
          </div>
          {panel.content}
        </section>
      ))}
    </div>
  )
}
