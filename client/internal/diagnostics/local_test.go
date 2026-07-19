package diagnostics

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWriteLocalCrashStaysOnDiskAndIsAnonymized(t *testing.T) {
	writer, err := NewWriter(filepath.Join(t.TempDir(), "crashes"))
	if err != nil {
		t.Fatal(err)
	}
	writer.now = func() time.Time { return time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC) }
	path, err := writer.WriteLocalCrash([]byte("panic at 100.115.10.21 host=legengen.top"))
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(path) != writer.Directory || strings.Contains(path, "http") {
		t.Fatalf("crash report path=%q", path)
	}
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(payload), "100.115.10.21") || strings.Contains(string(payload), "legengen.top") {
		t.Fatalf("crash report was not anonymized: %s", payload)
	}
}
