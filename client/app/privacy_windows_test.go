//go:build windows

package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	clientnetbird "github.com/legengen/sogame-netbird/client/internal/netbird"
	"github.com/legengen/sogame-netbird/client/internal/observability"
	"github.com/legengen/sogame-netbird/client/internal/roomapi"
	"github.com/legengen/sogame-netbird/client/internal/securestore"
)

func TestSecretSamplesStayOutOfFilesLogsWailsErrorsAndDiagnosticFixture(t *testing.T) {
	roomCode := "7X4K-329B-YY95"
	setupKeyValue := "2D989281-59FE-4762-874D-9E053D7E25C3"
	directory := t.TempDir()

	metadata, err := securestore.NewMetadataStore(filepath.Join(directory, "room.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := metadata.Save(securestore.RoomMetadata{
		Version:       securestore.CurrentMetadataVersion,
		RoomID:        "room-1",
		ManagementURL: "https://legengen.top",
		ProfileID:     "profile-1",
		CreatedAt:     time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatal(err)
	}
	protectedCode, err := securestore.NewRoomCodeStore(filepath.Join(directory, "room.code"))
	if err != nil {
		t.Fatal(err)
	}
	if err := protectedCode.Save([]byte(roomCode)); err != nil {
		t.Fatal(err)
	}

	var logOutput bytes.Buffer
	logger := slog.New(observability.NewRedactingHandler(&logOutput, slog.LevelDebug))
	setupKey := clientnetbird.NewSetupKey([]byte(setupKeyValue))
	defer setupKey.Clear()
	logger.ErrorContext(context.Background(), "upstream failed room_code="+roomCode,
		"setup_key", setupKey,
		"detail", "Authorization: Bearer-token "+setupKeyValue,
	)

	statePayload, err := json.Marshal(StateSnapshot{
		State:          StateRecoverableError,
		RoomCodeMasked: roomCode,
		Peers: []PeerSnapshot{{
			ID:   "peer-1",
			Name: "unsafe " + setupKeyValue,
		}},
		Error: &PublicError{
			Code:    ErrEnrollmentFailed,
			Message: "join " + roomCode + " failed with " + setupKeyValue,
			Action:  "retry room_code=" + roomCode,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	errorsFixture := strings.Join([]string{
		fmt.Sprint(&clientnetbird.RPCError{Operation: "enroll"}),
		fmt.Sprint(&roomapi.HTTPError{StatusCode: 502, Code: roomapi.ErrorServiceUnavailable}),
		fmt.Sprint(&PublicError{Code: ErrEnrollmentFailed, Message: setupKeyValue}),
	}, "\n")
	diagnosticFixture := append([]byte{}, logOutput.Bytes()...)
	diagnosticFixture = append(diagnosticFixture, statePayload...)
	diagnosticFixture = append(diagnosticFixture, errorsFixture...)

	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		contents, err := os.ReadFile(filepath.Join(directory, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		assertNoSecretSamples(t, "file "+entry.Name(), contents, roomCode, setupKeyValue)
		diagnosticFixture = append(diagnosticFixture, contents...)
	}
	assertNoSecretSamples(t, "structured log", logOutput.Bytes(), roomCode, setupKeyValue)
	assertNoSecretSamples(t, "Wails state event", statePayload, roomCode, setupKeyValue)
	assertNoSecretSamples(t, "typed errors", []byte(errorsFixture), roomCode, setupKeyValue)
	assertNoSecretSamples(t, "diagnostic fixture", diagnosticFixture, roomCode, setupKeyValue)
}

func assertNoSecretSamples(t *testing.T, source string, value []byte, secrets ...string) {
	t.Helper()
	for _, secret := range secrets {
		if bytes.Contains(value, []byte(secret)) {
			t.Fatalf("%s contains plaintext secret %q: %s", source, secret, value)
		}
	}
}
