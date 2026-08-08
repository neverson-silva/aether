use std::collections::HashMap;
use std::net::TcpListener;
use std::process;
use std::sync::{Arc, Condvar, Mutex};
use std::thread;
use std::time::Duration;

fn main() {
    // 12 workers ociosos bloqueados em Condvar (equivalentes a goroutines do Go)
    let pair = Arc::new((Mutex::new(false), Condvar::new()));
    let mut handles = Vec::new();
    for _ in 0..12 {
        let pair = Arc::clone(&pair);
        handles.push(thread::spawn(move || {
            let (lock, cvar) = &*pair;
            let mut stop = lock.lock().unwrap();
            while !*stop {
                let (guard, _t) = cvar.wait_timeout(stop, Duration::from_secs(3600)).unwrap();
                stop = guard;
            }
        }));
    }

    // cache LRU-ish em memória (10k entradas) — como o mem-lru do core
    let mut cache: HashMap<u64, String> = HashMap::with_capacity(10_000);
    for i in 0..10_000u64 {
        cache.insert(i, "x".repeat(16));
    }
    std::hint::black_box(&cache);

    let listener = TcpListener::bind("127.0.0.1:18080").unwrap();
    println!("READY {}", process::id());
    drop(listener);

    for h in handles {
        let _ = h;
    }
    loop {
        thread::sleep(Duration::from_secs(3600));
    }
}
