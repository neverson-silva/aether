import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useMemo, useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { apiGet } from "../../../api/client";
import {
  useInstallTemplate,
  useProjects,
  useTemplateCategories,
  useTemplatesFiltered,
  useTrendingTemplates,
  useTemplates,
} from "../../../hooks";
import type { TemplateItem } from "../../../hooks";
import { Badge, Button, Dialog, EmptyState, Field, Input, NativeSelect, useToast } from "@aether/design-system";
import { Plus, Star } from "@phosphor-icons/react";
import { Markdown } from "../../../components/Markdown";
import { TechIcon } from "../../../components/TechIcon";

const installSchema = z.object({
  project_id: z.string().min(1, "Project is required"),
});

function fmtCount(n: number): string {
  if (n >= 1000) return `${(n / 1000).toFixed(1)}k`;
  return String(n);
}

function TemplateCard({ t, onClick, fav, onFav }: { t: TemplateItem; onClick: () => void; fav?: boolean; onFav?: () => void }) {
  return (
    <div className="bg-surface-container-low border border-outline-variant rounded-lg p-lg flex flex-col gap-md hover:border-primary/40 transition-colors relative group">
      <button
        onClick={onFav}
        className={`absolute top-2 right-2 transition-colors ${fav ? "text-[#fbbf24]" : "text-on-surface-variant/30 opacity-0 group-hover:opacity-100 hover:text-[#fbbf24]"}`}
        aria-label="Favorite"
      >
        <Star size={18} weight={fav ? "fill" : "regular"} />
      </button>
      <button onClick={onClick} className="text-left flex flex-col gap-md flex-1">
      <div className="flex items-start justify-between pr-5">
        <TechIcon name={t.icon} size={32} className="text-primary" />
        <div className="flex items-center gap-xs">
          {t.featured && <Badge tone="success" dot>Featured</Badge>}
          {t.verified ? (
            <span className="px-1.5 py-0.5 rounded border border-[#4ade80]/30 font-code-md text-code-md text-[#4ade80]">verified</span>
          ) : (
            <span className="px-1.5 py-0.5 rounded border border-outline-variant font-code-md text-code-md text-on-surface-variant">community</span>
          )}
        </div>
      </div>
      <div>
        <h3 className="font-headline-sm text-headline-sm text-on-surface">{t.name}</h3>
        <p className="font-body-sm text-body-sm text-on-surface-variant line-clamp-2">{t.description}</p>
      </div>
      <div className="flex items-center justify-between mt-auto pt-md border-t border-outline-variant/40">
        <span className="font-code-md text-code-md text-on-surface-variant/60">{fmtCount(t.installs)} installs · {t.license || "MIT"}</span>
        <span className="font-label-caps text-label-caps text-primary">Install →</span>
      </div>
      </button>
    </div>
  );
}

function Marketplace() {
  const { data: categories } = useTemplateCategories();
  const { data: trending } = useTrendingTemplates();
  const [category, setCategory] = useState("");
  const [q, setQ] = useState("");
  const [featuredOnly, setFeaturedOnly] = useState(false);
  const [tab, setTab] = useState<"all" | "trending" | "latest" | "favorites">("all");
  const [favs, setFavs] = useState<Set<string>>(() => new Set(JSON.parse(localStorage.getItem("aether.tplfavs") || "[]")));
  const toggleFav = (id: string) => {
    setFavs((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      localStorage.setItem("aether.tplfavs", JSON.stringify([...next]));
      return next;
    });
  };
  const { data: templates } = useTemplatesFiltered({ category: category || undefined, q: q || undefined, featured: featuredOnly || undefined });
  const { data: allTemplates } = useTemplates();
  const latest = useMemo(() => [...(allTemplates ?? [])].sort((a, b) => (b.updated_at ?? "").localeCompare(a.updated_at ?? "")).slice(0, 8), [allTemplates]);
  const favTemplates = useMemo(() => (allTemplates ?? []).filter((t) => favs.has(t.id)), [allTemplates, favs]);
  const { data: projects } = useProjects();
  const navigate = useNavigate();
  const install = useInstallTemplate();
  const { add } = useToast();
  const [target, setTarget] = useState<TemplateItem | null>(null);
  const [detail, setDetail] = useState<TemplateItem | null>(null);
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<z.infer<typeof installSchema>>({
    resolver: zodResolver(installSchema),
    defaultValues: { project_id: "" },
  });

  const openInstall = (t: TemplateItem) => {
    setDetail(null);
    setTarget(t);
  };

  const submit = async (values: z.infer<typeof installSchema>) => {
    try {
      await install.mutateAsync({ id: target!.id, project_id: values.project_id });
      add({ title: "Template installed", description: `Template "${target!.name}" installed as a compose stack.`, tone: "success" });
      setTarget(null);
      const services = await apiGet<{ id: string; name: string; kind: string }[]>(`/api/v1/services?project_id=${encodeURIComponent(values.project_id)}`);
      const service = services.find((item) => item.name === target!.name && item.kind === "compose");
      if (service) navigate({ to: "/apps/$appId", params: { appId: service.id } });
    } catch (err) {
      add({ title: "Installation failed", description: err instanceof Error ? err.message : "Unable to install template.", tone: "error" });
    }
  };

  return (
    <div className="space-y-lg">
      <div><h1 className="text-headline-lg text-foreground">Marketplace</h1><p className="mt-1 text-body-md text-muted-foreground">Curated catalog of one-click apps. Featured, trending and community templates.</p></div>
      <div className="mb-lg flex flex-wrap items-center gap-md border-b border-outline-variant pb-0">
        {([["all", "All"], ["trending", "Trending"], ["latest", "Latest"], ["favorites", "Favorites"]] as const).map(([key, label]) => (
          <button
            key={key}
            onClick={() => setTab(key)}
            className={`px-3 py-2 font-label-caps text-label-caps uppercase border-b-2 -mb-px transition-colors ${tab === key ? "border-primary text-primary" : "border-transparent text-on-surface-variant hover:text-on-surface"}`}
          >
            {label}
          </button>
        ))}
      </div>

      <div className="flex items-center gap-sm flex-wrap">
        <Input
          value={q}
          onChange={(e) => setQ(e.target.value)}
          placeholder="Search templates..."
          className="w-64"
        />
        <NativeSelect
          value={category}
          onChange={(e) => setCategory(e.target.value)}
          className="w-56"
          options={[{ value: "", label: "All categories" }, ...(categories ?? []).map((c) => ({ value: c, label: c }))]}
        />
        <label className="flex items-center gap-sm cursor-pointer select-none">
          <input type="checkbox" checked={featuredOnly} onChange={(e) => setFeaturedOnly(e.target.checked)} className="w-4 h-4 rounded-sm bg-surface border-outline-variant text-primary" />
          <span className="font-body-sm text-body-sm text-on-surface-variant">Featured only</span>
        </label>
      </div>

      {(trending ?? []).length > 0 && tab === "all" && !category && !q && (
        <div>
          <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase mb-md">Trending</h2>
          <div className="grid grid-cols-2 md:grid-cols-3 xl:grid-cols-6 gap-md">
            {trending!.slice(0, 6).map((t) => (
              <button
                key={t.id}
                onClick={() => setDetail(t)}
                className="bg-surface-container-low border border-outline-variant rounded-lg p-md hover:border-primary/40 transition-colors text-center"
              >
                <TechIcon name={t.icon} size={28} className="block mb-sm text-primary" />
                <span className="block font-body-md text-body-md text-on-surface truncate">{t.name}</span>
                <span className="block font-code-md text-code-md text-on-surface-variant/60">{fmtCount(t.installs)} installs</span>
              </button>
            ))}
          </div>
        </div>
      )}

      {tab === "favorites" && favTemplates.length === 0 && (
        <EmptyState title="No favorites yet" description="Star templates to pin them here." className="border-0" />
      )}

      <div>
        <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase mb-md">
          {tab === "favorites" ? "Favorites" : tab === "latest" ? "Latest" : tab === "trending" ? "Trending" : category ? category : "All templates"} ·{" "}
          {(tab === "favorites" ? favTemplates : tab === "latest" ? latest : templates ?? []).length}
        </h2>
        <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 gap-md">
          {(tab === "favorites" ? favTemplates : tab === "latest" ? latest : templates ?? []).map((t) => (
            <TemplateCard key={t.id} t={t} onClick={() => setDetail(t)} fav={favs.has(t.id)} onFav={() => toggleFav(t.id)} />
          ))}
        </div>
        {(tab === "favorites" ? favTemplates : tab === "latest" ? latest : templates ?? []).length === 0 && (
          <p className="font-body-sm text-body-sm text-on-surface-variant">
            {tab === "favorites" ? "Star templates to see them here." : "No templates match your search."}
          </p>
        )}
      </div>

      <Dialog open={!!detail} trigger={<span />} onOpenChange={(open) => { if (!open) setDetail(null); }} title={detail?.name ?? ""}>
        {detail && (
          <div className="space-y-lg">
            <div className="flex items-center gap-md">
              <TechIcon name={detail.icon} size={40} className="text-primary" />
              <div>
                <p className="font-body-sm text-body-sm text-on-surface-variant">{detail.description}</p>
                <p className="font-code-md text-code-md text-on-surface-variant/60 mt-xs">
                  {fmtCount(detail.installs)} installs · v{detail.version} · {detail.license}
                </p>
              </div>
              <div className="flex-1" />
              <Button onClick={() => openInstall(detail)}>
                <Plus size={16} />
                Install
              </Button>
            </div>
            <div className="flex gap-sm flex-wrap">
              {(detail.tags ?? []).map((tag) => (
                <span key={tag} className="px-2 py-0.5 rounded border border-outline-variant font-code-md text-code-md text-on-surface-variant">
                  {tag}
                </span>
              ))}
            </div>
            <div className="bg-surface-container-lowest border border-outline-variant rounded-lg p-md max-h-72 overflow-y-auto sidebar-scroll">
              {detail.readme ? <Markdown text={detail.readme} /> : <p className="font-body-sm text-body-sm text-on-surface-variant">No readme provided for this template.</p>}
            </div>
          </div>
        )}
      </Dialog>

      <Dialog open={!!target} trigger={<span />} onOpenChange={(open) => { if (!open) setTarget(null); }} title={`Install ${target?.name ?? ""}`}>
        <form onSubmit={handleSubmit(submit)} className="space-y-lg" noValidate>
          <Field label="Project" error={errors.project_id?.message}>
            <NativeSelect {...register("project_id")} options={[{ value: "", label: "Select..." }, ...(projects ?? []).map((pr) => ({ value: pr.id, label: pr.name }))]} />
          </Field>
          <div className="flex justify-end gap-md">
            <Button type="button" variant="ghost" onClick={() => setTarget(null)}>
              Cancel
            </Button>
            <Button type="submit" disabled={isSubmitting}>
              {isSubmitting ? "Installing..." : "Install"}
            </Button>
          </div>
        </form>
      </Dialog>
    </div>
  );
}

export const Route = createFileRoute("/_shell/marketplace/")({
  component: Marketplace,
});
