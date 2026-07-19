package app

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/legengen/sogame-netbird/client/internal/diagnostics"
	clientnetbird "github.com/legengen/sogame-netbird/client/internal/netbird"
	"github.com/legengen/sogame-netbird/client/internal/platform"
	"github.com/legengen/sogame-netbird/client/internal/roomapi"
	"github.com/legengen/sogame-netbird/client/internal/securestore"
	"github.com/legengen/sogame-netbird/client/internal/session"
)

type fakeRoomSession struct {
	createName string
	joinCode   string
	joinName   string
	snapshot   session.Snapshot
	err        error
	view       session.RoomViewSnapshot
	roomCode   string
	calls      []string
	switchReq  session.SwitchRequest
}

type fakeServiceChecker struct {
	inspection platform.ServiceInspection
	err        error
}

func (f fakeServiceChecker) Inspect(context.Context) (platform.ServiceInspection, error) {
	return f.inspection, f.err
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

func (f *fakeRoomSession) View(context.Context) (session.RoomViewSnapshot, error) {
	return f.view, nil
}

func (f *fakeRoomSession) RevealRoomCode(context.Context) (string, error) {
	return f.roomCode, nil
}

func (f *fakeRoomSession) Connect(context.Context) (session.Snapshot, error) {
	f.calls = append(f.calls, "connect")
	return f.snapshot, f.err
}

func (f *fakeRoomSession) Disconnect(context.Context) (session.Snapshot, error) {
	f.calls = append(f.calls, "disconnect")
	return f.snapshot, f.err
}

func (f *fakeRoomSession) Leave(context.Context) (session.Snapshot, error) {
	f.calls = append(f.calls, "leave")
	return f.snapshot, f.err
}

func (f *fakeRoomSession) Switch(_ context.Context, request session.SwitchRequest) (session.Snapshot, error) {
	f.calls = append(f.calls, "switch")
	f.switchReq = request
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

func TestControllerPublishesActiveRoomViewAndExplicitReveal(t *testing.T) {
	rooms := &fakeRoomSession{
		view: session.RoomViewSnapshot{
			Session:         session.Snapshot{Revision: 4, State: session.StateConnectedP2P, Path: clientnetbird.PathP2P},
			Metadata:        testRoomMetadata(),
			RoomCodeMasked:  "****-****-CCCC",
			LocalNetBirdIP:  "100.115.10.21",
			Peers:           []roomapi.Peer{{ID: "peer-2", Name: "friend", NetBirdIP: "100.115.10.22", Connected: true}},
			LastPeerRefresh: time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC),
		},
		roomCode: "AAAA-BBBB-CCCC",
	}
	controller := testController(rooms)
	controller.refreshRoomView(context.Background())

	state := controller.GetState()
	if state.RoomCodeMasked != "****-****-CCCC" || state.LocalNetBirdIP != "100.115.10.21" || state.ConnectedPath != PathP2P {
		t.Fatalf("state=%+v", state)
	}
	if len(state.Peers) != 1 || state.Peers[0].Path != PathP2P || state.Peers[0].NetBirdIP != "100.115.10.22" {
		t.Fatalf("peers=%+v", state.Peers)
	}
	if stateBytes, err := state.MarshalJSON(); err != nil || string(stateBytes) == "" || string(stateBytes) == "AAAA-BBBB-CCCC" {
		t.Fatalf("state JSON=%s error=%v", stateBytes, err)
	}
	result := controller.RevealRoomCode()
	if result.Error != nil || result.RoomCode != "AAAA-BBBB-CCCC" {
		t.Fatalf("reveal=%+v", result)
	}
}

func TestControllerLifecycleCommandsUseStableBusyBoundary(t *testing.T) {
	rooms := &fakeRoomSession{snapshot: session.Snapshot{Revision: 5, State: session.StateNoRoom}}
	controller := testController(rooms)

	if state := controller.ConnectRoom(); state.State != StateNoRoom || state.BusyCommand != "" {
		t.Fatalf("connect state=%+v", state)
	}
	if state := controller.DisconnectRoom(); state.State != StateNoRoom || state.BusyCommand != "" {
		t.Fatalf("disconnect state=%+v", state)
	}
	if state := controller.SwitchRoom(SwitchRoomRequest{Mode: "join", RoomCode: "7X4K-329B-YY95", Confirmed: true}); state.State != StateNoRoom || state.BusyCommand != "" {
		t.Fatalf("switch state=%+v", state)
	}
	if len(rooms.calls) != 3 || rooms.calls[0] != "connect" || rooms.calls[1] != "disconnect" || rooms.calls[2] != "switch" {
		t.Fatalf("calls=%v", rooms.calls)
	}
	if rooms.switchReq.Mode != "join" || rooms.switchReq.RoomCode != "7X4K-329B-YY95" || !rooms.switchReq.Confirmed {
		t.Fatalf("switch request=%+v", rooms.switchReq)
	}
}

func TestControllerPublishesServiceRepairState(t *testing.T) {
	controller := testController(nil)
	controller.ConfigureService(fakeServiceChecker{inspection: platform.ServiceInspection{
		Health:          platform.ServiceMissing,
		ExpectedVersion: "0.74.7",
	}}, func(context.Context) error { return nil })
	controller.refreshService(context.Background())
	state := controller.GetState()
	if state.Error == nil || state.Error.Code != ErrServiceMissing || !state.Service.RepairRequired || state.Service.Installed {
		t.Fatalf("state=%+v", state)
	}
	state = controller.RepairService()
	if state.BusyCommand != "" || state.Error != nil {
		t.Fatalf("repair state=%+v", state)
	}
}

func TestControllerShutdownDoesNotMutateRoomSession(t *testing.T) {
	rooms := &fakeRoomSession{snapshot: session.Snapshot{State: session.StateConnectedP2P, Path: clientnetbird.PathP2P}}
	closed := false
	controller := NewWithSession(slog.New(slog.NewTextHandler(io.Discard, nil)), rooms, func() error {
		closed = true
		return nil
	})
	controller.Shutdown(context.Background())
	if !closed {
		t.Fatal("shutdown did not close the local RPC adapter")
	}
	if len(rooms.calls) != 0 {
		t.Fatalf("shutdown mutated room session: %v", rooms.calls)
	}
}

func TestControllerExportsDiagnosticsToConfiguredLocalWriter(t *testing.T) {
	writer, err := diagnostics.NewWriter(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	controller := testController(nil)
	controller.ConfigureDiagnostics(writer)
	result := controller.ExportDiagnostics()
	if result.Error != nil || result.Path == "" {
		t.Fatalf("result=%+v", result)
	}
	if _, err := os.Stat(result.Path); err != nil {
		t.Fatalf("diagnostic path=%q error=%v", result.Path, err)
	}
}

func testRoomMetadata() securestore.RoomMetadata {
	return securestore.RoomMetadata{
		Version:       securestore.CurrentMetadataVersion,
		RoomID:        "room-1",
		ManagementURL: "https://legengen.top",
		ProfileID:     "profile-1",
		CreatedAt:     time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC),
	}
}
