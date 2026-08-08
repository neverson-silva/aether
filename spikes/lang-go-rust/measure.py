#!/usr/bin/env python3
"""Mede: startup preciso, RSS idle, threads, binário — para Go e Rust mini-core."""
import subprocess, sys, time, os

def measure(path):
    size = os.path.getsize(path)
    t0 = time.perf_counter()
    p = subprocess.Popen([path], stdout=subprocess.PIPE, stderr=subprocess.DEVNULL)
    ready = False
    while time.perf_counter() - t0 < 10:
        line = p.stdout.readline()
        if b"READY" in line:
            ready = True
            break
    startup = (time.perf_counter() - t0) * 1000
    time.sleep(0.3)
    rss_kb = int(subprocess.run(["ps", "-o", "rss=", "-p", str(p.pid)],
                                capture_output=True, text=True).stdout.strip())
    nthreads = subprocess.run(["ps", "-M", "-p", str(p.pid)],
                              capture_output=True, text=True).stdout.count("\n") - 1
    p.terminate()
    try:
        p.wait(timeout=3)
    except subprocess.TimeoutExpired:
        p.kill()
    return {
        "binary_mb": size / 1048576,
        "startup_ms": startup,
        "rss_mb": rss_kb / 1024,
        "threads": nthreads,
        "ready": ready,
    }

for name, path in [("Go", sys.argv[1]), ("Rust", sys.argv[2])]:
    r = measure(path)
    print(f"=== {name} (mini-core) ===")
    print(f"  binário:  {r['binary_mb']:.1f} MB")
    print(f"  startup:  {r['startup_ms']:.0f} ms")
    print(f"  RSS idle: {r['rss_mb']:.1f} MB")
    print(f"  threads:  {r['threads']}")
    print(f"  ready:    {r['ready']}")
