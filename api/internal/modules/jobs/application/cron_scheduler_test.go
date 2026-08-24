package application

import "testing"

func TestNativeCronAddsSecondsField(t *testing.T) {
	if got := nativeCron("*/5 * * * *"); got != "0 */5 * * * *" {
		t.Fatalf("unexpected native cron expression: %q", got)
	}
}

func TestNativeCronPreservesSixFields(t *testing.T) {
	if got := nativeCron("*/5 * * * * *"); got != "*/5 * * * * *" {
		t.Fatalf("unexpected native cron expression: %q", got)
	}
}
