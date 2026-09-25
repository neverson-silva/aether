import { ArrowUpRight, Command } from './icons'
import { useElisyum } from './providers'

export function ElisyumPreview() {
  const { density, resolvedTheme } = useElisyum()

  return (
    <main className="min-h-screen bg-canvas px-6 py-16 font-interface text-text">
      <div className="mx-auto max-w-3xl space-y-8">
        <div className="flex items-center gap-3 text-sm font-semibold tracking-[0.16em] text-accent uppercase">
          <Command
            size={18}
            weight="bold"
            aria-hidden="true"
          />
          <span>Elisyum design system</span>
        </div>
        <div className="space-y-4">
          <h1 className="max-w-2xl text-display">
            Infrastructure, with a point of view.
          </h1>
          <p className="max-w-xl text-body text-text-muted">
            A token-driven foundation for long sessions, realtime operations and dense
            infrastructure data.
          </p>
        </div>
        <dl className="grid max-w-md grid-cols-2 gap-4 text-supporting">
          <div className="rounded-md border border-border-subtle bg-surface-1 p-4">
            <dt className="text-text-muted">Theme</dt>
            <dd className="mt-1 font-semibold text-text-strong capitalize">
              {resolvedTheme}
            </dd>
          </div>
          <div className="rounded-md border border-border-subtle bg-surface-1 p-4">
            <dt className="text-text-muted">Density</dt>
            <dd className="mt-1 font-semibold text-text-strong capitalize">
              {density}
            </dd>
          </div>
        </dl>
        <a
          className="inline-flex items-center gap-2 rounded-md bg-action px-4 py-2.5 font-semibold text-canvas transition-transform duration-[var(--ely-duration-fast)] ease-ely-out active:scale-[0.98]"
          href="/"
        >
          Explore the system
          <ArrowUpRight
            size={18}
            weight="bold"
            aria-hidden="true"
          />
        </a>
      </div>
    </main>
  )
}
