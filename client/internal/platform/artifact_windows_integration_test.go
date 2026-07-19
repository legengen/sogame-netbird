//go:build windows

package platform

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	releasebuild "github.com/legengen/sogame-netbird/client/build"
)

func TestOfficialNetBirdArtifactContract(t *testing.T) {
	path := os.Getenv("NETBIRD_TEST_MSI")
	if path == "" {
		t.Skip("NETBIRD_TEST_MSI is not set")
	}
	metadata, err := releasebuild.Load()
	if err != nil {
		t.Fatal(err)
	}
	verifier := NewArtifactVerifier(WindowsSignatureVerifier{})
	if err := verifier.Verify(context.Background(), path, metadata.WindowsX64); err != nil {
		t.Fatalf("official artifact rejected: %v", err)
	}

	tampered := filepath.Join(t.TempDir(), "tampered.msi")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content[len(content)/2] ^= 0xff
	if err := os.WriteFile(tampered, content, 0600); err != nil {
		t.Fatal(err)
	}
	if err := verifier.Verify(context.Background(), tampered, metadata.WindowsX64); !errors.Is(err, ErrArtifactDigest) {
		t.Fatalf("tampered artifact returned %v, want digest error", err)
	}
}
