package slack_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nlink-jp/scli/internal/slack"
)

// newTestClient creates a Client pointing at the given test server.
func newTestClient(t *testing.T, srv *httptest.Server) *slack.Client {
	t.Helper()
	c := slack.NewClient("xoxp-test-token")
	c.SetBaseURL(srv.URL)
	return c
}

func TestListChannels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/conversations.list" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"channels": []map[string]interface{}{
				{"id": "C001", "name": "general", "is_member": true, "purpose": map[string]string{"value": "General"}, "unread_count": 3},
				{"id": "C002", "name": "random", "is_member": true, "purpose": map[string]string{"value": ""}, "unread_count": 0},
				{"id": "C003", "name": "not-member", "is_member": false, "purpose": map[string]string{"value": ""}},
			},
			"response_metadata": map[string]string{"next_cursor": ""},
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	channels, err := c.ListChannels(t.Context())
	if err != nil {
		t.Fatalf("ListChannels: %v", err)
	}
	// All visible channels are returned regardless of membership.
	if len(channels) != 3 {
		t.Errorf("expected 3 channels, got %d", len(channels))
	}
	if channels[0].Name != "general" {
		t.Errorf("expected first channel 'general', got %q", channels[0].Name)
	}
	if channels[0].UnreadCount != 3 {
		t.Errorf("expected unread_count 3, got %d", channels[0].UnreadCount)
	}
	if channels[0].IsMember != true {
		t.Errorf("general: IsMember: got false, want true")
	}
	if channels[2].IsMember != false {
		t.Errorf("not-member: IsMember: got true, want false")
	}
}

func TestGetChannelHistory(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/conversations.history" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"messages": []map[string]interface{}{
				{"ts": "1000.000002", "user": "U1", "text": "second"},
				{"ts": "1000.000001", "user": "U2", "text": "first"},
			},
			"has_more": false,
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	msgs, err := c.GetChannelHistory(t.Context(), "C001", 10, "")
	if err != nil {
		t.Fatalf("GetChannelHistory: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	// Should be reversed to chronological order
	if msgs[0].Text != "first" {
		t.Errorf("expected first message 'first', got %q", msgs[0].Text)
	}
	if msgs[1].Text != "second" {
		t.Errorf("expected second message 'second', got %q", msgs[1].Text)
	}
}

func TestPostMessage(t *testing.T) {
	var gotChannel, gotText string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotChannel = r.URL.Query().Get("channel")
		gotText = r.URL.Query().Get("text")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"ts": "1234567890.000001",
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	ts, err := c.PostMessage(t.Context(), "C001", "hello world")
	if err != nil {
		t.Fatalf("PostMessage: %v", err)
	}
	if ts == "" {
		t.Error("expected non-empty timestamp")
	}
	if gotChannel != "C001" {
		t.Errorf("channel: got %q, want %q", gotChannel, "C001")
	}
	if gotText != "hello world" {
		t.Errorf("text: got %q, want %q", gotText, "hello world")
	}
}

func TestAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":    false,
			"error": "channel_not_found",
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.GetChannelHistory(t.Context(), "C999", 10, "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestResolveChannelID_ByID(t *testing.T) {
	c := slack.NewClient("token")
	// Channel IDs starting with C should be returned as-is without an API call
	id, err := c.ResolveChannelID(t.Context(), "C123ABC")
	if err != nil {
		t.Fatalf("ResolveChannelID: %v", err)
	}
	if id != "C123ABC" {
		t.Errorf("got %q, want %q", id, "C123ABC")
	}
}

func TestListDMs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/conversations.list" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"channels": []map[string]interface{}{
				{"id": "D001", "user": "U001"},
				{"id": "D002", "user": "U002"},
			},
			"response_metadata": map[string]string{"next_cursor": ""},
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	dms, err := c.ListDMs(t.Context())
	if err != nil {
		t.Fatalf("ListDMs: %v", err)
	}
	if len(dms) != 2 {
		t.Fatalf("expected 2 DMs, got %d", len(dms))
	}
	if dms[0].ID != "D001" || dms[0].UserID != "U001" {
		t.Errorf("unexpected first DM: %+v", dms[0])
	}
	if !dms[0].IsIM {
		t.Error("expected IsIM=true")
	}
}

func TestOpenDM(t *testing.T) {
	var gotUsers string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		gotUsers = r.FormValue("users")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":      true,
			"channel": map[string]string{"id": "D001"},
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	id, err := c.OpenDM(t.Context(), "U001")
	if err != nil {
		t.Fatalf("OpenDM: %v", err)
	}
	if id != "D001" {
		t.Errorf("got %q, want %q", id, "D001")
	}
	if gotUsers != "U001" {
		t.Errorf("users param: got %q, want %q", gotUsers, "U001")
	}
}

func TestResolveChannelID_ByName(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"channels": []map[string]interface{}{
				{"id": "C001", "name": "general", "is_member": true, "purpose": map[string]string{"value": ""}},
			},
			"response_metadata": map[string]string{"next_cursor": ""},
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	id, err := c.ResolveChannelID(t.Context(), "#general")
	if err != nil {
		t.Fatalf("ResolveChannelID: %v", err)
	}
	if id != "C001" {
		t.Errorf("got %q, want %q", id, "C001")
	}
}
