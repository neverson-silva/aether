import Editor from "@monaco-editor/react";

export function SqlEditor({
  value,
  onChange,
  language = "sql",
  height = "260px",
}: {
  value: string;
  onChange: (v: string) => void;
  language?: string;
  height?: string;
}) {
  return (
    <div className="h-full w-full overflow-hidden">
      <Editor
        height={height}
        language={language}
        value={value}
        onChange={(v) => onChange(v ?? "")}
        theme="vs-dark"
        options={{
          fontSize: 13,
          minimap: { enabled: false },
          scrollBeyondLastLine: false,
          wordWrap: "on",
          automaticLayout: true,
          padding: { top: 8 },
          lineNumbersMinChars: 3,
          renderLineHighlight: "none",
        }}
      />
    </div>
  );
}