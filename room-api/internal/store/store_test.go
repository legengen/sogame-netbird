package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestStorePersistsRoomsAndOperations(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rooms.db")
	ctx := context.Background()

	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if err := db.MustHealthy(ctx); err != nil {
		t.Fatalf("MustHealthy() error = %v", err)
	}
	created := time.Now().UTC().Truncate(time.Microsecond)
	room := Room{ID: "room-1", CodeHash: []byte("hash"), CodeCiphertext: []byte("code"), Status: "creating", CreatedAt: created}
	if err := db.CreateRoom(ctx, room); err != nil {
		t.Fatalf("CreateRoom() error = %v", err)
	}
	if ok, err := db.BeginOperation(ctx, "request-1", room.ID); err != nil || !ok {
		t.Fatalf("BeginOperation() = %v, %v", ok, err)
	}
	if err := db.UpdateExternalIDs(ctx, room.ID, "group-1", "key-1", "policy-1", []byte("encrypted-key")); err != nil {
		t.Fatalf("UpdateExternalIDs() error = %v", err)
	}
	if err := db.SetStatus(ctx, room.ID, "active", ""); err != nil {
		t.Fatalf("SetStatus() error = %v", err)
	}
	if err := db.SaveOperation(ctx, "request-1", []byte("response"), "active"); err != nil {
		t.Fatalf("SaveOperation() error = %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	db, err = Open(path)
	if err != nil {
		t.Fatalf("reopen error = %v", err)
	}
	defer db.Close()
	loaded, err := db.GetRoom(ctx, room.ID)
	if err != nil {
		t.Fatalf("GetRoom() error = %v", err)
	}
	if loaded.Status != "active" || loaded.GroupID != "group-1" || loaded.SetupKeyID != "key-1" || loaded.PolicyID != "policy-1" {
		t.Fatalf("loaded room = %+v", loaded)
	}
	op, err := db.GetOperation(ctx, "request-1")
	if err != nil {
		t.Fatalf("GetOperation() error = %v", err)
	}
	if op.Status != "active" || string(op.Response) != "response" {
		t.Fatalf("loaded operation = %+v", op)
	}
}

func TestStoreRejectsDuplicateCodeHash(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "rooms.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()
	room := Room{ID: "room-1", CodeHash: []byte("same"), CodeCiphertext: []byte("code"), Status: "creating", CreatedAt: time.Now()}
	if err := db.CreateRoom(context.Background(), room); err != nil {
		t.Fatalf("first CreateRoom() error = %v", err)
	}
	room.ID = "room-2"
	if err := db.CreateRoom(context.Background(), room); err == nil {
		t.Fatal("second CreateRoom() unexpectedly succeeded")
	}
}
