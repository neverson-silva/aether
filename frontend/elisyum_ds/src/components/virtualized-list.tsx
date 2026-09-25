import { useMemo, useState, type ReactNode, type UIEvent } from 'react'

export interface VirtualizedListProps<T> {
  items: T[]
  renderItem: (item: T, index: number) => ReactNode
  itemHeight?: number
  height?: number
  className?: string
}

export function VirtualizedList<T>({
  className = '',
  height = 360,
  itemHeight = 44,
  items,
  renderItem,
}: VirtualizedListProps<T>) {
  const [scrollTop, setScrollTop] = useState(0)
  const overscan = 4
  const start = Math.max(0, Math.floor(scrollTop / itemHeight) - overscan)
  const end = Math.min(
    items.length,
    Math.ceil((scrollTop + height) / itemHeight) + overscan,
  )
  const visible = useMemo(() => items.slice(start, end), [end, items, start])
  const handleScroll = (event: UIEvent<HTMLDivElement>) =>
    setScrollTop(event.currentTarget.scrollTop)
  return (
    <div
      aria-label="Virtualized list"
      className={`overflow-auto overscroll-contain border-y border-border-subtle bg-surface-1 ${className}`}
      onScroll={handleScroll}
      role="list"
      style={{ height }}
    >
      <div style={{ height: items.length * itemHeight, position: 'relative' }}>
        {visible.map((item, offset) => {
          const index = start + offset
          return (
            <div
              aria-setsize={items.length}
              aria-posinset={index + 1}
              className="transition-[background-color] duration-[var(--ely-duration-fast)] motion-reduce:transition-none hover:bg-surface-2"
              key={index}
              role="listitem"
              style={{
                height: itemHeight,
                left: 0,
                position: 'absolute',
                right: 0,
                top: index * itemHeight,
              }}
            >
              {renderItem(item, index)}
            </div>
          )
        })}
      </div>
    </div>
  )
}
