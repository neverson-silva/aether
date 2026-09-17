import { TerminalShell } from "@/components/TerminalShell";
import { useEffect, useState } from "react";
import { useServiceContainers } from "@/hooks";

export function Terminal({ serviceId }: { serviceId: string }) {
  const { data: containers } = useServiceContainers(serviceId);
  const [selected, setSelected] = useState("");
  const active =
    selected ||
    containers?.find((container) => container.status === "running")?.name ||
    "";

  useEffect(() => {
    if (
      containers?.length &&
      !containers.some((container) => container.name === selected)
    ) {
      setSelected(
        containers.find((container) => container.status === "running")?.name ??
          containers[0].name,
      );
    }
  }, [containers, selected]);

  const query = active ? `?container=${encodeURIComponent(active)}` : "";
  return (
    <div className="space-y-md">
      {(containers?.length ?? 0) > 1 && (
        <label className="flex flex-wrap items-center gap-sm text-sm text-on-surface-variant">
          Container
          <select
            value={active}
            onChange={(event) => setSelected(event.target.value)}
            className="rounded-xl border border-outline-variant bg-surface-container px-sm py-xs text-on-surface outline-none transition-[background-color,border-color,box-shadow] duration-150 hover:bg-surface-container-high focus:border-primary focus:ring-2 focus:ring-ring/20"
          >
            {containers?.map((container) => (
              <option key={container.id} value={container.name}>
                {container.name} · {container.status}
              </option>
            ))}
          </select>
        </label>
      )}
      <TerminalShell wsUrl={`/api/v1/ws/terminal/${serviceId}${query}`} />
    </div>
  );
}
