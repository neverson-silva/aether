package queue

import (
	"errors"
	"testing"
)

func TestPermanentErrorClassification(t *testing.T) {
	base := errors.New("invalid payload")
	err := PermanentError{Err: base}
	if !IsPermanent(err) {
		t.Fatal("permanent error was not classified")
	}
	if !errors.Is(err, base) {
		t.Fatal("permanent error did not unwrap")
	}
	if IsPermanent(errors.New("temporary failure")) {
		t.Fatal("temporary error was classified as permanent")
	}
}
