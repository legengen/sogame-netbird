//go:build windows

package platform

import (
	"context"
	"os"
	"testing"
)

func TestInstalledNetBirdServiceReadOnlyDiscovery(t *testing.T) {
	if os.Getenv("NETBIRD_TEST_SERVICE") == "" {
		t.Skip("NETBIRD_TEST_SERVICE is not set")
	}
	record, err := newServiceBackend("").Lookup(context.Background())
	if err != nil {
		t.Fatalf("service discovery failed for an unprivileged user: %v", err)
	}
	if !record.Installed || record.BinaryPath == "" {
		t.Fatalf("unexpected service record: %+v", record)
	}
}
