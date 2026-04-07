package slack_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nlink-jp/scli/internal/slack"
)

// ---------------------------------------------------------------------------
// HTTP 429 retry
// ---------------------------------------------------------------------------

func TestRetryOn429_SucceedsAfterRetry(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n < 3 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"ts": "111.000",
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.PostMessage(t.Context(), "C001", "hi")
	if err != nil {
		t.Fatalf("expected success after retry, got: %v", err)
	}
	if calls.Load() != 3 {
		t.Errorf("expected 3 calls (2 x 429 + 1 success), got %d", calls.Load())
	}
}

func TestRetryOn429_ExhaustedReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.PostMessage(t.Context(), "C001", "hi")
	if err == nil {
		t.Fatal("expected error after exhausting retries, got nil")
	}
}

// ---------------------------------------------------------------------------
// PostMessageWithBlocks
// ---------------------------------------------------------------------------

func TestPostMessageWithBlocks(t *testing.T) {
	var gotBlocks string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBlocks = r.URL.Query().Get("blocks")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": true, "ts": "1.0"})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	blocks := `[{"type":"section","text":{"type":"mrkdwn","text":"hello"}}]`
	ts, err := c.PostMessageWithBlocks(t.Context(), "C001", "fallback", blocks)
	if err != nil {
		t.Fatalf("PostMessageWithBlocks: %v", err)
	}
	if ts == "" {
		t.Error("expected non-empty timestamp")
	}
	if gotBlocks != blocks {
		t.Errorf("blocks param: got %q, want %q", gotBlocks, blocks)
	}
}

func TestPostThreadReplyWithBlocks(t *testing.T) {
	var gotThreadTS, gotBlocks string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotThreadTS = r.URL.Query().Get("thread_ts")
		gotBlocks = r.URL.Query().Get("blocks")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": true, "ts": "2.0"})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	blocks := `[{"type":"section"}]`
	_, err := c.PostThreadReplyWithBlocks(t.Context(), "C001", "1.0", "fallback", blocks)
	if err != nil {
		t.Fatalf("PostThreadReplyWithBlocks: %v", err)
	}
	if gotThreadTS != "1.0" {
		t.Errorf("thread_ts: got %q, want %q", gotThreadTS, "1.0")
	}
	if gotBlocks != blocks {
		t.Errorf("blocks: got %q, want %q", gotBlocks, blocks)
	}
}

// ---------------------------------------------------------------------------
// UploadFile thread support
// ---------------------------------------------------------------------------

func TestUploadFile_ThreadTS(t *testing.T) {
	var gotThreadTS string
	var uploadURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/files.getUploadURLExternal":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"ok": true, "upload_url": uploadURL + "/upload", "file_id": "F001",
			})
		case "/upload":
			w.WriteHeader(http.StatusOK)
		case "/files.completeUploadExternal":
			_ = r.ParseForm()
			gotThreadTS = r.FormValue("thread_ts")
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	uploadURL = srv.URL

	// Create a temp file to upload
	tmp := filepath.Join(t.TempDir(), "test.txt")
	if err := os.WriteFile(tmp, []byte("hello"), 0600); err != nil {
		t.Fatal(err)
	}

	c := newTestClient(t, srv)
	err := c.UploadFile(t.Context(), "C001", tmp, "comment", "1234567890.123456")
	if err != nil {
		t.Fatalf("UploadFile: %v", err)
	}
	if gotThreadTS != "1234567890.123456" {
		t.Errorf("thread_ts: got %q, want %q", gotThreadTS, "1234567890.123456")
	}
}

func TestUploadFile_NoThread(t *testing.T) {
	var gotThreadTS string
	var uploadURL2 string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/files.getUploadURLExternal":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"ok": true, "upload_url": uploadURL2 + "/upload", "file_id": "F001",
			})
		case "/upload":
			w.WriteHeader(http.StatusOK)
		case "/files.completeUploadExternal":
			_ = r.ParseForm()
			gotThreadTS = r.FormValue("thread_ts")
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	uploadURL2 = srv.URL

	tmp := filepath.Join(t.TempDir(), "test.txt")
	if err := os.WriteFile(tmp, []byte("hello"), 0600); err != nil {
		t.Fatal(err)
	}

	c := newTestClient(t, srv)
	err := c.UploadFile(t.Context(), "C001", tmp, "", "")
	if err != nil {
		t.Fatalf("UploadFile: %v", err)
	}
	if gotThreadTS != "" {
		t.Errorf("thread_ts should be empty when not specified, got %q", gotThreadTS)
	}
}

// ---------------------------------------------------------------------------
// GetUserProfile
// ---------------------------------------------------------------------------

func TestGetUserProfile(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users.info" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"user": map[string]interface{}{
				"id":       "U001",
				"name":     "alice",
				"tz_label": "Japan Standard Time",
				"profile": map[string]interface{}{
					"display_name": "Alice",
					"real_name":    "Alice Smith",
					"title":        "Engineer",
					"email":        "alice@example.com",
					"phone":        "090-0000-0000",
					"status_text":  "Coding",
					"status_emoji": ":computer:",
				},
			},
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	p, err := c.GetUserProfile(t.Context(), "U001")
	if err != nil {
		t.Fatalf("GetUserProfile: %v", err)
	}
	if p.ID != "U001" {
		t.Errorf("ID: got %q, want %q", p.ID, "U001")
	}
	if p.Title != "Engineer" {
		t.Errorf("Title: got %q, want %q", p.Title, "Engineer")
	}
	if p.Email != "alice@example.com" {
		t.Errorf("Email: got %q, want %q", p.Email, "alice@example.com")
	}
	if p.Status != "Coding" {
		t.Errorf("Status: got %q, want %q", p.Status, "Coding")
	}
	if p.Timezone != "Japan Standard Time" {
		t.Errorf("Timezone: got %q, want %q", p.Timezone, "Japan Standard Time")
	}
}

// ---------------------------------------------------------------------------
// GetChannelDetail
// ---------------------------------------------------------------------------

func TestGetChannelDetail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/conversations.info" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"channel": map[string]interface{}{
				"id":          "C001",
				"name":        "general",
				"is_general":  true,
				"is_private":  false,
				"is_archived": false,
				"num_members": 42,
				"creator":     "U001",
				"created":     1600000000,
				"topic":       map[string]string{"value": "Company news"},
				"purpose":     map[string]string{"value": "All hands"},
			},
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	d, err := c.GetChannelDetail(t.Context(), "C001")
	if err != nil {
		t.Fatalf("GetChannelDetail: %v", err)
	}
	if d.ID != "C001" {
		t.Errorf("ID: got %q, want %q", d.ID, "C001")
	}
	if d.NumMembers != 42 {
		t.Errorf("NumMembers: got %d, want 42", d.NumMembers)
	}
	if d.Topic != "Company news" {
		t.Errorf("Topic: got %q, want %q", d.Topic, "Company news")
	}
	if d.Purpose != "All hands" {
		t.Errorf("Purpose: got %q, want %q", d.Purpose, "All hands")
	}
	if !d.IsGeneral {
		t.Error("expected IsGeneral=true")
	}
}

// ---------------------------------------------------------------------------
// Disk cache integration: ListChannels and ListUsers
// ---------------------------------------------------------------------------

func TestListChannels_UsesDiskCache(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"channels": []map[string]interface{}{
				{"id": "C001", "name": "general", "is_member": true,
					"purpose": map[string]string{"value": "General"}},
			},
			"response_metadata": map[string]string{"next_cursor": ""},
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	c.SetCacheDir(t.TempDir())

	// First call: hits the API and writes cache.
	ch1, err := c.ListChannels(t.Context())
	if err != nil {
		t.Fatalf("first ListChannels: %v", err)
	}
	// Second call: should use the cache, not the API.
	ch2, err := c.ListChannels(t.Context())
	if err != nil {
		t.Fatalf("second ListChannels: %v", err)
	}
	if calls.Load() != 1 {
		t.Errorf("expected 1 API call, got %d", calls.Load())
	}
	if len(ch1) != len(ch2) || ch1[0].ID != ch2[0].ID {
		t.Errorf("cached result differs: %+v vs %+v", ch1, ch2)
	}
}

func TestListUsers_UsesDiskCache(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"members": []map[string]interface{}{
				{"id": "U001", "name": "alice", "deleted": false, "is_bot": false,
					"profile": map[string]string{"display_name": "Alice", "real_name": "Alice Smith"}},
			},
			"response_metadata": map[string]string{"next_cursor": ""},
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	c.SetCacheDir(t.TempDir())

	u1, err := c.ListUsers(t.Context())
	if err != nil {
		t.Fatalf("first ListUsers: %v", err)
	}
	u2, err := c.ListUsers(t.Context())
	if err != nil {
		t.Fatalf("second ListUsers: %v", err)
	}
	if calls.Load() != 1 {
		t.Errorf("expected 1 API call, got %d", calls.Load())
	}
	if len(u1) != len(u2) || u1[0].ID != u2[0].ID {
		t.Errorf("cached result differs: %+v vs %+v", u1, u2)
	}
}

func TestGetUser_UsesInMemoryCache(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"user": map[string]interface{}{
				"id": "U001", "name": "alice", "deleted": false, "is_bot": false,
				"profile": map[string]string{"display_name": "Alice", "real_name": "Alice Smith"},
			},
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	// First call: API.
	if _, err := c.GetUser(t.Context(), "U001"); err != nil {
		t.Fatalf("first GetUser: %v", err)
	}
	// Second call: in-memory cache, no API call.
	if _, err := c.GetUser(t.Context(), "U001"); err != nil {
		t.Fatalf("second GetUser: %v", err)
	}
	if calls.Load() != 1 {
		t.Errorf("expected 1 API call, got %d", calls.Load())
	}
}

func TestGetUser_FallsBackToDiskCache(t *testing.T) {
	// Pre-populate the disk cache with a user list.
	dir := t.TempDir()
	users := []slack.User{{ID: "U999", Name: "bob", DisplayName: "Bob", RealName: "Bob Jones"}}
	slack.SaveCacheForTest(filepath.Join(dir, "users.json"), users)

	// Server that should NOT be called.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("unexpected API call — user should have been found in disk cache")
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	c.SetCacheDir(dir)

	u, err := c.GetUser(t.Context(), "U999")
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}
	if u.Name != "bob" {
		t.Errorf("Name: got %q, want %q", u.Name, "bob")
	}
}

// ---------------------------------------------------------------------------
// retryAfterDuration helper
// ---------------------------------------------------------------------------

func TestRetryAfterDuration_ValidHeader(t *testing.T) {
	got := slack.RetryAfterDurationForTest("5")
	want := 6 * time.Second // 5 + 1 buffer
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestRetryAfterDuration_InvalidHeader(t *testing.T) {
	got := slack.RetryAfterDurationForTest("bad")
	if got != 5*time.Second {
		t.Errorf("got %v, want 5s fallback", got)
	}
}

// ---------------------------------------------------------------------------
// ListJoinedChannels: uses users.conversations, sets IsMember=true
// ---------------------------------------------------------------------------

func TestListJoinedChannels(t *testing.T) {
	var calledPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calledPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"channels": []map[string]interface{}{
				{"id": "C001", "name": "general",
					"purpose": map[string]string{"value": "General discussion"}},
				{"id": "G002", "name": "private-team",
					"purpose": map[string]string{"value": "Team only"}},
			},
			"response_metadata": map[string]string{"next_cursor": ""},
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	channels, err := c.ListJoinedChannels(t.Context())
	if err != nil {
		t.Fatalf("ListJoinedChannels: %v", err)
	}

	// Verify the correct API endpoint was called.
	if calledPath != "/users.conversations" {
		t.Errorf("endpoint: got %q, want %q", calledPath, "/users.conversations")
	}

	if len(channels) != 2 {
		t.Fatalf("expected 2 channels, got %d", len(channels))
	}
	// All returned channels must have IsMember=true.
	for _, ch := range channels {
		if !ch.IsMember {
			t.Errorf("channel %q: IsMember should be true", ch.Name)
		}
	}
}

func TestListJoinedChannels_UsesDiskCache(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"channels": []map[string]interface{}{
				{"id": "C001", "name": "general",
					"purpose": map[string]string{"value": "General discussion"}},
			},
			"response_metadata": map[string]string{"next_cursor": ""},
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	c.SetCacheDir(t.TempDir())

	// First call: hits the API and writes cache.
	ch1, err := c.ListJoinedChannels(t.Context())
	if err != nil {
		t.Fatalf("first ListJoinedChannels: %v", err)
	}
	// Second call: should use the cache, not the API.
	ch2, err := c.ListJoinedChannels(t.Context())
	if err != nil {
		t.Fatalf("second ListJoinedChannels: %v", err)
	}
	if calls.Load() != 1 {
		t.Errorf("expected 1 API call, got %d", calls.Load())
	}
	if len(ch1) != len(ch2) || ch1[0].ID != ch2[0].ID {
		t.Errorf("cached result differs: %+v vs %+v", ch1, ch2)
	}
}
