package netbird

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientUsesPATAndTypedRequests(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Token test-pat" {
			t.Fatalf("Authorization = %q", got)
		}
		switch r.URL.Path {
		case "/api/groups":
			if r.Method == http.MethodGet {
				_ = json.NewEncoder(w).Encode([]Group{{ID: "group-1", Name: "room-test"}})
				return
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode group request: %v", err)
			}
			if body["name"] != "room-new" {
				t.Fatalf("group request = %#v", body)
			}
			_ = json.NewEncoder(w).Encode(Group{ID: "group-2", Name: "room-new"})
		case "/api/setup-keys":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode setup-key request: %v", err)
			}
			if body["type"] != "reusable" || body["expires_in"] != float64(0) || body["usage_limit"] != float64(0) {
				t.Fatalf("setup-key request = %#v", body)
			}
			_ = json.NewEncoder(w).Encode(SetupKeyClear{SetupKey: SetupKey{ID: "key-1", Name: "room-new"}, Key: "clear-key"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := New(server.URL, "test-pat")
	group, found, err := client.FindGroupByName(context.Background(), "room-test")
	if err != nil || !found || group.ID != "group-1" {
		t.Fatalf("FindGroupByName() = %+v, %v, %v", group, found, err)
	}
	created, err := client.CreateGroup(context.Background(), "room-new")
	if err != nil || created.ID != "group-2" {
		t.Fatalf("CreateGroup() = %+v, %v", created, err)
	}
	key, err := client.CreateSetupKey(context.Background(), "room-new", "group-2")
	if err != nil || key.Key != "clear-key" {
		t.Fatalf("CreateSetupKey() = %+v, %v", key, err)
	}
}
