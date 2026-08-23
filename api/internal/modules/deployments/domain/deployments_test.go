package domain

import (
	"testing"

	"github.com/google/uuid"
)

func TestReadyIsTerminal(t *testing.T) {
	dep := &Deployment{ID: uuid.New(), Status: StatusReady}
	if !dep.Status.Terminal() {
		t.Fatalf("ready should be terminal")
	}
	if err := dep.Transition(StatusFailed); err == nil {
		t.Fatalf("ready->failed should be rejected")
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
	if !dep.Status.Terminal() {
		t.Fatalf("ready should be terminal")
	}
}

func TestReadySetsFinishedAt(t *testing.T) {
	dep := &Deployment{ID: uuid.New(), Status: StatusHealthChecking}
	if err := dep.Transition(StatusReady); err != nil {
		t.Fatalf("health_checking->ready: %v", err)
	}
	if dep.FinishedAt == nil {
		t.Fatalf("ready should set FinishedAt")
	}
	if dep.StartedAt != nil {
		t.Fatalf("ready should not set StartedAt")
	}
}
