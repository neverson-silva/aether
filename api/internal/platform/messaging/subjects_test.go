package messaging

import "testing"

func TestNotifyOrgUsesAuthorizedNamespace(t *testing.T) {
	if got := NotifyOrg("org-1"); got != "aether.notify.org.org-1" {
		t.Fatalf("unexpected notification subject: %s", got)
	}
}
