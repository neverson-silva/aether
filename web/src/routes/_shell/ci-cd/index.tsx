import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import {
  useApps,
  useCreatePipeline,
  useDeletePipeline,
  usePipelineRuns,
  usePipelines,
  useRunPipeline,
} from "../../../hooks";
import {
  Button,
  Card,
  Field,
  Input,
  Modal,
  Select,
  StatusPill,
  Table,
  useToast,
} from "../../../components/ui";
import { AppPageHeader } from "../../../components/ds";

const stageSchema = z.object({
  name: z.string().min(1),
  image: z.string().min(1),
  commands: z.string(),
});

const schema = z.object({
  name: z.string().min(1, "Name is required"),
  app_id: z.string(),
  trigger: z.enum(["manual", "deploy"]),
  stages: z.array(stageSchema).min(1, "Add at least one stage"),
});

type StageForm = z.input<typeof stageSchema>;
type StageFilled = z.output<typeof stageSchema>;

function stageTone(s: string): string {
  return s === "success" ? "active" : s === "failed" ? "failed" : "building";
}

function PipelineRow({ pipeline }: { pipeline: { id: string; name: string; app_id: string; trigger: string; stages: { name: string; image: string; commands: string[] }[] } }) {
  const runs = usePipelineRuns(pipeline.id);
  const run = useRunPipeline();
  const remove = useDeletePipeline();
  const { toast } = useToast();
  const last = runs.data?.[0];

  return (
    <Card>
      <div className="flex items-center justify-between mb-sm">
        <div className="flex items-center gap-sm">
          <h3 className="font-body-md text-body-md text-on-surface">{pipeline.name}</h3>
          <span className="font-code-md text-code-md text-on-surface-variant/60">trigger: {pipeline.trigger}</span>
        </div>
        <div className="flex items-center gap-sm">
          {last && <StatusPill status={stageTone(last.status)} />}
          <Button onClick={() => run.mutate(pipeline.id, { onSuccess: () => toast("Pipeline started") })}>
            <span className="material-symbols-outlined text-[16px]">play_arrow</span>
            Run
          </Button>
          <button
            onClick={() => remove.mutate(pipeline.id)}
            className="material-symbols-outlined text-[16px] text-on-surface-variant hover:text-error transition-colors"
          >
            delete
          </button>
        </div>
      </div>
      <div className="flex flex-wrap gap-sm mb-md">
        {pipeline.stages.map((st, i) => (
          <span key={i} className="inline-flex items-center gap-xs px-2 py-1 rounded border border-outline-variant font-code-md text-code-md text-on-surface-variant">
            <span className="text-primary">{i + 1}.</span> {st.name}
            <span className="text-on-surface-variant/60">({st.image})</span>
          </span>
        ))}
      </div>
      {last && last.log && (
        <pre className="bg-surface-container-high rounded-md p-sm font-code-md text-code-md text-on-surface-variant max-h-40 overflow-auto whitespace-pre-wrap">
          {last.log.split("\n").slice(-12).join("\n")}
        </pre>
      )}
    </Card>
  );
}

function CiCd() {
  const { data: pipelines } = usePipelines();
  const { data: apps } = useApps();
  const create = useCreatePipeline();
  const { toast } = useToast();
  const [open, setOpen] = useState(false);
  const [stageRows, setStageRows] = useState<StageFilled[]>([{ name: "test", image: "node:22", commands: "npm ci\nnpm test" }]);
  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<z.input<typeof schema>, any, z.output<typeof schema>>({
    resolver: zodResolver(schema),
    defaultValues: { trigger: "manual", stages: stageRows },
  });

  const submit = async (values: z.infer<typeof schema>) => {
    try {
      const validStages = stageRows.filter((st) => st.name && st.image);
      if (validStages.length === 0) {
        toast("Add at least one complete stage", "error");
        return;
      }
      await create.mutateAsync({
        app_id: values.app_id || "",
        name: values.name,
        trigger: values.trigger,
        stages: validStages.map((st) => ({ name: st.name, image: st.image, commands: st.commands.split("\n").filter(Boolean) })),
      });
      setOpen(false);
      reset();
    } catch (err) {
      toast(err instanceof Error ? err.message : "failed", "error");
    }
  };

  return (
    <div className="space-y-lg">
      <AppPageHeader
        title="CI/CD Pipelines"
        description="Stage-based pipelines running in ephemeral containers. Trigger manually or on deploy."
        actions={
          <Button leftIcon="add" onClick={() => setOpen(true)}>
            New pipeline
          </Button>
        }
      />

      <div className="grid grid-cols-1 xl:grid-cols-2 gap-lg">
        {(pipelines ?? []).map((p) => <PipelineRow key={p.id} pipeline={p} />)}
      </div>
      {(pipelines ?? []).length === 0 && (
        <Card>
          <p className="font-body-sm text-body-sm text-on-surface-variant">
            No pipelines yet. Create one to run tests/lint/deploy stages automatically.
          </p>
        </Card>
      )}

      <Modal open={open} onClose={() => setOpen(false)} title="New pipeline" wide>
        <form onSubmit={handleSubmit(submit)} className="space-y-lg" noValidate>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-lg">
            <Field label="Name" hint={errors.name?.message}>
              <Input icon="label" placeholder="ci" {...register("name")} />
            </Field>
            <Field label="App">
              <Select {...register("app_id")}>
                <option value="">(no app — pipeline only)</option>
                {(apps ?? []).map((a) => (
                  <option key={a.id} value={a.id}>{a.name}</option>
                ))}
              </Select>
            </Field>
            <Field label="Trigger">
              <Select {...register("trigger")}>
                <option value="manual">Manual</option>
                <option value="deploy">On deploy (webhook)</option>
              </Select>
            </Field>
          </div>
          <Field label="Stages" hint={errors.stages?.message}>
            <div className="space-y-sm">
              {stageRows.map((st, i) => (
                <div key={i} className="border border-outline-variant rounded-md p-sm space-y-sm">
                  <div className="grid grid-cols-2 gap-sm">
                    <Input placeholder={`stage ${i + 1} name`} value={st.name} onChange={(e) => {
                      const next = [...stageRows];
                      next[i] = { ...next[i], name: e.target.value };
                      setStageRows(next);
                    }} />
                    <Input placeholder="image (e.g. node:20)" value={st.image} onChange={(e) => {
                      const next = [...stageRows];
                      next[i] = { ...next[i], image: e.target.value };
                      setStageRows(next);
                    }} />
                  </div>
                  <textarea
                    className="w-full bg-surface-container-low border border-outline-variant rounded-md px-sm py-2 font-code-md text-code-md text-on-surface min-h-20"
                    placeholder={"commands (one per line)\nnpm ci\nnpm test"}
                    value={st.commands}
                    onChange={(e) => {
                      const next = [...stageRows];
                      next[i] = { ...next[i], commands: e.target.value };
                      setStageRows(next);
                    }}
                  />
                </div>
              ))}
              <Button
                type="button"
                variant="ghost"
                onClick={() => setStageRows([...stageRows, { name: "", image: "", commands: "" }])}
              >
                <span className="material-symbols-outlined text-[16px]">add</span>
                Add stage
              </Button>
            </div>
          </Field>
          <div className="flex justify-end gap-md border-t border-outline-variant pt-lg">
            <Button type="button" variant="ghost" onClick={() => setOpen(false)}>Cancel</Button>
            <Button type="submit">Create</Button>
          </div>
        </form>
      </Modal>
    </div>
  );
}

export const Route = createFileRoute("/_shell/ci-cd/")({
  component: CiCd,
});
