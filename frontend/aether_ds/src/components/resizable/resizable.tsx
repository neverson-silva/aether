import {
  type HTMLAttributes,
  type ReactNode,
  type PointerEvent as ReactPointerEvent,
  useEffect,
  useRef,
  useState,
} from 'react'

export interface ResizableProps extends HTMLAttributes<HTMLDivElement> {
  sidebar?: ReactNode
  children: ReactNode
  minWidth?: number
  maxWidth?: number
  defaultWidth?: number
  onWidthChange?: (width: number) => void
}

export function Resizable({
  children,
  className = '',
  defaultWidth = 320,
  maxWidth = 640,
  minWidth = 220,
  onWidthChange,
  sidebar,
  ...props
}: ResizableProps) {
  const [width, setWidth] = useState(defaultWidth)
  const [dragging, setDragging] = useState(false)
  const startX = useRef(0)
  const startWidth = useRef(defaultWidth)

  const update = (value: number) => {
    const next = Math.min(maxWidth, Math.max(minWidth, value))
    setWidth(next)
    onWidthChange?.(next)
  }

  const handlePointerDown = (event: ReactPointerEvent<HTMLDivElement>) => {
    event.preventDefault()
    startX.current = event.clientX
    startWidth.current = width
    setDragging(true)
    event.currentTarget.setPointerCapture(event.pointerId)
  }

  useEffect(() => {
    if (!dragging) return
    const handlePointerMove = (event: PointerEvent) => {
      update(startWidth.current + event.clientX - startX.current)
    }
    const handlePointerUp = () => setDragging(false)
    window.addEventListener('pointermove', handlePointerMove)
    window.addEventListener('pointerup', handlePointerUp)
    return () => {
      window.removeEventListener('pointermove', handlePointerMove)
      window.removeEventListener('pointerup', handlePointerUp)
    }
  }, [dragging])

  return (
    <div
      className={`flex min-h-0 ${dragging ? 'select-none' : ''} ${className}`}
      {...props}
    >
      {sidebar ? (
        <aside
          className="relative flex shrink-0 border-r border-border"
          style={{ width }}
        >
          <div className="min-w-0 flex-1 overflow-auto">{sidebar}</div>
          <input
            type="range"
            aria-label="Resize panel"
            min={minWidth}
            max={maxWidth}
            step={1}
            value={width}
            className="absolute -right-2 top-0 z-10 h-full w-4 cursor-col-resize touch-none opacity-0"
            onChange={(event) => update(Number(event.target.value))}
            onPointerDown={handlePointerDown}
            onPointerUp={() => setDragging(false)}
            onPointerCancel={() => setDragging(false)}
            onKeyDown={(event) => {
              if (event.key === 'ArrowLeft') update(width - 16)
              if (event.key === 'ArrowRight') update(width + 16)
              if (event.key === 'Home') update(minWidth)
              if (event.key === 'End') update(maxWidth)
            }}
          />
          <span
            className={`pointer-events-none absolute -right-0.5 top-1/2 z-0 h-12 w-1 -translate-y-1/2 rounded-full transition-colors ${dragging ? 'bg-primary' : 'bg-border'}`}
            aria-hidden="true"
          />
        </aside>
      ) : null}
      <main className="min-w-0 flex-1 overflow-auto">{children}</main>
    </div>
  )
}
