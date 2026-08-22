import type { Monaco } from "@monaco-editor/react";
import type { editor } from "monaco-editor";

type CursorPos = { lineNumber: number; column: number };
import type { SchemaSnapshot } from "./types";
import { extractContext } from "./context";
import { getCompletions, type EngineDeps } from "./ranker";

const KIND_MAP: Record<string, string> = {
  keyword: "Keyword",
  table: "Struct",
  column: "Field",
  schema: "Module",
  alias: "Variable",
  join: "Snippet",
  function: "Function",
};

// Registers the SQL completion provider once against the Monaco instance.
export function registerSqlCompletions(monaco: Monaco, engine: () => EngineDeps): void {
  monaco.languages.registerCompletionItemProvider("sql", {
    triggerCharacters: [".", " "],
    provideCompletionItems: async (model: editor.ITextModel, position: CursorPos) => {
      const cursor = model.getOffsetAt(position);
      const ctx = extractContext(model.getValue(), cursor);
      const items = await getCompletions(ctx, engine());
      if (import.meta.env.DEV && items.length > 0) {
        console.debug("[sql-intelligence]", ctx.clause, ctx.prefix, items.slice(0, 6).map((i) => `${i.label}(${i.score.toFixed(2)})[${i.reasons?.join(" ")}]`));
      }
      return {
        suggestions: items.map((it) => ({
          label: it.label,
          kind: (monaco.languages.CompletionItemKind as Record<string, number>)[KIND_MAP[it.kind] ?? "Text"] ?? 9,
          insertText: it.insertText,
          detail: it.detail,
          sortText: String(1000 - it.score).padStart(6, "0"),
          documentation: it.source === "inferred" || it.source === "history" ? `Suggested (${it.source})` : undefined,
        })),
      };
    },
  });
}

export function resolveAliasTarget(snapshot: SchemaSnapshot, alias: string): string | null {
  void snapshot;
  void alias;
  return null;
}