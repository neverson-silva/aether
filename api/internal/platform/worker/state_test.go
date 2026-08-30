package worker

import (
	"context"
	"sync"
	"testing"

	"github.com/google/uuid"

	deploydomain "aether/internal/modules/deployments/domain"
)

type fakeServiceStateStore struct {
	mu      sync.Mutex
	targets []RuntimeServiceTarget
	status  map[uuid.UUID]string
}

func (s *fakeServiceStateStore) ListRuntimeServiceTargets(context.Context) ([]RuntimeServiceTarget, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]RuntimeServiceTarget(nil), s.targets...), nil
}

func (s *fakeServiceStateStore) UpdateRuntimeStatus(_ context.Context, serviceID uuid.UUID, status string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.status == nil {
		s.status = map[uuid.UUID]string{}
	}
	if s.status[serviceID] == status {
		return false, nil
	}
	s.status[serviceID] = status
	return true, nil
}

type stateRuntime struct {
	fakeRuntime
	containers   []ContainerInfo
	subscription RuntimeEventSubscription
}

func (r *stateRuntime) ListContainers(context.Context) ([]ContainerInfo, error) {
	return append([]ContainerInfo(nil), r.containers...), nil
}

func (r *stateRuntime) SubscribeEvents(context.Context, map[string]string) (RuntimeEventSubscription, error) {
	return r.subscription, nil
}

type fakeRuntimeEventSubscription struct {
	events chan RuntimeEvent
	errors chan error
}

func (s *fakeRuntimeEventSubscription) Events() <-chan RuntimeEvent { return s.events }
func (s *fakeRuntimeEventSubscription) Errors() <-chan error        { return s.errors }
func (s *fakeRuntimeEventSubscription) Close() error                { return nil }

type stateNotifier struct {
	mu     sync.Mutex
	states []string
}

func (n *stateNotifier) NotifyDeploy(context.Context, deploydomain.DeployEvent) {}
func (n *stateNotifier) NotifyAppState(context.Context, uuid.UUID, string)      {}

func (n *stateNotifier) NotifyServiceState(_ context.Context, _ uuid.UUID, _ uuid.UUID, state string) {
	n.mu.Lock()
	n.states = append(n.states, state)
	n.mu.Unlock()
}

func TestWatcherProjectsRuntimeStateAndEmitsOnlyTransitions(t *testing.T) {
	serviceID := uuid.New()
	organizationID := uuid.New()
	store := &fakeServiceStateStore{targets: []RuntimeServiceTarget{{ID: serviceID, OrganizationID: organizationID, Kind: "app"}}}
	runtime := &stateRuntime{containers: []ContainerInfo{{ID: "container-1", Name: "web", State: "running", Labels: map[string]string{"aether.service-id": serviceID.String()}}}}
	notifier := &stateNotifier{}
	watcher := &Watcher{Runtime: runtime, ServiceStore: store, Notifier: notifier}

	watcher.reconcile(context.Background())
	watcher.reconcile(context.Background())
	store.mu.Lock()
	if got := store.status[serviceID]; got != "running" {
		t.Fatalf("projected status = %q, want running", got)
	}
	store.mu.Unlock()
	if len(notifier.states) != 1 || notifier.states[0] != "running" {
		t.Fatalf("notifications = %v, want one running transition", notifier.states)
	}

	runtime.containers[0].State = "exited"
	watcher.reconcile(context.Background())
	store.mu.Lock()
	if got := store.status[serviceID]; got != "stopped" {
		t.Fatalf("projected exited status = %q, want stopped", got)
	}
	store.mu.Unlock()
	if len(notifier.states) != 2 || notifier.states[1] != "stopped" {
		t.Fatalf("notifications = %v, want running and stopped", notifier.states)
	}
}

func TestWatcherProjectsPendingAndDeployingWithoutContainers(t *testing.T) {
	serviceID := uuid.New()
	store := &fakeServiceStateStore{targets: []RuntimeServiceTarget{{ID: serviceID, Kind: "database"}}}
	runtime := &stateRuntime{}
	watcher := &Watcher{Runtime: runtime, ServiceStore: store}

	watcher.reconcile(context.Background())
	store.mu.Lock()
	if got := store.status[serviceID]; got != "pending" {
		t.Fatalf("never-deployed status = %q, want pending", got)
	}
	store.targets[0].ActiveDeployment = true
	store.mu.Unlock()
	watcher.reconcile(context.Background())
	store.mu.Lock()
	if got := store.status[serviceID]; got != "deploying" {
		t.Fatalf("active-deployment status = %q, want deploying", got)
	}
	store.mu.Unlock()
}

func TestWatcherReconcilesAfterRuntimeEventStreamCloses(t *testing.T) {
	serviceID := uuid.New()
	store := &fakeServiceStateStore{targets: []RuntimeServiceTarget{{ID: serviceID, Kind: "app"}}}
	runtime := &stateRuntime{
		containers: []ContainerInfo{{ID: "container-1", State: "running", Labels: map[string]string{"aether.service-id": serviceID.String()}}},
		subscription: &fakeRuntimeEventSubscription{
			events: func() chan RuntimeEvent {
				out := make(chan RuntimeEvent, 1)
				out <- RuntimeEvent{Action: "start", ContainerID: "container-1"}
				close(out)
				return out
			}(),
			errors: make(chan error),
		},
	}
	watcher := &Watcher{Runtime: runtime, ServiceStore: store}
	if reconnect := watcher.consumeEvents(context.Background(), runtime.subscription); !reconnect {
		t.Fatal("closed event stream must request reconnect")
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if got := store.status[serviceID]; got != "running" {
		t.Fatalf("event-projected status = %q, want running", got)
	}
}

func TestWatcherStopsOnRuntimeEventError(t *testing.T) {
	subscription := &fakeRuntimeEventSubscription{events: make(chan RuntimeEvent), errors: make(chan error, 1)}
	subscription.errors <- context.Canceled
	watcher := &Watcher{}
	if reconnect := watcher.consumeEvents(context.Background(), subscription); !reconnect {
		t.Fatal("runtime error must request reconnect")
	}
}
