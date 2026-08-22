import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { Controller, useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { useEffect, useState } from "react";
import { ArrowRight, CheckCircle, Code, Database, Eye, EyeSlash, GithubLogo, Lightning, LockKey, RocketLaunch, UserCircle } from "@phosphor-icons/react";
import type { Icon as DesignIcon } from "@aether/design-system";
import { Badge, Button, Card, Checkbox, EmptyState, Field, Input, Progress, Typography, useToast } from "@aether/design-system";
import { api, apiGet, getServer, setToken } from "../../api/client";

const signupSchema = z.object({
  name: z.string().min(2, "Name is required"),
  email: z.string().email("Invalid email"),
  password: z.string().min(8, "Minimum 8 characters"),
  tos: z.boolean().refine((value) => value, "Accept the terms to continue"),
}).superRefine((value, context) => {
  if (value.password.length >= 8 && (!/[A-Z]/.test(value.password) || !/[0-9]/.test(value.password))) {
    context.addIssue({ code: "custom", path: ["password"], message: "Include uppercase and a number" });
  }
});

type SignupForm = z.infer<typeof signupSchema>;
const designIcon = (icon: typeof ArrowRight) => icon as unknown as DesignIcon;

function strengthOf(value: string) {
  let score = 0;
  if (value.length > 0) score += 1;
  if (value.length >= 8) score += 1;
  if (/[A-Z]/.test(value) && /[0-9]/.test(value)) score += 1;
  if (/[^A-Za-z0-9]/.test(value)) score += 1;
  return { score, label: score === 0 ? "" : score === 1 ? "Weak" : score === 2 ? "Fair" : score === 3 ? "Good" : "Strong" };
}

const benefits = [
  { icon: Code, title: "Open source foundation", description: "Build on standard primitives with full access to the engine." },
  { icon: Database, title: "Self-hosted capabilities", description: "Deploy to your own infrastructure with one-command configuration." },
  { icon: Lightning, title: "Premium developer experience", description: "Ship quickly with integrated delivery and observability." },
];

function Onboarding() {
  const navigate = useNavigate();
  const { add } = useToast();
  const [status, setStatus] = useState({ registered: false, sso: false });
  const [apiDown, setApiDown] = useState(false);
  const [showPassword, setShowPassword] = useState(false);
  const [providers, setProviders] = useState<{ id: string; name: string }[]>([]);
  const form = useForm<SignupForm>({ resolver: zodResolver(signupSchema), defaultValues: { name: "", email: "", password: "", tos: false } });
  const password = form.watch("password");
  const strength = strengthOf(password);

  useEffect(() => {
    apiGet<{ registered: boolean; sso: boolean }>("/api/v1/auth/status").then((value) => {
      setStatus(value);
      if (value.sso) apiGet<{ id: string; name: string }[]>("/api/v1/sso/public").then(setProviders);
    }).catch(() => setApiDown(true));
  }, []);

  const submit = async (values: SignupForm) => {
    try {
      const response = await api<{ token: string }>("/api/v1/auth/register", { method: "POST", body: { name: values.name, email: values.email, password: values.password } });
      setToken(response.token);
      add({ title: "Account created", tone: "success" });
      await navigate({ to: "/" });
    } catch (error) {
      add({ title: "Unable to create account", description: error instanceof Error ? error.message : "Try again.", tone: "error" });
    }
  };

  const ssoLogin = async (providerName: string) => {
    const provider = providers.find((item) => item.name.toLowerCase().includes(providerName));
    if (!provider) { add({ title: `No ${providerName} provider configured`, tone: "error" }); return; }
    const { url } = await api<{ url: string }>(`/api/v1/sso/public/${provider.id}/auth-url`);
    window.open(url, "_blank", "width=700,height=600");
  };

  return (
    <main className="min-h-screen bg-background px-4 py-6 text-foreground sm:px-6 lg:flex lg:items-center lg:justify-center">
      <div className="grid w-full max-w-6xl overflow-hidden rounded-2xl border border-border bg-surface-card shadow-xl lg:grid-cols-[0.82fr_1.18fr]">
        <aside className="hidden bg-surface-container p-10 lg:flex lg:flex-col lg:justify-between">
          <div className="space-y-10">
            <div className="flex items-center gap-3"><span className="flex size-10 items-center justify-center rounded-lg bg-primary text-primary-foreground"><RocketLaunch size={22} weight="fill" /></span><Typography as="span" level="heading">Aether</Typography></div>
            <div className="space-y-3"><Typography as="p" level="label" tone="primary">Get started</Typography><Typography as="h1" level="display">Build without limits.</Typography><Typography as="p" level="body" tone="muted">A focused control plane for teams that want ownership without sacrificing developer experience.</Typography></div>
            <div className="space-y-5">{benefits.map((benefit) => <div key={benefit.title} className="flex gap-3"><span className="flex size-9 shrink-0 items-center justify-center rounded-lg border border-border bg-surface-card text-primary"><benefit.icon size={18} weight="duotone" /></span><div><Typography as="h2" level="small" weight="semibold">{benefit.title}</Typography><Typography as="p" level="small" tone="muted">{benefit.description}</Typography></div></div>)}</div>
          </div>
          <Badge tone="info" icon={designIcon(CheckCircle)}>One workspace, full control</Badge>
        </aside>
        <Card as="section" variant="default" padding="lg" className="rounded-none border-0 sm:p-12">
          <div className="mx-auto w-full max-w-xl space-y-8">
            {status.registered ? <EmptyState icon={designIcon(CheckCircle)} title="Already set up" description="This platform already has an administrator account. Use the existing credentials to sign in." action={<Button icon={designIcon(ArrowRight)} onClick={() => navigate({ to: "/login" })}>Sign in</Button>} /> : <>
              {apiDown ? <Card variant="danger" padding="sm"><Typography as="p" level="small" tone="danger">API unreachable at {getServer() || "same origin"}. Start the backend before creating an account.</Typography></Card> : null}
              <header className="space-y-2"><Typography as="p" level="label" tone="primary">Create your workspace identity</Typography><Typography as="h2" level="display">Create account</Typography><Typography as="p" level="body" tone="muted">Already have an account? <Link to="/login" className="font-semibold text-primary hover:underline">Sign in</Link></Typography></header>
              <form onSubmit={form.handleSubmit(submit)} className="space-y-5" noValidate>
                <Field label="Full name" error={form.formState.errors.name?.message}><Input placeholder="Jane Doe" leadingIcon={designIcon(UserCircle)} autoComplete="name" {...form.register("name")} /></Field>
                <Field label="Email" error={form.formState.errors.email?.message}><Input type="email" placeholder="jane@company.com" leadingIcon={designIcon(UserCircle)} autoComplete="email" {...form.register("email")} /></Field>
                <Field label="Password" error={form.formState.errors.password?.message} description="Use at least 8 characters, one uppercase letter and one number."><div className="relative"><Input type={showPassword ? "text" : "password"} placeholder="Create a secure password" leadingIcon={designIcon(LockKey)} autoComplete="new-password" {...form.register("password")} /><button type="button" aria-label={showPassword ? "Hide password" : "Show password"} onClick={() => setShowPassword((value) => !value)} className="absolute inset-y-0 right-0 z-10 w-11 text-muted-foreground">{showPassword ? <EyeSlash size={18} className="mx-auto" /> : <Eye size={18} className="mx-auto" />}</button></div><div className="mt-3 flex items-center gap-3"><Progress value={strength.score} max={4} status={strength.score >= 3 ? "success" : strength.score === 2 ? "warning" : "danger"} size="sm" label="Password strength" /><Typography as="span" level="small" tone={strength.score >= 3 ? "success" : strength.score > 0 ? "danger" : "muted"}>{strength.label}</Typography></div></Field>
                <Controller name="tos" control={form.control} render={({ field }) => <Checkbox checked={field.value} onCheckedChange={(value) => field.onChange(value === true)} error={form.formState.errors.tos?.message} label="I agree to the Terms of Service and Privacy Policy." />} />
                <Button type="submit" fullWidth icon={designIcon(ArrowRight)} loading={form.formState.isSubmitting}>Create account</Button>
              </form>
              {status.sso && providers.length ? <div className="space-y-4 border-t border-border pt-5"><Typography as="p" level="label" tone="muted" align="center">Or continue with</Typography><div className="grid grid-cols-2 gap-3"><Button variant="outline" icon={designIcon(UserCircle)} onClick={() => ssoLogin("google")}>Google</Button><Button variant="outline" icon={designIcon(GithubLogo)} onClick={() => ssoLogin("github")}>GitHub</Button></div></div> : null}
            </>}
          </div>
        </Card>
      </div>
    </main>
  );
}

export const Route = createFileRoute("/onboarding/")({ component: Onboarding });
