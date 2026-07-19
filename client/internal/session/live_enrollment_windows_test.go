//go:build windows && amd64

package session

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"testing"
	"time"

	clientnetbird "github.com/legengen/sogame-netbird/client/internal/netbird"
	"github.com/legengen/sogame-netbird/client/internal/roomapi"
	"github.com/legengen/sogame-netbird/client/internal/securestore"
)

const liveEnrollmentContractEnvironment = "SOGAME_LIVE_ENROLLMENT_CONTRACT"

func TestLiveRoomEnrollmentContract(t *testing.T) {
	if os.Getenv(liveEnrollmentContractEnvironment) == "" {
		t.Skip(liveEnrollmentContractEnvironment + " is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	currentUser, err := user.Current()
	if err != nil {
		t.Fatalf("resolve current Windows user: %v", err)
	}
	raw, err := clientnetbird.DialLocalRPCAdapter(ctx, currentUser.Username)
	if err != nil {
		t.Fatalf("dial local NetBird RPC: %v", err)
	}
	defer raw.Close()
	previous, err := raw.ActiveProfile(ctx)
	if err != nil {
		t.Fatalf("read active profile: %v", err)
	}
	defer func() {
		if previous.ID != "" {
			if restoreErr := raw.SelectProfile(context.Background(), previous.ID); restoreErr != nil {
				t.Errorf("restore active profile %s: %v", previous.ID, restoreErr)
			}
		}
	}()

	rooms, err := roomapi.NewClient("https://legengen.top", &http.Client{})
	if err != nil {
		t.Fatalf("create Room API client: %v", err)
	}
	metadata, err := securestore.NewMetadataStore(filepath.Join(t.TempDir(), "room.json"))
	if err != nil {
		t.Fatalf("create metadata store: %v", err)
	}
	codes, err := securestore.NewRoomCodeStore(filepath.Join(t.TempDir(), "room-code.bin"))
	if err != nil {
		t.Fatalf("create room-code store: %v", err)
	}
	service := NewService(rooms, clientnetbird.EnforceExactVersion(raw, clientnetbird.ExpectedVersion), metadata, codes)

	snapshot, err := service.Create(ctx, "sogame-contract-test")
	if err != nil {
		for cause := err; cause != nil; cause = errors.Unwrap(cause) {
			t.Logf("enrollment error: %T: %v", cause, cause)
		}
		t.Fatalf("live enrollment failed in state %s", snapshot.State)
	}
	if snapshot.State == StateRecoverableError || snapshot.State == StateNoRoom {
		t.Fatalf("live enrollment returned unexpected state %s", snapshot.State)
	}
	if _, leaveErr := service.Leave(ctx); leaveErr != nil {
		t.Fatalf("clean up live enrollment: %v", leaveErr)
	}
}
