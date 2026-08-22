import Editor, { type OnMount } from "@monaco-editor/react";
import type { editor } from "monaco-editor";
import { registerSqlCompletions } from "../../../studio-intelligence/completion";
import type { SqlEngine } from "../../../studio-intelligence/engine";

export type SqlEditorApi = {
  getSelection: () => string | null;
  getCursorOffset: () => number;
};

export function SqlEditor({
  value,
  onChange,
  language = "sql",
  height = "260px",
  apiRef,
  engineRef,
  monacoRef,
  runHandlerRef,
}: {
  value: string;
  onChange: (v: string) => void;
  language?: string;
  height?: string;
  apiRef?: React.MutableRefObject<SqlEditorApi | null>;
  engineRef?: React.MutableRefObject<SqlEngine | null>;
  monacoRef?: React.MutableRefObject<Parameters<OnMount>[0] | null>;
  runHandlerRef?: React.MutableRefObject<(() => void) | null>;
}) {
  const handleMount: OnMount = (editor, monaco) => {
    editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.Enter, () => runHandlerRef?.current?.());
    if (apiRef) {
      apiRef.current = {
        getSelection: () => {
          const sel = editor.getSelection();
          if (!sel) return null;
          const model = editor.getModel();
          if (!model) return null;
          const text = model.getValueInRange(sel);
          return text && text.trim() ? text : null;
        },
        getCursorOffset: () => {
          const pos = editor.getPosition();
          const model = editor.getModel();
          if (!pos || !model) return 0;
          return model.getOffsetAt(pos);
        },
      };
    }
    if (monacoRef) monacoRef.current = monaco;
    if (engineRef?.current) {
      registerSqlCompletions(monaco, () => engineRef.current!.getDeps());
    }
  };

  return (
    <div className="h-full w-full overflow-hidden">
      <Editor
        height={height}
        language={language}
        value={value}
        onChange={(v) => onChange(v ?? "")}
        theme="vs-dark"
        onMount={handleMount}
        options={{
          fontSize: 13,
          minimap: { enabled: false },
          scrollBeyondLastLine: false,
          wordWrap: "on",
          automaticLayout: true,
          padding: { top: 8 },
          lineNumbersMinChars: 3,
          renderLineHighlight: "all",
        }}
      />
    </div>
  );
}