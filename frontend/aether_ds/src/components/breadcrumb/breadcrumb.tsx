import type { ReactNode } from 'react'
export interface BreadcrumbItem {
  label: ReactNode
  href?: string
  current?: boolean
}
export interface BreadcrumbProps {
  items: BreadcrumbItem[]
  maxItems?: number
  separator?: ReactNode
}
export function Breadcrumb({
  items,
  maxItems = 4,
  separator = '/',
}: BreadcrumbProps) {
  const visible =
    items.length > maxItems
      ? [items[0], { label: '…' }, ...items.slice(-maxItems + 1)]
      : items
  return (
    <nav aria-label="Breadcrumb">
      <ol className="flex min-w-0 items-center gap-2 text-body-sm text-muted-foreground">
        {visible.map((item, index) => (
          <li
            key={`${index}-${String(item.label)}`}
            className="flex min-w-0 items-center gap-2"
          >
            {index > 0 ? <span aria-hidden="true">{separator}</span> : null}
            {item.href && !item.current ? (
              <a
                href={item.href}
                className="truncate hover:text-foreground hover:underline"
              >
                {item.label}
              </a>
            ) : (
              <span
                className={
                  item.current
                    ? 'truncate font-semibold text-foreground'
                    : 'truncate'
                }
                aria-current={item.current ? 'page' : undefined}
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
