export interface SqlColumn {
  name: string;
  type: string;
  nullable: boolean;
  default?: string | null;
  primary_key?: boolean;
  unique?: boolean;
}

export interface SqlForeignKey {
  name: string;
  columns: string[];
  ref_table: string;
  ref_columns: string[];
}

export interface SqlTable {
  schema: string;
  name: string;
  type: string;
  columns: SqlColumn[];
  foreign_keys: SqlForeignKey[];
}

export interface SchemaSnapshot {
  version: number;
  dbId: string;
  fetchedAt: number;
  schemas: string[];
  tables: Record<string, SqlTable>;
}

export type SqlClause =
  | "select"
  | "from"
  | "join"
  | "on"
  | "where"
  | "group"
  | "having"
  | "order"
  | "limit"
  | "insert"
  | "update"
  | "set"
  | "unknown";

export interface SqlContext {
  sql: string;
  cursor: number;
  prefix: string;
  clause: SqlClause;
  aliases: Record<string, string>;
  tablesInScope: string[];
  afterDot: boolean;
  dotTable: string | null;
}

export interface CompletionCandidate {
  label: string;
  insertText: string;
  kind: "keyword" | "schema" | "table" | "column" | "alias" | "function" | "join";
  detail: string;
  score: number;
  source: "schema" | "foreign-key" | "inferred" | "history" | "keyword";
  reasons?: string[];
}

export interface RelationEdge {
  fromTable: string;
  fromCol: string;
  toTable: string;
  toCol: string;
  confidence: number;
  source: "foreign-key" | "inferred" | "history";
}

export interface UsageRecord {
  count: number;
  lastUsed: number;
}

export interface QueryPatternRecord {
  key: string;
  tables: string[];
  joins: [string, string][];
  count: number;
  lastUsed: number;
}

export interface JoinPairRecord {
  key: string;
  count: number;
  lastUsed: number;
}

export const VERSION = 1;
export const SCHEMA_TTL_MS = 7 * 24 * 60 * 60 * 1000;
export const USAGE_TTL_MS = 90 * 24 * 60 * 60 * 1000;
export const MAX_HISTORY = 200;
export const MAX_USAGE = 5000;
