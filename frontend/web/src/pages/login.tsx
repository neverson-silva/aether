import {
  Button,
  Checkbox,
  Field,
  IconButton,
  Input,
  Marker,
  Typography,
} from '@aether/elisyum-ds'
import {
  ArrowRight,
  CheckCircle,
  CloudArrowUp,
  Command,
  Database,
  Eye,
  EyeSlash,
  Fingerprint,
  GitBranch,
  HardDrives,
  LockKey,
  Pulse,
  RocketLaunch,
} from '@phosphor-icons/react'
import { Link, useNavigate } from '@tanstack/react-router'
import { type FormEvent, useState } from 'react'
import { apiGet } from '../api/client'
import type { Me } from '../api/types'
import { useLogin } from '../hooks/use-login'
import { useAuthStore } from '../stores/auth'

export function LoginSurface() {
  const navigate = useNavigate()
  const login = useLogin()
  const setUser = useAuthStore((state) => state.setUser)
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [remember, setRemember] = useState(true)
  const [showPassword, setShowPassword] = useState(false)
  const [error, setError] = useState('')

  const submit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setError('')
    try {
      await login.mutateAsync({ email, password, server: '' })
      setUser(await apiGet<Me>('/api/v1/me'))
      if (remember) localStorage.setItem('aether_session_preference', 'persistent')
      await navigate({ to: '/' })
    } catch (value) {
      setError(value instanceof Error ? value.message : 'Authentication failed')
    }
  }

  return (
    <main className="min-h-dvh bg-canvas text-text-primary">
      <div className="grid min-h-dvh w-full lg:grid-cols-[minmax(0,1.1fr)_minmax(30rem,0.9fr)]">
        <ProductShowcase />
        <section
          aria-labelledby="login-heading"
          className="grid min-h-dvh content-center bg-canvas px-6 py-[clamp(1rem,3vh,3rem)] sm:px-10 md:px-16 lg:px-10 xl:px-16 2xl:px-20"
        >
          <div className="mx-auto grid w-full max-w-[26rem] gap-[clamp(1.25rem,4vh,2.25rem)]">
            <div className="lg:hidden">
              <BrandMark />
              <p className="mt-3 hidden max-w-md text-supporting text-text-tertiary sm:block lg:hidden">
                Deploy and operate every service from one control plane.
              </p>
            </div>
            <header className="grid gap-3.5">
              <div className="flex items-center gap-2.5 text-label tracking-[0.16em] text-text-subtle">
                <Fingerprint
                  className="text-action"
                  size={17}
                  aria-hidden="true"
                />
                WORKSPACE ACCESS
              </div>
              <Typography
                as="h1"
                id="login-heading"
                role="page-title"
                style={{
                  fontSize: 'clamp(2rem, 1.35vw + 1rem, 2.5rem)',
                  lineHeight: 1.08,
                  letterSpacing: '-0.028em',
                }}
              >
                Welcome back.
              </Typography>
              <Typography
                className="max-w-lg"
                role="supporting"
              >
                Sign in to continue to your Aether workspace.
              </Typography>
            </header>

            <form
              className="grid gap-[clamp(1rem,3vh,1.5rem)]"
              onSubmit={submit}
            >
              <div className="[&_label]:text-supporting">
                <Field
                  id="email"
                  label="Work email"
                  required
                >
                  <Input
                    autoComplete="email"
                    autoCapitalize="none"
                    autoCorrect="off"
                    className="min-h-12 px-4"
                    onChange={(event) => setEmail(event.target.value)}
                    placeholder="you@company.com"
                    required
                    type="email"
                    value={email}
                  />
                </Field>
              </div>
              <div className="[&_label]:text-supporting">
                <Field
                  id="password"
                  label="Password"
                  required
                  error={error || undefined}
                >
                  <div className="relative">
                    <Input
                      autoComplete="current-password"
                      className="min-h-12 px-4 pr-12"
                      onChange={(event) => setPassword(event.target.value)}
                      placeholder="Enter your password"
                      required
                      type={showPassword ? 'text' : 'password'}
                      value={password}
                    />
                    <IconButton
                      aria-pressed={showPassword}
                      className="absolute right-1 top-1/2 -translate-y-1/2"
                      label={showPassword ? 'Hide password' : 'Show password'}
                      onClick={() => setShowPassword((current) => !current)}
                    >
                      {showPassword ? <EyeSlash size={18} /> : <Eye size={18} />}
                    </IconButton>
                  </div>
                </Field>
              </div>
              <label className="flex cursor-pointer items-center gap-3 py-1 text-supporting text-text-secondary">
                <Checkbox
                  checked={remember}
                  onChange={(event) => setRemember(event.target.checked)}
                />
                <span>Keep this session active</span>
              </label>
              <Button
                className="group mt-1 w-full justify-center shadow-elevation-1"
                loading={login.isPending}
                size="lg"
                type="submit"
              >
                {login.isPending ? 'Signing in…' : 'Continue'}
                {!login.isPending ? (
                  <ArrowRight
                    className="transition-transform duration-[var(--ely-duration-fast)] group-hover:translate-x-0.5"
                    size={17}
                    aria-hidden="true"
                  />
                ) : null}
              </Button>
            </form>

            <div className="grid gap-4 border-t border-border-subtle pt-5 text-supporting text-text-tertiary">
              <LockKey
                size={16}
                aria-hidden="true"
              />
              <span>
                Your session is protected by your organization’s access policy.
              </span>
              <p className="text-center text-text-secondary">
                New to Aether?{' '}
                <Link
                  className="font-medium text-action hover:underline focus-visible:outline-2 focus-visible:outline-focus"
                  to="/onboarding"
                >
                  Create the owner account
                </Link>
              </p>
            </div>
          </div>
        </section>
      </div>
    </main>
  )
}

export function ProductShowcase() {
  return (
    <aside
      aria-label="Aether platform preview"
      className="relative hidden min-h-dvh items-center overflow-y-auto border-r border-border-subtle bg-surface-1 lg:flex lg:flex-col lg:px-8 lg:py-[clamp(1rem,3vh,2.75rem)] xl:px-12 2xl:px-16"
    >
      <div className="pointer-events-none absolute left-0 top-1/4 h-80 w-3/4 rounded-full bg-action-soft/30 blur-3xl" />
      <div className="relative flex w-full max-w-[68rem] items-center justify-between gap-4">
        <BrandMark />
        <span className="inline-flex items-center gap-2 rounded-full border border-border-subtle bg-surface-2 px-3 py-1.5 text-label text-text-secondary">
          <span
            className="size-1.5 rounded-full bg-info"
            aria-hidden="true"
          />
          PRODUCT PREVIEW
        </span>
      </div>

      <div className="relative my-auto grid w-full max-w-[68rem] gap-[clamp(1rem,2.5vh,2rem)] py-[clamp(1rem,3vh,2.5rem)]">
        <div className="max-w-3xl">
          <div className="mb-4 flex items-center gap-3 text-label tracking-[0.18em] text-action">
            <Marker tone="accent" />
            ONE CONTROL PLANE · EVERY RELEASE
          </div>
          <Typography
            as="h2"
            className="max-w-3xl"
            role="display"
            style={{
              fontSize: 'clamp(2.5rem, 2vw + 1rem, 4rem)',
              lineHeight: 1.04,
              letterSpacing: '-0.035em',
            }}
          >
            Every service. Every signal. In view.
          </Typography>
          <Typography
            className="mt-4 max-w-2xl"
            role="body"
            style={{ fontSize: 'clamp(1rem, 0.35vw + 0.9rem, 1.125rem)' }}
          >
            Deploy applications, shape runtime and understand every release from one
            operational workspace.
          </Typography>
        </div>
        <RuntimePreview />
      </div>
    </aside>
  )
}

function RuntimePreview() {
  return (
    <section
      aria-label="Illustrative runtime interface using sample data"
      className="@container w-full overflow-hidden rounded-2xl border border-border-default/50 bg-surface-0 shadow-elevation-2"
    >
      <header className="flex flex-wrap items-center justify-between gap-4 border-b border-border-subtle bg-surface-1 px-4 py-4 @lg:flex-nowrap @lg:px-5">
        <div className="flex items-center gap-3">
          <span className="grid size-9 place-items-center rounded-lg bg-action-soft text-action">
            <HardDrives
              size={18}
              aria-hidden="true"
            />
          </span>
          <div>
            <p className="text-supporting font-semibold text-text-primary">
              Northstar Commerce
            </p>
            <p className="mt-0.5 text-label text-text-tertiary">Example organization</p>
          </div>
        </div>
        <div className="flex flex-wrap items-center gap-3 @lg:flex-nowrap">
          <span className="inline-flex shrink-0 items-center gap-2 whitespace-nowrap rounded-full border border-success/30 bg-success-soft px-3 py-1.5 text-label font-semibold text-success">
            <span
              className="size-1.5 rounded-full bg-success"
              aria-hidden="true"
            />
            All systems healthy
          </span>
          <span className="hidden shrink-0 whitespace-nowrap rounded-lg border border-border-subtle bg-surface-2 px-3 py-1.5 text-label font-medium text-text-primary sm:inline-flex">
            Production
          </span>
        </div>
      </header>
      <div className="grid gap-[clamp(0.75rem,1.5vh,1.5rem)] p-[clamp(0.75rem,1.4vh,1.25rem)]">
        <div className="flex flex-wrap items-end justify-between gap-4 rounded-xl border border-border-subtle bg-surface-1 p-[clamp(0.75rem,1.4vh,1.25rem)] @lg:flex-nowrap">
          <div className="flex min-w-0 items-center gap-3.5">
            <span className="grid size-11 shrink-0 place-items-center rounded-xl bg-action-soft text-action">
              <CloudArrowUp
                size={21}
                aria-hidden="true"
              />
            </span>
            <div className="min-w-0">
              <p className="text-label tracking-[0.12em] text-text-tertiary">
                APPLICATION · PRODUCTION
              </p>
              <p className="mt-1 truncate text-section-title font-semibold text-text-primary">
                storefront
              </p>
              <p className="mt-1 truncate font-technical text-log text-text-tertiary">
                shop.northstar.example
              </p>
            </div>
          </div>
          <div className="flex items-center gap-2 rounded-lg border border-success/25 bg-success-soft px-3 py-2 text-supporting font-medium text-success">
            <CheckCircle
              size={16}
              weight="fill"
              aria-hidden="true"
            />
            Running
          </div>
        </div>
        <div className="grid gap-2">
          <ServicePreview
            icon={RocketLaunch}
            name="api-core"
            detail="API · production"
            state="Running"
          />
          <ServicePreview
            icon={Database}
            name="postgres-primary"
            detail="Database · production"
            state="Healthy"
          />
        </div>
        <div className="grid gap-4 border-t border-border-subtle pt-4 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-center xl:pt-5">
          <div className="flex min-w-0 items-center gap-3">
            <span className="grid size-9 shrink-0 place-items-center rounded-lg bg-action-soft text-action">
              <GitBranch
                size={17}
                aria-hidden="true"
              />
            </span>
            <div className="min-w-0">
              <p className="truncate text-supporting font-medium text-text-primary">
                Production deployment
              </p>
              <p className="mt-0.5 truncate font-technical text-log text-text-tertiary">
                main · 7f3a9c2 · completed 42s
              </p>
            </div>
          </div>
          <div className="flex items-center gap-2 text-label text-text-tertiary">
            <Pulse
              size={15}
              className="text-action"
              aria-hidden="true"
            />
            <span>Signals nominal</span>
            <span
              className="ml-1 flex h-5 items-end gap-0.5"
              aria-hidden="true"
            >
              {[35, 58, 44, 76, 52, 88, 61, 72, 95, 68, 82, 100].map(
                (height, index) => (
                  <span
                    key={index}
                    className={`w-1 rounded-t-sm ${index > 8 ? 'bg-action' : 'bg-action/40'}`}
                    style={{ height: `${height}%` }}
                  />
                ),
              )}
            </span>
          </div>
        </div>
      </div>
      <footer className="flex flex-wrap items-center justify-between gap-3 border-t border-border-subtle bg-surface-1 px-4 py-[clamp(0.5rem,1.2vh,0.75rem)] text-label text-text-tertiary @lg:px-5">
        <span>Illustrative interface · sample data only</span>
        <span className="font-technical text-log">LIVE RUNTIME VIEW</span>
      </footer>
    </section>
  )
}

function ServicePreview({
  icon: Icon,
  name,
  detail,
  state,
}: {
  icon: typeof Database
  name: string
  detail: string
  state: string
}) {
  return (
    <div className="grid grid-cols-[2.25rem_minmax(0,1fr)_auto] items-center gap-3 border-b border-border-subtle px-1 py-2 last:border-b-0">
      <span className="grid size-9 place-items-center rounded-lg bg-surface-2 text-text-secondary">
        <Icon
          size={17}
          aria-hidden="true"
        />
      </span>
      <div className="min-w-0">
        <p className="truncate text-supporting font-semibold text-text-primary">
          {name}
        </p>
        <p className="mt-0.5 truncate text-label text-text-tertiary">{detail}</p>
      </div>
      <span className="inline-flex items-center gap-1.5 text-label font-medium text-success">
        <span
          className="size-1.5 rounded-full bg-success"
          aria-hidden="true"
        />
        {state}
      </span>
    </div>
  )
}

export function BrandMark() {
  return (
    <div className="flex items-center gap-3">
      <span className="grid size-9 place-items-center rounded-lg bg-action text-on-action">
        <Command
          size={18}
          weight="bold"
          aria-hidden="true"
        />
      </span>
      <span className="text-label tracking-[0.2em] text-text-primary">AETHER</span>
    </div>
  )
}
