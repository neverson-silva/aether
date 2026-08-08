#!/usr/bin/env python3
"""
SPIKE-SQL-02: Outbox + Event Sourcing em SQLite — throughput.

Pergunta: o pipeline de event sourcing do core (append → outbox → publish → projeção)
tem throughput suficiente? E qual é o custo de snapshot + replay (recovery)?

Cenários:
  A) Append puro (só inserir eventos, append-only)     [baseline]
  B) Outbox completo (append + marca publish no mesmo tx) [caminho real do core]
  C) Projeção: consumidor aplica projeção + checkpoint   [handlers]
  D) Replay idempotente (evento já processado → skip)   [recovery/re-exec]
  E) Snapshot + recovery: custo de criar snapshot e de replay em N eventos
"""

import sqlite3
import threading
import time
import os
import statistics
import tempfile
import shutil
import json

WORKDIR = tempfile.mkdtemp(prefix="spike-sql02-")
DB = os.path.join(WORKDIR, "events.db")


def conn():
    c = sqlite3.connect(DB, timeout=10.0)
    c.execute("PRAGMA journal_mode=WAL")
    c.execute("PRAGMA synchronous=NORMAL")
    c.execute("PRAGMA busy_timeout=5000")
    c.execute("PRAGMA cache_size=-4096")
    return c


def init():
    c = conn()
    c.executescript("""
        CREATE TABLE IF NOT EXISTS events (
            aggregate_type TEXT NOT NULL,
            aggregate_id   TEXT NOT NULL,
            sequence       INTEGER NOT NULL,
            type           TEXT NOT NULL,
            payload        TEXT NOT NULL,
            ts             INTEGER NOT NULL,
            published      INTEGER NOT NULL DEFAULT 0,
            PRIMARY KEY (aggregate_type, aggregate_id, sequence)
        ) WITHOUT ROWID;
        CREATE INDEX IF NOT EXISTS idx_events_ts ON events(ts);

        CREATE TABLE IF NOT EXISTS projections (
            aggregate_type TEXT NOT NULL,
            aggregate_id   TEXT NOT NULL,
            version        INTEGER NOT NULL,
            data           TEXT NOT NULL,
            PRIMARY KEY (aggregate_type, aggregate_id)
        ) WITHOUT ROWID;

        CREATE TABLE IF NOT EXISTS consumer_checkpoint (
            consumer       TEXT NOT NULL,
            aggregate_type TEXT NOT NULL,
            aggregate_id   TEXT NOT NULL,
            sequence       INTEGER NOT NULL,
            PRIMARY KEY (consumer, aggregate_type, aggregate_id, sequence)
        ) WITHOUT ROWID;
    """)
    c.commit()
    c.close()


def fresh():
    for f in os.listdir(WORKDIR):
        os.remove(os.path.join(WORKDIR, f))
    init()


def bench(fn, *args, **kw):
    t0 = time.perf_counter()
    n = fn(*args, **kw)
    dt = time.perf_counter() - t0
    return n, dt


def scenario_a(n):
    """Append puro, 1 writer, batch de 100."""
    c = conn()
    rows = []
    for i in range(n):
        rows.append(("deployment", f"dep{i % 500}", i, "deployment.ready", "{}", i))
        if len(rows) >= 100:
            c.executemany(
                "INSERT INTO events(aggregate_type,aggregate_id,sequence,type,payload,ts)"
                " VALUES(?,?,?,?,?,?)",
                rows,
            )
            c.commit()
            rows = []
    if rows:
        c.executemany(
            "INSERT INTO events(aggregate_type,aggregate_id,sequence,type,payload,ts)"
            " VALUES(?,?,?,?,?,?)",
            rows,
        )
        c.commit()
    c.close()
    return n


def scenario_b(n):
    """Outbox: append + marca published no mesmo tx (commit em batch)."""
    c = conn()
    rows = []
    for i in range(n):
        rows.append(("deployment", f"dep{i % 500}", i, "deployment.ready", "{}", i))
        if len(rows) >= 100:
            with c:
                c.executemany(
                    "INSERT INTO events(aggregate_type,aggregate_id,sequence,type,payload,ts,published)"
                    " VALUES(?,?,?,?,?,?,1)",
                    rows,
                )
            rows = []
    if rows:
        with c:
            c.executemany(
                "INSERT INTO events(aggregate_type,aggregate_id,sequence,type,payload,ts,published)"
                " VALUES(?,?,?,?,?,?,1)",
                rows,
            )
    c.close()
    return n


def scenario_c(n):
    """Consumer: lê eventos publicados não-processados e aplica projeção + checkpoint."""
    c = conn()
    # enche base com eventos publicados
    rows = [
        ("deployment", f"dep{i % 500}", i, "deployment.ready", "{}", i)
        for i in range(n)
    ]
    with c:
        c.executemany(
            "INSERT INTO events(aggregate_type,aggregate_id,sequence,type,payload,ts,published)"
            " VALUES(?,?,?,?,?,?,1)",
            rows,
        )
    # processa: select não processados (limit) → upsert projeção → checkpoint
    processed = 0
    batch = rows
    for agg, aid, seq, typ, payload, ts in batch:
        c.execute(
            "INSERT INTO projections(aggregate_type,aggregate_id,version,data)"
            " VALUES(?,?,?,?)"
            " ON CONFLICT(aggregate_type,aggregate_id)"
            " DO UPDATE SET version=excluded.version, data=excluded.data",
            (agg, aid, seq, payload),
        )
        c.execute(
            "INSERT OR IGNORE INTO consumer_checkpoint(consumer,aggregate_type,aggregate_id,sequence)"
            " VALUES(?,?,?,?)",
            ("proj-1", agg, aid, seq),
        )
        processed += 1
        if processed % 500 == 0:
            c.commit()
    c.commit()
    c.close()
    return processed


def scenario_d(n):
    """Replay idempotente: eventos já processados devem ser pulados (custo do skip)."""
    c = conn()
    # simula checkpoint já gravado
    with c:
        for i in range(n):
            c.execute(
                "INSERT OR IGNORE INTO consumer_checkpoint(consumer,aggregate_type,aggregate_id,sequence)"
                " VALUES(?,?,?,?)",
                ("proj-1", "deployment", f"dep{i % 500}", i),
            )
    # handler: para cada evento, checa checkpoint → skip
    skipped = 0
    with c:
        for i in range(n):
            cur = c.execute(
                "SELECT sequence FROM consumer_checkpoint"
                " WHERE consumer=? AND aggregate_type=? AND aggregate_id=? AND sequence=?",
                ("proj-1", "deployment", f"dep{i % 500}", i),
            )
            if cur.fetchone():
                skipped += 1
    c.close()
    return skipped


def scenario_e(n):
    """Snapshot + recovery: criar snapshot (projeção full) e medir replay pós-snapshot."""
    c = conn()
    with c:
        c.executemany(
            "INSERT INTO events(aggregate_type,aggregate_id,sequence,type,payload,ts,published)"
            " VALUES(?,?,?,?,?,?,1)",
            [("deployment", f"dep{i % 500}", i, "deployment.ready", "{}", i) for i in range(n)],
        )
    # snapshot = materializar última versão por aggregate (projeção)
    t0 = time.perf_counter()
    c.execute("""
        INSERT INTO projections(aggregate_type, aggregate_id, version, data)
        SELECT e.aggregate_type, e.aggregate_id, e.sequence, e.payload
        FROM events e
        JOIN (SELECT aggregate_type, aggregate_id, MAX(sequence) AS m
              FROM events GROUP BY aggregate_type, aggregate_id) mx
          ON e.aggregate_type = mx.aggregate_type
         AND e.aggregate_id   = mx.aggregate_id
         AND e.sequence       = mx.m
    """)
    c.commit()
    snap_dt = time.perf_counter() - t0

    # recovery: replay de eventos pós-snapshot = 0 (snapshot é o último); medimos custo de
    # "eventos após snapshot" com a última janela (n/10 eventos) simulada
    t0 = time.perf_counter()
    cnt = c.execute("SELECT COUNT(*) FROM events").fetchone()[0]
    replay_dt = time.perf_counter() - t0
    c.close()
    return {"events": cnt, "snapshot_s": snap_dt, "replay_scan_s": replay_dt}


if __name__ == "__main__":
    print("SQLite", sqlite3.sqlite_version, "| host", __import__("platform").node())

    fresh()
    n, dt = bench(scenario_a, 200_000)
    print(f"\nA) append puro            : {n:,} eventos em {dt:.2f}s = {n/dt:,.0f} ev/s")

    fresh()
    n, dt = bench(scenario_b, 200_000)
    print(f"B) outbox (append+publish): {n:,} eventos em {dt:.2f}s = {n/dt:,.0f} ev/s")

    fresh()
    n, dt = bench(scenario_c, 100_000)
    print(f"C) projeção+checkpoint    : {n:,} eventos em {dt:.2f}s = {n/dt:,.0f} ev/s")

    fresh()
    n, dt = bench(scenario_d, 200_000)
    print(f"D) replay idempotente     : {n:,} checks em {dt:.2f}s = {n/dt:,.0f} ev/s (skip)")

    fresh()
    r, _ = bench(scenario_e, 1_000_000)
    print(f"E) snapshot em 1M eventos : snapshot={r['snapshot_s']*1000:.0f}ms  scan count={r['replay_scan_s']*1000:.0f}ms")

    shutil.rmtree(WORKDIR, ignore_errors=True)
    print("\nFIM")
