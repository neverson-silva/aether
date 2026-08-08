#!/usr/bin/env python3
"""
SPIKE-SQL-01: SQLite WAL sob carga — contenção de escrita.

Pergunta: o modelo do core (1 processo, fila de escrita, batch commits, transações curtas)
sofre contenção inaceitável no SQLite WAL quando há N escritores concorrentes?

Cenários:
  A) 1 escritor (baseline — o core real serializa escritas via fila)
  B) 2 escritores concorrentes (múltiplos handlers de eventos escrevendo)
  C) 4 escritores concorrentes
  D) 8 escritores concorrentes (pior caso)
  E) baseline com synchronous=FULL vs NORMAL (durabilidade x throughput)

Métricas por cenário:
  - ops/s agregado
  - write latency p50/p95/p99 (ms)
  - lock/contention stalls (busy_timeout acionado)
  - crescimento do WAL (checkpoint)
"""

import sqlite3
import threading
import time
import os
import statistics
import tempfile
import shutil

WORKDIR = tempfile.mkdtemp(prefix="spike-sql01-")
DB = os.path.join(WORKDIR, "state.db")


def make_conn(sync="NORMAL", wal=True, busy_ms=5000, cache_mb=4):
    c = sqlite3.connect(DB, timeout=busy_ms / 1000.0)
    c.execute(f"PRAGMA synchronous={sync}")
    if wal:
        c.execute("PRAGMA journal_mode=WAL")
    c.execute(f"PRAGMA cache_size={-cache_mb * 1024}")
    c.execute("PRAGMA busy_timeout=%d" % busy_ms)
    return c


def init_schema(conn):
    conn.executescript("""
        CREATE TABLE IF NOT EXISTS state (
            aggregate_type TEXT NOT NULL,
            aggregate_id   TEXT NOT NULL,
            version        INTEGER NOT NULL,
            payload        TEXT NOT NULL,
            updated_at     TEXT NOT NULL,
            PRIMARY KEY (aggregate_type, aggregate_id)
        ) WITHOUT ROWID;
        CREATE TABLE IF NOT EXISTS events_outbox (
            id           INTEGER PRIMARY KEY AUTOINCREMENT,
            aggregate_type TEXT NOT NULL,
            aggregate_id   TEXT NOT NULL,
            sequence     INTEGER NOT NULL,
            type         TEXT NOT NULL,
            payload      TEXT NOT NULL,
            created_at   TEXT NOT NULL,
            published    INTEGER NOT NULL DEFAULT 0
        );
        CREATE INDEX IF NOT EXISTS idx_outbox_pub ON events_outbox(published, id);
    """)
    conn.commit()


def writer_worker(conn_factory, aggregate, total, batch, results, stop_flag):
    """Simula handler de domínio: transação curta, upsert de estado + outbox.
    A conexão é criada DENTRO da thread de trabalho (check_same_thread)."""
    conn = conn_factory()
    op = 0
    stalls = 0
    lat = []
    agg = aggregate % 50  # 50 aggregates distintos para reduzir lock de linha
    while op < total and not stop_flag.is_set():
        # batch: vários eventos por transação
        t0 = time.perf_counter()
        try:
            with conn:
                for i in range(batch):
                    ver = op * batch + i
                    conn.execute(
                        "INSERT INTO state(aggregate_type,aggregate_id,version,payload,updated_at)"
                        " VALUES(?,?,?,?,datetime('now'))"
                        " ON CONFLICT(aggregate_type,aggregate_id)"
                        " DO UPDATE SET version=excluded.version, payload=excluded.payload, updated_at=excluded.updated_at",
                        (f"agg{agg}", f"id{agg}-{ver % 20}", ver, "{}"),
                    )
                    conn.execute(
                        "INSERT INTO events_outbox(aggregate_type,aggregate_id,sequence,type,payload,created_at)"
                        " VALUES(?,?,?,?,?,datetime('now'))",
                        (f"agg{agg}", f"id{agg}", ver, "test.event", "{}"),
                    )
        except sqlite3.OperationalError as e:
            if "locked" in str(e).lower() or "busy" in str(e).lower():
                stalls += 1
            else:
                raise
        dt = (time.perf_counter() - t0) * 1000
        lat.append(dt)
        op += 1
    results[aggregate] = {"ops": op * batch, "stalls": stalls, "lat": lat}


def run_scenario(name, n_writers, per_writer_ops, batch, sync="NORMAL", wal=True):
    # fresh db per scenario
    for f in os.listdir(WORKDIR):
        os.remove(os.path.join(WORKDIR, f))
    conn0 = make_conn(sync=sync, wal=wal)
    init_schema(conn0)
    conn0.commit()
    conn0.close()

    conns = [None] * n_writers
    results = {}
    stop = threading.Event()
    factory = lambda: make_conn(sync=sync, wal=wal)
    threads = [
        threading.Thread(
            target=writer_worker,
            args=(factory, i, per_writer_ops, batch, results, stop),
        )
        for i in range(n_writers)
    ]

    t0 = time.perf_counter()
    for t in threads:
        t.start()
    for t in threads:
        t.join()
    elapsed = time.perf_counter() - t0

    total_ops = sum(r["ops"] for r in results.values())
    total_stalls = sum(r["stalls"] for r in results.values())
    all_lat = []
    for r in results.values():
        all_lat.extend(r["lat"])
    all_lat.sort()

    def pct(p):
        if not all_lat:
            return 0.0
        return all_lat[min(len(all_lat) - 1, int(len(all_lat) * p))]

    wal_size = 0
    for f in os.listdir(WORKDIR):
        if f.endswith("-wal"):
            wal_size += os.path.getsize(os.path.join(WORKDIR, f))

    print(f"\n=== {name} ===")
    print(f"  writers={n_writers}  per_writer_ops={per_writer_ops}  batch={batch}")
    print(f"  elapsed={elapsed:.2f}s  total_ops={total_ops}  ops/s={total_ops/elapsed:,.0f}")
    print(f"  write latency: p50={pct(0.50):.2f}ms p95={pct(0.95):.2f}ms p99={pct(0.99):.2f}ms")
    print(f"  contention stalls={total_stalls}  WAL size={wal_size/1024:.0f} KiB")

    # checkpoint (WAL→main) cost
    c = make_conn(sync=sync, wal=wal)
    t0 = time.perf_counter()
    c.execute("PRAGMA wal_checkpoint(PASSIVE)")
    cp = (time.perf_counter() - t0) * 1000
    c.close()
    print(f"  wal_checkpoint(PASSIVE)={cp:.2f}ms")
    return total_ops / elapsed, pct(0.95), total_stalls


if __name__ == "__main__":
    print("SQLite", sqlite3.sqlite_version, "| host", __import__("platform").node())
    print("DB em:", DB)

    scenarios = [
        ("A) 1 writer (fila serializada)", 1, 2000, 1, "NORMAL"),
        ("B) 2 writers, batch=1", 2, 2000, 1, "NORMAL"),
        ("C) 4 writers, batch=1", 4, 2000, 1, "NORMAL"),
        ("D) 8 writers, batch=1", 8, 2000, 1, "NORMAL"),
        ("E1) 4 writers, batch=8 (commit em batch)", 4, 500, 8, "NORMAL"),
        ("F) 1 writer, synchronous=FULL", 1, 2000, 1, "FULL"),
        ("G) 4 writers, synchronous=FULL", 4, 2000, 1, "FULL"),
    ]
    for name, nw, ops, batch, sync in scenarios:
        run_scenario(name, nw, ops, batch, sync=sync)

    shutil.rmtree(WORKDIR, ignore_errors=True)
    print("\nFIM")
