package domain

import (
	"testing"

	"github.com/google/uuid"
)

func TestReadyToFailedTransition(t *testing.T) {
	dep := &Deployment{ID: uuid.New(), Status: StatusReady}
	if err := dep.Transition(StatusFailed); err != nil {
		t.Fatalf("ready->failed deveria ser permitido: %v", err)
	}
	if dep.Status != StatusFailed {
		t.Fatalf("status: %v", dep.Status)
	}
	if dep.FinishedAt == nil {
		t.Fatalf("failed deveria marcar FinishedAt")
	}
}

func TestReadyTransitions(t *testing.T) {
	dep := &Deployment{ID: uuid.New(), Status: StatusReady}
	if err := dep.Transition(StatusReady); err == nil {
		t.Fatalf("ready->ready deveria falhar")
	}
	if dep.Transition(StatusCancelled) == nil {
		t.Fatalf("ready->cancelled deveria falhar")
	}
	if dep.Status.Terminal() {
		t.Fatalf("ready agora admite transição a failed, não é terminal")
	}
	if err := dep.Transition(StatusFailed); err != nil {
		t.Fatalf("ready->failed: %v", err)
	}
	if !dep.Status.Terminal() {
		t.Fatalf("failed deveria ser terminal")
	}
}

func TestReadySetsFinishedAt(t *testing.T) {
	dep := &Deployment{ID: uuid.New(), Status: StatusHealthChecking}
	if err := dep.Transition(StatusReady); err != nil {
		t.Fatalf("health_checking->ready: %v", err)
	}
	if dep.FinishedAt == nil {
		t.Fatalf("ready deveria marcar FinishedAt")
	}
	if dep.StartedAt != nil {
		t.Fatalf("ready não deveria marcar StartedAt")
	}
}