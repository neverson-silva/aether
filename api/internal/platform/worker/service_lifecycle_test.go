package worker

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	databasesdomain "aether/internal/modules/databases/domain"
	servicesdomain "aether/internal/modules/services/domain"
	"aether/internal/platform/druntime/queue"
)

type serviceLifecycleRuntime struct {
	Runtime
	containers []ContainerInfo
	err        error
}

func (r serviceLifecycleRuntime) ListContainers(context.Context) ([]ContainerInfo, error) {
	return r.containers, r.err
}

func TestServiceLifecycleVerificationUsesObservedRuntimeState(t *testing.T) {
	serviceID := uuid.New()
	operation := &servicesdomain.LifecycleOperation{ServiceID: serviceID, Action: servicesdomain.LifecycleStart}
	worker := &ServiceLifecycleWorker{
		Runtime: serviceLifecycleRuntime{containers: []ContainerInfo{{ID: "container-1", State: "running", Labels: map[string]string{"aether.service-id": serviceID.String()}}}},
		Execute: func(context.Context, *servicesdomain.LifecycleOperation) error {
			return errors.New("ambiguous runtime response")
		},
	}
	status, err := worker.executeAndVerify(context.Background(), operation)
	if err != nil || status != string(servicesdomain.StatusRunning) {
		t.Fatalf("expected verified running state, got %q and %v", status, err)
	}

	operation.Action = servicesdomain.LifecycleStop
	worker.Runtime = serviceLifecycleRuntime{containers: []ContainerInfo{{ID: "container-1", State: "running", Labels: map[string]string{"aether.service-id": serviceID.String()}}}}
	status, err = worker.executeAndVerify(context.Background(), operation)
	if err == nil || status != string(servicesdomain.StatusRunning) {
		t.Fatalf("expected failed stop to preserve observed running state, got %q and %v", status, err)
	}

	operation.Action = servicesdomain.LifecycleStart
	worker.Runtime = serviceLifecycleRuntime{containers: []ContainerInfo{{ID: "container-1", State: "restarting", Labels: map[string]string{"aether.service-id": serviceID.String()}}}}
	status, err = worker.executeAndVerify(context.Background(), operation)
	if err == nil || status != string(servicesdomain.StatusDegraded) {
		t.Fatalf("expected restarting runtime to remain unconfirmed, got %q and %v", status, err)
	}
}

type lifecycleExecutorRecorder struct {
	called string
}

func (r *lifecycleExecutorRecorder) StartService(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (string, error) {
	r.called = "app.start"
	return "", nil
}

func (r *lifecycleExecutorRecorder) StopService(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (string, error) {
	r.called = "app.stop"
	return "", nil
}

func (r *lifecycleExecutorRecorder) Start(context.Context, uuid.UUID, uuid.UUID) error {
	r.called = "compose.start"
	return nil
}

func (r *lifecycleExecutorRecorder) Stop(context.Context, uuid.UUID, uuid.UUID) error {
	r.called = "compose.stop"
	return nil
}

func (r *lifecycleExecutorRecorder) StartDatabase(context.Context, uuid.UUID, uuid.UUID) (*databasesdomain.Database, error) {
	r.called = "database.start"
	return &databasesdomain.Database{}, nil
}

func (r *lifecycleExecutorRecorder) StopDatabase(context.Context, uuid.UUID, uuid.UUID) (*databasesdomain.Database, error) {
	r.called = "database.stop"
	return &databasesdomain.Database{}, nil
}

type lifecycleDatabaseExecutor struct {
	recorder *lifecycleExecutorRecorder
}

func (d lifecycleDatabaseExecutor) Start(ctx context.Context, id, orgID uuid.UUID) (*databasesdomain.Database, error) {
	return d.recorder.StartDatabase(ctx, id, orgID)
}

func (d lifecycleDatabaseExecutor) Stop(ctx context.Context, id, orgID uuid.UUID) (*databasesdomain.Database, error) {
	return d.recorder.StopDatabase(ctx, id, orgID)
}

func TestServiceLifecycleExecutorRoutesEveryServiceKind(t *testing.T) {
	recorder := &lifecycleExecutorRecorder{}
	execute := NewServiceLifecycleExecutor(recorder, recorder, lifecycleDatabaseExecutor{recorder: recorder})
	for _, test := range []struct {
		kind   servicesdomain.Kind
		action servicesdomain.LifecycleAction
		want   string
	}{
		{kind: servicesdomain.KindApp, action: servicesdomain.LifecycleStart, want: "app.start"},
		{kind: servicesdomain.KindApp, action: servicesdomain.LifecycleStop, want: "app.stop"},
		{kind: servicesdomain.KindCompose, action: servicesdomain.LifecycleStart, want: "compose.start"},
		{kind: servicesdomain.KindCompose, action: servicesdomain.LifecycleStop, want: "compose.stop"},
		{kind: servicesdomain.KindDatabase, action: servicesdomain.LifecycleStart, want: "database.start"},
		{kind: servicesdomain.KindDatabase, action: servicesdomain.LifecycleStop, want: "database.stop"},
	} {
		t.Run(string(test.kind)+"."+string(test.action), func(t *testing.T) {
			err := execute(context.Background(), &servicesdomain.LifecycleOperation{Kind: test.kind, Action: test.action})
			if err != nil {
				t.Fatalf("execute lifecycle operation: %v", err)
			}
			if recorder.called != test.want {
				t.Fatalf("expected executor %q, got %q", test.want, recorder.called)
			}
		})
	}
}

type reconnectLifecycleQueue struct {
	queue.Queue
	attempts int
	cancel   context.CancelFunc
}

func (q *reconnectLifecycleQueue) NewConsumer(_ context.Context, _, _, _ string) (queue.Consumer, error) {
	q.attempts++
	if q.attempts == 1 {
		return nil, errors.New("temporary queue outage")
	}
	return reconnectLifecycleConsumer{cancel: q.cancel}, nil
}

type reconnectLifecycleConsumer struct {
	cancel context.CancelFunc
}

func (c reconnectLifecycleConsumer) Next(context.Context) (*queue.Job, error) {
	c.cancel()
	return nil, context.Canceled
}

func (reconnectLifecycleConsumer) Ack(context.Context, *queue.Job) error  { return nil }
func (reconnectLifecycleConsumer) Nack(context.Context, *queue.Job) error { return nil }
func (reconnectLifecycleConsumer) Close() error                           { return nil }

func TestServiceLifecycleWorkerRetriesQueueConsumerCreation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	q := &reconnectLifecycleQueue{cancel: cancel}
	(&ServiceLifecycleWorker{Queue: q}).Run(ctx)
	if q.attempts != 2 {
		t.Fatalf("expected consumer creation retry after outage, got %d attempts", q.attempts)
	}
}

type completedLifecycleStore struct {
	operation *servicesdomain.LifecycleOperation
	err       error
}

func (s completedLifecycleStore) GetLifecycleOperation(context.Context, uuid.UUID) (*servicesdomain.LifecycleOperation, error) {
	return s.operation, s.err
}

func (completedLifecycleStore) ClaimLifecycleOperation(context.Context, uuid.UUID) (*servicesdomain.LifecycleOperation, bool, error) {
	return nil, false, nil
}

func (completedLifecycleStore) RenewLifecycleOperation(context.Context, uuid.UUID, int) error {
	return nil
}

func (completedLifecycleStore) CompleteLifecycleOperation(context.Context, uuid.UUID, int, string, string) (*servicesdomain.LifecycleOperation, error) {
	return nil, nil
}

func (completedLifecycleStore) RecoverLifecycleOperations(context.Context, int) error {
	return nil
}

type lifecycleMessageConsumer struct {
	acked  int
	nacked int
}

func (*lifecycleMessageConsumer) Next(context.Context) (*queue.Job, error) { return nil, nil }
func (c *lifecycleMessageConsumer) Ack(context.Context, *queue.Job) error {
	c.acked++
	return nil
}
func (c *lifecycleMessageConsumer) Nack(context.Context, *queue.Job) error {
	c.nacked++
	return nil
}
func (*lifecycleMessageConsumer) Close() error { return nil }

func TestServiceLifecycleWorkerAcknowledgesDuplicateCompletedMessage(t *testing.T) {
	operationID := uuid.New()
	worker := &ServiceLifecycleWorker{
		Store: completedLifecycleStore{operation: &servicesdomain.LifecycleOperation{ID: operationID, Status: "succeeded"}},
		Execute: func(context.Context, *servicesdomain.LifecycleOperation) error {
			t.Fatal("completed duplicate must not execute runtime side effects")
			return nil
		},
	}
	consumer := &lifecycleMessageConsumer{}
	job := &queue.Job{Payload: []byte(`{"operation_id":"` + operationID.String() + `"}`)}
	worker.process(context.Background(), consumer, job)
	if consumer.acked != 1 || consumer.nacked != 0 {
		t.Fatalf("expected completed duplicate to be acknowledged, got %d ack and %d nack", consumer.acked, consumer.nacked)
	}
}

func TestServiceLifecycleWorkerAcknowledgesDeletedOperationMessage(t *testing.T) {
	operationID := uuid.New()
	worker := &ServiceLifecycleWorker{Store: completedLifecycleStore{err: servicesdomain.ErrNotFound}}
	consumer := &lifecycleMessageConsumer{}
	job := &queue.Job{Payload: []byte(`{"operation_id":"` + operationID.String() + `"}`)}
	worker.process(context.Background(), consumer, job)
	if consumer.acked != 1 || consumer.nacked != 0 {
		t.Fatalf("expected deleted operation message to be acknowledged, got %d ack and %d nack", consumer.acked, consumer.nacked)
	}
}
