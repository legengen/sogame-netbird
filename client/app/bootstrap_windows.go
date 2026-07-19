//go:build windows && amd64

package app

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os/user"
	"time"

	clientnetbird "github.com/legengen/sogame-netbird/client/internal/netbird"
	"github.com/legengen/sogame-netbird/client/internal/roomapi"
	"github.com/legengen/sogame-netbird/client/internal/securestore"
	"github.com/legengen/sogame-netbird/client/internal/session"
)

const roomAPIBaseURL = "https://legengen.top"

func NewWindowsController(logger *slog.Logger) *Controller {
	controller := New(logger)
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
	return NewWithSession(logger, session.NewService(rooms, adapter, metadata, codes), rpcAdapter.Close)
}

func controllerWithStartupError(controller *Controller, err error) *Controller {
	controller.state.State = StateRecoverableError
	controller.state.Error = publicError(err)
	controller.state.Service.RepairRequired = true
	return controller
}
