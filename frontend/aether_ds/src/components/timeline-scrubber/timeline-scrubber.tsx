import { Pause, Play } from '@phosphor-icons/react'
import { useEffect, useState } from 'react'
export interface TimelineMarker {
  id: string
  position: number
  label: string
  tone?: 'info' | 'warning' | 'danger'
}
export interface TimelineScrubberProps {
  start: number
  end: number
  value?: number
  markers?: TimelineMarker[]
  timezone?: string
  onChange?: (value: number) => void
  onRangeChange?: (range: [number, number]) => void
  playback?: boolean
}
export function TimelineScrubber({
  end,
  markers = [],
  onChange,
  playback,
  start,
  timezone,
  value = start,
}: TimelineScrubberProps) {
  const [playing, setPlaying] = useState(false)
  const [current, setCurrent] = useState(value)
  useEffect(() => {
    if (!playing) return
    const timer = window.setInterval(
      () =>
        setCurrent((previous) => {
          const next = Math.min(end, previous + (end - start) / 100)
          onChange?.(next)
          if (next >= end) setPlaying(false)
          return next
        }),
      100,
    )
    return () => window.clearInterval(timer)
  }, [end, onChange, playing, start])
  const percent = ((current - start) / (end - start)) * 100
  return (
    <section className="rounded-lg border border-border bg-surface-card p-4">
      <div className="mb-3 flex items-center justify-between text-body-sm">
        <span className="font-mono text-foreground">
          {new Date(current).toLocaleString()}
        </span>
        {timezone ? (
          <span className="text-muted-foreground">{timezone}</span>
        ) : null}
      </div>
      <div className="relative h-8">
        <input
          type="range"
          min={start}
          max={end}
          value={current}
          onChange={(event) => {
            const next = Number(event.target.value)
            setCurrent(next)
            onChange?.(next)
          }}
          aria-label="Timeline position"
          className="absolute inset-0 z-10 h-8 w-full cursor-pointer opacity-0"
        />
        <div className="absolute top-3 h-2 w-full rounded-full bg-surface-container" />
        <div
          className="absolute top-3 h-2 rounded-full bg-primary"
          style={{ width: `${percent}%` }}
        />
        {markers.map((marker) => (
          <span
            key={marker.id}
            title={marker.label}
            className={`absolute top-1 size-6 -translate-x-1/2 rounded-full border-2 border-surface-card ${marker.tone === 'danger' ? 'bg-status-danger' : marker.tone === 'warning' ? 'bg-status-warning' : 'bg-status-info'}`}
            style={{ left: `${marker.position}%` }}
          />
        ))}
      </div>
      {playback ? (
        <button
          type="button"
          onClick={() => setPlaying(!playing)}
          className="mt-2 inline-flex items-center gap-1 text-body-sm text-muted-foreground hover:text-foreground"
        >
          {playing ? <Pause size={16} /> : <Play size={16} />}
          {playing ? 'Pause' : 'Play'}
        </button>
      ) : null}
    </section>
  )
}
