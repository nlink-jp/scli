package output_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/nlink-jp/scli/internal/output"
	"github.com/nlink-jp/scli/internal/slack"
)

func newTestPrinter(buf *bytes.Buffer) *output.Printer {
	// noColor=true so output is deterministic in tests
	return output.New(buf, false, true)
}

func TestChannels_Text(t *testing.T) {
	buf := new(bytes.Buffer)
	p := newTestPrinter(buf)

	channels := []slack.Channel{
		{ID: "C1", Name: "general", Purpose: "General discussion"},
		{ID: "C2", Name: "random", Purpose: ""},
	}
	if err := p.Channels(channels, ""); err != nil {
		t.Fatalf("Channels: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "#general") {
		t.Errorf("expected '#general' in output, got: %q", out)
	}
	if !strings.Contains(out, "General discussion") {
		t.Errorf("expected purpose in output, got: %q", out)
	}
}

func TestChannels_JSON(t *testing.T) {
	buf := new(bytes.Buffer)
	p := output.New(buf, true, true)

	channels := []slack.Channel{
		{ID: "C1", Name: "general"},
	}
	if err := p.Channels(channels, ""); err != nil {
		t.Fatalf("Channels: %v", err)
	}

	var got []map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON output: %v\noutput: %s", err, buf.String())
	}
	if len(got) != 1 {
		t.Errorf("expected 1 channel in JSON, got %d", len(got))
	}
}

func TestMessages_Text(t *testing.T) {
	buf := new(bytes.Buffer)
	p := newTestPrinter(buf)

	msgs := []slack.Message{
		{Timestamp: "1000000000.000001", UserID: "U1", UserName: "alice", Text: "hello"},
		{Timestamp: "1000000001.000001", UserID: "U2", UserName: "bob", Text: "world"},
	}
	if err := p.Messages(msgs); err != nil {
		t.Fatalf("Messages: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "alice") {
		t.Errorf("expected 'alice' in output, got: %q", out)
	}
	if !strings.Contains(out, "hello") {
		t.Errorf("expected 'hello' in output, got: %q", out)
	}
}

func TestMessages_FallbackToUserID(t *testing.T) {
	buf := new(bytes.Buffer)
	p := newTestPrinter(buf)

	msgs := []slack.Message{
		{Timestamp: "1000000000.000001", UserID: "U999", UserName: "", Text: "hi"},
	}
	if err := p.Messages(msgs); err != nil {
		t.Fatalf("Messages: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "U999") {
		t.Errorf("expected fallback user ID 'U999' in output, got: %q", out)
	}
}

func TestPostResult_JSON(t *testing.T) {
	var buf bytes.Buffer
	p := output.New(&buf, true, false) // jsonOut=true
	err := p.PostResult("1234567890.123456", "C001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var result map[string]string
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result["ts"] != "1234567890.123456" {
		t.Errorf("ts: got %q, want %q", result["ts"], "1234567890.123456")
	}
	if result["channel"] != "C001" {
		t.Errorf("channel: got %q, want %q", result["channel"], "C001")
	}
}

func TestPostResult_Text(t *testing.T) {
	var buf bytes.Buffer
	p := output.New(&buf, false, false) // jsonOut=false
	_ = p.PostResult("1234567890.123456", "C001")
	got := buf.String()
	if !strings.Contains(got, "1234567890.123456") {
		t.Errorf("expected ts in output, got: %q", got)
	}
}

func TestFormatTimestamp(t *testing.T) {
	buf := new(bytes.Buffer)
	p := newTestPrinter(buf)

	msgs := []slack.Message{
		// Unix timestamp 0 = 1970-01-01 00:00 UTC
		{Timestamp: "0.000001", UserID: "U1", UserName: "u", Text: "t"},
	}
	if err := p.Messages(msgs); err != nil {
		t.Fatalf("Messages: %v", err)
	}
	// Just verify a date-like pattern is present (exact value depends on timezone)
	out := buf.String()
	if !strings.Contains(out, "1970") {
		t.Errorf("expected year '1970' in timestamp output, got: %q", out)
	}
}
