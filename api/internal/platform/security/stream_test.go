package security

import (
	"testing"

	"github.com/google/uuid"
)

func TestStreamLimiterScopesByOrganization(t *testing.T) {
	limiter := NewStreamLimiter(1)
	first, err := limiter.Acquire(uuid.New())
	if err != nil {
		t.Fatalf("first acquire: %v", err)
	}
	defer first()
	if _, err := limiter.Acquire(uuid.New()); err != nil {
		t.Fatalf("different organization rejected: %v", err)
	}
	if _, err := limiter.Acquire(uuid.Nil); err != nil {
		t.Fatalf("nil organization rejected: %v", err)
	}
}
