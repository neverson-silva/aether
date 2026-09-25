package infra

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"aether/internal/modules/services/domain"
	"aether/internal/platform/database"
	"aether/internal/platform/druntime"
	"aether/internal/platform/druntime/adapter"
	"aether/internal/platform/druntime/events"
	"aether/internal/platform/druntime/queue"
	"aether/internal/platform/worker"
)

func TestLifecycleStorePersistsCommandsForEveryServiceKind(t *testing.T) {
	ctx := context.Background()
	pool := lifecycleTestPool(t, ctx)
	store := NewLifecycleStore(pool)
	orgID, projectID := createLifecycleTestScope(t, ctx, pool)
	services := []struct {
		kind   domain.Kind
		status string
	}{
		{kind: domain.KindApp, status: "stopped"},
		{kind: domain.KindCompose, status: "stopped"},
		{kind: domain.KindDatabase, status: "stopped"},
	}
	for _, item := range services {
		t.Run(string(item.kind), func(t *testing.T) {
			specID, serviceID := createLifecycleTestService(t, ctx, pool, orgID, projectID, item.kind, item.status)
			resolvedID, err := store.ResolveServiceID(ctx, specID, orgID, item.kind)
			if err != nil || resolvedID != serviceID {
				t.Fatalf("resolve service identity: got %s and %v", resolvedID, err)
			}
			startRequest := domain.LifecycleRequest{ServiceID: serviceID, SpecID: specID, OrgID: orgID, Kind: item.kind, Action: domain.LifecycleStart}
			started, err := store.RequestLifecycle(ctx, startRequest)
			if err != nil {
				t.Fatalf("accept start command: %v", err)
			}
			if started.Status != "accepted" || started.ID == uuid.Nil {
				t.Fatalf("unexpected accepted operation: %+v", started)
			}
			assertLifecycleServiceStatus(t, ctx, pool, serviceID, "starting")
			assertLifecycleOutbox(t, ctx, pool, started.ID, serviceID)
			if _, err := pool.Exec(ctx, `UPDATE services SET status = 'running' WHERE id = $1`, serviceID); err != nil {
				t.Fatalf("simulate service status projection during startup: %v", err)
			}
			duplicate, err := store.RequestLifecycle(ctx, startRequest)
			if err != nil || duplicate.ID != started.ID {
				t.Fatalf("duplicate start did not reuse active operation: %+v, %v", duplicate, err)
			}
			assertLifecycleServiceStatus(t, ctx, pool, serviceID, "starting")
			stopRequest := startRequest
			stopRequest.Action = domain.LifecycleStop
			if _, err := store.RequestLifecycle(ctx, stopRequest); !errors.Is(err, domain.ErrLifecycleConflict) {
				t.Fatalf("opposite command during start should conflict, got %v", err)
			}
			claimed, didClaim, err := store.ClaimLifecycleOperation(ctx, started.ID)
			if err != nil || !didClaim || claimed.Attempts != 1 {
				t.Fatalf("claim start operation: %+v, claimed=%t, err=%v", claimed, didClaim, err)
			}
			if _, err := pool.Exec(ctx, `UPDATE services SET status = 'degraded' WHERE id = $1`, serviceID); err != nil {
				t.Fatalf("simulate a stale status projection before start confirmation: %v", err)
			}
			completed, err := store.CompleteLifecycleOperation(ctx, started.ID, claimed.Attempts, "running", "")
			if err != nil || completed.Status != "succeeded" {
				t.Fatalf("complete start operation: %+v, %v", completed, err)
			}
			assertLifecycleServiceStatus(t, ctx, pool, serviceID, "running")
			stopped, err := store.RequestLifecycle(ctx, stopRequest)
			if err != nil || stopped.Status != "accepted" {
				t.Fatalf("accept stop command: %+v, %v", stopped, err)
			}
			assertLifecycleServiceStatus(t, ctx, pool, serviceID, "stopping")
			if _, err := pool.Exec(ctx, `UPDATE services SET status = 'stopped' WHERE id = $1`, serviceID); err != nil {
				t.Fatalf("simulate service status projection during shutdown: %v", err)
			}
			duplicate, err = store.RequestLifecycle(ctx, stopRequest)
			if err != nil || duplicate.ID != stopped.ID {
				t.Fatalf("duplicate stop did not reuse active operation: %+v, %v", duplicate, err)
			}
			assertLifecycleServiceStatus(t, ctx, pool, serviceID, "stopping")
			claimed, didClaim, err = store.ClaimLifecycleOperation(ctx, stopped.ID)
			if err != nil || !didClaim {
				t.Fatalf("claim stop operation: claimed=%t, err=%v", didClaim, err)
			}
			if _, err := pool.Exec(ctx, `UPDATE services SET status = 'running' WHERE id = $1`, serviceID); err != nil {
				t.Fatalf("simulate a stale status projection before stop confirmation: %v", err)
			}
			completed, err = store.CompleteLifecycleOperation(ctx, stopped.ID, claimed.Attempts, "stopped", "")
			if err != nil || completed.Status != "succeeded" {
				t.Fatalf("complete stop operation: %+v, %v", completed, err)
			}
			assertLifecycleServiceStatus(t, ctx, pool, serviceID, "stopped")
		})
	}
}

func TestLifecycleStoreSerializesConcurrentCommandsAndRecoversStaleOperations(t *testing.T) {
	ctx := context.Background()
	pool := lifecycleTestPool(t, ctx)
	store := NewLifecycleStore(pool)
	orgID, projectID := createLifecycleTestScope(t, ctx, pool)
	_, serviceID := createLifecycleTestService(t, ctx, pool, orgID, projectID, domain.KindApp, "running")
	request := domain.LifecycleRequest{ServiceID: serviceID, OrgID: orgID, Kind: domain.KindApp, Action: domain.LifecycleStop}
	const requests = 12
	operations := make(chan *domain.LifecycleOperation, requests)
	errorsFound := make(chan error, requests)
	var wait sync.WaitGroup
	for range requests {
		wait.Add(1)
		go func() {
			defer wait.Done()
			operation, err := store.RequestLifecycle(ctx, request)
			if err != nil {
				errorsFound <- err
				return
			}
			operations <- operation
		}()
	}
	wait.Wait()
	close(operations)
	close(errorsFound)
	for err := range errorsFound {
		t.Fatalf("concurrent stop command failed: %v", err)
	}
	var operationID uuid.UUID
	for operation := range operations {
		if operationID == uuid.Nil {
			operationID = operation.ID
		}
		if operation.ID != operationID {
			t.Fatalf("concurrent requests created multiple operations: %s and %s", operationID, operation.ID)
		}
	}
	var operationCount, outboxCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM service_lifecycle_operations WHERE service_id = $1 AND status IN ('accepted', 'running')`, serviceID).Scan(&operationCount); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM outbox_events WHERE id = $1 AND topic = 'service-operations'`, operationID).Scan(&outboxCount); err != nil {
		t.Fatal(err)
	}
	if operationCount != 1 || outboxCount != 1 {
		t.Fatalf("expected one active operation and outbox command, got %d and %d", operationCount, outboxCount)
	}
	if _, err := pool.Exec(ctx, `UPDATE service_lifecycle_operations SET lease_until = now() - interval '1 second' WHERE id = $1`, operationID); err != nil {
		t.Fatal(err)
	}
	if err := store.RecoverLifecycleOperations(ctx, 16); err != nil {
		t.Fatalf("recover stale lifecycle operation: %v", err)
	}
	operation, err := store.GetLifecycleOperation(ctx, operationID)
	if err != nil || operation.Status != "accepted" {
		t.Fatalf("stale lifecycle operation was not re-accepted: %+v, %v", operation, err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM outbox_events WHERE topic = 'service-operations' AND aggregate_id = $1`, serviceID.String()).Scan(&outboxCount); err != nil {
		t.Fatal(err)
	}
	if outboxCount != 2 {
		t.Fatalf("expected recovery to enqueue a second durable command, got %d", outboxCount)
	}
}

func TestLifecycleStoreAllowsStopWhenDeploymentStatusProjectionIsStale(t *testing.T) {
	ctx := context.Background()
	pool := lifecycleTestPool(t, ctx)
	store := NewLifecycleStore(pool)
	orgID, projectID := createLifecycleTestScope(t, ctx, pool)
	specID, serviceID := createLifecycleTestService(t, ctx, pool, orgID, projectID, domain.KindApp, "deploying")
	if _, err := pool.Exec(ctx, `INSERT INTO deployments (id, app_id, service_id, number, status) VALUES ($1, $2, $3, 1, 'ready')`, uuid.New(), specID, serviceID); err != nil {
		t.Fatalf("create completed deployment: %v", err)
	}

	operation, err := store.RequestLifecycle(ctx, domain.LifecycleRequest{ServiceID: serviceID, SpecID: specID, OrgID: orgID, Kind: domain.KindApp, Action: domain.LifecycleStop})
	if err != nil {
		t.Fatalf("accept stop after completed deployment: %v", err)
	}
	if operation.Status != "accepted" || operation.ID == uuid.Nil {
		t.Fatalf("unexpected stop operation: %+v", operation)
	}
	assertLifecycleServiceStatus(t, ctx, pool, serviceID, "stopping")
}

func TestLifecycleStoreRejectsStopWhileDeploymentIsActive(t *testing.T) {
	ctx := context.Background()
	pool := lifecycleTestPool(t, ctx)
	store := NewLifecycleStore(pool)
	orgID, projectID := createLifecycleTestScope(t, ctx, pool)
	specID, serviceID := createLifecycleTestService(t, ctx, pool, orgID, projectID, domain.KindApp, "deploying")
	if _, err := pool.Exec(ctx, `INSERT INTO deployments (id, app_id, service_id, number, status) VALUES ($1, $2, $3, 1, 'building')`, uuid.New(), specID, serviceID); err != nil {
		t.Fatalf("create active deployment: %v", err)
	}

	_, err := store.RequestLifecycle(ctx, domain.LifecycleRequest{ServiceID: serviceID, SpecID: specID, OrgID: orgID, Kind: domain.KindApp, Action: domain.LifecycleStop})
	if !errors.Is(err, domain.ErrLifecycleConflict) {
		t.Fatalf("stop during an active deployment should conflict, got %v", err)
	}
}

func TestLifecycleCommandsCompleteThroughNATSWorkerForEveryServiceKind(t *testing.T) {
	url := os.Getenv("AETHER_NATS_TEST_URL")
	if url == "" {
		t.Skip("lifecycle end-to-end test requires AETHER_NATS_TEST_URL")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool := lifecycleTestPool(t, ctx)
	runtime, err := adapter.New(ctx, druntime.Config{Backend: "nats", NATSURL: url, NATSName: "aether-service-lifecycle-e2e"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = runtime.Close(context.Background()) }()
	store := NewLifecycleStore(pool)
	orgID, projectID := createLifecycleTestScope(t, ctx, pool)
	for _, kind := range []domain.Kind{domain.KindApp, domain.KindCompose, domain.KindDatabase} {
		t.Run(string(kind), func(t *testing.T) {
			specID, serviceID := createLifecycleTestService(t, ctx, pool, orgID, projectID, kind, "stopped")
			queueStream := "service-lifecycle-" + uuid.NewString()
			observedRuntime := &lifecycleObservedRuntime{serviceID: serviceID}
			stateNotifications := &lifecycleStateNotifications{states: make(chan string, 16)}
			executionCount := 0
			lifecycleWorker := &worker.ServiceLifecycleWorker{
				Queue: runtime.Queue, QueueStream: queueStream, ConsumerGroup: "service-lifecycle-" + uuid.NewString(), Store: store, Runtime: observedRuntime, Notifier: stateNotifications,
				Execute: func(_ context.Context, operation *domain.LifecycleOperation) error {
					executionCount++
					if operation.Action == domain.LifecycleStart {
						observedRuntime.state = "running"
					} else {
						observedRuntime.state = "exited"
					}
					return nil
				},
			}
			workerCtx, stopWorker := context.WithCancel(ctx)
			workerDone := make(chan struct{})
			go func() {
				defer close(workerDone)
				lifecycleWorker.Run(workerCtx)
			}()
			t.Cleanup(func() {
				stopWorker()
				<-workerDone
			})
			for _, action := range []domain.LifecycleAction{domain.LifecycleStart, domain.LifecycleStop} {
				operation, err := store.RequestLifecycle(ctx, domain.LifecycleRequest{ServiceID: serviceID, SpecID: specID, OrgID: orgID, Kind: kind, Action: action})
				if err != nil {
					t.Fatalf("request %s command: %v", action, err)
				}
				var serializedEvent []byte
				if err := pool.QueryRow(ctx, `SELECT payload FROM outbox_events WHERE id = $1`, operation.ID).Scan(&serializedEvent); err != nil {
					t.Fatalf("load durable %s command: %v", action, err)
				}
				var event events.Event
				if err := json.Unmarshal(serializedEvent, &event); err != nil {
					t.Fatalf("decode durable %s command: %v", action, err)
				}
				var job queue.Job
				if err := json.Unmarshal(event.Payload, &job); err != nil {
					t.Fatalf("decode %s command job: %v", action, err)
				}
				if err := runtime.Queue.Enqueue(ctx, queueStream, job); err != nil {
					t.Fatalf("enqueue %s command in NATS: %v", action, err)
				}
				finalStatus := "running"
				if action == domain.LifecycleStop {
					finalStatus = "stopped"
				}
				waitForLifecycleState(t, ctx, stateNotifications.states, finalStatus)
				assertLifecycleServiceStatus(t, ctx, pool, serviceID, finalStatus)
				completed, err := store.GetLifecycleOperation(ctx, operation.ID)
				if err != nil || completed.Status != "succeeded" {
					t.Fatalf("operation %s did not complete: %+v, %v", action, completed, err)
				}
				if executionCount != int(actionIndex(action))+1 {
					t.Fatalf("runtime execution count after %s = %d", action, executionCount)
				}
			}
		})
	}
}

type lifecycleStateNotifications struct {
	states chan string
}

func (n *lifecycleStateNotifications) NotifyServiceState(_ context.Context, _, _ uuid.UUID, state string) {
	n.states <- state
}

func waitForLifecycleState(t *testing.T, ctx context.Context, states <-chan string, expected string) {
	t.Helper()
	for {
		select {
		case state := <-states:
			if state == expected {
				return
			}
		case <-ctx.Done():
			t.Fatalf("timed out waiting for lifecycle state %q: %v", expected, ctx.Err())
		}
	}
}

func actionIndex(action domain.LifecycleAction) int {
	if action == domain.LifecycleStart {
		return 0
	}
	return 1
}

type lifecycleObservedRuntime struct {
	worker.Runtime
	serviceID uuid.UUID
	state     string
}

func (r *lifecycleObservedRuntime) ListContainers(context.Context) ([]worker.ContainerInfo, error) {
	return []worker.ContainerInfo{{ID: "lifecycle-container", State: r.state, Labels: map[string]string{"aether.service-id": r.serviceID.String()}}}, nil
}

func lifecycleTestPool(t *testing.T, ctx context.Context) *pgxpool.Pool {
	t.Helper()
	portValue := os.Getenv("AETHER_TEST_DATABASE_PORT")
	if portValue == "" {
		t.Skip("lifecycle integration tests require AETHER_TEST_DATABASE_PORT")
	}
	port, err := strconv.Atoi(portValue)
	if err != nil {
		t.Fatalf("invalid test database port: %v", err)
	}
	user := os.Getenv("AETHER_TEST_DATABASE_USER")
	password := os.Getenv("AETHER_TEST_DATABASE_PASSWORD")
	if user == "" && port == 5433 {
		user, password = "postgres", "postgres"
	}
	if user == "" {
		t.Fatalf("AETHER_TEST_DATABASE_USER must be set for lifecycle integration tests on port %d", port)
	}
	config := database.Config{Host: "127.0.0.1", Port: port, Name: "aether_service_lifecycle_test", User: user, Password: password, SSLMode: "disable", PoolMax: 16, ConnectTimeout: 5}
	if err := database.EnsureDatabase(ctx, config); err != nil {
		t.Fatalf("ensure dedicated lifecycle test database: %v", err)
	}
	pool, err := database.Open(ctx, config)
	if err != nil {
		t.Fatalf("open dedicated lifecycle test database: %v", err)
	}
	if err := database.Migrate(ctx, pool, "../../../../db/migrations"); err != nil {
		pool.Close()
		t.Fatalf("migrate dedicated lifecycle test database: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func createLifecycleTestScope(t *testing.T, ctx context.Context, pool *pgxpool.Pool) (uuid.UUID, uuid.UUID) {
	t.Helper()
	orgID := uuid.New()
	projectID := uuid.New()
	slug := "lifecycle-" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err := pool.Exec(ctx, `INSERT INTO orgs (id, name, slug, owner_user_id) VALUES ($1, 'Lifecycle test org', $2, NULL)`, orgID, slug); err != nil {
		t.Fatalf("create lifecycle test organization: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO projects (id, org_id, name, slug) VALUES ($1, $2, 'Lifecycle test project', $3)`, projectID, orgID, slug); err != nil {
		t.Fatalf("create lifecycle test project: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM outbox_events WHERE aggregate_type = 'service' AND aggregate_id IN (SELECT id::text FROM services WHERE org_id = $1)`, orgID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM orgs WHERE id = $1`, orgID)
	})
	return orgID, projectID
}

func createLifecycleTestService(t *testing.T, ctx context.Context, pool *pgxpool.Pool, orgID, projectID uuid.UUID, kind domain.Kind, status string) (uuid.UUID, uuid.UUID) {
	t.Helper()
	specID := uuid.New()
	serviceID := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO services (id, org_id, project_id, name, kind, status) VALUES ($1, $2, $3, $4, $5, $6)`, serviceID, orgID, projectID, "lifecycle-"+string(kind), kind, status); err != nil {
		t.Fatalf("create lifecycle service: %v", err)
	}
	switch kind {
	case domain.KindApp:
		_, err := pool.Exec(ctx, `INSERT INTO apps (id, org_id, project_id, name, service_id) VALUES ($1, $2, $3, $4, $5)`, specID, orgID, projectID, "lifecycle-app", serviceID)
		if err != nil {
			t.Fatalf("create lifecycle app: %v", err)
		}
	case domain.KindCompose:
		_, err := pool.Exec(ctx, `INSERT INTO compose_apps (id, org_id, project_id, name, compose, status, service_id) VALUES ($1, $2, $3, $4, 'services: {}', 'stopped', $5)`, specID, orgID, projectID, "lifecycle-compose", serviceID)
		if err != nil {
			t.Fatalf("create lifecycle compose service: %v", err)
		}
	case domain.KindDatabase:
		_, err := pool.Exec(ctx, `INSERT INTO databases (id, org_id, project_id, name, engine, version, db_name, db_user, pass_enc, status, service_id) VALUES ($1, $2, $3, $4, 'postgres', '17', 'lifecycle', 'lifecycle', 'encrypted', 'stopped', $5)`, specID, orgID, projectID, "lifecycle-database", serviceID)
		if err != nil {
			t.Fatalf("create lifecycle database: %v", err)
		}
	default:
		t.Fatalf("unsupported lifecycle test kind %q", kind)
	}
	return specID, serviceID
}

func assertLifecycleServiceStatus(t *testing.T, ctx context.Context, pool *pgxpool.Pool, serviceID uuid.UUID, expected string) {
	t.Helper()
	var status string
	if err := pool.QueryRow(ctx, `SELECT status FROM services WHERE id = $1`, serviceID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != expected {
		t.Fatalf("service status = %q, want %q", status, expected)
	}
}

func assertLifecycleOutbox(t *testing.T, ctx context.Context, pool *pgxpool.Pool, operationID, serviceID uuid.UUID) {
	t.Helper()
	var topic string
	var payload []byte
	if err := pool.QueryRow(ctx, `SELECT topic, payload FROM outbox_events WHERE id = $1`, operationID).Scan(&topic, &payload); err != nil {
		t.Fatalf("load lifecycle outbox event: %v", err)
	}
	var event events.Event
	if err := json.Unmarshal(payload, &event); err != nil {
		t.Fatalf("decode lifecycle outbox event: %v", err)
	}
	var job queue.Job
	if err := json.Unmarshal(event.Payload, &job); err != nil {
		t.Fatalf("decode lifecycle outbox job: %v", err)
	}
	var command struct {
		OperationID string `json:"operation_id"`
	}
	if err := json.Unmarshal(job.Payload, &command); err != nil {
		t.Fatalf("decode lifecycle command payload: %v", err)
	}
	if topic != "service-operations" || event.Type != "service.lifecycle.queued" || event.AggregateID != serviceID.String() || job.ID != operationID.String() || command.OperationID != operationID.String() {
		t.Fatalf("unexpected lifecycle outbox message: topic=%q event=%+v job=%+v operation_id=%q", topic, event, job, command.OperationID)
	}
}
