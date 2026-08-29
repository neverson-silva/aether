-- name: CreatePipeline :one
INSERT INTO pipelines (org_id, app_id, service_id, name, trigger, stages, enabled)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, org_id, app_id, name, trigger, stages, enabled, created_at, service_id;

-- name: GetPipeline :one
SELECT id, org_id, app_id, name, trigger, stages, enabled, created_at, service_id
FROM pipelines
WHERE id = $1;

-- name: ListPipelinesByOrg :many
SELECT id, org_id, app_id, name, trigger, stages, enabled, created_at, service_id
FROM pipelines
WHERE org_id = $1
ORDER BY name;

-- name: DeletePipeline :exec
DELETE FROM pipelines
WHERE id = $1 AND org_id = $2;

-- name: CreatePipelineRun :one
INSERT INTO pipeline_runs (pipeline_id, status, trigger)
VALUES ($1, $2, $3)
RETURNING id, pipeline_id, status, trigger, log, started_at, finished_at;

-- name: FinishPipelineRun :exec
UPDATE pipeline_runs
SET status = $2, log = $3, finished_at = now()
WHERE id = $1;

-- name: ListPipelineRuns :many
SELECT id, pipeline_id, status, trigger, log, started_at, finished_at
FROM pipeline_runs
WHERE pipeline_id = $1
ORDER BY started_at DESC
LIMIT $2;
