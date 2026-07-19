package rooms

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	roomcrypto "github.com/legengen/sogame-netbird/room-api/internal/crypto"
	"github.com/legengen/sogame-netbird/room-api/internal/netbird"
	"github.com/legengen/sogame-netbird/room-api/internal/store"
)

type fakeNetBird struct {
	groups        []netbird.Group
	peers         []netbird.Peer
	policies      []netbird.Policy
	deletedGroup  string
	deletedKey    string
	deletedPolicy string
	setupErr      bool
}

func (f *fakeNetBird) ListGroups(context.Context) ([]netbird.Group, error) { return f.groups, nil }
func (f *fakeNetBird) CreateGroup(_ context.Context, name string) (netbird.Group, error) {
	group := netbird.Group{ID: "group-1", Name: name}
	f.groups = append(f.groups, group)
	return group, nil
}
func (f *fakeNetBird) DeleteGroup(_ context.Context, id string) error {
	f.deletedGroup = id
	return nil
}
func (f *fakeNetBird) ListSetupKeys(context.Context) ([]netbird.SetupKey, error) { return nil, nil }
func (f *fakeNetBird) CreateSetupKey(_ context.Context, name, _ string) (netbird.SetupKeyClear, error) {
	if f.setupErr {
		return netbird.SetupKeyClear{}, errors.New("setup key unavailable")
	}
	return netbird.SetupKeyClear{SetupKey: netbird.SetupKey{ID: "key-1", Name: name}, Key: "clear-setup-key"}, nil
}
func (f *fakeNetBird) RevokeSetupKey(_ context.Context, id string, _ []string) error {
	f.deletedKey = id
	return nil
}
func (f *fakeNetBird) DeleteSetupKey(context.Context, string) error           { return nil }
func (f *fakeNetBird) ListPolicies(context.Context) ([]netbird.Policy, error) { return f.policies, nil }
func (f *fakeNetBird) CreateRoomPolicy(_ context.Context, name, _ string) (netbird.Policy, error) {
	policy := netbird.Policy{ID: "policy-1", Name: name, Enabled: true}
	f.policies = append(f.policies, policy)
	return policy, nil
}
func (f *fakeNetBird) DeletePolicy(_ context.Context, id string) error {
	f.deletedPolicy = id
	return nil
}
func (f *fakeNetBird) DisablePolicy(context.Context, netbird.Policy) error { return nil }
func (f *fakeNetBird) ListPeers(context.Context) ([]netbird.Peer, error)   { return f.peers, nil }

func TestCreateJoinPeersAndDisable(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "rooms.db"))
	if err != nil {
		t.Fatalf("store.Open() error = %v", err)
	}
	defer db.Close()
	fake := &fakeNetBird{peers: []netbird.Peer{
		{ID: "peer-in", Name: "inside", IP: "100.64.0.1", Connected: true, Groups: []netbird.GroupMinimum{{ID: "group-1"}}},
		{ID: "peer-out", Name: "outside", IP: "100.64.0.2", Groups: []netbird.GroupMinimum{{ID: "other"}}},
	}}
	service := New(db, fake, Config{ManagementURL: "https://legengen.top", EncryptionKey: []byte("01234567890123456789012345678901")})

	created, err := service.Create(context.Background(), "request-1")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.RoomCode == "" || created.SetupKey != "clear-setup-key" {
		t.Fatalf("Create() response = %+v", created)
	}
	retried, err := service.Create(context.Background(), "request-1")
	if err != nil || retried.RoomID != created.RoomID || retried.SetupKey != created.SetupKey {
		t.Fatalf("idempotent Create() = %+v, %v", retried, err)
	}

	joined, err := service.Join(context.Background(), created.RoomCode)
	if err != nil || joined.SetupKey != created.SetupKey || joined.RoomCode != "" {
		t.Fatalf("Join() = %+v, %v", joined, err)
	}
	peers, err := service.Peers(context.Background(), created.RoomCode)
	if err != nil || len(peers.Peers) != 1 || peers.Peers[0].ID != "peer-in" {
		t.Fatalf("Peers() = %+v, %v", peers, err)
	}
	if err := service.Disable(context.Background(), created.RoomCode); err != nil {
		t.Fatalf("Disable() error = %v", err)
	}
	if _, err := service.Join(context.Background(), created.RoomCode); err != ErrInvalidRoom {
		t.Fatalf("Join() after disable error = %v", err)
	}
	if fake.deletedKey != "key-1" || fake.deletedPolicy != "policy-1" {
		t.Fatalf("disable cleanup = key %q, policy %q", fake.deletedKey, fake.deletedPolicy)
	}
}

func TestReconcileInterruptedOperation(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "rooms.db"))
	if err != nil {
		t.Fatalf("store.Open() error = %v", err)
	}
	defer db.Close()
	fake := &fakeNetBird{}
	ctx := context.Background()
	if err := db.CreateRoom(ctx, store.Room{ID: "room-1", CodeHash: roomcrypto.Hash("code"), CodeCiphertext: []byte("cipher"), GroupID: "group-1", SetupKeyID: "key-1", PolicyID: "policy-1", Status: "creating"}); err != nil {
		t.Fatalf("CreateRoom() error = %v", err)
	}
	if err := db.UpdateExternalIDs(ctx, "room-1", "group-1", "key-1", "policy-1", []byte("encrypted-key")); err != nil {
		t.Fatalf("UpdateExternalIDs() error = %v", err)
	}
	if ok, err := db.BeginOperation(ctx, "request-1", "room-1"); err != nil || !ok {
		t.Fatalf("BeginOperation() = %v, %v", ok, err)
	}
	service := New(db, fake, Config{EncryptionKey: []byte("01234567890123456789012345678901")})
	if err := service.Reconcile(ctx); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	room, err := db.GetRoom(ctx, "room-1")
	if err != nil || room.Status != "error" {
		t.Fatalf("reconciled room = %+v, %v", room, err)
	}
	op, err := db.GetOperation(ctx, "request-1")
	if err != nil || op.Status != "error" {
		t.Fatalf("reconciled operation = %+v, %v", op, err)
	}
	if fake.deletedGroup != "group-1" || fake.deletedKey != "key-1" || fake.deletedPolicy != "policy-1" {
		t.Fatalf("reconcile cleanup = group %q, key %q, policy %q", fake.deletedGroup, fake.deletedKey, fake.deletedPolicy)
	}
}

func TestCreateCompensatesWhenSetupKeyFails(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "rooms.db"))
	if err != nil {
		t.Fatalf("store.Open() error = %v", err)
	}
	defer db.Close()
	fake := &fakeNetBird{setupErr: true}
	service := New(db, fake, Config{ManagementURL: "https://legengen.top", EncryptionKey: []byte("01234567890123456789012345678901")})
	if _, err := service.Create(context.Background(), ""); err == nil {
		t.Fatal("Create() unexpectedly succeeded")
	}
	if fake.deletedGroup != "group-1" {
		t.Fatalf("compensation deleted group = %q", fake.deletedGroup)
	}
	rooms, err := db.ListRoomsByStatus(context.Background(), "error")
	if err != nil || len(rooms) != 1 {
		t.Fatalf("error rooms = %+v, %v", rooms, err)
	}
}
