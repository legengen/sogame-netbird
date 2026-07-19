package diagnostics

import (
	"archive/zip"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type testSource struct {
	report Report
	err    error
}

func (s testSource) Collect(context.Context) (Report, error) { return s.report, s.err }

func TestWriterCreatesLocalBundleWithExpectedEntries(t *testing.T) {
	directory := t.TempDir()
	writer, err := NewWriter(filepath.Join(directory, "diagnostics"))
	if err != nil {
		t.Fatal(err)
	}
	writer.now = func() time.Time { return time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC) }
	path, err := writer.Write(context.Background(), testSource{report: Report{
		Application: map[string]any{"state": "WaitingForPeer"},
		NetBird:     map[string]any{"version": "0.74.7"},
		Logs:        []byte("local log\n"),
	}})
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(path) != filepath.Join(directory, "diagnostics") {
		t.Fatalf("path=%s", path)
	}
	archive, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()
	entries := map[string]bool{}
	for _, entry := range archive.File {
		entries[entry.Name] = true
	}
	for _, name := range []string{"application.json", "netbird.json", "logs.txt", "README.txt"} {
		if !entries[name] {
			t.Fatalf("missing entry %s", name)
		}
	}
}

func TestWriterRejectsMissingSourceAndOversizedLogs(t *testing.T) {
	writer, err := NewWriter(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write(context.Background(), nil); err == nil {
		t.Fatal("missing source was accepted")
	}
	if _, err := writer.Write(context.Background(), testSource{report: Report{Logs: make([]byte, maxReportBytes+1)}}); err == nil {
		t.Fatal("oversized logs were accepted")
	}
	if _, err := writer.Write(context.Background(), testSource{err: errors.New("source unavailable")}); err == nil {
		t.Fatal("source error was suppressed")
	}
	if _, err := os.Stat(filepath.Join(writer.Directory, "diagnostic-20260719-120000.000000000.zip")); !os.IsNotExist(err) {
		t.Fatal("failed diagnostics left a published bundle")
	}
}

func TestWriterAnonymizesAllBundleEntries(t *testing.T) {
	writer, err := NewWriter(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	path, err := writer.Write(context.Background(), testSource{report: Report{
		Application: map[string]any{"ip": "100.115.10.21", "host": "legengen.top"},
		NetBird:     map[string]any{"peer": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"},
		Logs:        []byte("Authorization: Bearer secret-value\n"),
	}})
	if err != nil {
		t.Fatal(err)
	}
	archive, err := zip.OpenReader(path)
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
		for _, secret := range []string{"100.115.10.21", "legengen.top", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", "Bearer secret-value"} {
			if string(payload) == secret || containsBytes(payload, []byte(secret)) {
				t.Fatalf("entry %s contains secret %q", entry.Name, secret)
			}
		}
	}
}

func containsBytes(payload, needle []byte) bool {
	for index := 0; index+len(needle) <= len(payload); index++ {
		match := true
		for offset := range needle {
			if payload[index+offset] != needle[offset] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
