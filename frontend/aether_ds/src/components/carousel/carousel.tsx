import { CaretLeft, CaretRight, Pause, Play } from '@phosphor-icons/react'
import { type ReactNode, useEffect, useState } from 'react'
export interface CarouselProps {
  items: ReactNode[]
  labels?: string[]
  autoplay?: boolean
  interval?: number
  onChange?: (index: number) => void
}
export function Carousel({
  autoplay,
  interval = 5000,
  items,
  labels,
  onChange,
}: CarouselProps) {
  const [index, setIndex] = useState(0)
  const [paused, setPaused] = useState(!autoplay)
  const update = (next: number) => {
    const value = (next + items.length) % items.length
    setIndex(value)
    onChange?.(value)
  }
  useEffect(() => {
    if (paused || items.length < 2) return
    const timer = window.setInterval(() => update(index + 1), interval)
    return () => window.clearInterval(timer)
  }, [index, interval, items.length, paused])
  if (!items.length) return null
  return (
    <section
      aria-roledescription="carousel"
      aria-label="Carousel"
      className="relative overflow-hidden rounded-lg border border-border"
    >
      <div className="min-h-48">{items[index]}</div>
      {items.length > 1 ? (
        <>
          <button
            type="button"
            aria-label="Previous slide"
            onClick={() => update(index - 1)}
            className="absolute left-3 top-1/2 inline-flex size-8 -translate-y-1/2 items-center justify-center rounded-full bg-surface-modal/80 text-foreground shadow-md hover:bg-surface-modal focus-visible:ring-2 focus-visible:ring-primary"
          >
            <CaretLeft size={18} />
          </button>
          <button
            type="button"
            aria-label="Next slide"
            onClick={() => update(index + 1)}
            className="absolute right-3 top-1/2 inline-flex size-8 -translate-y-1/2 items-center justify-center rounded-full bg-surface-modal/80 text-foreground shadow-md hover:bg-surface-modal focus-visible:ring-2 focus-visible:ring-primary"
          >
            <CaretRight size={18} />
          </button>
          <div className="absolute bottom-3 left-1/2 flex -translate-x-1/2 gap-1.5">
            {items.map((_, itemIndex) => (
              <button
                type="button"
                key={itemIndex}
                aria-label={
                  labels?.[itemIndex] ?? `Go to slide ${itemIndex + 1}`
                }
                aria-current={index === itemIndex}
                onClick={() => update(itemIndex)}
                className={`size-2 rounded-full transition-colors ${index === itemIndex ? 'bg-primary' : 'bg-border'}`}
              />
            ))}
          </div>
          {autoplay ? (
            <button
              type="button"
              aria-label={paused ? 'Play carousel' : 'Pause carousel'}
              onClick={() => setPaused(!paused)}
              className="absolute bottom-2 right-3 rounded p-1 text-muted-foreground hover:bg-surface-container"
            >
              {paused ? <Play size={14} /> : <Pause size={14} />}
            </button>
          ) : null}
        </>
      ) : null}
    </section>
  )
}
