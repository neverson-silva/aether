import { ArrowClockwise, Bug, CaretDown, CaretUp } from "@phosphor-icons/react";
import { type ReactNode, useState } from "react";
export interface ErrorBoundaryUIProps {
  error?: Error | null;
  reset?: () => void;
  title?: ReactNode;
  description?: ReactNode;
  reportId?: string;
  children?: ReactNode;
}
export function ErrorBoundaryUI({
  children,
  description = "Something went wrong while rendering this area.",
  error,
  reportId,
  reset,
  title = "Unable to load this area",
}: ErrorBoundaryUIProps) {
  const [details, setDetails] = useState(false);
  return (
    <section
      role="alert"
      className="rounded-2xl border border-status-danger/30 bg-status-danger-container/10 p-5 text-foreground shadow-sm sm:p-6"
    >
      <div className="flex gap-3">
        <Bug
          size={22}
          className="mt-0.5 shrink-0 text-status-danger"
          aria-hidden="true"
        />
        <div className="min-w-0 flex-1">
          <h2 className="font-semibold">{title}</h2>
          <p className="mt-1 text-body-sm text-muted-foreground">
            {description}
          </p>
          {reportId ? (
            <code className="mt-2 block text-code-md text-muted-foreground">
              Report ID: {reportId}
            </code>
          ) : null}
          <div className="mt-4 flex flex-wrap items-center gap-3">
            {reset ? (
              <button
                type="button"
                onClick={reset}
                className="inline-flex items-center gap-2 rounded-xl bg-primary px-3 py-2 text-body-sm text-primary-foreground outline-none transition-[filter,transform] duration-150 hover:brightness-110 focus-visible:ring-2 focus-visible:ring-ring active:scale-[0.985]"
              >
                <ArrowClockwise size={16} />
                Try again
              </button>
            ) : null}
            {error ? (
              <button
                type="button"
                onClick={() => setDetails(!details)}
                className="inline-flex items-center gap-1 rounded-lg px-2 py-1 text-body-sm text-muted-foreground underline underline-offset-2 outline-none transition-colors hover:bg-surface-container focus-visible:ring-2 focus-visible:ring-ring"
              >
                Technical details
                {details ? <CaretUp size={14} /> : <CaretDown size={14} />}
              </button>
            ) : null}
          </div>
          {details && error ? (
            <pre className="mt-4 max-h-40 overflow-auto rounded-xl border border-status-danger/20 bg-surface-lowest p-3 text-code-md text-status-danger">
              {error.stack ?? error.message}
            </pre>
          ) : null}
        </div>
      </div>
      {children}
    </section>
  );
}
