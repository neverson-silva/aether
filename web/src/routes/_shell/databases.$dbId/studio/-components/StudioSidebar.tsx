import { useState } from "react";
import { useRouter } from "@tanstack/react-router";
import {
  useDatabaseDetail,
  useStudioSchemas,
  useStudioObjects,
} from "../../../../../hooks";

interface Sel {
  schema: string;
  table: string;
}

const TYPE_ICON: Record<string, string> = {
  table: "table",
  view: "visibility",
  "materialized view": "view_column",
  collection: "view_list",
  key: "key",
  function: "code",
  procedure: "functions",
};

function typeColor(t: string): string {
  switch (t) {
    case "table":
      return "text-on-surface-variant";
    case "view":
    case "materialized view":
      return "text-[#ecc48d]";
    case "function":
    case "procedure":
      return "text-[#82aaff]";
    case "collection":
      return "text-[#f78c6c]";
    case "key":
      return "text-[#89ddff]";
    default:
      return "text-on-surface-variant";
  }
}

export function StudioSidebar({
  dbId,
  selected,
  onSelect,
}: {
  dbId: string;
  selected: Sel | null;
  onSelect: (sel: Sel) => void;
}) {
  const router = useRouter();
  const { data } = useDatabaseDetail(dbId);
  const { data: schemas } = useStudioSchemas(dbId);
  const [expandedSchemas, setExpandedSchemas] = useState<Record<string, boolean>>({});
  const [expandedGroups, setExpandedGroups] = useState<Record<string, boolean>>({});

  const db = data?.database;
  const running = db?.status === "running" || db?.status === "ready";

  const toggleSchema = (s: string) => setExpandedSchemas((p) => ({ ...p, [s]: !p[s] }));
  const toggleGroup = (k: string) => setExpandedGroups((p) => ({ ...p, [k]: !p[k] }));

  return (
    <aside className="w-60 flex-shrink-0 bg-surface-container-lowest border-r border-outline-variant h-full flex flex-col overflow-hidden">
      <div className="px-md py-lg border-b border-outline-variant">
        <div className="flex items-center gap-sm mb-1">
          <span className="material-symbols-outlined text-[16px] text-primary">dns</span>
          <span className="font-code-md text-[12px] font-bold text-on-surface truncate">{db?.name ?? "Database"}</span>
        </div>
        <div className="flex items-center gap-sm">
          <span className="material-symbols-outlined text-[12px] text-on-surface-variant">public</span>
          <span className="font-label-caps text-[10px] text-on-surface-variant">{db?.engine}</span>
          <span className="w-1 h-1 rounded-full bg-outline-variant" />
          <span className="w-1.5 h-1.5 rounded-full bg-[#82aaff]" />
          <span className="font-label-caps text-[10px] text-on-surface-variant capitalize">{running ? "online" : "offline"}</span>
        </div>
      </div>

      <div className="px-sm py-sm border-b border-outline-variant">
        <button
          onClick={() => router.history.back()}
          className="flex items-center gap-sm py-sm px-sm text-on-surface-variant hover:text-on-surface hover:bg-surface-container-high transition-all rounded-md font-label-caps text-label-caps cursor-pointer"
        >
          <span className="material-symbols-outlined text-[18px]">arrow_back</span>
          Back
        </button>
      </div>

      <div className="flex-1 overflow-y-auto sidebar-scroll py-md pr-xs">
        <div className="font-label-caps text-label-caps text-on-surface-variant mb-sm px-sm">Object Explorer</div>

        <div className="flex flex-col text-[12px] font-code-md">
          <div className="flex items-center gap-xs py-1 px-sm text-on-surface">
            <span className="material-symbols-outlined text-[14px] text-on-surface-variant">expand_more</span>
            <span className="material-symbols-outlined text-[14px] text-[#89ddff]">database</span>
            <span className="truncate">{db?.name ?? "Database"}</span>
          </div>

          {(schemas ?? []).map((schema) => {
            const open = expandedSchemas[schema];
            return (
              <div key={schema} className="ml-4 relative">
                <div className="flex items-center gap-xs py-1 px-sm hover:bg-surface-container-high rounded cursor-pointer group" onClick={() => toggleSchema(schema)}>
                  <span className="material-symbols-outlined text-[14px] text-on-surface-variant group-hover:text-primary transition-colors">
                    {open ? "expand_more" : "chevron_right"}
                  </span>
                  <span className="material-symbols-outlined text-[14px] text-[#c792ea]">schema</span>
                  <span className="truncate">{schema}</span>
                </div>

                {open && <SchemaGroup schema={schema} dbId={dbId} expanded={expandedGroups} onToggle={toggleGroup} selected={selected} onSelect={onSelect} />}
              </div>
            );
          })}
        </div>
      </div>

      <div className="mt-auto pt-md border-t border-outline-variant px-sm">
        <a className="flex items-center gap-sm py-sm px-sm text-on-surface-variant hover:text-on-surface hover:bg-surface-container-high transition-all rounded-md font-label-caps text-label-caps cursor-pointer" href="#">
          <span className="material-symbols-outlined text-[16px]">description</span>
          Documentation
        </a>
        <a className="flex items-center gap-sm py-sm px-sm text-on-surface-variant hover:text-on-surface hover:bg-surface-container-high transition-all rounded-md font-label-caps text-label-caps cursor-pointer" href="#">
          <span className="material-symbols-outlined text-[16px]">support_agent</span>
          Support
        </a>
      </div>
    </aside>
  );
}

function SchemaGroup({
  schema,
  dbId,
  expanded,
  onToggle,
  selected,
  onSelect,
}: {
  schema: string;
  dbId: string;
  expanded: Record<string, boolean>;
  onToggle: (k: string) => void;
  selected: Sel | null;
  onSelect: (sel: Sel) => void;
}) {
  const { data: objects } = useStudioObjects(dbId, schema);
  const tables = (objects ?? []).filter((o) => o.type === "table");
  const views = (objects ?? []).filter((o) => o.type === "view" || o.type === "materialized view");
  const funcs = (objects ?? []).filter((o) => o.type === "function" || o.type === "procedure");
  const others = (objects ?? []).filter((o) => !["table", "view", "materialized view", "function", "procedure"].includes(o.type));

  const tableKey = `${schema}:tables`;
  const viewKey = `${schema}:views`;
  const funcKey = `${schema}:functions`;
  const otherKey = `${schema}:others`;

  return (
    <div className="ml-4">
      <GroupRow
        label={`Tables (${tables.length})`}
        icon="table_chart"
        color="text-[#f78c6c]"
        open={!!expanded[tableKey]}
        onToggle={() => onToggle(tableKey)}
      >
        {tables.map((t) => {
          const active = selected?.schema === schema && selected?.table === t.name;
          return (
            <button
              key={t.name}
              onClick={() => onSelect({ schema, table: t.name })}
              className={`flex items-center gap-xs py-1 px-sm rounded cursor-pointer w-full text-left group ${active ? "bg-surface-container-high" : "hover:bg-surface-container-high"}`}
            >
              <span className="material-symbols-outlined text-[14px] opacity-0">chevron_right</span>
              <span className={`material-symbols-outlined text-[14px] ${active ? "text-primary" : "text-on-surface-variant group-hover:text-primary"}`}>table</span>
              <span className={`truncate ${active ? "text-primary" : ""}`}>{t.name}</span>
            </button>
          );
        })}
      </GroupRow>

      <GroupRow label={`Views (${views.length})`} icon="visibility" color="text-[#ecc48d]" open={!!expanded[viewKey]} onToggle={() => onToggle(viewKey)}>
        {views.map((v) => (
          <button
            key={v.name}
            onClick={() => onSelect({ schema, table: v.name })}
            className="flex items-center gap-xs py-1 px-sm hover:bg-surface-container-high rounded cursor-pointer w-full text-left group"
          >
            <span className="material-symbols-outlined text-[14px] opacity-0">chevron_right</span>
            <span className="material-symbols-outlined text-[14px] text-[#ecc48d] group-hover:text-primary">visibility</span>
            <span className="truncate">{v.name}</span>
          </button>
        ))}
      </GroupRow>

      <GroupRow label={`Functions (${funcs.length})`} icon="code" color="text-[#82aaff]" open={!!expanded[funcKey]} onToggle={() => onToggle(funcKey)}>
        {funcs.map((f) => (
          <div key={f.name} className="flex items-center gap-xs py-1 px-sm text-on-surface-variant">
            <span className="material-symbols-outlined text-[14px] opacity-0">chevron_right</span>
            <span className="material-symbols-outlined text-[14px] text-[#82aaff]">code</span>
            <span className="truncate">{f.name}</span>
          </div>
        ))}
      </GroupRow>

      {others.length > 0 && (
        <GroupRow label={`Other (${others.length})`} icon="category" color="text-on-surface-variant" open={!!expanded[otherKey]} onToggle={() => onToggle(otherKey)}>
          {others.map((o) => (
            <div key={o.name} className="flex items-center gap-xs py-1 px-sm text-on-surface-variant">
              <span className="material-symbols-outlined text-[14px] opacity-0">chevron_right</span>
              <span className={`material-symbols-outlined text-[14px] ${typeColor(o.type)}`}>{TYPE_ICON[o.type] ?? "category"}</span>
              <span className="truncate">{o.name}</span>
            </div>
          ))}
        </GroupRow>
      )}
    </div>
  );
}

function GroupRow({
  label,
  icon,
  color,
  open,
  onToggle,
  children,
}: {
  label: string;
  icon: string;
  color: string;
  open: boolean;
  onToggle: () => void;
  children: React.ReactNode;
}) {
  return (
    <div className="mt-0.5">
      <button onClick={onToggle} className="flex items-center gap-xs py-1 px-sm hover:bg-surface-container-high rounded cursor-pointer w-full text-left group">
        <span className="material-symbols-outlined text-[14px] text-on-surface-variant group-hover:text-primary transition-colors">{open ? "expand_more" : "chevron_right"}</span>
        <span className={`material-symbols-outlined text-[14px] ${color}`}>{icon}</span>
        <span className="truncate">{label}</span>
      </button>
      {open && <div className="ml-3 border-l border-outline-variant/40">{children}</div>}
    </div>
  );
}