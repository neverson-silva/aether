-- name: InsertMonitoringSample :exec
INSERT INTO monitoring_samples (
    ts, host_cpu, host_mem, aether_cpu, aether_mem, aether_mem_pct,
    user_cpu, user_mem, user_mem_pct, net_rx, net_tx
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11);

-- name: InsertMonitoringResourceSample :exec
INSERT INTO monitoring_resource_samples (ts, resource_id, name, owner, cpu, mem, net_rx, net_tx)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: ListMonitoringSamples :many
SELECT ts, host_cpu, host_mem, aether_cpu, aether_mem, aether_mem_pct,
       user_cpu, user_mem, user_mem_pct, net_rx, net_tx
FROM monitoring_samples
WHERE ts >= $1 AND ts <= $2
ORDER BY ts ASC;

-- name: ListMonitoringResourceSamples :many
SELECT ts, cpu, mem, net_rx, net_tx
FROM monitoring_resource_samples
WHERE resource_id = $1 AND ts >= $2 AND ts <= $3
ORDER BY ts ASC;

-- name: PurgeMonitoringSamples :exec
DELETE FROM monitoring_samples WHERE ts < $1;

-- name: PurgeMonitoringResourceSamples :exec
DELETE FROM monitoring_resource_samples WHERE ts < $1;
