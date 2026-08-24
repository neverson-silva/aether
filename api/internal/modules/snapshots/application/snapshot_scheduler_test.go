package application

import "testing"

func TestNativeSnapshotCronAddsSecondsField(t *testing.T) {
	if got := nativeSnapshotCron("0 3 * * *"); got != "0 0 3 * * *" {
		t.Fatalf("unexpected native snapshot cron expression: %q", got)
	}
}
