package diagnostics

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/legengen/sogame-netbird/client/internal/observability"
)

const (
	maxReportBytes = 256 << 10
	maxBundleBytes = 2 << 20
)

type Report struct {
	Application any
	NetBird     any
	Logs        []byte
}

type Source interface {
	Collect(context.Context) (Report, error)
}

type Writer struct {
	Directory string
	now       func() time.Time
}

func NewWriter(directory string) (*Writer, error) {
	if directory == "" {
		return nil, errors.New("diagnostic directory is required")
	}
	absolute, err := filepath.Abs(directory)
	if err != nil {
		return nil, errors.New("resolve diagnostic directory")
	}
	return &Writer{Directory: absolute, now: time.Now}, nil
}

func (w *Writer) Write(ctx context.Context, source Source) (string, error) {
	if source == nil {
		return "", errors.New("diagnostic source is required")
	}
	report, err := source.Collect(ctx)
	if err != nil {
		return "", err
	}
	application, err := boundedJSON(report.Application)
	if err != nil {
		return "", fmt.Errorf("encode application diagnostic: %w", err)
	}
	netbird, err := boundedJSON(report.NetBird)
	if err != nil {
		return "", fmt.Errorf("encode NetBird diagnostic: %w", err)
	}
	if len(report.Logs) > maxReportBytes {
		return "", errors.New("diagnostic logs exceed size limit")
	}
	application = []byte(observability.Anonymize(string(application)))
	netbird = []byte(observability.Anonymize(string(netbird)))
	logs := []byte(observability.Anonymize(string(report.Logs)))
	if err := os.MkdirAll(w.Directory, 0o700); err != nil {
		return "", fmt.Errorf("create diagnostic directory: %w", err)
	}
	name := "diagnostic-" + w.now().UTC().Format("20060102-150405.000000000") + ".zip"
	temporary, err := os.CreateTemp(w.Directory, ".diagnostic-*.tmp")
	if err != nil {
		return "", fmt.Errorf("create diagnostic bundle: %w", err)
	}
	temporaryPath := temporary.Name()
	keepTemporary := true
	defer func() {
		_ = temporary.Close()
		if keepTemporary {
			_ = os.Remove(temporaryPath)
		}
	}()
	archive := zip.NewWriter(temporary)
	if err := writeEntry(archive, "application.json", application); err != nil {
		return "", err
	}
	if err := writeEntry(archive, "netbird.json", netbird); err != nil {
		return "", err
	}
	if err := writeEntry(archive, "logs.txt", logs); err != nil {
		return "", err
	}
	if err := writeEntry(archive, "README.txt", []byte("This diagnostic bundle is local-only. No data was uploaded.\n")); err != nil {
		return "", err
	}
	if err := archive.Close(); err != nil {
		return "", fmt.Errorf("close diagnostic bundle: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		return "", fmt.Errorf("flush diagnostic bundle: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return "", fmt.Errorf("close diagnostic bundle: %w", err)
	}
	info, err := os.Stat(temporaryPath)
	if err != nil {
		return "", fmt.Errorf("stat diagnostic bundle: %w", err)
	}
	if info.Size() > maxBundleBytes {
		return "", errors.New("diagnostic bundle exceeds size limit")
	}
	path := filepath.Join(w.Directory, name)
	if err := os.Rename(temporaryPath, path); err != nil {
		return "", fmt.Errorf("publish diagnostic bundle: %w", err)
	}
	keepTemporary = false
	return path, nil
}

func (w *Writer) WriteLocalCrash(payload []byte) (string, error) {
	if len(payload) == 0 {
		return "", errors.New("crash report is empty")
	}
	if len(payload) > maxReportBytes {
		return "", errors.New("crash report exceeds size limit")
	}
	if err := os.MkdirAll(w.Directory, 0o700); err != nil {
		return "", fmt.Errorf("create crash report directory: %w", err)
	}
	name := "crash-" + w.now().UTC().Format("20060102-150405.000000000") + ".log"
	temporary, err := os.CreateTemp(w.Directory, ".crash-*.tmp")
	if err != nil {
		return "", fmt.Errorf("create local crash report: %w", err)
	}
	temporaryPath := temporary.Name()
	keepTemporary := true
	defer func() {
		_ = temporary.Close()
		if keepTemporary {
			_ = os.Remove(temporaryPath)
		}
	}()
	clean := []byte(observability.Anonymize(string(payload)))
	if _, err := temporary.Write(clean); err != nil {
		return "", fmt.Errorf("write local crash report: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		return "", fmt.Errorf("flush local crash report: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return "", fmt.Errorf("close local crash report: %w", err)
	}
	path := filepath.Join(w.Directory, name)
	if err := os.Rename(temporaryPath, path); err != nil {
		return "", fmt.Errorf("publish local crash report: %w", err)
	}
	keepTemporary = false
	return path, nil
}

func boundedJSON(value any) ([]byte, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	if len(payload) > maxReportBytes {
		return nil, errors.New("report exceeds size limit")
	}
	return payload, nil
}

func writeEntry(archive *zip.Writer, name string, payload []byte) error {
	entry, err := archive.Create(name)
	if err != nil {
		return fmt.Errorf("create diagnostic entry %s: %w", name, err)
	}
	if _, err := io.Copy(entry, bytesReader(payload)); err != nil {
		return fmt.Errorf("write diagnostic entry %s: %w", name, err)
	}
	return nil
}

func bytesReader(value []byte) io.Reader { return &byteReader{value: value} }

type byteReader struct {
	value []byte
	index int
}

func (r *byteReader) Read(target []byte) (int, error) {
	if r.index >= len(r.value) {
		return 0, io.EOF
	}
	n := copy(target, r.value[r.index:])
	r.index += n
	return n, nil
}
