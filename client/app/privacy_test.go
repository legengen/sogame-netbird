package app

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestStateAndPublicErrorJSONRedactSecretPatterns(t *testing.T) {
	roomCode := "7X4K-329B-YY95"
	setupKey := "2D989281-59FE-4762-874D-9E053D7E25C3"
	payload, err := json.Marshal(StateSnapshot{
		State:          StateRecoverableError,
		RoomCodeMasked: roomCode,
		Peers:          []PeerSnapshot{{Name: setupKey}},
		Error:          &PublicError{Code: ErrInternal, Message: roomCode + " " + setupKey},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{roomCode, setupKey} {
		if bytes.Contains(payload, []byte(secret)) {
			t.Fatalf("state JSON contains secret %q: %s", secret, payload)
		}
	}
	if !bytes.Contains(payload, []byte("[REDACTED]")) {
		t.Fatalf("state JSON has no redaction marker: %s", payload)
	}
}
