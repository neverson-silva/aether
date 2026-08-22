import type { CompletionCandidate, RelationEdge, SchemaSnapshot, SqlContext, SqlTable } from "./types";
import { buildGraph, relatedTables } from "./relationships";
import { joinScore, usageScore } from "./learning";

// Deterministic, composable ranking. Every candidate carries reasons (the
// "why"), kept internal for debugging. Higher score wins.
export interface EngineDeps {
  snapshot: SchemaSnapshot;
  dbId: string;
  graph: ReturnType<typeof buildGraph>;
  usage: (id: string) => Promise<number>;
  join: (a: string, b: string) => Promise<number>;
}

interface ScoreParts {
  lexical: number;
  context: number;
  relationship: number;
  learning: number;
}

function lexical(prefix: string, label: string): number {
  const l = label.toLowerCase();
  const p = prefix.toLowerCase();
  if (!p) return 0.5;
  if (l === p) return 1;
  if (l.startsWith(p)) return 0.9;
  if (new RegExp(`\\b${p.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")}`).test(l)) return 0.7;
  if (l.includes(p)) return 0.5;
  return 0;
}

function finalScore(parts: ScoreParts): number {
  return parts.lexical * 100 + parts.context * 10 + parts.relationship * 5 + parts.learning;
}

function reasons(p: ScoreParts, src: string): string[] {
  const out: string[] = [];
  if (p.relationship > 0) out.push(`${src === "foreign-key" ? "FK relationship" : src === "history" ? "learned join" : "inferred relationship"}: +${(p.relationship * 5).toFixed(2)}`);
  if (p.context > 0) out.push(`context relevance: +${(p.context * 10).toFixed(1)}`);
  if (p.learning > 0) out.push(`usage/history: +${p.learning.toFixed(2)}`);
  if (p.lexical > 0.5) out.push(`lexical match: +${(p.lexical * 100).toFixed(0)}`);
  return out;
}

function tableRef(t: SqlTable): string {
  return `${t.schema}.${t.name}`.toLowerCase();
}

function inScopeKeys(ctx: SqlContext, snapshot: SchemaSnapshot): string[] {
  const defaultSchema = snapshot.schemas[0] ?? "public";
  return ctx.tablesInScope.map((t) => (t.includes(".") ? t : `${defaultSchema}.${t}`).toLowerCase());
}

export async function getCompletions(ctx: SqlContext, deps: EngineDeps): Promise<CompletionCandidate[]> {
  const out: CompletionCandidate[] = [];
  const snapshot = deps.snapshot;
  const graph = deps.graph;
  const usage = deps.usage;
  const scopeKeys = new Set(inScopeKeys(ctx, snapshot));
  const aliasTarget = ctx.aliases[ctx.dotTable ?? ""];

  // Column completion after alias/table qualifier: u.| or table.|
  if (ctx.afterDot && ctx.dotTable) {
    const defaultSchema = snapshot.schemas[0] ?? "public";
    const qualified = (ctx.dotTable.includes(".") ? ctx.dotTable : `${defaultSchema}.${ctx.dotTable}`).toLowerCase();
    const byAlias = aliasTarget ? Object.values(snapshot.tables).find((t) => t.name.toLowerCase() === aliasTarget) : undefined;
    const table =
      (aliasTarget ? byAlias : undefined) ??
      snapshot.tables[qualified] ??
      snapshot.tables[aliasTarget?.toLowerCase() ?? ""] ??
      Object.values(snapshot.tables).find((t) => t.name.toLowerCase() === ctx.dotTable);
    if (table) {
      for (const col of table.columns) {
        const label = col.name;
        const p: ScoreParts = { lexical: lexical(ctx.prefix, label), context: 1, relationship: 0, learning: await usage(`${tableRef(table)}.${col.name}`) };
        out.push({
          label,
          insertText: col.name,
          kind: "column",
          detail: `${table.schema}.${table.name} · ${col.type}${col.primary_key ? " · PK" : ""}`,
          score: finalScore(p),
          source: "schema",
          reasons: reasons(p, "schema"),
        });
      }
    }
    return sortUnique(out);
  }

  const clause = ctx.clause;

  if (clause === "from" || clause === "join") {
    // Related tables first (graph + history), then the rest by learning/lexical.
    const related = new Map<string, RelationEdge>();
    for (const key of scopeKeys) {
      for (const e of relatedTables(graph, key)) {
        const prev = related.get(e.toTable);
        if (!prev || e.confidence > prev.confidence) related.set(e.toTable, e);
      }
    }
    for (const t of Object.values(snapshot.tables)) {
      const key = tableRef(t);
      const edge = related.get(key);
      const rel = edge ? edge.confidence : 0;
      const learning = await usage(key) + (scopeKeys.size ? await deps.join([...scopeKeys][0], key) : 0);
      const p: ScoreParts = {
        lexical: lexical(ctx.prefix, t.name),
        context: clause === "join" ? 0.5 : 1,
        relationship: rel,
        learning,
      };
      out.push({
        label: t.name,
        insertText: t.name,
        kind: "table",
        detail: `${t.schema} · ${t.type}`,
        score: finalScore(p),
        source: edge ? edge.source : learning > 0 ? "history" : "schema",
        reasons: reasons(p, edge ? edge.source : "schema"),
      });
    }
  } else if (clause === "select" || clause === "where" || clause === "group" || clause === "having" || clause === "order") {
    for (const [alias, tableName] of Object.entries(ctx.aliases)) {
      const table = Object.values(snapshot.tables).find((t) => t.name.toLowerCase() === tableName);
      if (!table) continue;
      for (const col of table.columns) {
        const label = `${alias}.${col.name}`;
        const p: ScoreParts = { lexical: lexical(ctx.prefix, label), context: 1, relationship: 0, learning: await usage(`${tableRef(table)}.${col.name}`) };
        out.push({
          label,
          insertText: label,
          kind: "column",
          detail: `${table.schema}.${table.name} · ${col.type}`,
          score: finalScore(p),
          source: "schema",
          reasons: reasons(p, "schema"),
        });
      }
    }
  } else if (clause === "on") {
    for (const [alias, tableName] of Object.entries(ctx.aliases)) {
      const table = Object.values(snapshot.tables).find((t) => t.name.toLowerCase() === tableName);
      if (!table) continue;
      for (const col of table.columns) {
        const label = `${alias}.${col.name}`;
        out.push({
          label,
          insertText: label,
          kind: "column",
          detail: `${table.schema}.${table.name} · ${col.type}`,
          score: finalScore({ lexical: lexical(ctx.prefix, label), context: 1, relationship: 0, learning: 0 }),
          source: "schema",
          reasons: ["on-clause column"],
        });
      }
    }
  }

  out.push(...keywords(clause, ctx.prefix));
  return sortUnique(out);
}

function keywords(clause: SqlContext["clause"], prefix: string): CompletionCandidate[] {
  const base = ["SELECT", "FROM", "WHERE", "JOIN", "LEFT", "RIGHT", "INNER", "ON", "GROUP", "ORDER", "HAVING", "LIMIT", "INSERT", "UPDATE", "DELETE", "AS", "AND", "OR", "NOT", "NULL", "IN", "EXISTS", "BETWEEN", "LIKE", "COUNT", "SUM", "AVG", "MIN", "MAX", "DISTINCT", "CASE", "WHEN", "THEN", "ELSE", "END"];
  return base
    .flatMap((k) => {
      const lx = lexical(prefix, k);
      if (lx <= 0) return [];
      const p: ScoreParts = { lexical: lx, context: clause === "unknown" ? 0.5 : 0.1, relationship: 0, learning: 0 };
      const c: CompletionCandidate = {
        label: k,
        insertText: k,
        kind: "keyword",
        detail: "SQL keyword",
        score: finalScore(p),
        source: "keyword",
        reasons: reasons(p, "keyword"),
      };
      return [c];
    });
}

function sortUnique(items: CompletionCandidate[]): CompletionCandidate[] {
  const seen = new Map<string, CompletionCandidate>();
  for (const it of items) {
    const prev = seen.get(it.label);
    if (!prev || it.score > prev.score) seen.set(it.label, it);
  }
  return Array.from(seen.values()).sort((a, b) => b.score - a.score).slice(0, 50);
}

export { finalScore, lexical };