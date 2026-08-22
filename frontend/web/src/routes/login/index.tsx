import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { Controller, useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { useState } from "react";
import { CheckCircle, Eye, EyeSlash, LockKey, SignIn, UserCircle } from "@phosphor-icons/react";
import type { Icon as DesignIcon } from "@aether/design-system";
import { Badge, Button, Card, Checkbox, Field, Input, Typography, useToast } from "@aether/design-system";
import { useLogin } from "../../hooks";
import { useAuthStore } from "../../stores/auth";
import { apiGet, getServer, setServer } from "../../api/client";
import type { Me } from "../../api/types";

const loginSchema = z.object({
  email: z.string().email("Invalid email"),
  password: z.string().min(1, "Password is required"),
  remember: z.boolean(),
});

type LoginForm = z.infer<typeof loginSchema>;
const designIcon = (icon: typeof SignIn) => icon as unknown as DesignIcon;

function Login() {
  const form = useForm<LoginForm>({ resolver: zodResolver(loginSchema), defaultValues: { email: "", password: "", remember: true } });
  const login = useLogin();
  const navigate = useNavigate();
  const { add } = useToast();
  const emailField = form.register("email");
  const passwordField = form.register("password");
  const [showPassword, setShowPassword] = useState(false);
  const [status, setStatus] = useState<"checking" | "ok" | "down">("checking");

  const checkStatus = async () => {
    try {
      const response = await fetch(`${getServer() || ""}/healthz`);
      setStatus(response.ok ? "ok" : "down");
    } catch {
      setStatus("down");
    }
  };

  const submit = async (values: LoginForm) => {
    try {
      console.log("Submitting login form with values:", values);
      await login.mutateAsync({ email: values.email, password: values.password, server: getServer() || "" });
      const me = await apiGet<Me>("/api/v1/me");
      useAuthStore.getState().setUser(me);
      if (values.remember) setServer(getServer() || "");
      add({ title: "Authenticated successfully", tone: "success" });
      await navigate({ to: "/" });
    } catch (error) {
      add({ title: "Authentication failed", description: error instanceof Error ? error.message : "Check your credentials and try again.", tone: "error" });
    }
  };

  return (
    <main className="min-h-screen bg-background px-4 py-10 text-foreground sm:px-6 lg:flex lg:items-center lg:justify-center">
      <div className="grid w-full max-w-5xl overflow-hidden rounded-2xl border border-border bg-surface-card shadow-xl lg:grid-cols-[0.9fr_1.1fr]">
        <aside className="hidden bg-surface-container p-10 lg:flex lg:flex-col lg:justify-between">
          <div className="space-y-8">
            <div className="flex items-center gap-3"><span className="flex size-10 items-center justify-center rounded-lg bg-primary text-primary-foreground"><CheckCircle size={24} weight="fill" /></span><Typography as="span" level="heading">Aether</Typography></div>
            <div className="space-y-3"><Typography as="p" level="label" tone="primary">Control plane</Typography><Typography as="h1" level="display">Operate infrastructure with clarity.</Typography><Typography as="p" level="body" tone="muted">Deploy services, manage databases and observe your platform from one focused workspace.</Typography></div>
          </div>
          <Badge tone="success" icon={designIcon(CheckCircle)}>Platform ready</Badge>
        </aside>
        <Card as="section" variant="default" padding="lg" className="rounded-none border-0 sm:p-12">
          <div className="mx-auto w-full max-w-[28rem] min-w-0 space-y-8">
            <header className="space-y-2"><Typography as="p" level="label" tone="primary">Welcome back</Typography><Typography as="h2" level="display">Sign in</Typography><Typography as="p" level="body" tone="muted">Access your Aether workspace and continue operating your services.</Typography></header>
            <form onSubmit={form.handleSubmit(submit)} className="space-y-5" noValidate>
              <Field label="Email" error={form.formState.errors.email?.message}>
                <Input id="email" type="email" placeholder="you@company.com" leadingIcon={designIcon(UserCircle)} autoComplete="email" name={emailField.name} onChange={emailField.onChange} onBlur={emailField.onBlur} inputRef={emailField.ref} />
              </Field>
              <Field label="Password" error={form.formState.errors.password?.message}>
                <div className="relative"><Input id="password" type={showPassword ? "text" : "password"} placeholder="Your password" leadingIcon={designIcon(LockKey)} autoComplete="current-password" name={passwordField.name} onChange={passwordField.onChange} onBlur={passwordField.onBlur} inputRef={passwordField.ref} /><button type="button" aria-label={showPassword ? "Hide password" : "Show password"} onClick={() => setShowPassword((value) => !value)} className="absolute inset-y-0 right-0 z-10 flex w-11 items-center justify-center text-muted-foreground hover:text-foreground">{showPassword ? <EyeSlash size={18} /> : <Eye size={18} />}</button></div>
              </Field>
              <Controller name="remember" control={form.control} render={({ field }) => <label className="flex cursor-pointer items-center gap-3 text-body-sm font-semibold text-foreground"><Checkbox checked={field.value} onCheckedChange={(value) => field.onChange(value === true)} /><span>Keep session active</span></label>} />
              <Button variant='primary' type="submit" fullWidth icon={designIcon(SignIn)} loading={form.formState.isSubmitting}>Sign in</Button>
            </form>
            <div className="flex flex-col gap-4 border-t border-border pt-6 text-center"><Typography as="p" level="small" tone="muted">No account yet? <Link to="/onboarding" className="font-semibold text-primary hover:underline">Create account</Link></Typography><button type="button" onClick={checkStatus} className="mx-auto inline-flex items-center gap-2 text-body-sm text-muted-foreground hover:text-foreground"><span className={`size-2 rounded-full ${status === "ok" ? "bg-status-success" : status === "down" ? "bg-status-danger" : "bg-muted-foreground"}`} />{status === "checking" ? "Check system status" : status === "ok" ? "System operational" : "System unreachable"}</button></div>
          </div>
        </Card>
      </div>
    </main>
  );
}

export const Route = createFileRoute("/login/")({ component: Login });
