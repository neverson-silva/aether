import { createFileRoute, Link } from "@tanstack/react-router";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { useState } from "react";
import { useNavigate } from "@tanstack/react-router";
import { useLogin } from "../../hooks";
import { getServer, setServer } from "../../api/client";
import { useToast } from "../../components/ui";

const loginSchema = z.object({
  email: z.string().email("Invalid email"),
  password: z.string().min(1, "Password is required"),
  remember: z.boolean(),
});

type LoginForm = z.infer<typeof loginSchema>;

function Login() {
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<LoginForm>({
    resolver: zodResolver(loginSchema),
    defaultValues: {
      email: "",
      password: "",
      remember: true,
    },
  });
  const login = useLogin();
  const navigate = useNavigate();
  const { toast } = useToast();
  const [showPassword, setShowPassword] = useState(false);
  const [status, setStatus] = useState<"checking" | "ok" | "down">("checking");

  const checkStatus = async () => {
    try {
      const res = await fetch(`${getServer() || ""}/healthz`);
      setStatus(res.ok ? "ok" : "down");
    } catch {
      setStatus("down");
    }
  };

  const submit = async (values: LoginForm) => {
    try {
      await login.mutateAsync({
        email: values.email,
        password: values.password,
        server: getServer() || "",
      });
      if (values.remember) setServer(getServer() || "");
      toast("Authenticated successfully");
      navigate({ to: "/" });
    } catch (err) {
      toast(err instanceof Error ? err.message : "authentication failed", "error");
    }
  };

  return (
    <div
      className="font-body-md min-h-screen flex items-center justify-center relative overflow-hidden antialiased"
      style={{ backgroundColor: "#050507", color: "#e5e2e1" }}
    >
      <div className="fixed inset-0 z-0 pointer-events-none overflow-hidden flex items-center justify-center">
        <div className="absolute w-[800px] h-[800px] bg-primary/10 rounded-full mix-blend-screen blur-[100px] opacity-40 animate-blob top-[-20%] left-[-10%]" />
        <div className="absolute w-[600px] h-[600px] bg-secondary-container/20 rounded-full mix-blend-screen blur-[120px] opacity-30 animate-blob-delayed bottom-[-10%] right-[-10%]" />
        <div className="absolute inset-0 bg-[url('data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHdpZHRoPSI0MCIgaGVpZ2h0PSI0MCI+PGRlZnM+PHBhdHRlcm4gaWQ9ImdyaWQiIHdpZHRoPSI0MCBoZWlnaHQ9IjQwIiBwYXR0ZXJuVW5pdHM9InVzZXJTcGFjZU9uVXNlIj48cGF0aCBkPSJNIDQwIDAgTC AwIDAgMCA0MCIgZmlsbD0ibm9uZSIgc3Ryb2tlPSJyZ2JhKDI1NSwyNTUsMjU1LDAuMDIpIiBzdHJva2Utd2lkdGg9IjEiLz48L3BhdHRlcm4+PC9kZWZzPjxyZWN0IHdpZHRoPSIxMDAlIiBoZWlnaHQ9IjEwMCUiIGZpbGw9InVybCgjZ3JpZCkiLz48L3N2Zz4=')] opacity-50" />
      </div>

      <main className="relative z-10 w-full max-w-[420px] px-margin-mobile md:px-0">
        <div className="glass-card rounded-2xl p-[48px] backdrop-blur-xl relative overflow-hidden group">
          <div className="absolute top-0 left-0 right-0 h-px bg-gradient-to-r from-transparent via-primary/30 to-transparent" />

          <div className="flex flex-col items-center mb-10 relative">
            <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-20 h-20 bg-primary/20 blur-xl rounded-full" />
            <div className="w-14 h-14 bg-gradient-to-br from-primary-container to-secondary-container rounded-xl flex items-center justify-center mb-6 shadow-lg border border-primary/20 animate-float relative z-10">
              <span
                className="material-symbols-outlined text-on-primary-container text-[28px]"
                style={{ fontVariationSettings: "'FILL' 1" }}
              >
                cloud_done
              </span>
            </div>
            <h1 className="font-headline-sm text-[28px] text-white mb-2 tracking-tight glow-text">Aether</h1>
            <p className="font-body-sm text-[14px] text-on-surface-variant/80 tracking-wide font-light">
              PaaS Control Plane
            </p>
          </div>

          <form onSubmit={handleSubmit(submit)} className="space-y-6 flex flex-col w-full" noValidate>
            <div className="space-y-2">
              <label className="font-label-caps text-label-caps text-on-surface-variant/70 block uppercase tracking-widest pl-1" htmlFor="email">
                Account Email
              </label>
              <div className="relative group glow-input rounded-xl transition-all duration-300">
                <div className="absolute inset-0 bg-surface-container-highest/50 rounded-xl blur-sm transition-opacity opacity-0 group-focus-within:opacity-100" />
                <span className="material-symbols-outlined absolute left-4 top-1/2 -translate-y-1/2 text-on-surface-variant/60 group-focus-within:text-primary transition-colors text-[20px] z-10">
                  mail
                </span>
                <input
                  className="w-full bg-surface/50 border border-glass-border rounded-xl py-3.5 pl-[46px] pr-4 font-body-md text-white placeholder:text-on-surface-variant/40 focus:border-transparent focus:ring-0 focus:outline-none transition-all relative z-10 shadow-inner"
                  id="email"
                  placeholder="admin@organization.com"
                  type="email"
                  {...register("email")}
                />
              </div>
              {errors.email?.message && (
                <p className="pl-1 font-body-sm text-[12px] text-error">{errors.email.message}</p>
              )}
            </div>

            <div className="space-y-2">
              <label className="font-label-caps text-label-caps text-on-surface-variant/70 block uppercase tracking-widest pl-1" htmlFor="password">
                Password
              </label>
              <div className="relative group glow-input rounded-xl transition-all duration-300">
                <div className="absolute inset-0 bg-surface-container-highest/50 rounded-xl blur-sm transition-opacity opacity-0 group-focus-within:opacity-100" />
                <span className="material-symbols-outlined absolute left-4 top-1/2 -translate-y-1/2 text-on-surface-variant/60 group-focus-within:text-primary transition-colors text-[20px] z-10">
                  key
                </span>
                <input
                  className="w-full bg-surface/50 border border-glass-border rounded-xl py-3.5 pl-[46px] pr-12 font-code-md text-code-md text-white placeholder:text-on-surface-variant/40 focus:border-transparent focus:ring-0 focus:outline-none transition-all tracking-[0.2em] relative z-10 shadow-inner"
                  id="password"
                  placeholder="••••••••••••"
                  type={showPassword ? "text" : "password"}
                  {...register("password")}
                />
                <button
                  type="button"
                  onClick={() => setShowPassword((v) => !v)}
                  className="absolute right-4 top-1/2 -translate-y-1/2 text-on-surface-variant/60 hover:text-primary transition-colors z-10"
                  aria-label={showPassword ? "Hide password" : "Show password"}
                >
                  <span className="material-symbols-outlined text-[20px]">
                    {showPassword ? "visibility_off" : "visibility"}
                  </span>
                </button>
              </div>
              {errors.password?.message && (
                <p className="pl-1 font-body-sm text-[12px] text-error">{errors.password.message}</p>
              )}
            </div>

            <div className="flex items-center pt-2">
              <input
                className="w-5 h-5 rounded-[6px] bg-surface/80 border-glass-border text-primary focus:ring-primary/50 focus:ring-offset-0 cursor-pointer transition-all shadow-inner"
                id="remember"
                type="checkbox"
                {...register("remember")}
              />
              <label className="ml-3 font-body-sm text-[13px] text-on-surface-variant/80 cursor-pointer select-none font-medium hover:text-white transition-colors" htmlFor="remember">
                Keep session active
              </label>
            </div>

            <button
              type="submit"
              disabled={isSubmitting}
              className="w-full bg-gradient-to-r from-primary to-primary-container hover:from-primary-fixed hover:to-primary-fixed-dim text-on-primary-container font-label-caps text-label-caps uppercase rounded-xl py-4 flex items-center justify-center gap-3 transition-all duration-300 active:scale-[0.98] mt-2 font-bold tracking-widest border border-white/10 disabled:opacity-60"
            >
              <span>{isSubmitting ? "Authenticating…" : "Authenticate"}</span>
              <span className="material-symbols-outlined text-[18px]">arrow_forward</span>
            </button>
          </form>

          <div className="mt-10 pt-8 border-t border-glass-border text-center">
            <p className="font-body-sm text-[13px] text-on-surface-variant/70">
              No account yet?
              <Link to="/onboarding" className="text-primary hover:text-primary-fixed transition-all font-semibold ml-1 hover:drop-shadow-[0_0_8px_rgba(176,198,255,0.5)]">
                Create account
              </Link>
            </p>
          </div>
        </div>

        <div className="mt-8 flex justify-center gap-6 font-body-sm text-[13px] text-on-surface-variant/50">
          <button
            onClick={checkStatus}
            className="hover:text-white transition-all flex items-center gap-2"
          >
            <span
              className={`w-1.5 h-1.5 rounded-full ${
                status === "ok" ? "bg-[#4ade80]" : status === "down" ? "bg-error" : "bg-on-surface-variant/50"
              }`}
            />
            {status === "checking" ? "Checking system…" : status === "ok" ? "System Operational" : "System Unreachable"}
          </button>
          <span className="text-on-surface-variant/30">•</span>
          <span className="text-on-surface-variant/50">v1.0</span>
        </div>
      </main>
    </div>
  );
}

export const Route = createFileRoute("/login/")({
  component: Login,
});
