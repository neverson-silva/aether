import type { RelationEdge, SchemaSnapshot, SqlTable } from "./types";

function typeCompat(a: string, b: string): number {
  const norm = (s: string) => s.toLowerCase().replace(/\(\d+\)/g, "").replace(/\s*unsigned/g, "").trim();
  const x = norm(a);
  const y = norm(b);
  if (!x || !y) return 0.5;
  if (x === y) return 1;
  if ((x.includes("uuid") && y.includes("uuid")) || (x.includes("int") && y.includes("int"))) return 0.8;
  return 0.2;
}

export interface GraphInput {
  snapshot: SchemaSnapshot;
  // join-pair usage learned from query history: "a<->b" -> score
  history?: Record<string, number>;
}

// Relationship graph. Confidence combines declared foreign keys (1.0), naming
// conventions (x_id → x.id) adjusted by column type compatibility, and learned
// join patterns from query history. Inferred/history edges are never facts.
export function buildGraph(input: GraphInput): { edges: RelationEdge[]; byTable: Record<string, RelationEdge[]> } {
  const { snapshot, history = {} } = input;
  const edges: RelationEdge[] = [];
  const byTable: Record<string, RelationEdge[]> = {};

  const addEdge = (fromTable: string, fromCol: string, toTable: string, toCol: string, confidence: number, source: RelationEdge["source"]) => {
    if (!byTable[fromTable]) byTable[fromTable] = [];
    byTable[fromTable].push({ fromTable, fromCol, toTable, toCol, confidence, source });
  };

  const tables = Object.values(snapshot.tables);
  const tableByName = new Map(tables.map((t) => [t.name.toLowerCase(), t]));

  for (const t of tables) {
    const key = `${t.schema}.${t.name}`.toLowerCase();
    for (const fk of t.foreign_keys) {
      const fromCol = fk.columns[0] ?? "";
      const toCol = fk.ref_columns[0] ?? "";
      const ref = tableByName.get(fk.ref_table.toLowerCase());
      if (!ref) continue;
      const toKey = `${ref.schema}.${ref.name}`.toLowerCase();
      edges.push({ fromTable: key, fromCol, toTable: toKey, toCol, confidence: 1, source: "foreign-key" });
      addEdge(key, fromCol, toKey, toCol, 1, "foreign-key");
      addEdge(toKey, toCol, key, fromCol, 1, "foreign-key");
    }
  }

  for (const t of tables) {
    const key = `${t.schema}.${t.name}`.toLowerCase();
    for (const col of t.columns) {
      const m = /^(.+?)_(id|uuid)$/.exec(col.name.toLowerCase()) ?? /^(.+?)Id$/.exec(col.name);
      if (!m) continue;
      const ref = tableByName.get(m[1]);
      if (!ref) continue;
      const refKey = `${ref.schema}.${ref.name}`.toLowerCase();
      const refPk = ref.columns.find((c) => c.primary_key) ?? ref.columns.find((c) => c.name.toLowerCase() === "id");
      if (!refPk) continue;
      const already = edges.some((e) => e.fromTable === key && e.fromCol === col.name && e.toTable === refKey);
      if (already) continue;
      const tc = typeCompat(col.type, refPk.type);
      let confidence = 0.5 + 0.35 * tc;
      confidence = Math.max(0.1, Math.min(0.9, confidence));
      addEdge(key, col.name, refKey, refPk.name, confidence, "inferred");
      addEdge(refKey, refPk.name, key, col.name, confidence, "inferred");
    }
  }

  // Learned join patterns boost confidence (and create edges when no schema
  // relationship was declared). History is directional-independent.
  for (const [pairKey, score] of Object.entries(history)) {
    const [a, b] = pairKey.split("<->");
    if (!a || !b) continue;
    const boost = Math.min(0.6, score / 50);
    for (const from of [a, b]) {
      const to = from === a ? b : a;
      const existing = (byTable[from] ?? []).find((e) => e.toTable === to);
      if (existing) {
        existing.confidence = Math.min(1, existing.confidence + boost);
      } else {
        addEdge(from, "", to, "", Math.max(0.3, boost), "history");
      }
    }
  }

  return { edges, byTable };
}

export function relatedTables(graph: ReturnType<typeof buildGraph>, tableKey: string): RelationEdge[] {
  return graph.byTable[tableKey] ?? [];
}

export function pairKey(a: string, b: string): string {
  return [a, b].sort().join("<->");
}

export function tableKey(t: SqlTable): string {
  return `${t.schema}.${t.name}`.toLowerCase();
}