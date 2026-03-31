package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

// ---------------------------------------------------------------------------
// resolveDMBlocksJSON
// ---------------------------------------------------------------------------

func TestResolveDMBlocksJSON_Empty(t *testing.T) {
	dmSendBlocks = ""
	dmSendBlockFl = ""
	got, err := resolveDMBlocksJSON()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestResolveDMBlocksJSON_InlineString(t *testing.T) {
	dmSendBlocks = `[{"type":"section","text":{"type":"mrkdwn","text":"hi"}}]`
	dmSendBlockFl = ""
	defer func() { dmSendBlocks = "" }()

	got, err := resolveDMBlocksJSON()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != dmSendBlocks {
		t.Errorf("got %q, want %q", got, dmSendBlocks)
	}
}

func TestResolveDMBlocksJSON_FromFile(t *testing.T) {
	dmSendBlocks = ""
	blocks := `[{"type":"section"}]`

	tmp := filepath.Join(t.TempDir(), "blocks.json")
	if err := os.WriteFile(tmp, []byte(blocks), 0600); err != nil {
		t.Fatal(err)
	}
	dmSendBlockFl = tmp
	defer func() { dmSendBlockFl = "" }()

	got, err := resolveDMBlocksJSON()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != blocks {
		t.Errorf("got %q, want %q", got, blocks)
	}
}

func TestResolveDMBlocksJSON_InvalidJSON(t *testing.T) {
	dmSendBlocks = `not-json`
	dmSendBlockFl = ""
	defer func() { dmSendBlocks = "" }()

	_, err := resolveDMBlocksJSON()
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestResolveDMBlocksJSON_MutuallyExclusive(t *testing.T) {
	dmSendBlocks = `[{}]`
	dmSendBlockFl = "some-file"
	defer func() { dmSendBlocks = ""; dmSendBlockFl = "" }()

	_, err := resolveDMBlocksJSON()
	if err == nil {
		t.Fatal("expected error for mutually exclusive flags")
	}
}
