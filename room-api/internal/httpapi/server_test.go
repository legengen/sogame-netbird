package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/legengen/sogame-netbird/room-api/internal/netbird"
	"github.com/legengen/sogame-netbird/room-api/internal/rooms"
	"github.com/legengen/sogame-netbird/room-api/internal/store"
)

type httpFakeNetBird struct{}

func (httpFakeNetBird) ListGroups(context.Context) ([]netbird.Group, error) { return nil, nil }
func (httpFakeNetBird) CreateGroup(context.Context, string) (netbird.Group, error) {
	return netbird.Group{ID: "group-1"}, nil
}
func (httpFakeNetBird) DeleteGroup(context.Context, string) error                 { return nil }
func (httpFakeNetBird) ListSetupKeys(context.Context) ([]netbird.SetupKey, error) { return nil, nil }
func (httpFakeNetBird) CreateSetupKey(context.Context, string, string) (netbird.SetupKeyClear, error) {
	return netbird.SetupKeyClear{SetupKey: netbird.SetupKey{ID: "key-1"}, Key: "clear-key"}, nil
}
func (httpFakeNetBird) RevokeSetupKey(context.Context, string, []string) error { return nil }
func (httpFakeNetBird) DeleteSetupKey(context.Context, string) error           { return nil }
func (httpFakeNetBird) ListPolicies(context.Context) ([]netbird.Policy, error) { return nil, nil }
func (httpFakeNetBird) CreateRoomPolicy(context.Context, string, string) (netbird.Policy, error) {
	return netbird.Policy{ID: "policy-1"}, nil
}
func (httpFakeNetBird) DeletePolicy(context.Context, string) error          { return nil }
func (httpFakeNetBird) DisablePolicy(context.Context, netbird.Policy) error { return nil }
func (httpFakeNetBird) ListPeers(context.Context) ([]netbird.Peer, error)   { return nil, nil }

func newTestServer(t *testing.T, maxBody int64) *Server {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "rooms.db"))
	if err != nil {
		t.Fatalf("store.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	service := rooms.New(db, httpFakeNetBird{}, rooms.Config{ManagementURL: "https://legengen.top", EncryptionKey: []byte("01234567890123456789012345678901")})
	return New(service, Config{AdminToken: "admin-token", MaxBodyBytes: maxBody, CreateRatePerMinute: 5, JoinRatePerMinute: 5, PeerRatePerMinute: 5, ProvisionConcurrency: 1})
}

func TestPublicRoomRoutesAndAdminBoundary(t *testing.T) {
	server := newTestServer(t, 1024)

	health := httptest.NewRecorder()
	server.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if health.Code != http.StatusOK {
		t.Fatalf("health status = %d", health.Code)
	}
	roomHealth := httptest.NewRecorder()
	server.ServeHTTP(roomHealth, httptest.NewRequest(http.MethodGet, "/rooms/healthz", nil))
	if roomHealth.Code != http.StatusOK {
		t.Fatalf("room health status = %d", roomHealth.Code)
	}

	create := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/rooms", nil)
	request.Header.Set("Idempotency-Key", "http-request-1")
	server.ServeHTTP(create, request)
	if create.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", create.Code, create.Body.String())
	}
	var room struct {
		RoomCode string `json:"room_code"`
	}
	if err := json.Unmarshal(create.Body.Bytes(), &room); err != nil || room.RoomCode == "" {
		t.Fatalf("create response = %s, error = %v", create.Body.String(), err)
	}

	join := httptest.NewRecorder()
	server.ServeHTTP(join, httptest.NewRequest(http.MethodPost, "/rooms/join", bytes.NewBufferString(`{"room_code":"`+room.RoomCode+`"}`)))
	if join.Code != http.StatusOK {
		t.Fatalf("join status = %d, body = %s", join.Code, join.Body.String())
	}
	invalid := httptest.NewRecorder()
	server.ServeHTTP(invalid, httptest.NewRequest(http.MethodPost, "/rooms/join", bytes.NewBufferString(`{"room_code":"AAAA-BBBB-CCCC"}`)))
	if invalid.Code != http.StatusNotFound || !strings.Contains(invalid.Body.String(), `"room unavailable"`) {
		t.Fatalf("invalid join = %d, body = %q", invalid.Code, invalid.Body.String())
	}

	unauthorized := httptest.NewRecorder()
	server.ServeHTTP(unauthorized, httptest.NewRequest(http.MethodPost, "/rooms/"+room.RoomCode+"/disable", nil))
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized disable status = %d", unauthorized.Code)
	}
}

func TestCreateRejectsOversizedBody(t *testing.T) {
	server := newTestServer(t, 16)
	record := httptest.NewRecorder()
	server.ServeHTTP(record, httptest.NewRequest(http.MethodPost, "/rooms", bytes.NewBufferString("12345678901234567890")))
	if record.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized create status = %d", record.Code)
	}
}
