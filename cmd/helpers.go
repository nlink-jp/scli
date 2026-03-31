package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/nlink-jp/scli/internal/output"
	"github.com/nlink-jp/scli/internal/slack"
	"github.com/spf13/cobra"
)

// newSlackClient resolves the current workspace token and returns a Slack client
// with the workspace-specific disk cache configured.
func newSlackClient() (*slack.Client, error) {
	if newSlackClientOverride != nil {
		return newSlackClientOverride()
	}
	token, cacheDir, err := resolveTokenAndCacheDir()
	if err != nil {
		return nil, err
	}
	client := slack.NewClient(token)
	client.SetCacheDir(cacheDir)
	return client, nil
}

// resolveTokenAndCacheDir returns the token and the workspace-specific cache
// directory (~/.config/scli/cache/<workspace>/) for the effective workspace.
func resolveTokenAndCacheDir() (token, cacheDir string, err error) {
	mgr, err := newConfigManager()
	if err != nil {
		return "", "", err
	}

	ws := workspace
	if ws == "" {
		cfg, err := mgr.Load()
		if err != nil {
			return "", "", err
		}
		ws = cfg.DefaultWorkspace
		if ws == "" {
			ws = "default"
		}
	}

	ks := newKeychainStore()
	tok, err := mgr.ResolveToken(ws, ks)
	if err != nil {
		return "", "", fmt.Errorf("%w\nRun: scli auth login --workspace %s", err, ws)
	}
	return tok, filepath.Join(mgr.ConfigDir(), "cache", ws), nil
}

// unescapeText converts escape sequences in user-supplied message text.
// Currently handles \n (newline) and \t (tab).
func unescapeText(s string) string {
	s = strings.ReplaceAll(s, `\n`, "\n")
	s = strings.ReplaceAll(s, `\t`, "\t")
	return s
}

// newPrinter returns an output.Printer configured from the global flags.
func newPrinter(cmd *cobra.Command) *output.Printer {
	return output.New(cmd.OutOrStdout(), jsonOutput, noColor)
}

// loadBlocksJSON reads Block Kit JSON from the given flag/file pair and returns
// a valid JSON array string suitable for the Slack API `blocks` parameter.
//
// Accepts both formats:
//   - JSON array: [{"type":"section",...}]
//   - Wrapped object: {"blocks":[{"type":"section",...}]}  (e.g. md-to-slack output)
//
// The wrapped format is automatically unwrapped to extract the array.
func loadBlocksJSON(inline, filePath string) (string, error) {
	if inline != "" && filePath != "" {
		return "", fmt.Errorf("--blocks and --blocks-file are mutually exclusive")
	}

	var raw string

	switch {
	case inline != "":
		raw = inline
	case filePath != "":
		var data []byte
		var err error
		if filePath == "-" {
			data, err = io.ReadAll(os.Stdin)
		} else {
			data, err = os.ReadFile(filePath) //nolint:gosec
		}
		if err != nil {
			return "", fmt.Errorf("read blocks file: %w", err)
		}
		raw = strings.TrimSpace(string(data))
	default:
		return "", nil
	}

	if !json.Valid([]byte(raw)) {
		return "", fmt.Errorf("blocks JSON is invalid")
	}

	// If the input is a {"blocks": [...]} wrapper (e.g. md-to-slack output),
	// extract the blocks array.
	raw = unwrapBlocksObject(raw)

	return raw, nil
}

// unwrapBlocksObject checks if the JSON is an object with a "blocks" key and
// extracts the array. Returns the input unchanged if it is already an array.
func unwrapBlocksObject(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] == '[' {
		return raw
	}

	var wrapper struct {
		Blocks json.RawMessage `json:"blocks"`
	}
	if err := json.Unmarshal([]byte(trimmed), &wrapper); err == nil && wrapper.Blocks != nil {
		return string(wrapper.Blocks)
	}
	return raw
}
