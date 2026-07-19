package app

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	clientnetbird "github.com/legengen/sogame-netbird/client/internal/netbird"
	"github.com/legengen/sogame-netbird/client/internal/roomapi"
	"github.com/legengen/sogame-netbird/client/internal/session"
)

type fakeRoomSession struct {
	createName string
	joinCode   string
	joinName   string
	snapshot   session.Snapshot
	err        error
}

func (f *fakeRoomSession) Create(_ context.Context, displayName string) (session.Snapshot, error) {
	f.createName = displayName
	return f.snapshot, f.err
}

func (f *fakeRoomSession) Join(_ context.Context, roomCode, displayName string) (session.Snapshot, error) {
	f.joinCode = roomCode
	f.joinName = displayName
	return f.snapshot, f.err
}

func testController(rooms RoomSession) *Controller {
	return NewWithSession(slog.New(slog.NewTextHandler(io.Discard, nil)), rooms, nil)
}

func TestCreateRoomRunsSessionWorkflowAndPublishesState(t *testing.T) {
	rooms := &fakeRoomSession{snapshot: session.Snapshot{Revision: 3, State: session.StateWaitingForPeer, Path: clientnetbird.PathNone}}
	controller := testController(rooms)

	state := controller.CreateRoom(CreateRoomRequest{})
	if rooms.createName == "" {
		t.Fatal("default device name was not supplied")
	}
	if state.State != StateWaitingForPeer || state.Revision != 3 || state.BusyCommand != "" || state.Error != nil {
		t.Fatalf("state=%+v", state)
	}
}

func TestJoinRoomPassesNormalizedBoundaryRequest(t *testing.T) {
	rooms := &fakeRoomSession{snapshot: session.Snapshot{Revision: 2, State: session.StateConnectingPeer}}
	controller := testController(rooms)

	state := controller.JoinRoom(JoinRoomRequest{RoomCode: "7X4K-329B-YY95", DisplayName: "gaming-pc"})
	if rooms.joinCode != "7X4K-329B-YY95" || rooms.joinName != "gaming-pc" {
		t.Fatalf("join code=%q name=%q", rooms.joinCode, rooms.joinName)
	}
	if state.State != StateConnectingPeer || state.BusyCommand != "" || state.Error != nil {
		t.Fatalf("state=%+v", state)
	}
}

func TestRoomWorkflowMapsTypedErrorsWithoutUpstreamDetails(t *testing.T) {
	rooms := &fakeRoomSession{err: &roomapi.HTTPError{StatusCode: 404, Code: roomapi.ErrorRoomUnavailable}}
	state := testController(rooms).JoinRoom(JoinRoomRequest{RoomCode: "7X4K-329B-YY95", DisplayName: "gaming-pc"})
	if state.State != StateRecoverableError || state.Error == nil || state.Error.Code != ErrRoomUnavailable {
		t.Fatalf("state=%+v", state)
	}
	if errors.Is(state.Error, rooms.err) || state.Error.Message == rooms.err.Error() {
		t.Fatalf("public error exposed upstream detail: %+v", state.Error)
	}
}

func TestRoomWorkflowWithoutRuntimeReportsServiceUnavailable(t *testing.T) {
	state := testController(nil).CreateRoom(CreateRoomRequest{DisplayName: "gaming-pc"})
	if state.Error == nil || state.Error.Code != ErrServiceUnavailable || !state.Error.Retryable {
		t.Fatalf("state=%+v", state)
	}
}
