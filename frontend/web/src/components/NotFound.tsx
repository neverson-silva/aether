import { Link } from "@tanstack/react-router";
import { CloudSlash, House, MapTrifold, Database } from "@phosphor-icons/react";
import { Button } from "@aether/design-system";

export function NotFound() {
  return (
    <div className="relative min-h-dvh w-full flex items-center justify-center overflow-hidden bg-background text-on-background font-body-sm text-body-sm">
      <div className="pointer-events-none absolute -top-40 left-1/2 -translate-x-1/2 h-[420px] w-[640px] rounded-full bg-primary/10 blur-3xl" />
      <div className="pointer-events-none absolute bottom-0 right-0 h-80 w-80 rounded-full bg-secondary/10 blur-3xl" />
      <div className="pointer-events-none absolute -left-16 top-1/3 h-64 w-64 rounded-full bg-tertiary/10 blur-3xl" />

      <div className="relative z-10 flex flex-col items-center text-center px-md max-w-xl">
        <div className="relative mb-lg flex items-center justify-center">
          <CloudSlash size={120} weight="duotone" className="text-primary/15 absolute animate-pulse select-none" aria-hidden="true" />
          <span className="relative text-[clamp(4.5rem,16vw,9rem)] font-display-lg font-bold leading-none tracking-tighter bg-gradient-to-r from-primary via-on-surface to-primary bg-clip-text text-transparent">
            404
          </span>
        </div>

        <div className="flex items-center gap-sm mb-sm">
          <MapTrifold size={18} className="text-primary" aria-hidden="true" />
          <h1 className="font-headline-sm text-headline-sm text-on-surface">Page not found</h1>
        </div>

        <p className="text-body-md text-on-surface-variant mb-lg leading-relaxed">
          The page you are looking for drifted into the void — it may have been moved, renamed, or never existed.
        </p>

        <div className="flex items-center gap-sm flex-wrap justify-center">
          <Link to="/">
            <Button variant="primary">
              <House size={18} aria-hidden="true" />
              Back to home
            </Button>
          </Link>
          <Link to="/databases">
            <Button variant="secondary">
              <Database size={18} aria-hidden="true" />
              Browse databases
            </Button>
          </Link>
        </div>

        <div className="mt-xl font-code-md text-code-md text-outline-variant">
          HTTP · <span className="text-primary">404</span> · not_found
        </div>
      </div>
    </div>
  );
}
