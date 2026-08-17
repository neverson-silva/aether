import { createFileRoute, Link } from "@tanstack/react-router";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { useEffect, useState } from "react";
import { useNavigate } from "@tanstack/react-router";
import { api, apiGet, getServer, setToken } from "../../api/client";
import { useToast } from "../../components/ui";

const signupSchema = z
  .object({
    name: z.string().min(2, "Name is required"),
    email: z.string().email("Invalid email"),
    password: z.string().min(8, "Minimum 8 characters"),
    tos: z.boolean().refine((v) => v, "Accept the terms to continue"),
  })
  .superRefine((v, ctx) => {
    if (v.password.length >= 8) {
      if (!/[A-Z]/.test(v.password) || !/[0-9]/.test(v.password)) {
        ctx.addIssue({ code: "custom", path: ["password"], message: "Include uppercase and a number" });
      }
    }
  });

type SignupForm = z.infer<typeof signupSchema>;

function strengthOf(val: string): { score: number; label: string; color: string } {
  let score = 0;
  if (val.length > 0) score++;
  if (val.length >= 8) score++;
  if (/[A-Z]/.test(val) && /[0-9]/.test(val)) score++;
  if (/[^A-Za-z0-9]/.test(val)) score++;
  if (score === 0) return { score: 0, label: "", color: "" };
  if (score === 1) return { score: 1, label: "Weak", color: "bg-error" };
  if (score === 2) return { score: 2, label: "Fair", color: "bg-tertiary-container" };
  return { score, label: score === 4 ? "Strong" : "Good", color: "bg-primary" };
}

function Onboarding() {
  const navigate = useNavigate();
  const { toast } = useToast();
  const [status, setStatus] = useState<{ registered: boolean; sso: boolean }>({ registered: false, sso: false });
  const [registered, setRegistered] = useState(false);
  const [apiDown, setApiDown] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [showPassword, setShowPassword] = useState(false);
  const [providers, setProviders] = useState<{ id: string; name: string }[]>([]);

  const {
    register,
    handleSubmit,
    watch,
    formState: { errors },
  } = useForm<SignupForm>({
    resolver: zodResolver(signupSchema),
    defaultValues: { name: "", email: "", password: "", tos: false },
  });

  const password = watch("password");
  const strength = strengthOf(password);

  useEffect(() => {
    apiGet<{ registered: boolean; sso: boolean }>("/api/v1/auth/status")
      .then((s) => {
        setStatus(s);
        if (s.registered) {
          setRegistered(true);
        }
        if (s.sso) {
          apiGet<{ id: string; name: string }[]>("/api/v1/sso/public").then(setProviders);
        }
      })
      .catch(() => setApiDown(true));
  }, [navigate]);

  const submit = async (values: SignupForm) => {
    setSubmitting(true);
    try {
      const res = await api<{ token: string }>("/api/v1/auth/register", {
        method: "POST",
        body: { name: values.name, email: values.email, password: values.password },
      });
      setToken(res.token);
      navigate({ to: "/" });
    } catch (err) {
      toast(err instanceof Error ? err.message : "failed to create account", "error");
      setSubmitting(false);
    }
  };

  const ssoLogin = async (providerName: string) => {
    const p = providers.find((x) => x.name.toLowerCase().includes(providerName));
    if (!p) {
      toast(`No ${providerName} provider configured`, "error");
      return;
    }
    const { url } = await api<{ url: string }>(`/api/v1/sso/public/${p.id}/auth-url`);
    window.open(url, "_blank", "width=700,height=600");
  };

  return (
    <div
      className="font-body-md min-h-screen flex items-center justify-center p-md sm:p-lg antialiased"
      style={{ backgroundColor: "#050505", color: "#e5e2e1" }}
    >
      <main className="w-full max-w-[1000px] flex flex-col md:flex-row rounded-lg overflow-hidden shadow-2xl relative z-10" style={{ border: "1px solid #1F1F1F", backgroundColor: "#1c1b1b" }}>
        <aside className="w-full md:w-[400px] p-xl flex flex-col justify-between relative overflow-hidden group" style={{ backgroundColor: "#201f1f", borderRight: "1px solid #1F1F1F" }}>
          <div className="absolute -top-32 -left-32 w-64 h-64 bg-primary-container/20 rounded-full blur-3xl opacity-50 group-hover:opacity-70 transition-opacity duration-1000" />
          <div>
            <div className="flex items-center gap-sm mb-lg relative z-10">
              <div className="w-8 h-8 rounded bg-primary flex items-center justify-center text-on-primary">
                <span className="material-symbols-outlined" style={{ fontVariationSettings: "'FILL' 1" }}>rocket_launch</span>
              </div>
              <h1 className="font-headline-sm text-headline-sm text-on-surface">Aether</h1>
            </div>
            <div className="space-y-lg relative z-10 mt-xl">
              <h2 className="font-headline-sm text-headline-sm text-on-surface mb-md">Build without limits.</h2>
              <ul className="space-y-md">
                <li className="flex items-start gap-md">
                  <div className="w-6 h-6 rounded bg-surface-container-highest flex items-center justify-center mt-1 border border-outline-variant">
                    <span className="material-symbols-outlined text-[16px] text-primary">code</span>
                  </div>
                  <div>
                    <h3 className="font-label-caps text-label-caps text-on-surface mb-xs">Open Source Foundation</h3>
                    <p className="font-body-sm text-body-sm text-on-surface-variant">Built on standard primitives. No vendor lock-in. Full access to the engine.</p>
                  </div>
                </li>
                <li className="flex items-start gap-md">
                  <div className="w-6 h-6 rounded bg-surface-container-highest flex items-center justify-center mt-1 border border-outline-variant">
                    <span className="material-symbols-outlined text-[16px] text-primary">dns</span>
                  </div>
                  <div>
                    <h3 className="font-label-caps text-label-caps text-on-surface mb-xs">Self-Hosted Capabilities</h3>
                    <p className="font-body-sm text-body-sm text-on-surface-variant">Deploy to your own infrastructure with single-command configurations.</p>
                  </div>
                </li>
                <li className="flex items-start gap-md">
                  <div className="w-6 h-6 rounded bg-surface-container-highest flex items-center justify-center mt-1 border border-outline-variant">
                    <span className="material-symbols-outlined text-[16px] text-primary">bolt</span>
                  </div>
                  <div>
                    <h3 className="font-label-caps text-label-caps text-on-surface mb-xs">Premium DX</h3>
                    <p className="font-body-sm text-body-sm text-on-surface-variant">Sub-second deployments. Integrated CI/CD. Comprehensive observability.</p>
                  </div>
                </li>
              </ul>
            </div>
          </div>
          <div className="mt-xl relative z-10 pt-lg" style={{ borderTop: "1px solid rgba(66,70,85,0.3)" }}>
            <blockquote className="font-body-sm text-body-sm text-on-surface-variant italic border-l-2 border-primary pl-md">
              "Aether fundamentally changed how we deploy. The abstraction is perfectly balanced between power and simplicity."
            </blockquote>
            <div className="mt-md flex items-center gap-sm">
              <div className="w-6 h-6 rounded-full bg-surface-container-highest flex items-center justify-center font-label-caps text-label-caps" style={{ border: "1px solid #1F1F1F" }}>
                SC
              </div>
              <span className="font-label-caps text-label-caps text-on-surface">Sarah Chen, Tech Lead</span>
            </div>
          </div>
        </aside>

        <section className="flex-1 p-xl lg:p-[48px] flex flex-col justify-center" style={{ backgroundColor: "#131313" }}>
          {registered ? (
            <div className="max-w-[448px] w-full mx-auto text-center space-y-lg">
              <div className="w-14 h-14 mx-auto rounded-xl flex items-center justify-center" style={{ backgroundColor: "rgba(176,198,255,0.12)", border: "1px solid rgba(176,198,255,0.25)" }}>
                <span className="material-symbols-outlined text-primary text-[28px]">verified_user</span>
              </div>
              <div>
                <h2 className="font-headline-sm text-headline-sm text-on-surface mb-xs">Already set up</h2>
                <p className="font-body-md text-body-md text-on-surface-variant">
                  This platform already has an admin account. Only one account can be created here — use the existing credentials to sign in.
                </p>
              </div>
              <Link
                to="/login"
                className="w-full bg-primary hover:bg-primary-container text-on-primary font-label-caps text-label-caps py-md px-lg rounded flex items-center justify-center gap-sm transition-all duration-200 active:scale-[0.98]"
              >
                Sign in
                <span className="material-symbols-outlined text-[18px]">arrow_forward</span>
              </Link>
            </div>
          ) : (
          <>
          {apiDown && (
            <p className="mb-md px-md py-sm rounded font-body-sm text-body-sm text-error" style={{ border: "1px solid rgba(255,180,171,0.3)", backgroundColor: "rgba(147,0,10,0.2)" }}>
              API unreachable at {getServer() || "same origin"} - is the backend running? (aether serve, port 8080)
            </p>
          )}
          <header className="mb-xl">
            <h2 className="font-headline-sm text-headline-sm text-on-surface mb-xs">Create your account</h2>
            <p className="font-body-md text-body-md text-on-surface-variant">
              Get started with Aether in seconds. Already have an account?{" "}
              <Link to="/login" className="text-primary hover:text-primary-container transition-colors">Sign in</Link>
            </p>
          </header>
          <form onSubmit={handleSubmit(submit)} className="space-y-lg max-w-[448px] w-full" noValidate>
            <div className="space-y-xs">
              <label className="block font-label-caps text-label-caps text-on-surface-variant" htmlFor="name">Full Name</label>
              <div className="relative input-glow rounded transition-all duration-200 flex items-center" style={{ border: "1px solid #1F1F1F", backgroundColor: "#201f1f" }}>
                <span className="material-symbols-outlined text-outline ml-sm absolute pointer-events-none">person</span>
                <input
                  className="w-full bg-transparent border-none text-on-surface font-body-md text-body-md py-sm pl-[48px] pr-md focus:ring-0 placeholder:text-outline/50"
                  id="name"
                  placeholder="Jane Doe"
                  {...register("name")}
                />
              </div>
              {errors.name?.message && <p className="font-body-sm text-[12px] text-error">{errors.name.message}</p>}
            </div>

            <div className="space-y-xs">
              <label className="block font-label-caps text-label-caps text-on-surface-variant" htmlFor="email">Email</label>
              <div className="relative input-glow rounded transition-all duration-200 flex items-center" style={{ border: "1px solid #1F1F1F", backgroundColor: "#201f1f" }}>
                <span className="material-symbols-outlined text-outline ml-sm absolute pointer-events-none">mail</span>
                <input
                  className="w-full bg-transparent border-none text-on-surface font-body-md text-body-md py-sm pl-[48px] pr-md focus:ring-0 placeholder:text-outline/50"
                  id="email"
                  placeholder="jane@company.com"
                  type="email"
                  {...register("email")}
                />
              </div>
              {errors.email?.message && <p className="font-body-sm text-[12px] text-error">{errors.email.message}</p>}
            </div>

            <div className="space-y-xs">
              <label className="block font-label-caps text-label-caps text-on-surface-variant" htmlFor="password">Password</label>
              <div className="relative input-glow rounded transition-all duration-200 flex items-center" style={{ border: "1px solid #1F1F1F", backgroundColor: "#201f1f" }}>
                <span className="material-symbols-outlined text-outline ml-sm absolute pointer-events-none">lock</span>
                <input
                  className="w-full bg-transparent border-none text-on-surface font-code-md text-code-md py-sm pl-[48px] pr-12 focus:ring-0 placeholder:text-outline/50"
                  id="password"
                  placeholder="••••••••"
                  type={showPassword ? "text" : "password"}
                  {...register("password")}
                />
                <button
                  type="button"
                  onClick={() => setShowPassword((v) => !v)}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-outline hover:text-primary transition-colors"
                  aria-label={showPassword ? "Hide password" : "Show password"}
                >
                  <span className="material-symbols-outlined text-[18px]">{showPassword ? "visibility_off" : "visibility"}</span>
                </button>
              </div>
              <div className="pt-sm flex items-center justify-between gap-md">
                <div className="flex-1 flex gap-xs h-1">
                  {[1, 2, 3, 4].map((i) => (
                    <div
                      key={i}
                      className={`h-full flex-1 rounded-full transition-colors duration-300 ${
                        strength.score >= i ? strength.color : "bg-outline-variant/30"
                      }`}
                    />
                  ))}
                </div>
                <span className={`font-label-caps text-[10px] uppercase tracking-wider ${strength.color || "text-on-surface-variant"}`}>
                  {strength.label}
                </span>
              </div>
              {errors.password?.message && <p className="font-body-sm text-[12px] text-error">{errors.password.message}</p>}
            </div>

            <div className="flex items-start gap-sm pt-sm">
              <div className="flex items-center h-5">
                <input className="w-4 h-4 rounded bg-surface-container text-primary focus:ring-primary focus:ring-offset-background cursor-pointer" id="tos" type="checkbox" {...register("tos")} />
              </div>
              <label className="font-body-sm text-body-sm text-on-surface-variant cursor-pointer" htmlFor="tos">
                I agree to the <a className="text-primary hover:underline" href="#">Terms of Service</a> and <a className="text-primary hover:underline" href="#">Privacy Policy</a>.
              </label>
            </div>
            {errors.tos?.message && <p className="font-body-sm text-[12px] text-error">{errors.tos.message}</p>}

            <div className="pt-md">
              <button
                type="submit"
                disabled={submitting}
                className="w-full bg-primary hover:bg-primary-container text-on-primary font-label-caps text-label-caps py-md px-lg rounded flex items-center justify-center gap-sm transition-all duration-200 active:scale-[0.98] disabled:opacity-60"
              >
                {submitting ? "Creating account…" : "Create Account"}
                <span className="material-symbols-outlined text-[18px]">arrow_forward</span>
              </button>
            </div>

            {status?.sso && (
              <>
                <div className="relative flex items-center py-sm">
                  <div className="flex-grow border-t border-outline-variant/50" />
                  <span className="flex-shrink-0 mx-4 font-label-caps text-label-caps text-on-surface-variant">OR</span>
                  <div className="flex-grow border-t border-outline-variant/50" />
                </div>
                <div className="flex gap-md">
                  <button
                    type="button"
                    onClick={() => ssoLogin("google")}
                    className="flex-1 hover:bg-surface-container-high rounded py-sm flex items-center justify-center gap-sm transition-colors text-on-surface font-body-sm text-body-sm"
                    style={{ backgroundColor: "#201f1f", border: "1px solid #1F1F1F" }}
                  >
                    <span className="material-symbols-outlined text-[18px]">account_circle</span>
                    Google
                  </button>
                  <button
                    type="button"
                    onClick={() => ssoLogin("github")}
                    className="flex-1 hover:bg-surface-container-high rounded py-sm flex items-center justify-center gap-sm transition-colors text-on-surface font-body-sm text-body-sm"
                    style={{ backgroundColor: "#201f1f", border: "1px solid #1F1F1F" }}
                  >
                    <span className="material-symbols-outlined text-[18px]">code_blocks</span>
                    GitHub
                  </button>
                </div>
              </>
            )}
          </form>
          </>
          )}
        </section>
      </main>
    </div>
  );
}

export const Route = createFileRoute("/onboarding/")({
  component: Onboarding,
});
