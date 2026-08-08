import { useRef, useState } from "react";
import Editor from "@monaco-editor/react";
import { Button, Card } from "../../../../components/ui";
import { useAppCompose } from "../../../../api/hooks";
import { AppLoading, AppEmptyState } from "../../../../components/ds";

const THEME = {
  base: "vs-dark" as const,
  inherit: true,
  rules: [
    { token: "key", foreground: "b0c6ff" },
    { token: "string", foreground: "9ece6a" },
    { token: "number", foreground: "ff9e64" },
    { token: "comment", foreground: "565f89" },
    { token: "type", foreground: "2ac3de" },
  ],
  colors: { "editor.background": "#0d0d0d" },
};

export function ComposeTab({ appID }: { appID: string }) {
  const { data, isLoading, error } = useAppCompose(appID);
  const [copied, setCopied] = useState(false);
  const [fullscreen, setFullscreen] = useState(false);
  const [wrap, setWrap] = useState(false);
  const editorRef = useRef<{ setValue?: (v: string) => void } | null>(null);

  const compose = data?.compose ?? "";
  const lines = compose ? compose.split("\n").length : 0;

  const copy = async () => {
    await navigator.clipboard.writeText(compose);
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  };

  const download = () => {
    const blob = new Blob([compose], { type: "application/x-yaml" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "docker-compose.yml";
    a.click();
    URL.revokeObjectURL(url);
  };

  if (isLoading) return <AppLoading label="Generating compose from Deployment Spec..." />;
  if (error || !compose) {
    return (
      <AppEmptyState
        icon="deployed_code"
        title="No compose generated"
        description="This service has no deployment yet. Deploy once to capture its docker-compose.yml."
      />
    );
  }

  return (
    <Card className="p-0 overflow-hidden">
      <div className="flex items-center justify-between px-md py-2 border-b border-outline-variant bg-surface-container-low/50">
        <div className="flex items-center gap-2">
          <span className="material-symbols-outlined text-[16px] text-primary">deployed_code</span>
          <span className="font-code-md text-[12px] text-on-surface">docker-compose.yml</span>
          <span className="font-code-md text-[11px] text-on-surface-variant/60">{lines} lines</span>
          <span className="px-1.5 py-0.5 rounded bg-[#4ade80]/10 border border-[#4ade80]/20 font-code-md text-[10px] text-[#4ade80]">
            generated from spec
          </span>
        </div>
        <div className="flex items-center gap-1.5">
          <button onClick={() => setWrap((v) => !v)} className={`px-2 py-1 rounded font-code-md text-[11px] transition-colors ${wrap ? "bg-primary/15 text-primary" : "text-on-surface-variant hover:text-on-surface"}`} title="Toggle word wrap">
            wrap
          </button>
          <button onClick={copy} className="flex items-center gap-1 px-2 py-1 rounded font-code-md text-[11px] text-on-surface-variant hover:text-on-surface transition-colors" title="Copy">
            <span className="material-symbols-outlined text-[14px]">{copied ? "check" : "content_copy"}</span>
            {copied ? "Copied" : "Copy"}
          </button>
          <button onClick={download} className="flex items-center gap-1 px-2 py-1 rounded font-code-md text-[11px] text-on-surface-variant hover:text-on-surface transition-colors" title="Download">
            <span className="material-symbols-outlined text-[14px]">download</span>
            Download
          </button>
          <button onClick={() => window.open(`/api/v1/apps/${appID}/export?runtime=kubernetes`, "_blank")} className="flex items-center gap-1 px-2 py-1 rounded font-code-md text-[11px] text-on-surface-variant hover:text-on-surface transition-colors" title="Export Kubernetes manifest">
            <span className="material-symbols-outlined text-[14px]">workspaces</span>
            Kubernetes
          </button>
          <button onClick={() => window.open(`/api/v1/apps/${appID}/export?runtime=nomad`, "_blank")} className="flex items-center gap-1 px-2 py-1 rounded font-code-md text-[11px] text-on-surface-variant hover:text-on-surface transition-colors" title="Export Nomad job">
            <span className="material-symbols-outlined text-[14px]">terminal</span>
            Nomad
          </button>
          <Button variant="ghost" size="sm" leftIcon={fullscreen ? "fullscreen_exit" : "fullscreen"} onClick={() => setFullscreen((v) => !v)}>
            {fullscreen ? "Exit" : "Fullscreen"}
          </Button>
        </div>
      </div>
      <div className={fullscreen ? "h-[80vh]" : "h-[65vh] min-h-[360px]"}>
        <Editor
          language="yaml"
          value={compose}
          theme="vs-dark"
          options={{
            readOnly: true,
            minimap: { enabled: false },
            fontSize: 12.5,
            fontFamily: '"JetBrains Mono", Menlo, Consolas, monospace',
            lineNumbers: "on",
            folding: true,
            wordWrap: wrap ? "on" : "off",
            scrollBeyondLastLine: false,
            renderWhitespace: "none",
            smoothScrolling: true,
            cursorBlinking: "blink",
            automaticLayout: true,
            scrollbar: { verticalScrollbarSize: 8, horizontalScrollbarSize: 8 },
            contextmenu: true,
            quickSuggestions: false,
            guides: { indentation: false },
          }}
          beforeMount={(monaco) => {
            monaco.editor.defineTheme("aether", THEME);
          }}
          onMount={(editor, monaco) => {
            monaco.editor.setTheme("aether");
            editorRef.current = editor;
          }}
        />
      </div>
    </Card>
  );
}
