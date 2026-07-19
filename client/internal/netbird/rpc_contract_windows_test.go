//go:build windows

package netbird

import (
	"context"
	"os"
	"os/user"
	"testing"
	"time"
)

const contractTestEnvironment = "NETBIRD_V0747_RPC_CONTRACT"

// TestOfficialV0747DaemonReadOnlyContract verifies the pinned local RPC surface
// without changing profile selection, enrollment, or tunnel state.
func TestOfficialV0747DaemonReadOnlyContract(t *testing.T) {
	if os.Getenv(contractTestEnvironment) == "" {
		t.Skip(contractTestEnvironment + " is not set")
	}
	currentUser, err := user.Current()
	if err != nil {
		t.Fatalf("resolve current Windows user: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	raw, err := DialLocalRPCAdapter(ctx, currentUser.Username)
	if err != nil {
		t.Fatalf("dial official NetBird daemon at %s: %v", LocalDaemonAddress, err)
	}
	defer func() {
		if err := raw.Close(); err != nil {
			t.Errorf("close daemon RPC connection: %v", err)
		}
	}()
	adapter := EnforceExactVersion(raw, ExpectedVersion)

	snapshot, err := adapter.Status(ctx)
	if err != nil {
		t.Fatalf("v%s Status contract failed: %v (detected %q)", ExpectedVersion, err, snapshot.DaemonVersion)
	}
	if snapshot.DaemonVersion != ExpectedVersion {
		t.Fatalf("daemon version=%q, want %q", snapshot.DaemonVersion, ExpectedVersion)
	}
	profiles, err := adapter.ListProfiles(ctx)
	if err != nil {
		t.Fatalf("v%s ListProfiles contract failed: %v", ExpectedVersion, err)
	}
	if profiles == nil {
		t.Fatal("ListProfiles returned a nil normalized slice")
	}
	active, err := adapter.ActiveProfile(ctx)
	if err != nil {
		t.Fatalf("v%s GetActiveProfile contract failed: %v", ExpectedVersion, err)
	}
	if active.ID == "" || active.Name == "" {
		t.Fatalf("active profile lacks concrete identity: %+v", active)
	}
}
