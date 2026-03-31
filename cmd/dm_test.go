package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

// ---------------------------------------------------------------------------
// loadBlocksJSON (shared by post and dm send)
// ---------------------------------------------------------------------------

func TestLoadBlocksJSON_Empty(t *testing.T) {
	got, err := loadBlocksJSON("", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestLoadBlocksJSON_InlineArray(t *testing.T) {
	blocks := `[{"type":"section","text":{"type":"mrkdwn","text":"hi"}}]`
	got, err := loadBlocksJSON(blocks, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != blocks {
		t.Errorf("got %q, want %q", got, blocks)
	}
}

func TestLoadBlocksJSON_FromFile(t *testing.T) {
	blocks := `[{"type":"section"}]`
	tmp := filepath.Join(t.TempDir(), "blocks.json")
	if err := os.WriteFile(tmp, []byte(blocks), 0600); err != nil {
		t.Fatal(err)
	}

	got, err := loadBlocksJSON("", tmp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != blocks {
		t.Errorf("got %q, want %q", got, blocks)
	}
}

func TestLoadBlocksJSON_InvalidJSON(t *testing.T) {
	_, err := loadBlocksJSON("not-json", "")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestLoadBlocksJSON_MutuallyExclusive(t *testing.T) {
	_, err := loadBlocksJSON(`[{}]`, "some-file")
	if err == nil {
		t.Fatal("expected error for mutually exclusive flags")
	}
}

func TestLoadBlocksJSON_UnwrapObject(t *testing.T) {
	// md-to-slack outputs {"blocks": [...]}, should be unwrapped to just the array.
	wrapped := `{"blocks":[{"type":"header","text":{"type":"plain_text","text":"Title"}}]}`
	want := `[{"type":"header","text":{"type":"plain_text","text":"Title"}}]`

	got, err := loadBlocksJSON(wrapped, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestLoadBlocksJSON_UnwrapFromFile(t *testing.T) {
	wrapped := `{"blocks": [{"type":"section"}]}`
	want := `[{"type":"section"}]`

	tmp := filepath.Join(t.TempDir(), "md.json")
	if err := os.WriteFile(tmp, []byte(wrapped), 0600); err != nil {
		t.Fatal(err)
	}

	got, err := loadBlocksJSON("", tmp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestLoadBlocksJSON_ArrayPassedThrough(t *testing.T) {
	// Already an array — should not be modified.
	array := `[{"type":"section"}]`
	got, err := loadBlocksJSON(array, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != array {
		t.Errorf("got %q, want %q", got, array)
	}
}

func TestLoadBlocksJSON_UnwrapNull(t *testing.T) {
	// {"blocks": null} should be treated as no blocks (return the raw input)
	got, err := loadBlocksJSON(`{"blocks": null}`, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != `{"blocks": null}` {
		t.Errorf("got %q, want original input", got)
	}
}

func TestLoadBlocksJSON_EmptyArray(t *testing.T) {
	got, err := loadBlocksJSON(`[]`, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != `[]` {
		t.Errorf("got %q, want %q", got, `[]`)
	}
}
