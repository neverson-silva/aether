import {
  ArrowRight,
  CheckCircle,
  Eye,
  EyeSlash,
  Fingerprint,
  LockKey,
} from '@phosphor-icons/react'
import { Link, useNavigate } from '@tanstack/react-router'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { type FormEvent, useState } from 'react'
import {
  Button,
  Checkbox,
  Field,
  IconButton,
  Input,
  Typography,
} from '@aether/elisyum-ds'
import { ApiError, apiGet, apiPost } from '../api/client'
import type { Me } from '../api/types'
import { BrandMark, ProductShowcase } from './login'
import { useAuthStore } from '../stores/auth'

const statusKey = ['auth', 'public-status']

interface PublicAuthStatus {
  registered: boolean
  sso: boolean
}

function passwordStrength(password: string) {
  let score = 0
  if (password.length > 0) score += 1
  if (password.length >= 8) score += 1
  if (/[A-Z]/.test(password) && /[0-9]/.test(password)) score += 1
  if (/[^A-Za-z0-9]/.test(password)) score += 1
  return {
    score,
    label: ['', 'Weak', 'Fair', 'Good', 'Strong'][score],
  }
}

export function OnboardingSurface() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const setUser = useAuthStore((state) => state.setUser)
  const status = useQuery({
    queryKey: statusKey,
    queryFn: () => apiGet<PublicAuthStatus>('/api/v1/auth/status'),
    refetchOnMount: 'always',
    retry: false,
  })
  const [ownerRegistered, setOwnerRegistered] = useState(false)
  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [acceptedTerms, setAcceptedTerms] = useState(false)
  const [showPassword, setShowPassword] = useState(false)
  const [error, setError] = useState('')
  const [validationError, setValidationError] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const strength = passwordStrength(password)
  const isConfigured = ownerRegistered || status.data?.registered === true

  const submit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setError('')
    setValidationError('')
    if (name.trim().length < 2) {
      setValidationError('Enter your full name.')
      return
    }
    if (
      password.length < 8 ||
      !/[A-Z]/.test(password) ||
      !/[0-9]/.test(password)
    ) {
      setValidationError('Use at least 8 characters, one uppercase letter and one number.')
      return
    }
    if (!acceptedTerms) {
      setValidationError('Accept the terms to create the owner account.')
      return
    }
    setSubmitting(true)
    try {
      await apiPost<{ user: { id: string } }>('/api/v1/auth/register', {
        name: name.trim(),
        email: email.trim(),
        password,
      })
      setOwnerRegistered(true)
      queryClient.setQueryData<PublicAuthStatus>(statusKey, {
        registered: true,
        sso: status.data?.sso ?? false,
      })
      const me = await apiGet<Me>('/api/v1/me')
      setUser(me)
      window.dispatchEvent(new Event('aether:auth'))
      await navigate({ to: '/' })
    } catch (reason) {
      if (reason instanceof ApiError && reason.status === 409) {
        setOwnerRegistered(true)
        queryClient.setQueryData<PublicAuthStatus>(statusKey, {
          registered: true,
          sso: status.data?.sso ?? false,
        })
        setError('An owner account is already registered for this instance. Sign in with that account to continue.')
      } else if (ownerRegistered) {
        setError('The owner account was created, but sign-in could not be completed. Sign in with the account you just created.')
      } else {
        setError(reason instanceof Error ? reason.message : 'The owner account could not be created. Try again.')
      }
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <main className="min-h-dvh bg-canvas text-text-primary">
      <div className="grid min-h-dvh w-full lg:grid-cols-[minmax(0,1.1fr)_minmax(30rem,0.9fr)]">
        <ProductShowcase />
        <section
          aria-labelledby="onboarding-heading"
          className="grid min-h-dvh content-center bg-canvas px-6 py-[clamp(1rem,3vh,3rem)] sm:px-10 md:px-16 lg:px-10 xl:px-16 2xl:px-20"
        >
          <div className="mx-auto grid w-full max-w-[26rem] gap-[clamp(1.25rem,4vh,2.25rem)]">
            <div className="lg:hidden">
              <BrandMark />
              <p className="mt-3 hidden max-w-md text-supporting text-text-tertiary sm:block lg:hidden">
                Establish the first owner account for your Aether instance.
              </p>
            </div>
            {status.isPending ? (
              <div className="grid justify-items-center gap-4 py-12 text-center" role="status">
                <span className="size-6 animate-spin rounded-full border-2 border-action border-t-transparent" />
                <Typography role="supporting">Checking instance setup…</Typography>
              </div>
            ) : status.isError ? (
              <div className="grid gap-5">
                <header className="grid gap-3.5">
                  <div className="flex items-center gap-2.5 text-label tracking-[0.16em] text-text-subtle">
                    <Fingerprint className="text-action" size={17} aria-hidden="true" />
                    INSTANCE SETUP
                  </div>
                  <Typography as="h1" id="onboarding-heading" role="page-title">
                    Setup unavailable.
                  </Typography>
                  <Typography role="supporting">
                    Aether could not check whether an owner account already exists.
                  </Typography>
                </header>
                <p className="text-supporting text-danger" role="alert">
                  {status.error instanceof Error ? status.error.message : 'The API could not be reached.'}
                </p>
                <Button onClick={() => void status.refetch()} loading={status.isFetching} size="lg">
                  Retry setup check
                </Button>
              </div>
            ) : isConfigured ? (
              <div className="grid justify-items-center gap-6 text-center">
                <span className="grid size-14 place-items-center rounded-full border border-success/30 bg-success-soft text-success">
                  <CheckCircle size={30} weight="duotone" aria-hidden="true" />
                </span>
                <header className="grid gap-3">
                  <Typography as="p" role="supporting" className="text-action">
                    INSTANCE CONFIGURED
                  </Typography>
                  <Typography as="h1" id="onboarding-heading" role="page-title">
                    Owner account already exists.
                  </Typography>
                  <Typography role="supporting">
                    This Aether instance already has an owner account. Sign in with the existing owner credentials to continue.
                  </Typography>
                </header>
                <Button
                  className="group w-full justify-center"
                  onClick={() => void navigate({ to: '/login' })}
                  size="lg"
                >
                  Sign in
                  <ArrowRight className="transition-transform duration-[var(--ely-duration-fast)] group-hover:translate-x-0.5" size={17} aria-hidden="true" />
                </Button>
              </div>
            ) : (
              <>
                <header className="grid gap-3.5">
                  <div className="flex items-center gap-2.5 text-label tracking-[0.16em] text-text-subtle">
                    <Fingerprint className="text-action" size={17} aria-hidden="true" />
                    FIRST-TIME SETUP
                  </div>
                  <Typography as="h1" id="onboarding-heading" role="page-title">
                    Create the owner account.
                  </Typography>
                  <Typography role="supporting">
                    Set up the first identity that will own this Aether instance.
                  </Typography>
                  <p className="text-supporting text-text-tertiary">
                    Already have an account?{' '}
                    <Link className="font-medium text-action hover:underline focus-visible:outline-2 focus-visible:outline-focus" to="/login">
                      Sign in
                    </Link>
                  </p>
                </header>
                <form className="grid gap-[clamp(1rem,3vh,1.5rem)]" onSubmit={submit}>
                  <Field id="owner-name" label="Full name" required>
                    <Input
                      autoComplete="name"
                      className="min-h-12 px-4"
                      minLength={2}
                      onChange={(event) => setName(event.target.value)}
                      placeholder="Jane Doe"
                      required
                      value={name}
                    />
                  </Field>
                  <Field id="owner-email" label="Work email" required>
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
                  <Field
                    id="owner-password"
                    label="Password"
                    description="Use at least 8 characters, one uppercase letter and one number."
                    required
                  >
                    <div className="relative">
                      <Input
                        autoComplete="new-password"
                        className="min-h-12 px-4 pr-12"
                        minLength={8}
                        onChange={(event) => setPassword(event.target.value)}
                        placeholder="Create a secure password"
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
                    <div className="mt-2 flex items-center gap-3" aria-label={`Password strength: ${strength.label || 'empty'}`}>
                      <div className="flex flex-1 gap-1" aria-hidden="true">
                        {[1, 2, 3, 4].map((level) => (
                          <span
                            className={`h-1.5 flex-1 rounded-full ${strength.score >= level ? 'bg-action' : 'bg-surface-3'}`}
                            key={level}
                          />
                        ))}
                      </div>
                      <span className="text-label text-text-tertiary">{strength.label}</span>
                    </div>
                  </Field>
                  <label className="flex cursor-pointer items-start gap-3 text-supporting text-text-secondary">
                    <Checkbox
                      checked={acceptedTerms}
                      onChange={(event) => setAcceptedTerms(event.target.checked)}
                    />
                    <span>I agree to the Terms of Service and Privacy Policy.</span>
                  </label>
                  {validationError ? (
                    <p className="text-supporting text-danger" role="alert">{validationError}</p>
                  ) : null}
                  {error ? <p className="text-supporting text-danger" role="alert">{error}</p> : null}
                  <Button
                    className="group mt-1 w-full justify-center shadow-elevation-1"
                    loading={submitting}
                    size="lg"
                    type="submit"
                  >
                    {submitting ? 'Creating owner account…' : 'Create owner account'}
                    {!submitting ? <ArrowRight className="transition-transform duration-[var(--ely-duration-fast)] group-hover:translate-x-0.5" size={17} aria-hidden="true" /> : null}
                  </Button>
                </form>
                <div className="flex items-center gap-3 border-t border-border-subtle pt-5 text-supporting text-text-tertiary">
                  <LockKey size={16} aria-hidden="true" />
                  <span>This account becomes the owner of the Aether instance.</span>
                </div>
              </>
            )}
          </div>
        </section>
      </div>
    </main>
  )
}
