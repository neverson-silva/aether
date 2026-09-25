import { CaretLeft, CaretRight } from '@phosphor-icons/react'
import { useState, type ReactNode } from 'react'
import { IconButton } from './icon-button'

export interface CarouselProps {
  items: ReactNode[]
  label?: string
  className?: string
}
export function Carousel({ className = '', items, label = 'Carousel' }: CarouselProps) {
  const [index, setIndex] = useState(0)
  if (!items.length)
    return (
      <section
        aria-label={label}
        className={`grid place-items-center rounded-lg border border-dashed border-border-default bg-surface-1 px-4 py-8 text-supporting text-text-tertiary ${className}`}
      >
        No slides available.
      </section>
    )
  const current = items[index]
  return (
    <section
      aria-label={label}
      aria-roledescription="carousel"
      className={`grid gap-3 ${className}`}
    >
      <div
        aria-live="polite"
        className="overflow-hidden border-y border-border-subtle bg-surface-1 transition-[opacity,transform] duration-[var(--ely-duration-standard)] ease-ely-out motion-reduce:transition-opacity"
        key={index}
      >
        {current}
      </div>
      <div className="flex items-center justify-between">
        <IconButton
          disabled={index === 0}
          label="Previous slide"
          onClick={() => setIndex((value) => Math.max(0, value - 1))}
        >
          <CaretLeft
            aria-hidden="true"
            size={18}
          />
        </IconButton>
        <span className="font-technical text-log text-text-tertiary">
          {String(index + 1).padStart(2, '0')} / {String(items.length).padStart(2, '0')}
        </span>
        <IconButton
          disabled={index === items.length - 1}
          label="Next slide"
          onClick={() => setIndex((value) => Math.min(items.length - 1, value + 1))}
        >
          <CaretRight
            aria-hidden="true"
            size={18}
          />
        </IconButton>
      </div>
    </section>
  )
}
