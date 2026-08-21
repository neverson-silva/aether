import type { SchemaSnapshot } from "./types";
import { buildGraph } from "./relationships";
import { bumpJoin, bumpUsage, joinScore, usageScore } from "./learning";
import { idbAll } from "./store";
import { extractPattern } from "./context";
import { getCompletions, type EngineDeps } from "./ranker";
import type { SqlContext, CompletionCandidate } from "./types";

export interface SqlEngine {
  setSnapshot: (snapshot: SchemaSnapshot) => void;
  getDeps: () => EngineDeps;
  getCompletions: (ctx: SqlContext) => Promise<CompletionCandidate[]>;
  recordQuery: (sql: string) => void;
  recordCompletion: (label: string) => void;
  reset: () => void;
}

const emptySnapshot = (dbId: string): SchemaSnapshot => ({ version: 1, dbId, fetchedAt: Date.now(), schemas: [], tables: {} });

// Facade the editor consumes. It owns the schema snapshot, the relationship
// graph (enriched with learned join patterns) and the learning store.
export function createEngine(dbId: string): SqlEngine {
  let snapshot: SchemaSnapshot = emptySnapshot(dbId);
  let graph: ReturnType<typeof buildGraph> = { edges: [], byTable: {} };

  const rebuildGraph = (snap: SchemaSnapshot) => {
    void idbAll<{ count: number; lastUsed: number }>("join-pairs").then((rows) => {
      const history: Record<string, number> = {};
      for (const row of rows) {
        if (row.key.startsWith(dbId + ":")) history[row.key.slice(dbId.length + 1)] = row.value.count;
      }
      graph = buildGraph({ snapshot: snap, history });
    }).catch(() => {
      graph = buildGraph({ snapshot: snap });
    });
  };

  const getDeps = (): EngineDeps => ({
    snapshot,
    dbId,
    graph,
    usage: (id) => usageScore(dbId, id),
    join: (a, b) => joinScore(dbId, a, b),
  });

  const getCompletionsFor = (ctx: SqlContext) => getCompletions(ctx, getDeps());

  const recordQuery = (sql: string) => {
    const p = extractPattern(sql);
    for (const t of p.tables) void bumpUsage(dbId, t);
    for (const [a, b] of p.joins) void bumpJoin(dbId, a, b);
  };

  const recordCompletion = (label: string) => {
    void bumpUsage(dbId, label);
  };

  return {
    setSnapshot: (s) => {
      snapshot = s;
      rebuildGraph(s);
    },
    getDeps,
    getCompletions: getCompletionsFor,
    recordQuery,
    recordCompletion,
    reset: () => {
      snapshot = emptySnapshot(dbId);
      graph = { edges: [], byTable: {} };
    },
  };
}