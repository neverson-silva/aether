import { apiGet } from "../api/client";
import { idbGet, idbPut } from "./store";
import { SCHEMA_TTL_MS, VERSION, type SchemaSnapshot, type SqlColumn, type SqlForeignKey } from "./types";

interface CatalogEntry {
  schema: string;
  name: string;
  type: string;
  columns: SqlColumn[];
  foreign_keys: SqlForeignKey[];
}

export async function fetchCatalog(dbId: string): Promise<SchemaSnapshot> {
  const { entries } = await apiGet<{ entries: CatalogEntry[] }>(`/api/v1/databases/${dbId}/studio/catalog`);
  const schemas = Array.from(new Set(entries.map((e) => e.schema)));
  const tables: Record<string, { schema: string; name: string; type: string; columns: SqlColumn[]; foreign_keys: SqlForeignKey[] }> = {};
  for (const e of entries) {
    tables[`${e.schema}.${e.name}`.toLowerCase()] = {
      schema: e.schema,
      name: e.name,
      type: e.type,
      columns: e.columns ?? [],
      foreign_keys: e.foreign_keys ?? [],
    };
  }
  return { version: VERSION, dbId, fetchedAt: Date.now(), schemas, tables };
}

export async function loadSnapshot(dbId: string): Promise<SchemaSnapshot | null> {
  const cached = await idbGet<SchemaSnapshot>("schema-snapshots", dbId);
  if (cached && cached.fetchedAt + SCHEMA_TTL_MS > Date.now()) return cached;
  return null;
}

export async function refreshSnapshot(dbId: string): Promise<SchemaSnapshot> {
  const snap = await fetchCatalog(dbId);
  await idbPut("schema-snapshots", dbId, snap, SCHEMA_TTL_MS);
  return snap;
}

export async function getSnapshot(dbId: string): Promise<SchemaSnapshot> {
  const cached = await loadSnapshot(dbId);
  if (cached) return cached;
  return refreshSnapshot(dbId);
}

export async function invalidateSnapshot(dbId: string): Promise<SchemaSnapshot> {
  return refreshSnapshot(dbId);
}