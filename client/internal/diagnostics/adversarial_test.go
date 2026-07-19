package diagnostics

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"path/filepath"
	"testing"
)

func TestAdversarialUpstreamErrorsAndRPCFieldsNeverReachBundle(t *testing.T) {
	secrets := [][]byte{
		[]byte("2D989281-59FE-4762-874D-9E053D7E25C3"),
		[]byte("7X4K-329B-YY95"),
		[]byte("Bearer super-secret-token"),
		[]byte("100.115.10.21"),
		[]byte("legengen.top"),
		[]byte("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"),
		[]byte("-----BEGIN PRIVATE KEY-----private-material-----END PRIVATE KEY-----"),
	}
	writer, err := NewWriter(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	path, err := writer.Write(context.Background(), testSource{report: Report{
		Application: map[string]any{
			"error": "RPC rejected setup_key=2D989281-59FE-4762-874D-9E053D7E25C3",
			"rpc":   "peer=0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		},
		NetBird: map[string]any{
			"message": "Authorization: Bearer super-secret-token from legengen.top",
			"key":     "-----BEGIN PRIVATE KEY-----private-material-----END PRIVATE KEY-----",
		},
		Logs: []byte("room_code=7X4K-329B-YY95 ip=100.115.10.21"),
	}})
	if err != nil {
		t.Fatal(err)
	}
	archive, err := zip.OpenReader(filepath.Clean(path))
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()
	for _, entry := range archive.File {
		reader, err := entry.Open()
		if err != nil {
			t.Fatal(err)
		}
		payload, err := io.ReadAll(reader)
		reader.Close()
		if err != nil {
			t.Fatal(err)
		}
		for _, secret := range secrets {
			if bytes.Contains(payload, secret) {
				t.Fatalf("entry %s contains adversarial secret %q: %s", entry.Name, secret, payload)
			}
		}
	}
}
