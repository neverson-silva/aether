import { type ReactNode, useRef, useState } from 'react'
export interface VirtualizedListProps<T> {
  items: T[]
  rowHeight: number
  height?: number
  overscan?: number
  renderItem: (item: T, index: number) => ReactNode
  loading?: boolean
  empty?: ReactNode
  getKey?: (item: T, index: number) => string
}
export function VirtualizedList<T>({
  empty = 'No items.',
  getKey,
  height = 400,
  items,
  loading,
  overscan = 4,
  renderItem,
  rowHeight,
}: VirtualizedListProps<T>) {
  const scrollRef = useRef<HTMLDivElement>(null)
  const [scrollTop, setScrollTop] = useState(0)
  const start = Math.max(0, Math.floor(scrollTop / rowHeight) - overscan)
  const end = Math.min(
    items.length,
    Math.ceil((scrollTop + height) / rowHeight) + overscan,
  )
  return (
    <div
      ref={scrollRef}
      onScroll={(event) => setScrollTop(event.currentTarget.scrollTop)}
      style={{ height }}
      className="overflow-auto rounded-lg border border-border"
    >
      {loading ? (
        <div className="p-6 text-center text-body-sm text-muted-foreground">
          Loading...
        </div>
      ) : items.length ? (
        <div style={{ height: items.length * rowHeight, position: 'relative' }}>
          <div
            style={{
              position: 'absolute',
              top: start * rowHeight,
              left: 0,
              right: 0,
            }}
          >
            {items.slice(start, end).map((item, index) => {
              const actualIndex = start + index
              return (
                <div
                  key={getKey?.(item, actualIndex) ?? actualIndex}
                  style={{ minHeight: rowHeight }}
                >
                  {renderItem(item, actualIndex)}
                </div>
              )
            })}
          </div>
        </div>
      ) : (
        <div className="p-8 text-center text-body-sm text-muted-foreground">
          {empty}
        </div>
      )}
    </div>
  )
}
