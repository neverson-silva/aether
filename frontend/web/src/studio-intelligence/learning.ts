import { idbAll, idbDelete, idbGet, idbPut } from "./store";
import { MAX_USAGE, USAGE_TTL_MS, type JoinPairRecord, type UsageRecord } from "./types";

const usageKey = (dbId: string, id: string) => `${dbId}:${id}`;

function decay(count: number, lastUsed: number): number {
  const ageDays = (Date.now() - lastUsed) / (24 * 60 * 60 * 1000);
  const recency = Math.max(0.05, Math.pow(0.5, ageDays / 30));
  return count * recency;
}

export async function bumpUsage(dbId: string, id: string): Promise<void> {
  const key = usageKey(dbId, id);
  const rec = await idbGet<UsageRecord>("usage", key);
  const next = rec ? { count: rec.count + 1, lastUsed: Date.now() } : { count: 1, lastUsed: Date.now() };
  await idbPut("usage", key, next, USAGE_TTL_MS);
  await enforceUsageBounds();
}

export async function bumpJoin(dbId: string, a: string, b: string): Promise<void> {
  const key = usageKey(dbId, [a, b].sort().join("<->"));
  const rec = await idbGet<JoinPairRecord>("join-pairs", key);
  const next = rec ? { count: rec.count + 1, lastUsed: Date.now() } : { count: 1, lastUsed: Date.now() };
  await idbPut("join-pairs", key, next, USAGE_TTL_MS);
}

export async function usageScore(dbId: string, id: string): Promise<number> {
  const rec = await idbGet<UsageRecord>("usage", usageKey(dbId, id));
  if (!rec) return 0;
  return decay(rec.count, rec.lastUsed);
}

export async function joinScore(dbId: string, a: string, b: string): Promise<number> {
  const rec = await idbGet<JoinPairRecord>("join-pairs", usageKey(dbId, [a, b].sort().join("<->")));
  if (!rec) return 0;
  return decay(rec.count, rec.lastUsed);
}

async function enforceUsageBounds(): Promise<void> {
  try {
    const all = await idbAll<UsageRecord>("usage");
    if (all.length <= MAX_USAGE) return;
    const sorted = [...all].sort((a, b) => a.value.lastUsed - b.value.lastUsed);
    for (const s of sorted.slice(0, all.length - MAX_USAGE)) {
      await idbDelete("usage", s.key);
    }
  } catch {
    /* ignore */
  }
}