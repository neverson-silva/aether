import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { apiGet, apiPost, apiDelete } from "../../../api/client";
import {
  Button,
  Card,
  CodeBlock,
  Modal,
  StatusPill,
  Table,
  useToast,
} from "../../../components/ui";

export interface ServerInfo {
  id: string;
  name: string;
  host: string;
  role: string;
  status: string;
  version: string;
  labels: string[];
  cpu_cores: number;
  mem_total_bytes: number;
  load: number;
  last_heartbeat: string;
  created_at: string;
}

function useServers() {
  return useQuery({ queryKey: ["servers"], queryFn: () => apiGet<ServerInfo[]>("/api/v1/servers") });
}

function fmtMem(n: number): string {
  if (n <= 0) return "—";
  const gb = n / (1024 * 1024 * 1024);
  return `${gb.toFixed(1)} GiB`;
}

function Servers() {
  const { data: servers, isLoading } = useServers();
  const qc = useQueryClient();
  const { toast } = useToast();
  const [tokenOpen, setTokenOpen] = useState(false);
  const [token, setToken] = useState<{ token: string; core: string } | null>(null);
  const [name, setName] = useState("");

  const createToken = useMutation({
    mutationFn: () => apiPost<{ token: string; core: string }>("/api/v1/servers/token", { name }),
  });
  const removeServer = useMutation({
    mutationFn: (id: string) => apiDelete(`/api/v1/servers/${id}`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["servers"] });
      toast("Server removed");
    },
  });

  const genToken = () => {
    if (!name.trim()) {
      toast("Give the server a name first", "error");
      return;
    }
    createToken.mutate(undefined, {
      onSuccess: (data) => setToken(data),
      onError: (err) => toast(err.message, "error"),
    });
  };

  return (
    <div className="space-y-lg">
      <div className="flex items-start justify-between">
        <div>
          <h1 className="font-headline-sm text-headline-sm text-on-surface">Servers</h1>
          <p className="font-body-sm text-body-sm text-on-surface-variant">
            Execution nodes managed by Aether agents with mTLS.
          </p>
        </div>
        <Button onClick={() => setTokenOpen(true)}>
          <span className="material-symbols-outlined text-[16px]">add</span>
          Add server
        </Button>
      </div>

      <Card>
        <div className="flex items-center justify-between mb-md">
          <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase">Local server</h2>
          <StatusPill status="connected" pulse />
        </div>
        <p className="font-body-sm text-body-sm text-on-surface-variant">
          The core embeds its own runtime. Deploys without a server assignment run here.
        </p>
      </Card>

      <div className="bg-surface-container-low border border-outline-variant rounded-lg">
        <Table headers={["Name", "Status", "Host", "Version", "CPU", "Memory", "Load", ""]}>
          {isLoading && (
            <tr>
              <td colSpan={8} className="px-sm py-lg font-body-sm text-body-sm text-on-surface-variant text-center">
                Loading…
              </td>
            </tr>
          )}
          {(servers ?? []).map((srv) => (
            <tr key={srv.id} className="hover:bg-surface-container-high transition-colors">
              <td className="px-sm py-2">
                <div className="font-body-md text-body-md text-on-surface">{srv.name}</div>
                <div className="font-code-md text-code-md text-on-surface-variant/60">{srv.id}</div>
              </td>
              <td className="px-sm py-2">
                <StatusPill status={srv.status} pulse={srv.status === "healthy"} />
              </td>
              <td className="px-sm py-2 font-code-md text-code-md text-on-surface-variant">{srv.host}</td>
              <td className="px-sm py-2 font-code-md text-code-md text-on-surface-variant">{srv.version || "—"}</td>
              <td className="px-sm py-2 font-code-md text-code-md text-on-surface-variant">{srv.cpu_cores || "—"}</td>
              <td className="px-sm py-2 font-code-md text-code-md text-on-surface-variant">{fmtMem(srv.mem_total_bytes)}</td>
              <td className="px-sm py-2">
                <div className="flex items-center gap-sm">
                  <div className="w-16 h-1.5 rounded bg-surface-container-high overflow-hidden">
                    <div
                      className="h-full rounded bg-primary"
                      style={{ width: `${Math.min(100, Math.round(srv.load * 20))}%` }}
                    />
                  </div>
                  <span className="font-code-md text-code-md text-on-surface-variant">{srv.load.toFixed(2)}</span>
                </div>
              </td>
              <td className="px-sm py-2 text-right">
                <button
                  onClick={() => removeServer.mutate(srv.id)}
                  className="material-symbols-outlined text-[16px] text-on-surface-variant hover:text-error transition-colors"
                >
                  delete
                </button>
              </td>
            </tr>
          ))}
        </Table>
        {(servers ?? []).length === 0 && !isLoading && (
          <p className="font-body-sm text-body-sm text-on-surface-variant p-sm">
            No agents registered yet.
          </p>
        )}
      </div>

      <Modal open={tokenOpen} onClose={() => { setTokenOpen(false); setToken(null); }} title="Add server">
        {!token ? (
          <div className="space-y-lg">
            <div className="flex gap-md">
              <input
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="server name (e.g. worker-1)"
                className="flex-1 bg-surface-container-low border border-outline-variant rounded-md px-sm py-2 font-body-md text-body-md text-on-surface"
              />
              <Button onClick={genToken} disabled={createToken.isPending}>
                Generate token
              </Button>
            </div>
            <p className="font-body-sm text-body-sm text-on-surface-variant">
              The token is single-use and expires in 24h. Run the command below on the worker server.
            </p>
          </div>
        ) : (
          <div className="space-y-lg">
            <p className="font-body-sm text-body-sm text-on-surface-variant">
              Run this on the remote server to enroll it:
            </p>
            <CodeBlock>{`aether agent --core ${token.core} \\
  --token ${token.token} \\
  --name ${name.trim() || "worker"}`}</CodeBlock>
            <p className="font-body-sm text-body-sm text-on-surface-variant">
              The agent auto-provisions Docker/Podman, reports heartbeats every 5s and starts
              executing remote deploys.
            </p>
            <div className="flex justify-end">
              <Button variant="ghost" onClick={() => { setTokenOpen(false); setToken(null); }}>
                Done
              </Button>
            </div>
          </div>
        )}
      </Modal>
    </div>
  );
}

export const Route = createFileRoute("/_shell/servers/")({
  component: Servers,
});
