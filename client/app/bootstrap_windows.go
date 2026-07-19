//go:build windows && amd64

package app

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"time"

	releasebuild "github.com/legengen/sogame-netbird/client/build"
	"github.com/legengen/sogame-netbird/client/internal/diagnostics"
	clientnetbird "github.com/legengen/sogame-netbird/client/internal/netbird"
	"github.com/legengen/sogame-netbird/client/internal/platform"
	"github.com/legengen/sogame-netbird/client/internal/roomapi"
	"github.com/legengen/sogame-netbird/client/internal/securestore"
	"github.com/legengen/sogame-netbird/client/internal/session"
)

const roomAPIBaseURL = "https://legengen.top"

func NewWindowsController(logger *slog.Logger) *Controller {
	controller := New(logger)
	release, releaseErr := releasebuild.Load()
	if releaseErr == nil {
		controller.ConfigureService(
			platform.NewServiceInspector(clientnetbird.ExpectedVersion, release.WindowsX64.Install.ProductCode, nil),
			windowsRepairFunc(release),
		)
	}
	rooms, err := roomapi.NewClient(roomAPIBaseURL, &http.Client{})
	if err != nil {
		return controllerWithStartupError(controller, err)
	}
	metadataPath, err := securestore.DefaultMetadataPath()
	if err != nil {
		return controllerWithStartupError(controller, err)
	}
	metadata, err := securestore.NewMetadataStore(metadataPath)
	if err != nil {
		return controllerWithStartupError(controller, err)
	}
	if diagnosticWriter, writerErr := diagnostics.NewWriter(filepath.Join(filepath.Dir(metadataPath), "diagnostics")); writerErr == nil {
		controller.ConfigureDiagnostics(diagnosticWriter)
	}
	roomCodePath, err := securestore.DefaultRoomCodePath()
	if err != nil {
		return controllerWithStartupError(controller, err)
	}
	codes, err := securestore.NewRoomCodeStore(roomCodePath)
	if err != nil {
		return controllerWithStartupError(controller, err)
	}
	currentUser, err := user.Current()
	if err != nil {
		return controllerWithStartupError(controller, errors.New("resolve current Windows user"))
	}
	dialContext, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	rpcAdapter, err := clientnetbird.DialLocalRPCAdapter(dialContext, currentUser.Username)
	if err != nil {
		return controllerWithStartupError(controller, err)
	}
	adapter := clientnetbird.EnforceExactVersion(rpcAdapter, clientnetbird.ExpectedVersion)
	controller = NewWithSession(logger, session.NewService(rooms, adapter, metadata, codes), rpcAdapter.Close)
	if releaseErr == nil {
		controller.ConfigureService(
			platform.NewServiceInspector(clientnetbird.ExpectedVersion, release.WindowsX64.Install.ProductCode, adapter),
			windowsRepairFunc(release),
		)
	}
	return controller
}

func windowsRepairFunc(release releasebuild.Metadata) func(context.Context) error {
	return func(ctx context.Context) error {
		executable, err := os.Executable()
		if err != nil {
			return errors.New("resolve Sogame installation directory")
		}
		root := filepath.Dir(executable)
		logRoot, err := securestore.DefaultMetadataPath()
		if err != nil {
			return err
		}
		logPath := filepath.Join(filepath.Dir(logRoot), "netbird-repair.log")
		return platform.RequestInstallerElevation(
			filepath.Join(root, "sogame-helper.exe"),
			platform.MSIRepair,
			filepath.Join(root, release.WindowsX64.Artifact),
			logPath,
		)
	}
}

func controllerWithStartupError(controller *Controller, err error) *Controller {
	controller.state.State = StateRecoverableError
	controller.state.Error = publicError(err)
	controller.state.Service.RepairRequired = true
	return controller
}
