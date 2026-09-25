import { CaretRight } from '@phosphor-icons/react'
import type { ReactNode } from 'react'

export interface BreadcrumbItem {
  label: ReactNode
  href?: string
}
export interface BreadcrumbProps {
  items: BreadcrumbItem[]
  className?: string
}

export function Breadcrumb({ className = '', items }: BreadcrumbProps) {
  return (
    <nav
      aria-label="Breadcrumb"
      className={className}
    >
      <ol className="flex flex-wrap items-center gap-1 text-supporting text-text-tertiary">
        {items.map((item, index) => (
          <li
            className="flex items-center gap-1"
            key={`${index}-${String(item.label)}`}
          >
            {index > 0 ? (
              <CaretRight
                aria-hidden="true"
                className="text-text-subtle"
                size={14}
              />
            ) : null}
            {item.href && index < items.length - 1 ? (
              <a
                className="cursor-pointer rounded-sm px-1 transition-[background-color,color] duration-[var(--ely-duration-fast)] motion-reduce:transition-none hover:bg-surface-2 hover:text-text-primary focus-visible:outline-2 focus-visible:outline-focus"
                href={item.href}
              >
                {item.label}
              </a>
            ) : (
              <span
                aria-current={index === items.length - 1 ? 'page' : undefined}
                className={`px-1 ${index === items.length - 1 ? 'text-text-primary' : ''}`}
              >
                {item.label}
              </span>
            )}
          </li>
        ))}
      </ol>
    </nav>
  )
}
