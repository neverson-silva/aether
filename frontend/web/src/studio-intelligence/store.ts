const DB_NAME = "aether-sql-intelligence";
const DB_VERSION = 1;

const STORES = ["schema-snapshots", "usage", "join-pairs", "patterns"] as const;

function openDB(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const req = indexedDB.open(DB_NAME, DB_VERSION);
    req.onupgradeneeded = () => {
      const db = req.result;
      for (const s of STORES) {
        if (!db.objectStoreNames.contains(s)) {
          db.createObjectStore(s, { keyPath: "key" });
        }
      }
    };
    req.onsuccess = () => resolve(req.result);
    req.onerror = () => reject(req.error);
  });
}

// Thin, versioned IndexedDB wrapper. Every read/write degrades to in-memory
// (a no-op cache) when IndexedDB is unavailable so the editor always works.
let memory = new Map<string, { value: unknown; ttl: number }>();
let idb: Promise<IDBDatabase> | null = null;

function db(): Promise<IDBDatabase> {
  if (!idb) idb = openDB().catch(() => null as unknown as IDBDatabase);
  return idb;
}

function tx<T>(store: string, mode: IDBTransactionMode, fn: (s: IDBObjectStore) => IDBRequest): Promise<T> {
  return db().then((d) => {
    if (!d) {
      return Promise.reject(new Error("indexeddb unavailable"));
    }
    return new Promise<T>((resolve, reject) => {
      const t = d.transaction(store, mode);
      const req = fn(t.objectStore(store));
      req.onsuccess = () => resolve(req.result as T);
      req.onerror = () => reject(req.error);
    });
  });
}

export async function idbGet<T>(store: string, key: string): Promise<T | null> {
  try {
    const row = await tx<{ key: string; value: T; ttl: number } | undefined>(store, "readonly", (s) => s.get(key));
    if (!row) return null;
    if (row.ttl > 0 && Date.now() > row.ttl) {
      await idbDelete(store, key);
      return null;
    }
    return row.value;
  } catch {
    const row = memory.get(store + ":" + key);
    if (!row) return null;
    if (row.ttl > 0 && Date.now() > row.ttl) {
      memory.delete(store + ":" + key);
      return null;
    }
    return row.value as T;
  }
}

export async function idbPut<T>(store: string, key: string, value: T, ttlMs = 0): Promise<void> {
  try {
    await tx(store, "readwrite", (s) => s.put({ key, value, ttl: ttlMs ? Date.now() + ttlMs : 0 }));
  } catch {
    memory.set(store + ":" + key, { value, ttl: ttlMs ? Date.now() + ttlMs : 0 });
  }
}

export async function idbDelete(store: string, key: string): Promise<void> {
  try {
    await tx(store, "readwrite", (s) => s.delete(key));
  } catch {
    memory.delete(store + ":" + key);
  }
}

export async function idbAll<T>(store: string): Promise<{ key: string; value: T }[]> {
  try {
    const rows = await tx<{ key: string; value: T; ttl: number }[]>(store, "readonly", (s) => s.getAll());
    const now = Date.now();
    return rows
      .filter((r) => r.ttl === 0 || r.ttl > now)
      .map((r) => ({ key: r.key, value: r.value }));
  } catch {
    const now = Date.now();
    return Array.from(memory.entries())
      .filter(([k]) => k.startsWith(store + ":"))
      .filter(([, v]) => v.ttl === 0 || v.ttl > now)
      .map(([k, v]) => ({ key: k.slice(store.length + 1), value: v.value as T }));
  }
}

export async function idbCount(store: string): Promise<number> {
  try {
    return await tx<number>(store, "readonly", (s) => s.count());
  } catch {
    return Array.from(memory.keys()).filter((k) => k.startsWith(store + ":")).length;
  }
}